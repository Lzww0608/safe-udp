/*
@Author: Lzww
@LastEditTime: 2025-8-29 21:15:32
@Description: FEC (Forward Error Correction) implementation for SafeUDP protocol
@Language: Go 1.23.4
*/

package safeudp

import (
	"container/heap"
	"encoding/binary"

	"github.com/klauspost/reedsolomon"
)

const (
	fecHeaderSize     = 6
	fecHeaderSizePlus = fecHeaderSize + 2
	typeData          = 0xf1
	typeParity        = 0xf2
	maxShardSets      = 3
)

type fecPacket []byte

func (fec fecPacket) seqid() uint32 {
	return binary.LittleEndian.Uint32(fec)
}

func (fec fecPacket) flag() uint16 {
	return binary.LittleEndian.Uint16(fec[4:])
}

func (fec fecPacket) data() []byte {
	return fec[6:]
}

type shardHeap struct {
	elements []fecPacket
	marks    map[uint32]struct{} // to avoid duplicates
}

func (h *shardHeap) Len() int {
	return len(h.elements)
}

func (h *shardHeap) Less(i, j int) bool {
	return timediff(h.elements[i].seqid(), h.elements[j].seqid()) < 0
}

func (h *shardHeap) Swap(i, j int) {
	h.elements[i], h.elements[j] = h.elements[j], h.elements[i]
}

func (h *shardHeap) Push(x any) {
	h.elements = append(h.elements, x.(fecPacket))
	h.marks[x.(fecPacket).seqid()] = struct{}{}
}

func (h *shardHeap) Pop() any {
	old := h.elements
	n := len(old)
	x := old[n-1]
	h.elements = old[0 : n-1]
	delete(h.marks, x.seqid())
	return x
}

func (h *shardHeap) Contains(seqid uint32) bool {
	_, ok := h.marks[seqid]
	return ok
}

func newShardHeap() *shardHeap {
	h := &shardHeap{
		elements: []fecPacket{},
		marks:    make(map[uint32]struct{}),
	}
	heap.Init(h)
	return h
}

type fecDecoder struct {
	rxlimit      int
	dataShards   int
	parityShards int
	shardSize    int
	shardSet     map[uint32]*shardHeap

	minShardId uint32

	decodeCache [][]byte
	flagCache   []bool

	codec reedsolomon.Encoder

	autoTune   autoTune
	shouldTune bool
}

func newFecDecoder(dataShards, parityShards int) *fecDecoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}

	dec := new(fecDecoder)
	dec.dataShards = dataShards
	dec.parityShards = parityShards
	dec.shardSize = dataShards + parityShards
	dec.shardSet = make(map[uint32]*shardHeap)
	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil
	}

	dec.codec = codec
	dec.decodeCache = make([][]byte, dec.shardSize)
	dec.flagCache = make([]bool, dec.shardSize)
	return dec
}
