package main

import (
	"flag"
	"fmt"
	"kubebpfbox/global"
	"kubebpfbox/internal/endpoint2pod"
	"kubebpfbox/internal/influxdb"
	"kubebpfbox/internal/k8s"
	"kubebpfbox/internal/metric"
	"kubebpfbox/internal/pid2pod"
	"kubebpfbox/internal/plugin"
	"kubebpfbox/pkg/logger"
	"kubebpfbox/pkg/setting"
	_ "kubebpfbox/plugins"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	config  string
	Storage string
	db      *influxdb.Influxdb
)

func init() {
	if err := setupFlag(); err != nil {
		log.Fatalf("Init setupFlag failed: %v", err)
	}
	if err := setupSetting(); err != nil {
		log.Fatalf("Init setupSetting failed: %v", err)
	}
	if err := setupLogger(); err != nil {
		log.Fatalf("Init setupLogger failed: %v", err)
	}
}

func setupFlag() error {
	flag.StringVar(&config, "config", "configs/", "Specify the configuration file path to use")
	flag.StringVar(&Storage, "storage", "storage/", "Specify the storage directory to use")
	flag.Parse()
	return nil
}

func setupSetting() error {
	s, err := setting.NewSetting(strings.Split(config, ",")...)
	if err != nil {
		return err
	}

	err = s.ReadSection("Cluster", &global.ClusterSetting)
	if err != nil {
		return err
	}

	err = s.ReadSection("App", &global.AppSetting)
	if err != nil {
		return err
	}

	err = s.ReadSection("Database", &global.DatabaseSetting)
	if err != nil {
		return err
	}

	return nil
}

func setupLogger() error {
	fileName := Storage + global.AppSetting.LogSavePath + "/" + global.AppSetting.LogFileName + global.AppSetting.LogFileExt
	global.Logger = logger.NewLogger(&lumberjack.Logger{
		Filename:  fileName,
		MaxSize:   500,
		MaxAge:    10,
		LocalTime: true,
	}, "", log.LstdFlags).WithCaller(2)

	return nil
}

func setupK8s() error {
	endpoint2pod.GetEndpoint2Pod().Registry()
	go k8s.GetServiceController().Run()
	pid2pod.GetPid2Pod().Registry()
	go k8s.GetPodController().Run()
	return nil
}

func setupInfluxdb() error {
	influxAddr := os.Getenv("INFLUX_ADDR")
	influxOrg := os.Getenv("INFLUX_ORG")
	influxBucket := os.Getenv("INFLUX_BUCKET")
	influxToken := os.Getenv("INFLUX_TOKEN")
	if influxAddr == "" || influxOrg == "" || influxBucket == "" || influxToken == "" {
		return fmt.Errorf("influxdb config error")
	}
	db = influxdb.NewInfluxdb(influxAddr, influxOrg, influxBucket, influxToken).Run()
	return nil
}

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	if err := setupInfluxdb(); err != nil {
		global.Logger.Fatalf("Init setupInfluxdb failed: %v", err)
	}

	if err := setupK8s(); err != nil {
		global.Logger.Fatalf("Init setupK8s failed: %v", err)
	}

	ch := make(chan metric.Metric, 1000)
	global.Logger.Infof("Start gather metrics: %d", len(plugin.Plugins))
	for _, p := range plugin.Plugins {
		global.Logger.Infof("Gather %s metrics", p.Name())
		go func(p plugin.Plugin) {
			if err := p.Gather(ch); err != nil {
				global.Logger.Errorf("Gather %s metrics failed: %v", p.Name(), err)
			}
		}(p)
	}

	for {
		select {
		case m := <-ch:
			db.Write(m)
			log.Printf("Get metric: %s", m.String())
		case <-stopper:
			log.Print("Get stop signal, exit\n")
			return
		}
	}
}
