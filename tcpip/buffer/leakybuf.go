// Copyright (C) 2017-2018  DawnDIY<dawndiy.dev@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package buffer

import (
	"container/list"
	"sync"
)

const (
	// ViewArr 大小 和 个数，占用空间： defaultVASize * 指针占用的空间 * defaultVALength = 8 * 4 * 1000
	defaultVASize   = 8
	defaultVALength = 100

	block64Len   = 100
	block512Len  = 100
	block1600Len = 100
	block2KLen   = 100
)

var (
	lb = newLeakyBuf()
)

type pktInfo struct {
	content []byte
}

type leakyBuf struct {
	blocksize       int
	viewarrSize     int
	view64FreeList  *list.List
	view512FreeList *list.List
	view1KFreeList  *list.List
	view2KFreeList  *list.List
	viewArrFreeList *list.List
	mutex           sync.RWMutex
}

func newLeakyBuf() (lb *leakyBuf) {
	lb = &leakyBuf{
		viewarrSize:     defaultVASize,
		viewArrFreeList: list.New().Init(),
		view64FreeList:  list.New().Init(),
		view512FreeList: list.New().Init(),
		view1KFreeList:  list.New().Init(),
		view2KFreeList:  list.New().Init(),
		mutex:           sync.RWMutex{},
	}

	for i := 0; i < defaultVALength; i++ {
		lb.viewArrFreeList.PushBack(make([]View, defaultVASize))
	}
	for i := 0; i < block64Len; i++ {
		lb.view64FreeList.PushBack(make([]byte, 64))
	}
	for i := 0; i < block512Len; i++ {
		lb.view512FreeList.PushBack(make([]byte, 512))
	}
	for i := 0; i < block1600Len; i++ {
		lb.view1KFreeList.PushBack(make([]byte, 1600))
	}
	for i := 0; i < block2KLen; i++ {
		lb.view2KFreeList.PushBack(make([]byte, 2048))
	}

	return lb
}

var view64Index = 1
var view512Index = 1
var view1KIndex = 1
var view2KIndex = 1

func (lb *leakyBuf) getView(size int) (b []byte) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	var l *list.List

	if size <= 64 {
		l = lb.view64FreeList
		view64Index++
		if view64Index > block64Len {
			view64Index = 1
			//println("64 is full")
		}
	} else if size <= 512 {
		l = lb.view512FreeList
		view512Index++
		if view512Index > block512Len {
			view512Index = 1
			//println("512 is full")
		}
	} else if size <= 1600 {
		l = lb.view1KFreeList
		view1KIndex++
		if view1KIndex > block1600Len {
			view1KIndex = 1
			//println("1600 is full")
		}
	} else if size <= 2048 {
		l = lb.view2KFreeList
		view2KIndex++
		if view2KIndex > block2KLen {
			view2KIndex = 1
			//println("2K is full")
		}
	} else {
		return nil
	}

	firstElem := l.Front()
	b = firstElem.Value.([]byte)
	l.MoveToBack(firstElem)
	return
}

var viewArrIndex = 1

func (lb *leakyBuf) getViewArr() (va []View) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	//println(viewArrIndex)
	viewArrIndex++
	if viewArrIndex > defaultVALength {
		viewArrIndex = 1
	}
	firstElem := lb.viewArrFreeList.Front()
	va = (firstElem.Value.([]View))
	lb.viewArrFreeList.MoveToBack(firstElem)
	return
}

func (lb *leakyBuf) maxBlockSize() int {
	return 2048
}

func (lb *leakyBuf) viewArrSize() int {
	return lb.viewarrSize
}
