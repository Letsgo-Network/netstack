package fdbased

import (
	"fmt"
	"syscall"

	"github.com/Letsgo-Network/netstack/tcpip"
	"github.com/Letsgo-Network/netstack/tcpip/buffer"
	"github.com/Letsgo-Network/netstack/tcpip/header"
	"github.com/Letsgo-Network/netstack/tcpip/link/rawfile"
	"github.com/Letsgo-Network/netstack/tcpip/stack"
)

// New creates a new fd-based endpoint.
//
// Makes fd non-blocking, but does not take ownership of fd, which must remain
// open for the lifetime of the returned endpoint.
func New(opts *Options) (tcpip.LinkEndpointID, error) {
	if err := syscall.SetNonblock(opts.FD, true); err != nil {
		return 0, fmt.Errorf("syscall.SetNonblock(%v) failed: %v", opts.FD, err)
	}

	caps := stack.LinkEndpointCapabilities(0)
	if opts.RXChecksumOffload {
		caps |= stack.CapabilityRXChecksumOffload
	}

	if opts.TXChecksumOffload {
		caps |= stack.CapabilityTXChecksumOffload
	}

	hdrSize := 0
	if opts.EthernetHeader {
		hdrSize = header.EthernetMinimumSize
		caps |= stack.CapabilityResolutionRequired
	}

	if opts.SaveRestore {
		caps |= stack.CapabilitySaveRestore
	}

	if opts.DisconnectOk {
		caps |= stack.CapabilityDisconnectOk
	}

	e := &endpoint{
		fd:                 opts.FD,
		mtu:                opts.MTU,
		caps:               caps,
		closed:             opts.ClosedFunc,
		addr:               opts.Address,
		hdrSize:            hdrSize,
		packetDispatchMode: opts.PacketDispatchMode,
	}

	// For non-socket FDs we read one packet a time (e.g. TAP devices).
	msgsPerRecv := 1
	e.inboundDispatcher = e.dispatch

	isSocket, err := isSocketFD(opts.FD)
	if err != nil {
		return 0, err
	}
	if isSocket {
		if opts.GSOMaxSize != 0 {
			e.caps |= stack.CapabilityGSO
			e.gsoMaxSize = opts.GSOMaxSize
		}

		switch e.packetDispatchMode {
		case PacketMMap:
			if err := e.setupPacketRXRing(); err != nil {
				return 0, fmt.Errorf("e.setupPacketRXRing failed: %v", err)
			}
			e.inboundDispatcher = e.packetMMapDispatch
			return stack.RegisterLinkEndpoint(e), nil

		case RecvMMsg:
			// If the provided FD is a socket then we optimize
			// packet reads by using recvmmsg() instead of read() to
			// read packets in a batch.
			e.inboundDispatcher = e.recvMMsgDispatch
			msgsPerRecv = MaxMsgsPerRecv
		}
	}

	e.views = make([][]buffer.View, msgsPerRecv)
	for i := range e.views {
		e.views[i] = make([]buffer.View, len(BufConfig))
	}
	e.iovecs = make([][]syscall.Iovec, msgsPerRecv)
	iovLen := len(BufConfig)
	if e.Capabilities()&stack.CapabilityGSO != 0 {
		// virtioNetHdr is prepended before each packet.
		iovLen++
	}
	for i := range e.iovecs {
		e.iovecs[i] = make([]syscall.Iovec, iovLen)
	}
	e.msgHdrs = make([]rawfile.MMsgHdr, msgsPerRecv)
	for i := range e.msgHdrs {
		e.msgHdrs[i].Msg.Iov = &e.iovecs[i][0]
		e.msgHdrs[i].Msg.Iovlen = int32(iovLen)
	}

	return stack.RegisterLinkEndpoint(e), nil
}
