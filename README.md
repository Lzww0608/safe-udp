# Safe-UDP

A lightweight, high-performance secure UDP component that provides reliable, encrypted communication over UDP with built-in Forward Error Correction (FEC) capabilities.

## Overview

Safe-UDP is a Go library that combines the reliability of KCP protocol with strong encryption and error correction, providing a secure and efficient alternative to TCP for applications that require low-latency, reliable communication. It offers familiar `net.Conn` and `net.Listener` interfaces for easy integration into existing Go applications.

## Features

- **Reliable UDP**: Built on KCP protocol for reliable data transmission over UDP
- **Strong Encryption**: AES-GCM encryption with PSK-based key derivation (HKDF)
- **Forward Error Correction**: Built-in FEC for enhanced packet loss recovery
- **Connection Multiplexing**: Multiple streams over a single UDP connection via smux
- **Standard Interfaces**: Compatible with `net.Conn` and `net.Listener` interfaces
- **High Performance**: Optimized buffer management with sync.Pool
- **Lightweight**: Minimal overhead with pure Go implementation

## Architecture

### Core Components

```
safeudp/
├── safeudp.go          # Public API (Dial, Listen, Config)
├── listener.go         # Listener implementation with session management
├── conn.go             # Connection wrapper for smux.Stream
├── secure_conn.go      # Internal secure connection with KCP + crypto + FEC
├── buffer.go           # Optimized buffer management (sync.Pool)
├── crypto/
│   ├── crypto.go       # BlockCrypt interface and HKDF key derivation
│   └── aesgcm.go       # AES-GCM implementation
└── kcp/
    └── ikcp.go         # Pure Go KCP implementation
```

### Data Flow

1. **Client**: `Dial()` → `secureConn` → KCP + Encryption → UDP
2. **Server**: UDP → Decryption + KCP → `secureConn` → `Listener.Accept()`
3. **Multiplexing**: Multiple `Conn` instances over single `secureConn` via smux

## Installation

```bash
go get github.com/yourusername/safe-udp
```

## Configuration

### Config Structure

```go
type Config struct {
    // Pre-shared key for encryption (32 bytes for AES-256)
    Key []byte
    
    // FEC settings
    FECData   int // Number of data packets in FEC group
    FECParity int // Number of parity packets in FEC group
    
    // KCP settings
    NoDelay    int // Enable nodelay mode
    Interval   int // Internal update timer interval in millisec
    Resend     int // Fast resend mode
    NoCongestion int // Disable congestion control
    
    // Buffer settings
    SendBuffer int // Send buffer size
    RecvBuffer int // Receive buffer size
}
```

### Default Configuration

```go
func DefaultConfig() *Config {
    return &Config{
        FECData:   10,
        FECParity: 3,
        NoDelay:   1,
        Interval:  10,
        Resend:    2,
        NoCongestion: 1,
        SendBuffer: 4194304, // 4MB
        RecvBuffer: 4194304, // 4MB
    }
}
```

## API Reference

### Core Functions

- `func Dial(network, address string, config *Config) (net.Conn, error)`
- `func Listen(network, address string, config *Config) (net.Listener, error)`

### Interfaces

The library implements standard Go network interfaces:

- `net.Conn` - for client/server connections
- `net.Listener` - for server listeners

## Performance Considerations

1. **Buffer Management**: Uses `sync.Pool` for efficient memory allocation
2. **FEC Tuning**: Adjust FECData/FECParity based on network conditions
3. **KCP Parameters**: Fine-tune NoDelay, Interval, Resend for your use case
4. **Concurrent Connections**: Each connection runs in its own goroutine

## Security

- **Encryption**: AES-GCM provides both confidentiality and authenticity
- **Key Derivation**: HKDF ensures proper key material derivation from PSK
- **Perfect Forward Secrecy**: Consider implementing session key exchange for PFS

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [KCP](https://github.com/skywind3000/kcp) - Reliable UDP protocol implementation
- [smux](https://github.com/xtaci/smux) - Stream multiplexing library
- Go crypto packages for encryption implementations