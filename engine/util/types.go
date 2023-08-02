package util

import "github.com/go-gl/mathgl/mgl32"

type Transform struct {
	Position mgl32.Vec3
	Rotation mgl32.Quat
	Scale    mgl32.Vec3
}
type VirtualInterface interface {
	Write(bytes []byte) (int, error)
	Read(p []byte) (n int, err error)
	Close() error
}
type ChannelWrapper struct {
	inputChannel  chan string
	outputChannel chan string
	readBuffer    []byte
}

func NewReverseChannelWrapper(originalWrapper *ChannelWrapper) *ChannelWrapper {
	c := &ChannelWrapper{inputChannel: originalWrapper.outputChannel, outputChannel: originalWrapper.inputChannel, readBuffer: []byte{}}
	return c
}
func NewChannelWrapper() *ChannelWrapper {
	c := &ChannelWrapper{inputChannel: make(chan string), outputChannel: make(chan string), readBuffer: []byte{}}
	return c
}

func (c *ChannelWrapper) Close() error {
	close(c.inputChannel)
	return nil
}

// Read reads up to len(p) bytes into p. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered. Even if Read
// returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally
// returns what is available instead of waiting for more.
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of
// bytes read. It may return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil. The next Read should
// return 0, EOF.
//
// Callers should always process the n > 0 bytes returned before
// considering the error err. Doing so correctly handles I/O errors
// that happen after reading some bytes and also both of the
// allowed EOF behaviors.
//
// Implementations of Read are discouraged from returning a
// zero byte count with a nil error, except when len(p) == 0.
// Callers should treat a return of 0 and nil as indicating that
// nothing happened; in particular it does not indicate EOF.
//
// Implementations must not retain p.

func (c *ChannelWrapper) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(c.readBuffer) == 0 {
		c.waitForData()
	}

	if len(c.readBuffer) < len(p) {
		bytesRead := len(c.readBuffer)
		copy(p, c.readBuffer)
		c.readBuffer = []byte{}
		return bytesRead, nil
	} else {
		copy(p, c.readBuffer[:len(p)])
		c.readBuffer = c.readBuffer[len(p):]
		return len(p), nil
	}
}

func (c *ChannelWrapper) waitForData() {
	message := <-c.inputChannel
	asBytes := []byte(message)
	c.readBuffer = append(c.readBuffer, asBytes...)
}

// Write writes len(p) bytes from p to the underlying data stream.
// It returns the number of bytes written from p (0 <= n <= len(p))
// and any error encountered that caused the write to stop early.
// Write must return a non-nil error if it returns n < len(p).
// Write must not modify the slice data, even temporarily.
//
// Implementations must not retain p.
func (c *ChannelWrapper) Write(p []byte) (int, error) {
	message := string(p)
	c.outputChannel <- message
	return len(p), nil
}

type Collider interface {
	FindFurthestPoint(direction mgl32.Vec3) mgl32.Vec3
	ToString() string
	Draw()
	GetName() string
	SetName(name string)
	IntersectsRay(start mgl32.Vec3, end mgl32.Vec3) (bool, mgl32.Vec3)
}
