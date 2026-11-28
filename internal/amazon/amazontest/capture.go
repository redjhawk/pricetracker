package amazontest

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"testing"
)

// Capture holds what the code under test logs (standard logger) and prints (os.Stdout) from
// StartCapture until End, which the test's cleanup also calls.
type Capture struct {
	logs, printed  outputBuffer
	previousLog    io.Writer
	previousStdout *os.File
	writer, reader *os.File
	copied         chan struct{}
	once           sync.Once
	text           string
}

// StartCapture starts capturing the logged and printed text. The pipe is created before anything
// is redirected, so a failed setup fails the test and leaves both destinations unchanged.
func StartCapture(t testing.TB) *Capture {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("capture the standard output: %v", err)
	}
	c := &Capture{writer: writer, reader: reader, copied: make(chan struct{})}
	c.previousLog = log.Writer()
	log.SetOutput(&c.logs)
	c.previousStdout = os.Stdout
	os.Stdout = writer
	go func() {
		_, _ = io.Copy(&c.printed, reader)
		close(c.copied)
	}()
	t.Cleanup(func() { c.End() })
	return c
}

// Live returns the text captured so far, logged text first, without ending the capture. Text still
// in the pipe appears in a later read.
func (c *Capture) Live() string {
	return c.logs.String() + c.printed.String()
}

// End ends the capture on its first call, restoring the previous destinations once every byte
// printed before it is read, and returns the logged text followed by the printed text. Later calls
// return the same text.
func (c *Capture) End() string {
	c.once.Do(func() {
		os.Stdout = c.previousStdout
		_ = c.writer.Close()
		<-c.copied
		_ = c.reader.Close()
		log.SetOutput(c.previousLog)
		c.text = c.logs.String() + c.printed.String()
	})
	return c.text
}

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
