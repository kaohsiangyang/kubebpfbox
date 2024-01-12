// go:build ignore
#include "../include/vmlinux.h"
#include "../include/bpf_tracing.h"
#include "../include/bpf_helpers.h"

char __license[] SEC("license") = "Dual MIT/GPL";

#define TASK_COMM_LEN 16

struct event
{
    __u32 fpid;
    __u32 tpid;
    __u64 pages;
    __u8 fcomm[TASK_COMM_LEN];
    __u8 tcomm[TASK_COMM_LEN];
    __u8 message[128];
};

// RingBuf output oom kill event
struct
{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

// Force emitting struct event into the ELF.
const struct event *unused __attribute__((unused));

SEC("kprobe/oom_kill_process")
int kprobe__oom_kill_process(struct pt_regs *ctx)
{
    struct oom_control *oc = (struct oom_control *)PT_REGS_PARM1(ctx);
    __u8 *message = (__u8 *)PT_REGS_PARM2(ctx);
    struct task_struct *p;
    if (bpf_probe_read_kernel(&p, sizeof(p), &oc->chosen) != 0)
    {
        return 0;
    }
    // if(!p) {
    //     return 0;
    // }

    struct event oomkill_info = {};

    oomkill_info.fpid = bpf_get_current_pid_tgid() >> 32;
    bpf_get_current_comm(&oomkill_info.fcomm, TASK_COMM_LEN);

    if (bpf_probe_read_kernel(&oomkill_info.tpid, sizeof(oomkill_info.tpid), &p->tgid) != 0)
    {
        return 0;
    }
    if (bpf_probe_read_kernel(&oomkill_info.tcomm, sizeof(oomkill_info.tcomm), p->comm) != 0)
    {
        return 0;
    }
    if (bpf_probe_read_kernel(&oomkill_info.pages, sizeof(oomkill_info.pages), &oc->totalpages) != 0)
    {
        return 0;
    }
    if (bpf_probe_read_kernel_str(oomkill_info.message, sizeof(oomkill_info.message), message) < 0)
    {
        return 0;
    }

    bpf_ringbuf_output(&events, &oomkill_info, sizeof(oomkill_info), 0);

    return 0;
}