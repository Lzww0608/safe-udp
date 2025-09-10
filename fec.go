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
	"sync/atomic"
	"time"

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

func newFECDecoder(dataShards, parityShards int) *fecDecoder {
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

func (dec *fecDecoder) decode(in fecPacket) (recovered [][]byte) {
	if in.flag() == typeData {
		dec.autoTune.Sample(true, in.seqid())
	} else {
		dec.autoTune.Sample(false, in.seqid())
	}

	if int(in.seqid())%dec.shardSize < dec.dataShards {
		if in.flag() != typeData {
			dec.shouldTune = true
		}
	} else {
		if in.flag() != typeParity {
			dec.shouldTune = true
		}
	}

	if dec.shouldTune {
		autoDS := dec.autoTune.FindPeriod(true)
		autoPS := dec.autoTune.FindPeriod(false)

		if autoDS > 0 && autoPS > 0 && autoDS < 256 && autoPS < 256 {
			dec.dataShards = autoDS
			dec.parityShards = autoPS
			dec.shardSize = dec.dataShards + dec.parityShards
			dec.shardSet = make(map[uint32]*shardHeap)
			codec, err := reedsolomon.New(dec.dataShards, dec.parityShards)
			if err != nil {
				return nil
			}
			dec.codec = codec
			dec.decodeCache = make([][]byte, dec.shardSize)
			dec.flagCache = make([]bool, dec.shardSize)
			dec.shouldTune = false
		}

		return nil
	}

	shardId := dec.getShardId(in.seqid())
	if timediff(shardId, dec.minShardId) < 0 {
		return nil
	}

	shard, ok := dec.shardSet[shardId]
	if !ok {
		shard = newShardHeap()
		dec.shardSet[shardId] = shard
		atomic.AddUint64(&DefaultSnmp.FECShardSet, 1)
	}

	if shard.Contains(in.seqid()) {
		return nil
	}

	if in.flag() == typeParity {
		atomic.AddUint64(&DefaultSnmp.FECParityShards, 1)
	}

	pkt := fecPacket(xmitBuf.Get().([]byte)[:len(in)])
	copy(pkt, in)
	shard.Push(pkt)

	if shard.Len() >= dec.dataShards {
		var numDataShard, maxLen int

		shards := dec.decodeCache
		shardsFlag := dec.flagCache
		for k := range dec.decodeCache {
			shards[k] = nil
			shardsFlag[k] = false
		}

		for shard.Len() > 0 {
			pkt := shard.Pop().(fecPacket)
			seqid := pkt.seqid()
			shards[seqid%uint32(dec.shardSize)] = pkt.data()
			shardsFlag[seqid%uint32(dec.shardSize)] = true
			if pkt.flag() == typeData {
				numDataShard++
			}

			if len(pkt.data()) > maxLen {
				maxLen = len(pkt.data())
			}
			if numDataShard == dec.dataShards {
				// do nothing if all shards are present
				atomic.AddUint64(&DefaultSnmp.FECFullShardSet, 1)
			} else { // case 2: loss on data shards, but it's recoverable from parity shards
				// make the bytes length of each shard equal
				for k := range shards {
					if shards[k] != nil {
						dlen := len(shards[k])
						shards[k] = shards[k][:maxLen]
						clear(shards[k][dlen:])
					} else if k < dec.dataShards {
						// prepare memory for the data recovery
						shards[k] = xmitBuf.Get().([]byte)[:0]
					}
				}

				// Reed-Solomon recovery
				if err := dec.codec.ReconstructData(shards); err == nil {
					for k := range shards[:dec.dataShards] {
						if !shardsFlag[k] {
							// recovered data should be recycled
							recovered = append(recovered, shards[k])
						}
					}
				} else {
					// record the error, and still keep the seqid monotonic increasing
					atomic.AddUint64(&DefaultSnmp.FECErrs, 1)
				}

				atomic.AddUint64(&DefaultSnmp.FECRecovered, uint64(len(recovered)))
			}

		}
	}

	if timediff(shardId, dec.minShardId) > 0 {
		dec.minShardId = shardId
		atomic.StoreUint64(&DefaultSnmp.FECShardMin, uint64(dec.minShardId))
	}

	dec.flushShards()

	return
}

func (dec *fecDecoder) getShardId(seqid uint32) uint32 {
	return seqid / uint32(dec.shardSize)
}

func (dec *fecDecoder) flushShards() {
	for shardId := range dec.shardSet {
		// discard shards that are too old
		if timediff(dec.minShardId, shardId) > maxShardSets {
			//println("flushing shard", shardId, "minShardId", dec.minShardId, _itimediff(dec.minShardId, shardId))
			delete(dec.shardSet, shardId)
		}
	}

	atomic.StoreUint64(&DefaultSnmp.FECShardSet, uint64(len(dec.shardSet)))
}

type (
	// fecEncoder for encoding outgoing packets
	fecEncoder struct {
		dataShards   int
		parityShards int
		shardSize    int
		paws         uint32 // Protect Against Wrapped Sequence numbers
		next         uint32 // next seqid

		shardCount int // count the number of datashards collected
		maxSize    int // track maximum data length in datashard

		headerOffset  int // FEC header offset
		payloadOffset int // FEC payload offset

		// caches
		shardCache     [][]byte
		encodeCache    [][]byte
		tsLatestPacket int64

		// RS encoder
		codec reedsolomon.Encoder
	}
)

func newFECEncoder(dataShards, parityShards, offset int) *fecEncoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}
	enc := new(fecEncoder)
	enc.dataShards = dataShards
	enc.parityShards = parityShards
	enc.shardSize = dataShards + parityShards
	enc.paws = 0xffffffff / uint32(enc.shardSize) * uint32(enc.shardSize)
	enc.headerOffset = offset
	enc.payloadOffset = enc.headerOffset + fecHeaderSize

	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil
	}
	enc.codec = codec

	// caches
	enc.encodeCache = make([][]byte, enc.shardSize)
	enc.shardCache = make([][]byte, enc.shardSize)
	for k := range enc.shardCache {
		enc.shardCache[k] = make([]byte, mtuLimit)
	}
	return enc
}

// encodes the packet, outputs parity shards if we have collected quorum datashards
// notice: the contents of 'ps' will be re-written in successive calling
func (enc *fecEncoder) encode(b []byte, rto uint32) (ps [][]byte) {
	// The header format:
	// | FEC SEQID(4B) | FEC TYPE(2B) | SIZE (2B) | PAYLOAD(SIZE-2) |
	// |<-headerOffset                |<-payloadOffset
	enc.sealData(b[enc.headerOffset:])
	binary.LittleEndian.PutUint16(b[enc.payloadOffset:], uint16(len(b[enc.payloadOffset:])))

	// copy data from payloadOffset to fec shard cache
	sz := len(b)
	enc.shardCache[enc.shardCount] = enc.shardCache[enc.shardCount][:sz]
	copy(enc.shardCache[enc.shardCount][enc.payloadOffset:], b[enc.payloadOffset:])
	enc.shardCount++

	// track max datashard length
	if sz > enc.maxSize {
		enc.maxSize = sz
	}

	// Generation of Reed-Solomon Erasure Code when we have enough datashards
	now := time.Now().UnixMilli()
	if enc.shardCount == enc.dataShards {
		// generate the rs-code only if the data is continuous.
		if now-enc.tsLatestPacket < int64(rto) {
			// fill '0' into the tail of each datashard
			for i := 0; i < enc.dataShards; i++ {
				shard := enc.shardCache[i]
				slen := len(shard)
				clear(shard[slen:enc.maxSize])
			}

			// construct equal-sized slice with stripped header
			cache := enc.encodeCache
			for k := range cache {
				cache[k] = enc.shardCache[k][enc.payloadOffset:enc.maxSize]
			}

			// encoding
			if err := enc.codec.Encode(cache); err == nil {
				ps = enc.shardCache[enc.dataShards:]
				for k := range ps {
					enc.sealParity(ps[k][enc.headerOffset:]) // NOTE(x): seal parity will increase the seqid by 1
					ps[k] = ps[k][:enc.maxSize]
				}
			} else {
				// record the error, and still keep the seqid monotonic increasing
				atomic.AddUint64(&DefaultSnmp.FECErrs, 1)
				enc.skipParity()
			}
		} else {
			// through we do not send non-continuous parity shard, we still increase the next value
			// to keep the seqid aligned with 0 start
			enc.skipParity()
		}

		// Resetting the shard count and max size
		enc.shardCount = 0
		enc.maxSize = 0
	}

	// record the time of the latest packet
	enc.tsLatestPacket = now

	return
}

// sealData and sealParity write the sequence number and type into the header
func (enc *fecEncoder) sealData(data []byte) {
	binary.LittleEndian.PutUint32(data, enc.next)
	binary.LittleEndian.PutUint16(data[4:], typeData)
	enc.next = (enc.next + 1) % enc.paws
}

func (enc *fecEncoder) sealParity(data []byte) {
	binary.LittleEndian.PutUint32(data, enc.next)
	binary.LittleEndian.PutUint16(data[4:], typeParity)
	enc.next = (enc.next + 1) % enc.paws
}

// skipParity skips the parity shards in the sequence
func (enc *fecEncoder) skipParity() {
	enc.next = (enc.next + uint32(enc.parityShards)) % enc.paws
}
