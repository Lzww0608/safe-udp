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

type Listener struct {
	listener net.Listener
	config   *Config
}

func (l *Listener) Accept() (net.Conn, error) {
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

func (l *Listener) Close() error {
	return l.listener.Close()
}

func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}
