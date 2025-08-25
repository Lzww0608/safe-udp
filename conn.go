/*
@Author: Lzww
@LastEditTime: 2025-8-25 22:08:25
@Description: Conn
@Language: Go 1.23.4
*/

package safeudp

import (
	"net"
	"time"

	"github.com/xtaci/smux"
)

type Conn struct {
	// point to the underlying smux stream
	stream *smux.Stream
	// point to the parent session
	sess *smux.Session
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.stream.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.stream.Write(b)
}

func (c *Conn) Close() error {
	return c.stream.Close()
}

func (c *Conn) LocalAddr() net.Addr {
	return c.sess.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.sess.RemoteAddr()
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
