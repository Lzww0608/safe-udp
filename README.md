# Safe-UDP

An experimental UDP communication library that implements KCP protocol with optional encryption and Forward Error Correction (FEC) capabilities.

## Project Status

This is a research/experimental project. The following components have been implemented and tested:

### ✅ Completed Components

- **KCP Protocol Implementation**: Custom implementation of KCP reliable UDP protocol
- **FEC (Forward Error Correction)**: Reed-Solomon based error correction with auto-tuning
- **Multiple Encryption Support**: Various cipher implementations (AES, Salsa20, SM4, etc.)
- **Session Management**: UDP session handling with proper connection lifecycle
- **Buffer Management**: Ring buffer implementation for efficient data handling
- **Statistics Collection**: SNMP-style statistics for protocol monitoring
- **Timer System**: Multi-goroutine timer management system
- **Batch Operations**: Optimized batch UDP read/write operations

### 🔧 Recently Fixed Issues

- Fixed naming inconsistencies across modules
- Resolved type conflicts between different Listener implementations
- Added missing transmission methods (tx, batchTx, defaultTx)
- Implemented missing readLoop and monitor methods
- Fixed SNMP field name mismatches
- Corrected timer system references

### ⚠️ Known Limitations

- Some encryption algorithms marked as deprecated by Go crypto standards
- Unused legacy interfaces retained for potential future use
- No high-level public API implemented yet
- Limited documentation and examples

## Architecture

### File Structure
```
safe-udp/
├── safeudp.go          # Core KCP protocol and utilities
├── session.go          # UDP session management and I/O operations
├── fec.go              # Forward Error Correction implementation  
├── tx.go               # Packet transmission methods
├── crypt.go            # Multiple encryption algorithm support
├── snmp.go             # Statistics collection and monitoring
├── timers.go           # Timer management system
├── ringbuffer.go       # Efficient ring buffer implementation
├── batchconn.go        # Batch UDP operations interface
├── entropy.go          # Nonce generation for encryption
├── autotune.go         # FEC parameter auto-tuning
├── listener.go         # High-level stream listener wrapper
├── conn.go             # Connection wrapper interface
└── crypto/crypto.go    # Encryption interface definition
```

### Core Features

1. **KCP Reliable UDP**
   - Implemented ARQ (Automatic Repeat reQuest)
   - Congestion control with customizable parameters
   - Fast retransmission and early retransmission
   - Window-based flow control

2. **Forward Error Correction**
   - Reed-Solomon error correction codes
   - Auto-tuning mechanism for optimal performance
   - Configurable data/parity shard ratios
   - Packet recovery without retransmission

3. **Encryption Support**
   - Multiple cipher implementations available
   - Block cipher modes with proper padding
   - Nonce generation for secure encryption
   - Configurable encryption algorithms

4. **Performance Features**
   - Memory pool management to reduce GC pressure
   - Batch packet operations for improved throughput
   - Ring buffer for efficient data handling
   - Multi-threaded timer system

## Configuration

The `Config` struct in `safeudp.go` supports the following parameters:

```go
type Config struct {
    Key []byte    // Encryption key (32 bytes recommended)
    
    // FEC settings
    FECData   int // Number of data packets in FEC group
    FECParity int // Number of parity packets in FEC group
    
    // KCP settings  
    NoDelay      int // Enable nodelay mode
    Interval     int // Internal update timer interval (ms)
    Resend       int // Fast resend mode
    NoCongestion int // Disable congestion control
    
    // Buffer settings
    SendBuffer int // Send buffer size
    RecvBuffer int // Receive buffer size
}
```

## Testing

The project includes comprehensive unit tests:

```bash
go test -v
```

Test coverage includes:
- Ring buffer operations
- Transmission method functionality  
- Session readLoop and monitor operations
- Naming consistency verification
- Error handling scenarios

## Dependencies

- `github.com/klauspost/reedsolomon` - Reed-Solomon FEC implementation
- `github.com/pkg/errors` - Enhanced error handling
- `github.com/xtaci/smux` - Stream multiplexing (used by high-level wrapper)
- `golang.org/x/crypto` - Additional cryptographic functions
- `golang.org/x/net` - Network utilities for batch operations

## Development Status

This project is in active development. The core protocol implementation is functional and tested, but lacks a stable public API. Use at your own risk in production environments.

## License

MIT License - see [LICENSE](LICENSE) file for details.