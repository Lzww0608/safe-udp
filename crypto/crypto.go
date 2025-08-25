/*
@Author: Lzww
@LastEditTime: 2025-8-25 22:08:28
@Description: Crypto
@Language: Go 1.23.4
*/

package crypto

type BlockCrypt interface {
	// Encrypt a KCP segment. Input is a buffer from the buffer pool.
	// Returns a new buffer (from buffer pool) containing wire format data.
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt a wire format packet. Input is a buffer from the buffer pool.
	// Returns a new buffer (from buffer pool) containing KCP segment.
	Decrypt(ciphertext []byte) ([]byte, error)
}
