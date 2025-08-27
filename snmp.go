/*
@Author: Lzww
@LastEditTime: 2025-8-27 22:19:56
@Description: SNMP statistics collection for SafeUDP protocol
@Language: Go 1.23.4
*/

package safeudp

import (
	"fmt"
	"sync/atomic"
)

// Snmp contains all statistical counters for SafeUDP protocol monitoring
// All fields are uint64 and should be accessed using atomic operations for thread safety
type Snmp struct {
	// Basic traffic statistics
	BytesSent     uint64 // Total bytes sent through the protocol
	BytesReceived uint64 // Total bytes received through the protocol

	// Connection management statistics
	MaxConn      uint64 // Maximum number of concurrent connections
	ActiveOpens  uint64 // Number of connections opened by this endpoint (client-side)
	PassiveOpens uint64 // Number of connections accepted by this endpoint (server-side)
	CurrEstab    uint64 // Current number of established connections

	// Error statistics
	InErrs          uint64 // Total input errors
	InCsumErrors    uint64 // Input checksum errors
	SafeUdpInErrors uint64 // SafeUDP specific input errors

	// Packet-level statistics
	InPkts  uint64 // Total input packets
	OutPkts uint64 // Total output packets

	// Segment-level statistics (protocol data units)
	InSegs   uint64 // Total input segments
	OutSegs  uint64 // Total output segments
	InBytes  uint64 // Total input bytes at segment level
	OutBytes uint64 // Total output bytes at segment level

	// Retransmission statistics
	RetransSegs      uint64 // Total retransmitted segments
	FastRetransSegs  uint64 // Fast retransmitted segments (duplicate ACK triggered)
	EarlyRetransSegs uint64 // Early retransmitted segments (timeout triggered)
	LostSegs         uint64 // Segments detected as lost
	RepeatSegs       uint64 // Duplicate segments received

	// Forward Error Correction (FEC) statistics
	FECFullShardSet uint64 // Complete FEC shard sets processed
	FECRecovered    uint64 // Data recovered using FEC
	FECErrs         uint64 // FEC processing errors
	FECParityShards uint64 // Parity shards processed
	FECShardSet     uint64 // Total FEC shard sets processed
	FECShardMin     uint64 // Minimum shards required for recovery

	// Ring buffer statistics for internal queues
	RingBufferSndQueue  uint64 // Send queue ring buffer utilization
	RingBufferRcvQueue  uint64 // Receive queue ring buffer utilization
	RingBufferSndBuffer uint64 // Send buffer ring buffer utilization
}

// NewSnmp creates and initializes a new SNMP statistics structure
// All counters are initialized to zero
func NewSnmp() *Snmp {
	return new(Snmp)
}

// Header returns the column headers for SNMP statistics display
// The order matches the ToSlice() method output for consistent reporting
func (s *Snmp) Header() []string {
	return []string{
		"BytesSent",
		"BytesReceived",
		"MaxConn",
		"ActiveOpens",
		"PassiveOpens",
		"CurrEstab",
		"InErrs",
		"InCsumErrors",
		"KCPInErrors", // Legacy name kept for compatibility
		"InPkts",
		"OutPkts",
		"InSegs",
		"OutSegs",
		"InBytes",
		"OutBytes",
		"RetransSegs",
		"FastRetransSegs",
		"EarlyRetransSegs",
		"LostSegs",
		"RepeatSegs",
		"FECFullShards",
		"FECParityShards",
		"FECErrs",
		"FECRecovered",
		"FECShardSet",
		"FECShardMin",
		"RingBufferSndQueue",
		"RingBufferRcvQueue",
		"RingBufferSndBuffer",
	}
}

// ToSlice converts all SNMP statistics to a string slice for display purposes
// Creates a thread-safe copy of all counters before conversion to avoid inconsistent reads
func (s *Snmp) ToSlice() []string {
	snmp := s.Copy()
	return []string{
		fmt.Sprint(snmp.BytesSent),
		fmt.Sprint(snmp.BytesReceived),
		fmt.Sprint(snmp.MaxConn),
		fmt.Sprint(snmp.ActiveOpens),
		fmt.Sprint(snmp.PassiveOpens),
		fmt.Sprint(snmp.CurrEstab),
		fmt.Sprint(snmp.InErrs),
		fmt.Sprint(snmp.InCsumErrors),
		fmt.Sprint(snmp.SafeUdpInErrors),
		fmt.Sprint(snmp.InPkts),
		fmt.Sprint(snmp.OutPkts),
		fmt.Sprint(snmp.InSegs),
		fmt.Sprint(snmp.OutSegs),
		fmt.Sprint(snmp.InBytes),
		fmt.Sprint(snmp.OutBytes),
		fmt.Sprint(snmp.RetransSegs),
		fmt.Sprint(snmp.FastRetransSegs),
		fmt.Sprint(snmp.EarlyRetransSegs),
		fmt.Sprint(snmp.LostSegs),
		fmt.Sprint(snmp.RepeatSegs),
		fmt.Sprint(snmp.FECFullShardSet),
		fmt.Sprint(snmp.FECParityShards),
		fmt.Sprint(snmp.FECErrs),
		fmt.Sprint(snmp.FECRecovered),
		fmt.Sprint(snmp.FECShardSet),
		fmt.Sprint(snmp.FECShardMin),
		fmt.Sprint(snmp.RingBufferSndQueue),
		fmt.Sprint(snmp.RingBufferRcvQueue),
		fmt.Sprint(snmp.RingBufferSndBuffer),
	}
}

// Copy creates a thread-safe snapshot of all SNMP statistics
// Uses atomic operations to ensure consistent reads across all counters
// Returns a new Snmp instance with copied values
func (s *Snmp) Copy() *Snmp {
	d := NewSnmp()
	d.BytesSent = atomic.LoadUint64(&s.BytesSent)
	d.BytesReceived = atomic.LoadUint64(&s.BytesReceived)
	d.MaxConn = atomic.LoadUint64(&s.MaxConn)
	d.ActiveOpens = atomic.LoadUint64(&s.ActiveOpens)
	d.PassiveOpens = atomic.LoadUint64(&s.PassiveOpens)
	d.CurrEstab = atomic.LoadUint64(&s.CurrEstab)
	d.InErrs = atomic.LoadUint64(&s.InErrs)
	d.InCsumErrors = atomic.LoadUint64(&s.InCsumErrors)
	d.SafeUdpInErrors = atomic.LoadUint64(&s.SafeUdpInErrors)
	d.InPkts = atomic.LoadUint64(&s.InPkts)
	d.OutPkts = atomic.LoadUint64(&s.OutPkts)
	d.InSegs = atomic.LoadUint64(&s.InSegs)
	d.OutSegs = atomic.LoadUint64(&s.OutSegs)
	d.InBytes = atomic.LoadUint64(&s.InBytes)
	d.OutBytes = atomic.LoadUint64(&s.OutBytes)
	d.RetransSegs = atomic.LoadUint64(&s.RetransSegs)
	d.FastRetransSegs = atomic.LoadUint64(&s.FastRetransSegs)
	d.EarlyRetransSegs = atomic.LoadUint64(&s.EarlyRetransSegs)
	d.LostSegs = atomic.LoadUint64(&s.LostSegs)
	d.RepeatSegs = atomic.LoadUint64(&s.RepeatSegs)
	d.FECFullShardSet = atomic.LoadUint64(&s.FECFullShardSet)
	d.FECParityShards = atomic.LoadUint64(&s.FECParityShards)
	d.FECErrs = atomic.LoadUint64(&s.FECErrs)
	d.FECRecovered = atomic.LoadUint64(&s.FECRecovered)
	d.FECShardSet = atomic.LoadUint64(&s.FECShardSet)
	d.FECShardMin = atomic.LoadUint64(&s.FECShardMin)
	d.RingBufferSndQueue = atomic.LoadUint64(&s.RingBufferSndQueue)
	d.RingBufferRcvQueue = atomic.LoadUint64(&s.RingBufferRcvQueue)
	d.RingBufferSndBuffer = atomic.LoadUint64(&s.RingBufferSndBuffer)
	return d
}

// Reset atomically sets all SNMP statistics counters to zero
// This is useful for clearing statistics during testing or periodic resets
// Uses atomic operations to ensure thread-safe reset of all counters
func (s *Snmp) Reset() {
	atomic.StoreUint64(&s.BytesSent, 0)
	atomic.StoreUint64(&s.BytesReceived, 0)
	atomic.StoreUint64(&s.MaxConn, 0)
	atomic.StoreUint64(&s.ActiveOpens, 0)
	atomic.StoreUint64(&s.PassiveOpens, 0)
	atomic.StoreUint64(&s.CurrEstab, 0)
	atomic.StoreUint64(&s.InErrs, 0)
	atomic.StoreUint64(&s.InCsumErrors, 0)
	atomic.StoreUint64(&s.SafeUdpInErrors, 0)
	atomic.StoreUint64(&s.InPkts, 0)
	atomic.StoreUint64(&s.OutPkts, 0)
	atomic.StoreUint64(&s.InSegs, 0)
	atomic.StoreUint64(&s.OutSegs, 0)
	atomic.StoreUint64(&s.InBytes, 0)
	atomic.StoreUint64(&s.OutBytes, 0)
	atomic.StoreUint64(&s.RetransSegs, 0)
	atomic.StoreUint64(&s.FastRetransSegs, 0)
	atomic.StoreUint64(&s.EarlyRetransSegs, 0)
	atomic.StoreUint64(&s.LostSegs, 0)
	atomic.StoreUint64(&s.RepeatSegs, 0)
	atomic.StoreUint64(&s.FECFullShardSet, 0)
	atomic.StoreUint64(&s.FECParityShards, 0)
	atomic.StoreUint64(&s.FECErrs, 0)
	atomic.StoreUint64(&s.FECRecovered, 0)
	atomic.StoreUint64(&s.FECShardSet, 0)
	atomic.StoreUint64(&s.FECShardMin, 0)
	atomic.StoreUint64(&s.RingBufferSndQueue, 0)
	atomic.StoreUint64(&s.RingBufferRcvQueue, 0)
	atomic.StoreUint64(&s.RingBufferSndBuffer, 0)
}

// DefaultSnmp is the global default SNMP statistics instance
// This can be used for collecting system-wide SafeUDP statistics
var DefaultSnmp *Snmp

// init initializes the default SNMP statistics instance
// Called automatically when the package is imported
func init() {
	DefaultSnmp = NewSnmp()
}
