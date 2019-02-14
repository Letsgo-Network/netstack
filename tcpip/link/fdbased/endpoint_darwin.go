package fdbased

import (
	"fmt"
	"syscall"

	"github.com/FlowerWrong/netstack/tcpip"
	"github.com/FlowerWrong/netstack/tcpip/buffer"
	"github.com/FlowerWrong/netstack/tcpip/header"
	"github.com/FlowerWrong/netstack/tcpip/link/rawfile"
	"github.com/FlowerWrong/netstack/tcpip/stack"
)

// New creates a new fd-based endpoint.
//
// Makes fd non-blocking, but does not take ownership of fd, which must remain
// open for the lifetime of the returned endpoint.
func New(opts *Options) tcpip.LinkEndpointID {
	if err := syscall.SetNonblock(opts.FD, true); err != nil {
		// TODO : replace panic with an error return.
		panic(fmt.Sprintf("syscall.SetNonblock(%v) failed: %v", opts.FD, err))
	}

	caps := stack.LinkEndpointCapabilities(0)
	if opts.ChecksumOffload {
		caps |= stack.CapabilityChecksumOffload
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
		handleLocal:        opts.HandleLocal,
		packetDispatchMode: opts.PacketDispatchMode,
	}

	if isSocketFD(opts.FD) && e.packetDispatchMode == PacketMMap {
		if err := e.setupPacketRXRing(); err != nil {
			// TODO: replace panic with an error return.
			panic(fmt.Sprintf("e.setupPacketRXRing failed: %v", err))
		}
		e.inboundDispatcher = e.packetMMapDispatch
		return stack.RegisterLinkEndpoint(e)
	}

	// For non-socket FDs we read one packet a time (e.g. TAP devices)
	msgsPerRecv := 1
	e.inboundDispatcher = e.dispatch
	// If the provided FD is a socket then we optimize packet reads by
	// using recvmmsg() instead of read() to read packets in a batch.
	if isSocketFD(opts.FD) && e.packetDispatchMode == RecvMMsg {
		e.inboundDispatcher = e.recvMMsgDispatch
		msgsPerRecv = MaxMsgsPerRecv
	}

	e.views = make([][]buffer.View, msgsPerRecv)
	for i, _ := range e.views {
		e.views[i] = make([]buffer.View, len(BufConfig))
	}
	e.iovecs = make([][]syscall.Iovec, msgsPerRecv)
	for i, _ := range e.iovecs {
		e.iovecs[i] = make([]syscall.Iovec, len(BufConfig))
	}
	e.msgHdrs = make([]rawfile.MMsgHdr, msgsPerRecv)
	for i, _ := range e.msgHdrs {
		e.msgHdrs[i].Msg.Iov = &e.iovecs[i][0]
		e.msgHdrs[i].Msg.Iovlen = int32(len(BufConfig))
	}

	return stack.RegisterLinkEndpoint(e)
}
