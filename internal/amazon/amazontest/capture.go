package amazontest

import (
	"bytes"
	"sync"
	"testing"
)

// Capture holds what the code under test logs (standard logger) and prints (os.Stdout) from
// StartCapture until End, which the test's cleanup also calls. Issue #218: not implemented yet.
type Capture struct{}

// StartCapture starts capturing the logged and printed text. Not implemented yet.
func StartCapture(t testing.TB) *Capture {
	t.Helper()
	return &Capture{}
}

// Live returns the text captured so far without ending the capture. Not implemented yet.
func (c *Capture) Live() string { return "" }

// End ends the capture and returns the logged text followed by the printed text. Not implemented yet.
func (c *Capture) End() string { return "" }

// CaptureOutput captures the logged and printed text until the returned function is first called
// or the test ends; the function returns the logged text followed by the printed text.
func CaptureOutput(t *testing.T) func() string {
	t.Helper()
	return StartCapture(t).End
}

// outputBuffer is a bytes.Buffer safe for concurrent writes and reads.
type outputBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *outputBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *outputBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
