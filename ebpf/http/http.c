//go:build ignore
#include <linux/bpf.h>
#include "../include/bpf_helpers.h"
// #include "../include/common.h"
#include "../include/bpf_endian.h"
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <linux/ip.h>
// #include <linux/in.h>
#include <linux/string.h>
#include <linux/tcp.h>
#include <linux/types.h>
#include <netinet/in.h>

char __license[] SEC("license") = "Dual MIT/GPL";

#define MAX_MAP_CONN 1024 * 16
#define MAX_MAP_FILTER_IP 300

struct key 
{
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
};

enum packet_type
{
    T_HTTP = 1,
    T_RPC = 2,
    T_MYSQL = 3,
};

enum http_request_method
{
    M_GET = 1,
    M_POST = 2,
    M_PUT = 3,
    M_DELETE = 4,
    M_HEAD = 5,
    M_PATCH = 6,
};

enum packet_phase
{
    P_REQUEST = 1,
    P_RESPONSE = 2,
};

struct packet
{
    __u32 type;
    __u32 method;
    __u32 dst_ip;
	__u16 dst_port;
	__u32 src_ip;
	__u16 src_port;
	__u32 duration;
	// char req_payload[180];
	// char rsp_payload[180];
    __u32 req_payload_size;
    __u32 rsp_payload_size;
	__u8 status[3];
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_MAP_FILTER_IP);
    __type(key, __u32);
    __type(value, __u32);
} filter_ip SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_MAP_CONN);
    __type(key, struct key);
    __type(value, struct packet);
} request_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} packets SEC(".maps");

// Force emitting struct event into the ELF.
const struct packet *unused __attribute__((unused));


int __is_http_request(char p[13])
{
	// GET
	if ((p[0] == 'G') && (p[1] == 'E') && (p[2] == 'T'))
	{
		return M_GET;
	}
	// POST
	if ((p[0] == 'P') && (p[1] == 'O') && (p[2] == 'S') && (p[3] == 'T'))
	{
		return M_POST;
	}
	// PUT
	if ((p[0] == 'P') && (p[1] == 'U') && (p[2] == 'T'))
	{
		return M_PUT;
	}
	// DELETE
	if ((p[0] == 'D') && (p[1] == 'E') && (p[2] == 'L') && (p[3] == 'E') && (p[4] == 'T') && (p[5] == 'E'))
	{
		return M_DELETE;
	}
	// HEAD
	if ((p[0] == 'H') && (p[1] == 'E') && (p[2] == 'A') && (p[3] == 'D'))
	{
		return M_HEAD;
	}
	// PATCH
	if ((p[0] == 'P') && (p[1] == 'A') && (p[2] == 'T') && (p[3] == 'C') && (p[4] == 'H'))
	{
		return M_PATCH;
	}
	return 0;
}

int __is_http_response(char p[13])
{
	// HTTP
	if ((p[0] == 'H') && (p[1] == 'T') && (p[2] == 'T') && (p[3] == 'P'))
	{
		return 1;
	}
	return 0;
}

int __check_target_ip(__u32 ip)
{
    __u32 *result = bpf_map_lookup_elem(&filter_ip, &ip);
    if(result == 0)
    {
        return 1;
    }
    else
    {
        return 0;
    }
}

SEC("socket")
int socket__filter_http(struct __sk_buff *skb)
{
	// Skip non-IP packets
	__u16 eth_proto;
	bpf_skb_load_bytes(skb, offsetof(struct ethhdr, h_proto), &eth_proto, sizeof(eth_proto));
	if(ntohs(eth_proto) != ETH_P_IP)
		return -1;

	// Skip non-TCP packets
	__u8 ip_proto;
	bpf_skb_load_bytes(skb, ETH_HLEN + offsetof(struct iphdr, protocol), &ip_proto, sizeof(ip_proto));
	if(ip_proto != IPPROTO_TCP)
		return -1;

	struct packet p = {};
	__u32 poffset = 0;

    // Get IP packet header
	struct iphdr iph;
	bpf_skb_load_bytes(skb, ETH_HLEN, &iph, sizeof(iph));

    // Get TCP packet header
	struct tcphdr tcph;
	bpf_skb_load_bytes(skb, ETH_HLEN + sizeof(iph), &tcph, sizeof(tcph));

    // Get length of header
	__u32 tcp_hlen = tcph.doff;
	__u32 ip_hlen = iph.ihl;
	ip_hlen = ip_hlen << 2;
	tcp_hlen = tcp_hlen << 2;

    if(!__check_target_ip(iph.saddr) && !__check_target_ip(iph.daddr)){
        return -1;
    }

	poffset = ETH_HLEN + ip_hlen + tcp_hlen;

    // Get the first 12 bytes of the TCP packet body
	char pre_char[13];
	bpf_skb_load_bytes(skb, poffset, pre_char, 13);

	p.src_ip = iph.saddr;
	p.dst_ip = iph.daddr;
	p.src_port = tcph.source;
	p.dst_port = tcph.dest;

    struct key k = {};

    int result = __is_http_request(pre_char);
    // Store the HTTP request package in request_map and wait for matching.
	if (result)
	{
		p.type = T_HTTP;
        p.method = result;

		p.duration = bpf_ktime_get_ns();

        /* NOTE:IP data packets may be transmitted in fragments,
            so the calculated payload size of the current data packet
            may not be equal to the HTTP message length.
        */ 
        p.req_payload_size = ntohs(iph.tot_len) - ip_hlen - tcp_hlen;

        k.src_ip = iph.saddr;
	    k.dst_ip = iph.daddr;
	    k.src_port = tcph.source;
	    k.dst_port = tcph.dest;

		bpf_map_update_elem(&request_map, &k, &p, BPF_ANY);
	}
    // Match HTTP requests and responses, sending packets to user space
	else if (__is_http_response(pre_char))
	{
        k.src_ip = iph.daddr;
	    k.dst_ip = iph.saddr;
	    k.src_port = tcph.dest;
	    k.dst_port = tcph.source;
        struct packet *req;
		req = bpf_map_lookup_elem(&request_map, &k);
		if (!req)
		{
			return -1;
		}
        struct packet *matric;
        matric = bpf_ringbuf_reserve(&packets, sizeof(struct packet), 0);
        if (!matric){
            return -1;
        }
        matric->type = req->type;
        matric->method = req->method;
        matric->src_ip = req->src_ip;
        matric->dst_ip = req->dst_ip;
        matric->src_port = req->src_port;
        matric->dst_port = req->dst_port;
		matric->duration = bpf_ktime_get_ns() - req->duration;
		matric->req_payload_size = req->req_payload_size;
        matric->rsp_payload_size = ntohs(iph.tot_len) - ip_hlen - tcp_hlen;
        matric->status[0] = pre_char[9];
        matric->status[1] = pre_char[10];
        matric->status[2] = pre_char[11];

		bpf_ringbuf_submit(matric, 0);
        bpf_map_delete_elem(&request_map, &k);
	}
	else
	{
		return -1;
	}
    return 0;
}