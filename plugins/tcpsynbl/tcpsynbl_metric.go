package tcpsynbl

import (
	"fmt"
	"kubebpfbox/internal/metric"
	"strconv"
	"time"
)

type Event struct {
	PodName       string
	NodeName      string
	Namespace     string
	PodUID        string
	ContainerID   string
	ContainerName string
	Pid           uint32
	Comm          string
	MaxAckBacklog uint32
	AckBacklog    uint32
}

// String return tcpsynbl metric string
func (m *Event) String() string {
	return fmt.Sprintf("%s PID %d(%s), TCP SYN backlog queue full %d / %d === %s %s %s %s\n",
		time.Now().Format("2006-01-02 15:04:05"), m.Pid, m.Comm, m.AckBacklog, m.MaxAckBacklog,
		m.NodeName, m.Namespace, m.PodName, m.ContainerName,
	)
}

// CovertMetric convert tcpsynbl to metric
func (m *Event) CovertMetric() metric.Metric {
	var metric metric.Metric
	metric.Measurement = "tcpsynbl"
	metric.AddTags("poduid", m.PodUID)
	metric.AddTags("podname", m.PodName)
	metric.AddTags("nodename", m.NodeName)
	metric.AddTags("namespace", m.Namespace)
	metric.AddTags("containerid", m.ContainerID)
	metric.AddTags("containername", m.ContainerName)
	metric.AddTags("pid", strconv.Itoa(int(m.Pid)))
	metric.AddTags("comm", m.Comm)
	metric.AddField("maxackbacklog", strconv.Itoa(int(m.MaxAckBacklog)))
	metric.AddField("ackbacklog", strconv.Itoa(int(m.AckBacklog)))
	return metric
}
