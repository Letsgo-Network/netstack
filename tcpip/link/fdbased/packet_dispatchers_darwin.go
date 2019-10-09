package fdbased

import (
	"syscall"

	"github.com/FlowerWrong/netstack/tcpip/buffer"
	"github.com/FlowerWrong/netstack/tcpip/link/rawfile"
	"github.com/FlowerWrong/netstack/tcpip/stack"
)

func newRecvMMsgDispatcher(fd int, e *endpoint) (linkDispatcher, error) {
	d := &recvMMsgDispatcher{
		fd: fd,
		e:  e,
	}
	d.views = make([][]buffer.View, MaxMsgsPerRecv)
	for i := range d.views {
		d.views[i] = make([]buffer.View, len(BufConfig))
	}
	d.iovecs = make([][]syscall.Iovec, MaxMsgsPerRecv)
	iovLen := len(BufConfig)
	if d.e.Capabilities()&stack.CapabilityGSO != 0 {
		// virtioNetHdr is prepended before each packet.
		iovLen++
	}
	for i := range d.iovecs {
		d.iovecs[i] = make([]syscall.Iovec, iovLen)
	}
	d.msgHdrs = make([]rawfile.MMsgHdr, MaxMsgsPerRecv)
	for i := range d.msgHdrs {
		d.msgHdrs[i].Msg.Iov = &d.iovecs[i][0]
		d.msgHdrs[i].Msg.Iovlen = int32(iovLen)
	}
	return d, nil
}
