/*
@Author: Lzww
@LastEditTime: 2025-8-25 21:58:48
@Description: Safe UDP
@Language: Go 1.23.4
*/

package safeudp

import (
	"net"

	"github.com/xtaci/smux"
)

type Config struct {
	// Pre-shared key for encryption (32 bytes for AES-256)
	Key []byte

	// FEC settings
	FECData   int // Number of data packets in FEC group
	FECParity int // Number of parity packets in FEC group

	// KCP settings
	NoDelay      int // Enable nodelay mode
	Interval     int // Internal update timer interval in millisec
	Resend       int // Fast resend mode
	NoCongestion int // Disable congestion control

	// Buffer settings
	SendBuffer int // Send buffer size
	RecvBuffer int // Receive buffer size
}

func Dial(addr string, config *Config) (*Conn, error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	session, err := smux.Client(conn, nil)
	if err != nil {
		return nil, err
	}

	stream, err := session.OpenStream()

	if err != nil {
		return nil, err
	}

	return &Conn{
		stream: stream,
		sess:   session,
	}, nil
}

func Listen(addr string, config *Config) (*Listener, error) {
	listener, err := net.Listen("udp", addr)
	if err != nil {
		return nil, err
	}

	return &Listener{listener: listener, config: config}, nil
}
