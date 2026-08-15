package tun

import "net/netip"

type Options struct {
	Name string
	MTU  int
	// 	macOS (Darwin) 和其他 BSD 系统在处理 utun 设备时，底层内核协议栈强制要求一个 4 字节的“元数据头”。
	// 这 4 个字节被称为 Packet Information (PI) 或 Family Header。
	Offset int
	IP     netip.Prefix
}
