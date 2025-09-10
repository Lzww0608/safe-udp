/*
@Author: Lzww
@LastEditTime: 2025-9-10 21:00:00
@Description: Unit tests for session functionality and fixes
@Language: Go 1.23.4
*/

package safeudp

import (
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/ipv4"
)

// MockPacketConn 模拟 PacketConn 用于测试
type MockPacketConn struct {
	readData   []byte
	readAddr   net.Addr
	readError  error
	writeError error
	writeCount int
	writeMutex sync.Mutex
}

func (m *MockPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if m.readError != nil {
		return 0, nil, m.readError
	}
	n = copy(p, m.readData)
	return n, m.readAddr, nil
}

func (m *MockPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	m.writeMutex.Lock()
	defer m.writeMutex.Unlock()
	m.writeCount++
	if m.writeError != nil {
		return 0, m.writeError
	}
	return len(p), nil
}

func (m *MockPacketConn) Close() error {
	return nil
}

func (m *MockPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (m *MockPacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockPacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockPacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// MockBatchConn 模拟 batchConn 接口用于测试批量传输
type MockBatchConn struct {
	*MockPacketConn
	batchWriteError error
	batchWriteCount int
}

func (m *MockBatchConn) WriteBatch(msgs []ipv4.Message, flags int) (int, error) {
	if m.batchWriteError != nil {
		return 0, m.batchWriteError
	}
	m.batchWriteCount += len(msgs)
	return len(msgs), nil
}

func (m *MockBatchConn) ReadBatch(msgs []ipv4.Message, flags int) (int, error) {
	// 简单的模拟实现，返回一条消息
	if len(msgs) > 0 && m.readData != nil {
		copy(msgs[0].Buffers[0], m.readData)
		msgs[0].Addr = m.readAddr
		return 1, nil
	}
	return 0, m.readError
}

// Test helper function to create a test UDP session
func createTestSession(t *testing.T, mockConn *MockPacketConn) *UDPSession {
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 9999}

	// Create a mock listener to prevent automatic readLoop startup
	mockListener := &Listener{
		conn:              mockConn,
		sessions:          make(map[string]*UDPSession),
		chAccepts:         make(chan *UDPSession, 10),
		chSessionClosed:   make(chan net.Addr, 10),
		die:               make(chan struct{}),
		chSocketReadError: make(chan struct{}),
	}

	sess := newUDPSession(12345, 10, 3, mockListener, mockConn, false, remoteAddr, nil)
	return sess
}

// Test helper function to create a client session (with readLoop)
func createClientSession(t *testing.T, mockConn *MockPacketConn) *UDPSession {
	remoteAddr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 9999}
	sess := newUDPSession(12345, 10, 3, nil, mockConn, false, remoteAddr, nil)
	return sess
}

// TestTxMethods 测试 tx、defaultTx 和 batchTx 方法
func TestTxMethods(t *testing.T) {
	// 测试 defaultTx
	t.Run("DefaultTx", func(t *testing.T) {
		mockConn := &MockPacketConn{}
		sess := createTestSession(t, mockConn)

		// 准备测试消息队列
		msgs := []ipv4.Message{
			{
				Buffers: [][]byte{[]byte("test message 1")},
				Addr:    sess.remote,
			},
			{
				Buffers: [][]byte{[]byte("test message 2")},
				Addr:    sess.remote,
			},
		}

		// 执行 defaultTx
		sess.defaultTx(msgs)

		// 验证写入次数
		if mockConn.writeCount != 2 {
			t.Errorf("Expected 2 writes, got %d", mockConn.writeCount)
		}
	})

	// 测试 batchTx
	t.Run("BatchTx", func(t *testing.T) {
		mockConn := &MockPacketConn{}
		mockBatchConn := &MockBatchConn{MockPacketConn: mockConn}
		sess := createTestSession(t, mockConn)
		sess.xconn = mockBatchConn

		// 准备测试消息队列
		msgs := []ipv4.Message{
			{
				Buffers: [][]byte{[]byte("batch message 1")},
				Addr:    sess.remote,
			},
			{
				Buffers: [][]byte{[]byte("batch message 2")},
				Addr:    sess.remote,
			},
		}

		// 执行 batchTx
		sess.batchTx(msgs)

		// 验证批量写入次数
		if mockBatchConn.batchWriteCount != 2 {
			t.Errorf("Expected 2 batch writes, got %d", mockBatchConn.batchWriteCount)
		}
	})

	// 测试 tx 方法的路由逻辑
	t.Run("TxRouting", func(t *testing.T) {
		mockConn := &MockPacketConn{}
		sess := createTestSession(t, mockConn)

		msgs := []ipv4.Message{
			{
				Buffers: [][]byte{[]byte("routing test")},
				Addr:    sess.remote,
			},
		}

		// 测试没有 xconn 时使用 defaultTx
		sess.xconn = nil
		sess.tx(msgs)
		if mockConn.writeCount != 1 {
			t.Errorf("Expected 1 default write, got %d", mockConn.writeCount)
		}

		// 重置计数器
		mockConn.writeCount = 0

		// 测试有 xconn 时使用 batchTx
		mockBatchConn := &MockBatchConn{MockPacketConn: mockConn}
		sess.xconn = mockBatchConn
		sess.tx(msgs)
		if mockBatchConn.batchWriteCount != 1 {
			t.Errorf("Expected 1 batch write, got %d", mockBatchConn.batchWriteCount)
		}
	})
}

// TestReadLoop 测试 readLoop 方法
func TestReadLoop(t *testing.T) {
	t.Run("ReadLoopBasic", func(t *testing.T) {
		mockConn := &MockPacketConn{
			readData: []byte("test packet data"),
			readAddr: &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 9999},
		}
		sess := createClientSession(t, mockConn)

		// 启动 readLoop 并在短时间后停止
		go sess.readLoop()

		// 等待一小段时间让 readLoop 运行
		time.Sleep(10 * time.Millisecond)

		// 关闭会话停止 readLoop
		sess.Close()
	})

	t.Run("ReadLoopWithError", func(t *testing.T) {
		mockConn := &MockPacketConn{
			readError: &net.OpError{Op: "read", Err: &net.AddrError{Err: "connection refused"}},
		}
		sess := createClientSession(t, mockConn)

		// 启动 readLoop
		go sess.readLoop()

		// 等待错误处理
		time.Sleep(10 * time.Millisecond)

		// 验证错误通知已设置
		select {
		case <-sess.chSocketReadError:
			// 错误通知正常工作
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected read error notification")
		}
	})
}

// TestMonitor 测试 monitor 方法
func TestMonitor(t *testing.T) {
	t.Run("MonitorBasic", func(t *testing.T) {
		mockConn := &MockPacketConn{
			readData: []byte("monitor test data"),
			readAddr: &net.UDPAddr{IP: net.ParseIP("192.168.1.200"), Port: 8888},
		}

		listener := &Listener{
			conn:              mockConn,
			sessions:          make(map[string]*UDPSession),
			chAccepts:         make(chan *UDPSession, 10),
			chSessionClosed:   make(chan net.Addr, 10),
			die:               make(chan struct{}),
			chSocketReadError: make(chan struct{}),
		}

		// 启动 monitor
		go listener.monitor()

		// 等待一小段时间让 monitor 运行
		time.Sleep(10 * time.Millisecond)

		// 关闭监听器停止 monitor
		close(listener.die)
	})
}

// TestNamingFixes 测试命名修复的正确性
func TestNamingFixes(t *testing.T) {
	t.Run("SystemTimerAccess", func(t *testing.T) {
		// 验证 SystemTimer 全局变量可以正常访问
		if SystemTimer == nil {
			t.Error("SystemTimer should be initialized")
		}

		// 测试 Put 方法调用
		called := false
		testFunc := func() {
			called = true
		}

		SystemTimer.Put(testFunc, time.Now().Add(time.Millisecond))

		// 等待执行
		time.Sleep(5 * time.Millisecond)

		if !called {
			t.Error("SystemTimer.Put should execute the function")
		}
	})

	t.Run("DefaultSnmpFields", func(t *testing.T) {
		// 验证 DefaultSnmp 全局变量和字段可以正常访问
		if DefaultSnmp == nil {
			t.Error("DefaultSnmp should be initialized")
		}

		// 测试 SafeUdpInErrors 字段存在且可以操作
		originalValue := DefaultSnmp.SafeUdpInErrors
		DefaultSnmp.SafeUdpInErrors = 42

		if DefaultSnmp.SafeUdpInErrors != 42 {
			t.Error("SafeUdpInErrors field should be accessible and modifiable")
		}

		// 恢复原值
		DefaultSnmp.SafeUdpInErrors = originalValue
	})
}

// TestErrorHandling 测试错误处理场景
func TestErrorHandling(t *testing.T) {
	t.Run("TxWithWriteError", func(t *testing.T) {
		mockConn := &MockPacketConn{
			writeError: &net.OpError{Op: "write", Err: &net.AddrError{Err: "network unreachable"}},
		}
		sess := createTestSession(t, mockConn)

		msgs := []ipv4.Message{
			{
				Buffers: [][]byte{[]byte("error test")},
				Addr:    sess.remote,
			},
		}

		// 执行 defaultTx 应该触发错误通知
		sess.defaultTx(msgs)

		// 验证错误通知
		select {
		case <-sess.chSocketWriteError:
			// 写入错误通知正常工作
		case <-time.After(100 * time.Millisecond):
			t.Error("Expected write error notification")
		}
	})

	t.Run("BatchTxFallback", func(t *testing.T) {
		mockConn := &MockPacketConn{}
		mockBatchConn := &MockBatchConn{
			MockPacketConn:  mockConn,
			batchWriteError: &net.OpError{Op: "batch_write", Err: &net.AddrError{Err: "batch not supported"}},
		}
		sess := createTestSession(t, mockConn)
		sess.xconn = mockBatchConn

		msgs := []ipv4.Message{
			{
				Buffers: [][]byte{[]byte("fallback test")},
				Addr:    sess.remote,
			},
		}

		// 执行 batchTx，应该回退到 defaultTx
		sess.batchTx(msgs)

		// 验证回退到默认传输方法
		if mockConn.writeCount != 1 {
			t.Errorf("Expected fallback to defaultTx, got writeCount: %d", mockConn.writeCount)
		}

		// 验证错误记录
		if sess.xconnWriteError == nil {
			t.Error("Expected xconnWriteError to be set")
		}
	})
}
