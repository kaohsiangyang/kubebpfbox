// go:build ignore
#include "../include/vmlinux.h"
#include "../include/bpf_helpers.h"
#include "../include/bpf_tracing.h"
#include "../include/bpf_endian.h"

char __license[] SEC("license") = "Dual MIT/GPL";

#define TASK_COMM_LEN 16

struct event
{
    __u32 pid;
    __u32 max_ack_backlog;
    __u32 ack_backlog;
};

// RingBuf output tcp syn backlog full event
struct
{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} events SEC(".maps");

// Force emitting struct event into the ELF.
const struct event *unused __attribute__((unused));

// NOTE: bpf_get_current_pid_tgid() cannot obtain the pid of the current listening sock.
// sock->socket->file->f_owner->pid is the owner pid of the current listening sock.
// But f_owner struct is used for sending signals to the process that initiated an operation like fcntl(F_SETOWN).
// So, we can use f_owner->pid to get the pid of the current listening sock while the socket has been set F_SETOWN.
SEC("kprobe/tcp_v4_syn_recv_sock")
int do_entry(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);

    struct event backlog = {};

    if (bpf_probe_read_kernel(&backlog.max_ack_backlog, sizeof(backlog.max_ack_backlog), &sk->sk_max_ack_backlog) != 0)
    {
        return 0;
    }
    if (bpf_probe_read_kernel(&backlog.ack_backlog, sizeof(backlog.ack_backlog), &sk->sk_ack_backlog) != 0)
    {
        return 0;
    }
    if (backlog.max_ack_backlog != 0 && backlog.ack_backlog >= backlog.max_ack_backlog)
    {
        struct socket *sk_socket;
        if (bpf_probe_read_kernel(&sk_socket, sizeof(sk_socket), &sk->sk_socket) != 0)
        {
            return 0;
        }
        struct file *file;
        if (bpf_probe_read_kernel(&file, sizeof(file), &sk_socket->file) != 0)
        {
            return 0;
        }
        struct fown_struct owner;
        if (bpf_probe_read_kernel(&owner, sizeof(owner), &file->f_owner) != 0)
        {
            return 0;
        }
        if (owner.pid)
        {
            struct pid *pid_struct;
            if (bpf_probe_read_kernel(&pid_struct, sizeof(pid_struct), &owner.pid) != 0)
            {
                return 0;
            }
            int pid_nr;
            if (bpf_probe_read_kernel(&pid_nr, sizeof(pid_nr), &pid_struct->numbers[0].nr) != 0)
            {
                return 0;
            }
            backlog.pid = pid_nr;
            bpf_ringbuf_output(&events, &backlog, sizeof(backlog), 0);
        }
    }
    return 0;
};

SEC("kprobe/tcp_v6_syn_recv_sock")
int do_entry_v6(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);

    struct event backlog = {};

    if (bpf_probe_read_kernel(&backlog.max_ack_backlog, sizeof(backlog.max_ack_backlog), &sk->sk_max_ack_backlog) != 0)
    {
        return 0;
    }
    if (bpf_probe_read_kernel(&backlog.ack_backlog, sizeof(backlog.ack_backlog), &sk->sk_ack_backlog) != 0)
    {
        return 0;
    }
    if (backlog.max_ack_backlog != 0 && backlog.ack_backlog >= backlog.max_ack_backlog)
    {
        struct socket *sk_socket;
        if (bpf_probe_read_kernel(&sk_socket, sizeof(sk_socket), &sk->sk_socket) != 0)
        {
            return 0;
        }
        struct file *file;
        if (bpf_probe_read_kernel(&file, sizeof(file), &sk_socket->file) != 0)
        {
            return 0;
        }
        struct fown_struct owner;
        if (bpf_probe_read_kernel(&owner, sizeof(owner), &file->f_owner) != 0)
        {
            return 0;
        }
        if (owner.pid)
        {
            struct pid *pid_struct;
            if (bpf_probe_read_kernel(&pid_struct, sizeof(pid_struct), &owner.pid) != 0)
            {
                return 0;
            }
            int pid_nr;
            if (bpf_probe_read_kernel(&pid_nr, sizeof(pid_nr), &pid_struct->numbers[0].nr) != 0)
            {
                return 0;
            }
            backlog.pid = pid_nr;
            bpf_ringbuf_output(&events, &backlog, sizeof(backlog), 0);
        }
    }
    return 0;
};