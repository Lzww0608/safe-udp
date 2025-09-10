/*
@Author: Lzww
@LastEditTime: 2025-8-25 22:08:25
@Description: Listener
@Language: Go 1.23.4
*/

package safeudp

import (
	"net"

	"github.com/xtaci/smux"
)

type StreamListener struct {
	listener net.Listener
}

func (l *StreamListener) Accept() (net.Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}

	session, err := smux.Server(conn, nil)
	if err != nil {
		return nil, err
	}

	stream, err := session.AcceptStream()
	if err != nil {
		return nil, err
	}

	return &Conn{
		stream: stream,
		sess:   session,
	}, nil
}

func (l *StreamListener) Close() error {
	return l.listener.Close()
}

func (l *StreamListener) Addr() net.Addr {
	return l.listener.Addr()
}
