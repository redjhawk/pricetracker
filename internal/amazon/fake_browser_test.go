package amazon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Fake browser of issue #233: the test binary run again in place of Chromium. It speaks the DevTools
// protocol on the remote-debugging pipe, follows a script for Page.navigate and Network.clearBrowserCache,
// and records each call received and each answer sent, so a test sees the order of the clear and Browser.close.

// fakeBrowserVariable holds the JSON script of the fake browser; set, the test binary runs as the fake.
const fakeBrowserVariable = "PRICEFOLLOWER_FAKE_BROWSER"

func TestMain(m *testing.M) {
	if script := os.Getenv(fakeBrowserVariable); script != "" {
		os.Exit(runFakeBrowser(script))
	}
	os.Exit(m.Run())
}

// fakeBrowserScript tells the fake browser how to answer.
type fakeBrowserScript struct {
	Record     string        `json:"record"`     // file receiving one JSON line per call received and per answer sent
	SetUpUntil int64         `json:"setUpUntil"` // Unix nanoseconds at which the last set-up call is answered; 0 = at once
	Navigate   string        `json:"navigate"`   // "error": answered at once with errorText; "hold": answered when Browser.close arrives; "late": answered at NavigateAt
	NavigateAt int64         `json:"navigateAt"` // "late": Unix nanoseconds of the Page.navigate answer
	Response   int           `json:"response"`   // status of the Document response event sent after the Page.navigate answer; 0 = none
	Clear      string        `json:"clear"`      // "answer": after ClearDelay; "never"; "error": protocol error ClearError after ClearDelay
	ClearDelay time.Duration `json:"clearDelay"`
	ClearError string        `json:"clearError"`
}

const (
	fakeFrameID   = "fake-frame"
	fakeSessionID = "fake-session"
	fakeErrorText = "net::ERR_ABORTED"
)

// fakeEvent is one line of the fake browser's record.
type fakeEvent struct {
	Method string `json:"method"`
	Kind   string `json:"kind"` // "received" or "answered"
	At     int64  `json:"at"`   // Unix nanoseconds
}

type fakeMessage struct {
	ID        int64           `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
}

type fakeBrowser struct {
	script fakeBrowserScript
	mu     sync.Mutex // serializes the record and the writes
	output *os.File
	record *os.File
	held   *fakeMessage // Page.navigate held until Browser.close
}

// runFakeBrowser serves the protocol on file descriptors 3 (read) and 4 (write) until Browser.close or
// the end of its input, and returns the exit code.
func runFakeBrowser(raw string) int {
	var script fakeBrowserScript
	if err := json.Unmarshal([]byte(raw), &script); err != nil {
		fmt.Fprintln(os.Stderr, "fake browser script:", err)
		return 2
	}
	record, err := os.OpenFile(script.Record, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- file of the test
	if err != nil {
		fmt.Fprintln(os.Stderr, "fake browser record:", err)
		return 2
	}
	defer record.Close()
	fake := &fakeBrowser{script: script, output: os.NewFile(4, "cdp-output"), record: record}
	reader := bufio.NewReaderSize(os.NewFile(3, "cdp-input"), 1<<20)
	for {
		data, err := reader.ReadBytes(0)
		if err != nil {
			return 0
		}
		var message fakeMessage
		if json.Unmarshal(data[:len(data)-1], &message) != nil {
			continue
		}
		fake.log(message.Method, "received")
		if fake.serve(message) {
			return 0
		}
	}
}

// serve answers one call; it returns true once the browser must exit.
func (f *fakeBrowser) serve(message fakeMessage) bool {
	switch message.Method {
	case "Browser.getVersion":
		f.answer(message, map[string]any{"product": "HeadlessChrome/120.0.0.0",
			"userAgent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36"})
	case "Target.createTarget":
		f.answer(message, map[string]any{"targetId": "fake-target"})
	case "Target.attachToTarget":
		f.answer(message, map[string]any{"sessionId": fakeSessionID})
	case "Emulation.setUserAgentOverride":
		go func() {
			time.Sleep(time.Until(time.Unix(0, f.script.SetUpUntil)))
			f.answer(message, map[string]any{})
		}()
	case "Runtime.evaluate":
		f.answer(message, map[string]any{"result": map[string]any{"value": ""}})
	case "Page.navigate":
		f.navigate(message)
	case "Network.clearBrowserCache":
		f.clear(message)
	case "Browser.close":
		f.mu.Lock()
		held := f.held
		f.held = nil
		f.mu.Unlock()
		if held != nil {
			f.answerNavigate(*held)
		}
		f.answer(message, map[string]any{})
		return true
	default: // Browser.setDownloadBehavior, Network.enable, Page.enable
		f.answer(message, map[string]any{})
	}
	return false
}

func (f *fakeBrowser) navigate(message fakeMessage) {
	switch f.script.Navigate {
	case "hold":
		f.mu.Lock()
		f.held = &message
		f.mu.Unlock()
	case "late":
		go func() {
			time.Sleep(time.Until(time.Unix(0, f.script.NavigateAt)))
			f.answerNavigate(message)
		}()
	default:
		f.answerNavigate(message)
		if f.script.Response != 0 {
			var params struct {
				URL string `json:"url"`
			}
			_ = json.Unmarshal(message.Params, &params)
			f.send("", map[string]any{"method": "Network.responseReceived", "sessionId": message.SessionID, "params": map[string]any{
				"requestId": "fake-request", "frameId": fakeFrameID, "type": "Document",
				"response": map[string]any{"url": params.URL, "status": f.script.Response, "mimeType": "text/html"}}})
		}
	}
}

func (f *fakeBrowser) answerNavigate(message fakeMessage) {
	f.answer(message, map[string]any{"frameId": fakeFrameID, "loaderId": "fake-loader", "errorText": fakeErrorText})
}

func (f *fakeBrowser) clear(message fakeMessage) {
	if f.script.Clear == "never" {
		return
	}
	go func() {
		time.Sleep(f.script.ClearDelay)
		if f.script.Clear == "error" {
			f.send(message.Method, map[string]any{"id": message.ID, "sessionId": message.SessionID,
				"error": map[string]any{"code": -32000, "message": f.script.ClearError}})
			return
		}
		f.answer(message, map[string]any{})
	}()
}

func (f *fakeBrowser) answer(message fakeMessage, result map[string]any) {
	reply := map[string]any{"id": message.ID, "result": result}
	if message.SessionID != "" {
		reply["sessionId"] = message.SessionID
	}
	f.send(message.Method, reply)
}

// send writes a message. An answer (method set) is recorded before it is written, so the record shows
// it before any call the collector makes after reading it.
func (f *fakeBrowser) send(method string, message map[string]any) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if method != "" {
		f.writeRecord(method, "answered")
	}
	_, _ = f.output.Write(append(data, 0))
}

func (f *fakeBrowser) log(method, kind string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeRecord(method, kind)
}

func (f *fakeBrowser) writeRecord(method, kind string) {
	data, _ := json.Marshal(fakeEvent{Method: method, Kind: kind, At: time.Now().UnixNano()})
	_, _ = f.record.Write(append(data, '\n'))
}

// fakeBrowserOptions returns the options of a collector starting the fake browser.
func fakeBrowserOptions(t *testing.T) BrowserOptions {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return BrowserOptions{ExecPath: executable, ProfileDir: filepath.Join(t.TempDir(), "profile"), NavigationTimeout: 30 * time.Second,
		Pause: func() time.Duration { return 0 }, Origin: "http://127.0.0.1:9"}
}

// useFakeBrowser makes the next fake browser started follow script and returns the path of its record.
func useFakeBrowser(t *testing.T, script fakeBrowserScript) string {
	t.Helper()
	script.Record = filepath.Join(t.TempDir(), "record.jsonl")
	data, err := json.Marshal(script)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeBrowserVariable, string(data))
	return script.Record
}

// fakeRecord is the record of a fake browser, in order.
type fakeRecord []fakeEvent

func readFakeRecord(t *testing.T, path string) fakeRecord {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- file of the test
	if err != nil {
		t.Fatalf("read the fake browser's record: %v", err)
	}
	var record fakeRecord
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var event fakeEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("fake browser record line %q: %v", line, err)
		}
		record = append(record, event)
	}
	return record
}

// first returns the index and time of the first event of method and kind, or -1.
func (r fakeRecord) first(method, kind string) (int, time.Time) {
	for index, event := range r {
		if event.Method == method && event.Kind == kind {
			return index, time.Unix(0, event.At)
		}
	}
	return -1, time.Time{}
}

// count returns the number of events of method and kind.
func (r fakeRecord) count(method, kind string) int {
	count := 0
	for _, event := range r {
		if event.Method == method && event.Kind == kind {
			count++
		}
	}
	return count
}

func (r fakeRecord) String() string {
	lines := make([]string, 0, len(r))
	for _, event := range r {
		lines = append(lines, fmt.Sprintf("%s %s %s", time.Unix(0, event.At).Format("15:04:05.000"), event.Kind, event.Method))
	}
	return strings.Join(lines, "\n")
}
