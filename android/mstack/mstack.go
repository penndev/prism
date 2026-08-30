//go:build linux

package mstack

import (
	"errors"
	"net"
	"sync"
	"syscall"

	"github.com/penndev/prism/stack"
	"github.com/penndev/prism/transport"
	"github.com/penndev/prism/transport/dialer"
	"golang.org/x/sys/unix"
	gvisorstack "gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	mu         sync.Mutex
	ep         gvisorstack.LinkEndpoint
	tunFD      int
	started    bool
	origTCP    dialer.Dialer
	origUDP    dialer.Dialer
	protector  Protector
	logger     Logger
	errProtect = errors.New("protect failed")
	errStarted = errors.New("already started")
)

// Protector wraps VpnService.protect so local dials skip the TUN.
type Protector interface {
	Protect(fd int32) bool
}

// Logger reports each forwarded TCP/UDP request.
type Logger interface {
	OnConnect(network, address string)
}

// Start attaches gVisor to a TUN fd and forwards TCP/UDP with transport.Local.
// mtu <= 0 uses 1500. Android must pass VpnService.protect or outbound sockets loop back into the VPN.
func Start(fd int32, mtu int32, p Protector, log Logger) error {
	mu.Lock()
	defer mu.Unlock()
	if started {
		return errStarted
	}
	if mtu <= 0 {
		mtu = 1500
	}

	endpoint, err := createTUN(int(fd), uint32(mtu))
	if err != nil {
		return err
	}

	installProtect(p)
	handle := transport.Local()
	stack.New(stack.Option{
		EndPoint: endpoint,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			relay(handle, log, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			relay(handle, log, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})

	ep = endpoint
	tunFD = int(fd)
	protector = p
	logger = log
	started = true
	return nil
}

// Stop closes the TUN fd so gVisor dispatch exits.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if !started {
		return
	}
	if tunFD >= 0 {
		_ = unix.Close(tunFD)
		tunFD = -1
	}
	if ep != nil {
		ep.Wait()
		ep = nil
	}
	restoreProtect()
	protector = nil
	logger = nil
	started = false
}

func relay(handle transport.HandleConnect, log Logger, conn net.Conn, network, address string) {
	defer func() { recover() }()
	if conn == nil {
		return
	}
	if log != nil {
		log.OnConnect(network, address)
	}
	if err := handle(conn, network, address); err != nil {
		if log != nil {
			log.OnConnect(network, address+" "+err.Error())
		}
		_ = conn.Close()
	}
}

func installProtect(p Protector) {
	origTCP = dialer.TCPDialer
	origUDP = dialer.UDPDialer
	if p == nil {
		return
	}
	d := &net.Dialer{Control: protectControl(p)}
	dialer.TCPDialer = d
	dialer.UDPDialer = d
}

func restoreProtect() {
	if origTCP != nil {
		dialer.TCPDialer = origTCP
		origTCP = nil
	}
	if origUDP != nil {
		dialer.UDPDialer = origUDP
		origUDP = nil
	}
}

func protectControl(p Protector) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var fail error
		if err := c.Control(func(fd uintptr) {
			if p == nil || !p.Protect(int32(fd)) {
				fail = errProtect
			}
		}); err != nil {
			return err
		}
		return fail
	}
}
