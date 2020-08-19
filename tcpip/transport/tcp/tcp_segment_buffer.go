package tcp

import (
	"container/list"
	"sync"
	"unsafe"
)

const (
	// segment 大小（552） 和 个数（50），大约 27K
	// 这里我们固定维护的segment数量，不再过多维护，否则会造成内存负担。那么多余出来的申请内存就由GC控制，不至于造成空闲内存累积不释放
	defaultSegLength = 50
)

var (
	sb = newSegBuf()
)

type segBuf struct {
	segList *list.List
	mutex   sync.Mutex
}

func newSegBuf() (sb *segBuf) {
	sb = &segBuf{
		segList: list.New(),
		mutex:   sync.Mutex{},
	}
	return
}

var total uint64 = 0

func (sb *segBuf) newSeg() (seg *segment) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if sb.segList.Len() == 0 {
		total++
		return &segment{}
	}

	// 用首个空闲节点
	firstElem := sb.segList.Front()
	seg = (firstElem.Value.(*segment))
	sb.segList.Remove(firstElem)
	return
}

func (sb *segBuf) putSeg(seg *segment) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if sb.segList.Len() >= defaultSegLength {
		return
	}
	// 将数据清空
	var ptr uintptr
	var i uintptr
	ptr = uintptr(unsafe.Pointer(seg))
	for i = 0; i < unsafe.Sizeof(*seg); i++ {
		*((*byte)(unsafe.Pointer(ptr + i))) = 0
	}
	// 放入列表中
	sb.segList.PushBack(seg)
}
