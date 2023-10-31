package cni

type ByteBuffer interface {
	// Write data from src to buffer
	// panic if no enough space
	Write(src []byte)

	// Read data from buffer to dst
	// returns the number of bytes actually read
	Read(dst []byte) int

	// Capacity  of this buffer
	Capacity() int

	// ReadableBytes how many bytes can be read
	ReadableBytes() int

	// ReadIndex read index
	ReadIndex() int

	// WriteIndex write index for next byte
	WriteIndex() int

	// Mark  current buffer status for subsequent recovery
	Mark()

	// Recover  buffer status to mark point
	// if the Mark method has not been called before, the behavior is unknown
	Recover()

	// Clean  status
	Clean()
}
