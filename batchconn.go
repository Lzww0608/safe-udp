/*
@Author: Lzww
@LastEditTime: 2025-9-1 22:06:51
@Description: Batch UDP connection handling module that provides efficient batch read/write operations interface
@Language: Go 1.23.4
*/

package safeudp

import "golang.org/x/net/ipv4"

const batchSize = 16

type batchConn interface {
	WriteBatch(ms []ipv4.Message, flags int) (int, error)
	ReadBatch(ms []ipv4.Message, flags int) (int, error)
}
