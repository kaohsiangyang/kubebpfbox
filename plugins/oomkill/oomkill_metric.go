package oomkill

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
	TriggerPid    uint32
	TriggerComm   string
	Message       string
	Pages         uint64
}

// String return oomkill metric string
func (m *Event) String() string {
	return fmt.Sprintf("%s Triggered by PID %d (%s), OOM kill of PID %d (%s), %d pages, message: %s === %s %s %s %s\n",
		time.Now().Format("2006-01-02 15:04:05"), m.TriggerPid, m.TriggerComm, m.Pid, m.Comm, m.Pages, m.Message,
		m.NodeName, m.Namespace, m.PodName, m.ContainerName,
	)
}

// CovertMetric convert oomkill to metric
func (m *Event) CovertMetric() metric.Metric {
	var metric metric.Metric
	metric.Measurement = "oomkill"
	metric.AddTags("poduid", m.PodUID)
	metric.AddTags("podname", m.PodName)
	metric.AddTags("nodename", m.NodeName)
	metric.AddTags("namespace", m.Namespace)
	metric.AddTags("containerid", m.ContainerID)
	metric.AddTags("containername", m.ContainerName)
	metric.AddTags("pid", strconv.Itoa(int(m.Pid)))
	metric.AddTags("comm", m.Comm)
	metric.AddTags("triggerpid", strconv.Itoa(int(m.TriggerPid)))
	metric.AddTags("triggercomm", m.TriggerComm)
	metric.AddTags("message", m.Message)
	metric.AddField("pages", m.Pages)
	return metric
}
