package main

import (
	"flag"
	"fmt"
	"kubebpfbox/global"
	"kubebpfbox/internal/ip2pod"
	"kubebpfbox/internal/k8s"
	"kubebpfbox/internal/metric"
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
	ip2pod.GetIP2Pod().Registry()
	go k8s.GetPodController().Run()
	return nil
}

func main() {
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	if err := setupK8s(); err != nil {
		global.Logger.Fatalf("Init setupK8s failed: %v", err)
	}

	ch := make(chan metric.Metric, 1000)
	for _, plugin := range plugin.Plugins {
		global.Logger.Infof("Gather %s metrics", plugin.Name())
		go func() {
			if err := plugin.Gather(ch); err != nil {
				global.Logger.Errorf("Gather %s metrics failed: %v", plugin.Name(), err)
			}
		}()
	}

	for {
		select {
		case m := <-ch:
			fmt.Printf("Get metric: %s", m.String())
		case <-stopper:
			fmt.Print("Get stop signal, exit\n")
			return
		}
	}
}
