package amazon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
)

// cdpBrowser is a Chromium process driven through the Chrome DevTools Protocol over
// --remote-debugging-pipe: JSON messages separated by NUL bytes, read by the browser on file
// descriptor 3 and written by it on file descriptor 4.
type cdpBrowser struct {
	cmd      *exec.Cmd
	input    *os.File // written by us, read by the browser
	output   *os.File // written by the browser, read by us
	writeMu  sync.Mutex
	mu       sync.Mutex
	nextID   int64
	pending  map[int64]chan cdpMessage
	handlers map[string]func(method string, params json.RawMessage) // by session ID
	read     chan struct{}                                          // closed when the reader stops
	exited   chan struct{}                                          // closed when the process has exited
	stderr   *tailBuffer                                            // end of the browser's error output
}

// tailBuffer keeps the last bytes written to it.
type tailBuffer struct {
	mu   sync.Mutex
	data []byte
}

const stderrTailBytes = 2048

func (t *tailBuffer) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, data...)
	if len(t.data) > stderrTailBytes {
		t.data = append([]byte(nil), t.data[len(t.data)-stderrTailBytes:]...)
	}
	return len(data), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.data)
}

type cdpMessage struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

var errBrowserExited = errors.New("the browser exited")

// launchBrowser starts the browser executable with args. The browser runs in its own process
// group, so close can end it with every child process.
func launchBrowser(executable string, args []string) (*cdpBrowser, error) {
	browserInput, input, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	output, browserOutput, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(err, browserInput.Close(), input.Close())
	}
	cmd := exec.Command(executable, args...) // #nosec G204 -- executable from the configuration, arguments built by the code
	cmd.ExtraFiles = []*os.File{browserInput, browserOutput}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stderr := &tailBuffer{}
	cmd.Stderr = stderr
	cmd.WaitDelay = time.Second // child processes keeping the error output open delay Wait at most this long
	err = cmd.Start()
	// The browser has its own copies of these ends; failing to close the parent's changes nothing.
	_ = browserInput.Close()
	_ = browserOutput.Close()
	if err != nil {
		return nil, errors.Join(err, input.Close(), output.Close())
	}
	b := &cdpBrowser{cmd: cmd, input: input, output: output, pending: map[int64]chan cdpMessage{},
		handlers: map[string]func(string, json.RawMessage){}, read: make(chan struct{}), exited: make(chan struct{}), stderr: stderr}
	go func() { _ = cmd.Wait(); close(b.exited) }() // the exit status is read from cmd.ProcessState
	go b.readMessages()
	return b, nil
}

// exitError returns errBrowserExited with the browser's exit status and the end of its error output,
// waiting up to 2 s for the process to exit.
func (b *cdpBrowser) exitError() error {
	select {
	case <-b.exited:
	case <-time.After(2 * time.Second):
		return errBrowserExited
	}
	status := b.cmd.ProcessState.String() // set by Wait before exited is closed
	if tail := strings.TrimSpace(b.stderr.String()); tail != "" {
		return fmt.Errorf("%w (%s): %s", errBrowserExited, status, lastBytes(tail, exitReasonBytes))
	}
	return fmt.Errorf("%w (%s)", errBrowserExited, status)
}

// exitReasonBytes bounds the error output in an exit error so that, with the cause prefixes, it fits
// in the 1,024 bytes of the logged cause: the browser usually prints its fatal reason last.
const exitReasonBytes = 768

// lastBytes returns the end of text in at most limit bytes, starting on a rune boundary and marked
// with a leading ellipsis when the start was dropped.
func lastBytes(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	const ellipsis = "…"
	start := len(text) - (limit - len(ellipsis))
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return ellipsis + text[start:]
}

// readMessages dispatches the browser's messages: responses to their call, events to the handler of
// their session.
func (b *cdpBrowser) readMessages() {
	defer close(b.read)
	reader := bufio.NewReaderSize(b.output, 1<<20)
	for {
		data, err := reader.ReadBytes(0)
		if err != nil {
			return
		}
		var message cdpMessage
		if json.Unmarshal(data[:len(data)-1], &message) != nil {
			continue
		}
		b.mu.Lock()
		if message.ID != 0 {
			if response, ok := b.pending[message.ID]; ok {
				response <- message
				delete(b.pending, message.ID)
			}
		}
		handler := b.handlers[message.SessionID]
		b.mu.Unlock()
		if message.ID == 0 && handler != nil {
			handler(message.Method, message.Params)
		}
	}
}

// on sets the event handler of a session. It runs on the reader goroutine and must not block.
func (b *cdpBrowser) on(sessionID string, handler func(method string, params json.RawMessage)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[sessionID] = handler
}

// call sends method to the browser (sessionID "" for the browser itself) and stores the answer in
// result unless result is nil.
func (b *cdpBrowser) call(ctx context.Context, sessionID, method string, params, result any) error {
	response := make(chan cdpMessage, 1)
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.pending[id] = response
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
	}()
	request := map[string]any{"id": id, "method": method}
	if params != nil {
		request["params"] = params
	}
	if sessionID != "" {
		request["sessionId"] = sessionID
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	b.writeMu.Lock()
	_, err = b.input.Write(append(data, 0))
	b.writeMu.Unlock()
	if err != nil {
		return b.exitError()
	}
	select {
	case message := <-response:
		if message.Error != nil {
			return fmt.Errorf("%s: %s", method, message.Error.Message)
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(message.Result, result)
	case <-b.read:
		return b.exitError()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// close ends the browser: it asks it to close, waits up to 5 s, then kills its process group. It
// returns once the process has exited and the reader has stopped.
func (b *cdpBrowser) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = b.call(ctx, "", "Browser.close", nil, nil) // the process group is killed below anyway
	cancel()
	select {
	case <-b.exited:
	case <-time.After(5 * time.Second):
	}
	// Also ends the child processes still running after the browser itself exited.
	// The group may already be gone; the browser has exited either way once b.exited is closed.
	_ = syscall.Kill(-b.cmd.Process.Pid, syscall.SIGKILL)
	<-b.exited
	_ = b.input.Close()
	_ = b.output.Close()
	<-b.read
}
