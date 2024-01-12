package oomkill

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -type event -target amd64 oomkill ../../ebpf/oomkill/oomkill.c

import (
	"bytes"
	"encoding/binary"
	"errors"
	"kubebpfbox/global"
	"kubebpfbox/internal/pid2pod"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type OomkillEbpf struct {
	stopper chan struct{}
	objs    oomkillObjects
	kp      link.Link
	ch      chan Event
}

func NewOomkillEbpf(ch chan Event) *OomkillEbpf {
	return &OomkillEbpf{
		stopper: make(chan struct{}),
		ch:      ch,
	}
}

// Load loads the oomkill eBPF program into the kernel.
func (o *OomkillEbpf) Load() (err error) {
	// Remove the memlock limit.
	if err = rlimit.RemoveMemlock(); err != nil {
		global.Logger.Fatalf("Removing memlock: %v", err)
		return errors.New("removing memlock")
	}

	// Load the oomkill eBPF program into the kernel.
	o.objs = oomkillObjects{}
	if err = loadOomkillObjects(&o.objs, nil); err != nil {
		var verr *ebpf.VerifierError
		if errors.As(err, &verr) {
			global.Logger.Fatalf("%+v\n", verr)
		}
		global.Logger.Fatalf("Loading objects failed: %v", err)
		return errors.New("loading objects failed")
	}

	// Attach the oomkill eBPF program to the kernel.
	if o.kp, err = link.Kprobe("oom_kill_process", o.objs.KprobeOomKillProcess, nil); err != nil {
		global.Logger.Fatalf("Attaching kprobe failed: %v", err)
		return errors.New("attaching kprobe failed")
	}

	return nil
}

// Start starts the oomkill eBPF program.
func (o *OomkillEbpf) Start() error {
	// Create a ring buffer reader for the packets map.
	rd, err := ringbuf.NewReader(o.objs.Events)
	if err != nil {
		global.Logger.Fatalf("opening ringbuf reader error: %s", err)
		return errors.New("opening ringbuf reader error:" + err.Error())
	}
	defer rd.Close()

	// Close the reader when unload, which will exit the read loop.
	go func() {
		<-o.stopper
		o.Unload()
		if err := rd.Close(); err != nil {
			global.Logger.Errorf("closing ringbuf reader: %s", err)
		}
	}()

	// oomkillEvent is generated by bpf2go.
	var event oomkillEvent
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				o.stopper <- struct{}{}
				global.Logger.Info("Exiting..")
				return errors.New("exiting")
			}
			global.Logger.Errorf("Reading from reader failed: %s", err)
			continue
		}

		// Parse the ringbuf event entry into a oomkillEventData structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			global.Logger.Errorf("Parsing ringbuf event failed: %s", err)
			continue
		}

		// Decode oomkillEvent to oomkill metric
		if oomkill, err := DecodeMapItem(&event); err != nil {
			global.Logger.Errorf("Decode map item error: %v", err)
		} else {
			global.Logger.Infof("Get oomkill metric: %s", oomkill.String())
			o.ch <- *oomkill
		}
	}
}

// Unload unloads the oomkill eBPF program from the kernel.
func (o *OomkillEbpf) Unload() error {
	if err := o.kp.Close(); err != nil {
		global.Logger.Errorf("Detach BPF program failed: %v", err)
		return errors.New("detach BPF program failed")
	}
	if err := o.objs.Close(); err != nil {
		global.Logger.Errorf("Closing objects failed: %v", err)
		return errors.New("closing objects failed")
	}
	return nil
}

// DecodeMapItem decode oomkillEvent form ringbuf to oomkill metric
func DecodeMapItem(event *oomkillEvent) (*Event, error) {
	e := new(Event)
	e.TriggerPid = event.Fpid
	e.TriggerComm = string(event.Fcomm[:bytes.IndexByte(event.Fcomm[:], 0)])
	if e.TriggerComm == "" {
		e.TriggerComm = "unknown"
	}
	e.Pid = event.Tpid
	e.Comm = string(event.Tcomm[:bytes.IndexByte(event.Tcomm[:], 0)])
	if e.Comm == "" {
		e.Comm = "unknown"
	}
	e.Pages = event.Pages
	e.Message = string(event.Message[:bytes.IndexByte(event.Message[:], 0)])
	if e.Message == "" {
		e.Message = "Out of memory"
	}

	if podInfo, err := pid2pod.GetPid2Pod().GetPodInfoByPid(e.Pid); err != nil {
		global.Logger.Errorf("Get pod info error: %v", err)
	} else {
		e.Namespace = podInfo.Namespace
		e.NodeName = podInfo.NodeName
		e.PodName = podInfo.PodName
		e.PodUID = podInfo.PodUID
		e.ContainerID = podInfo.ContainerID
		e.ContainerName = podInfo.ContainerName
	}
	return e, nil
}
