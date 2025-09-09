/*
@Author: Lzww
@LastEditTime: 2025-9-9 20:35:46
@Description: Session
@Language: Go 1.23.4
*/

package safeudp

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	// 16-bytes nonce for each packet
	nonceSize = 16

	// 4-bytes packet checksum
	crcSize = 4

	// overall crypto header size
	cryptHeaderSize = nonceSize + crcSize

	// maximum packet size
	mtuLimit = 1500

	// accept backlog
	acceptBacklog = 128

	// maximum latency for consecutive FEC encoding, in milliseconds
	maxFECEncodingLatency = 500
)

var (
	errInvalidOperation = errors.New("invalid operation")
	errTimeout          = errors.New("timeout")
	errNotOwner         = errors.New("not owner")
)

type timeoutError struct{}

func (timeoutError) Error() string {
	return "timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return true
}

var (
	// a system-wide packet buffer shared among sending, receiving and FEC
	// to mitigate high-frequency memory allocation for packets, bytes from xmitBuf
	// is aligned to 64bit
	xmitBuf sync.Pool
)

func init() {
	xmitBuf.New = func() any {
		return make([]byte, mtuLimit)
	}
}

type (
	UDPSession struct {
		conn    net.PacketConn
		ownConn bool
		kcp     *KCP
		l       *Listener
		block   BlockCrypt

		recvbuf []byte
		bufptr  []byte

		fecDecoder *fecDecoder
		fecEncode  *fecEncoder

		remote     net.Addr
		rd         time.Time
		wd         time.Time
		headerSize int
		ackNoDelay bool
		writeDelay bool
		dup        int

		die          chan struct{}
		dieOnce      sync.Once
		chReadEvent  chan struct{}
		chWriteEvent chan struct{}

		socketReadError      atomic.Value
		socketWriteError     atomic.Value
		chSocketReadError    chan struct{}
		chSocketWriteError   chan struct{}
		socketReadErrorOnce  sync.Once
		socketWriteErrorOnce sync.Once

		nonce Entropy

		chPostProcessing chan []byte

		xconn           batchConn
		xconnWriteError error

		mu sync.Mutex
	}

	setReadBuffer interface {
		SetReadBuffer(bytes int) error
	}

	setWriteBuffer interface {
		SetWriteBuffer(bytes int) error
	}

	setDSCP interface {
		SetDSCP(int) error
	}
)
