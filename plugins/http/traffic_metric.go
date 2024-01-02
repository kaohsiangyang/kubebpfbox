package http

import (
	"fmt"
	"kubebpfbox/pkg/metric"
	"strconv"
	"time"
)

type Traffic struct {
	//HTTP, RPC, MySQL etc.
	Type uint32
	// 1: ingress, 2: egress
	Flow      int
	DstIP     string
	DstPort   uint16
	SrcIP     string
	SrcPort   uint16
	Duration  uint32
	Method    string
	URL       string
	Code      string
	PodName   string
	NodeName  string
	NameSpace string
}

func (m *Traffic) String() string {
	return fmt.Sprintf("%s %d %d [%s][%d] --> [%s][%d][%s %s] ====> %s [%dms] \n%s %s %s\n",
		time.Now().Format("2006-01-02 15:04:05"), m.Type, m.Flow, m.SrcIP, m.SrcPort, m.DstIP, m.DstPort, m.Method, m.URL, m.Code, m.Duration,
		m.NodeName, m.NameSpace, m.PodName,
	)
}

func (m *Traffic) CovertMetric() metric.Metric {
	var metric metric.Metric
	metric.Measurement = "traffic"
	metric.AddTags("podname", m.PodName)
	metric.AddTags("nodename", m.NodeName)
	metric.AddTags("namespace", m.NameSpace)
	metric.AddTags("type", strconv.Itoa(int(m.Type)))
	metric.AddTags("flow", strconv.Itoa(m.Flow))
	metric.AddTags("dstip", m.DstIP)
	metric.AddTags("dstport", strconv.Itoa(int(m.DstPort)))
	metric.AddTags("srcip", m.SrcIP)
	metric.AddTags("srcport", strconv.Itoa(int(m.SrcPort)))
	metric.AddTags("method", m.Method)
	metric.AddTags("url", m.URL)
	metric.AddTags("code", m.Code)
	metric.AddField("duration", m.Duration)
	return metric
}
