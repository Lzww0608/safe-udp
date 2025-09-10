/*
@Author: Lzww
@LastEditTime: 2025-9-9 20:27:32
@Description: Crypt
@Language: Go 1.23.4
*/

package safeudp

import (
	"sync/atomic"

	"github.com/pkg/errors"
	"golang.org/x/net/ipv4"
)

// tx sends packets using the appropriate transmission method
func (s *UDPSession) tx(txqueue []ipv4.Message) {
	// Check if we have batch connection capability
	if s.xconn != nil {
		s.batchTx(txqueue)
	} else {
		s.defaultTx(txqueue)
	}
}

func (s *UDPSession) defaultTx(txqueue []ipv4.Message) {
	nbytes, npkts := 0, 0

	for k := range txqueue {
		if n, err := s.conn.WriteTo(txqueue[k].Buffers[0], txqueue[k].Addr); err == nil {
			nbytes += n
			npkts++
		} else {
			s.notifyWriteError(errors.WithStack(err))
			break
		}
	}

	atomic.AddUint64(&DefaultSnmp.OutPkts, uint64(npkts))
	atomic.AddUint64(&DefaultSnmp.OutBytes, uint64(nbytes))
}

func (s *UDPSession) batchTx(txqueue []ipv4.Message) {
	nbytes, npkts := 0, 0

	if _, err := s.xconn.WriteBatch(txqueue, 0); err == nil {
		for k := range txqueue {
			nbytes += len(txqueue[k].Buffers[0])
		}
		npkts = len(txqueue)
		atomic.AddUint64(&DefaultSnmp.OutPkts, uint64(npkts))
		atomic.AddUint64(&DefaultSnmp.OutBytes, uint64(nbytes))
	} else {
		// fall back to default transmission method
		s.xconnWriteError = err
		s.defaultTx(txqueue)
	}
}
