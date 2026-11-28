package amazontest

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Tests of issue #218: CaptureOutput (and StartCapture with its live and ending reads) captures what
// the code under test logs and prints, and restores the previous destinations when it ends. TC-010
// (no local capture copy left), TC-011 (the former callers' tests of internal/amazon, internal/service
// and internal/httpapi list the same tests as on master and pass, and the reverted Live fault
// injection fails waitStartFailures), TC-012 (TestServerDoesNotDependOnTestHelper still passes) and
// TC-013 (diff, gofmt, vet and go test) are repository checks and runs, recorded by the development
// stage. No test is parallel: all of them change the standard logger and os.Stdout.

const setupFailureEnv = "AMAZONTEST_CAPTURE_SETUP_FAILURE"

// previousDestinations points the standard logger to a new buffer and os.Stdout to a new file of
// t.TempDir() until the end of the test, and returns both.
func previousDestinations(t *testing.T) (*outputBuffer, *os.File) {
	t.Helper()
	file, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	logs := &outputBuffer{}
	originalLog, originalStdout := log.Writer(), os.Stdout
	log.SetOutput(logs)
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = originalStdout
		log.SetOutput(originalLog)
		file.Close()
	})
	return logs, file
}

// received returns what the previous destinations received.
func received(t *testing.T, logs *outputBuffer, file *os.File) (logged, printed string) {
	t.Helper()
	content, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return logs.String(), string(content)
}

// assertUntouched checks that the previous destinations received nothing.
func assertUntouched(t *testing.T, logs *outputBuffer, file *os.File) {
	t.Helper()
	if logged, printed := received(t, logs, file); logged != "" || printed != "" {
		t.Errorf("the previous destinations received text during the capture: logged %q, printed %q", logged, printed)
	}
}

// assertRestored checks that the standard logger and os.Stdout are the previous destinations again.
func assertRestored(t *testing.T, logs *outputBuffer, file *os.File) {
	t.Helper()
	if log.Writer() != io.Writer(logs) {
		t.Errorf("log writes to %T %p, want the previous buffer %p", log.Writer(), log.Writer(), logs)
	}
	if os.Stdout != file {
		t.Errorf("os.Stdout is %v, want the previous file %s", os.Stdout.Name(), file.Name())
	}
}

// within returns the result of read, failing the test when it takes 1 s or more.
func within(t *testing.T, name string, read func() string) string {
	t.Helper()
	result := make(chan string, 1)
	start := time.Now()
	go func() { result <- read() }()
	select {
	case text := <-result:
		if elapsed := time.Since(start); elapsed >= time.Second {
			t.Errorf("%s took %s, want less than 1 s", name, elapsed)
		}
		return text
	case <-time.After(time.Second):
		t.Fatalf("%s did not return within 1 s", name)
		return ""
	}
}

// completeLines returns the lines of text that end with a newline.
func completeLines(text string) []string {
	end := strings.LastIndex(text, "\n")
	if end < 0 {
		return nil
	}
	return strings.Split(text[:end], "\n")
}

// TC-001 (issue #218)
func TestCaptureOutput_Issue218_TC001_Signature(t *testing.T) {
	var capture func(*testing.T) func() string = CaptureOutput
	if capture == nil {
		t.Fatal("CaptureOutput is nil")
	}
}

// TC-002 (issue #218)
func TestCaptureOutput_Issue218_TC002_LoggedThenPrinted(t *testing.T) {
	t.Run("logged text before printed text", func(t *testing.T) {
		logs, file := previousDestinations(t)
		read := CaptureOutput(t)
		fmt.Println("printed")
		log.Print("logged")
		assertUntouched(t, logs, file)
		if text := read(); !strings.HasSuffix(text, "logged\nprinted\n") {
			t.Errorf("ending read %q, want it to end with %q", text, "logged\nprinted\n")
		}
		assertRestored(t, logs, file)
		assertUntouched(t, logs, file)
	})

	t.Run("several lines keep their order in each stream", func(t *testing.T) {
		logs, file := previousDestinations(t)
		read := CaptureOutput(t)
		log.Print("log-1")
		fmt.Println("print-1")
		log.Print("log-2")
		fmt.Println("print-2")
		text := read()
		lines := completeLines(text)
		if len(lines) != 4 || !strings.HasSuffix(lines[0], " log-1") || !strings.HasSuffix(lines[1], " log-2") ||
			lines[2] != "print-1" || lines[3] != "print-2" || !strings.HasSuffix(text, "\n") {
			t.Errorf("ending read %q, want the lines log-1, log-2 (logged) then print-1, print-2", text)
		}
		assertRestored(t, logs, file)
		assertUntouched(t, logs, file)
	})

	t.Run("text printed without a final newline", func(t *testing.T) {
		logs, file := previousDestinations(t)
		read := CaptureOutput(t)
		fmt.Print("partial")
		if text := read(); text != "partial" {
			t.Errorf("ending read %q, want %q", text, "partial")
		}
		assertRestored(t, logs, file)
		assertUntouched(t, logs, file)
	})

	t.Run("no write", func(t *testing.T) {
		logs, file := previousDestinations(t)
		read := CaptureOutput(t)
		if text := read(); text != "" {
			t.Errorf("ending read %q, want the empty string", text)
		}
		assertRestored(t, logs, file)
	})
}

// TC-003 (issue #218)
func TestCaptureOutput_Issue218_TC003_LaterCallsSameText(t *testing.T) {
	t.Run("later calls", func(t *testing.T) {
		logs, file := previousDestinations(t)
		read := CaptureOutput(t)
		log.Print("logged")
		fmt.Println("printed")
		first := read()
		if !strings.HasSuffix(first, "logged\nprinted\n") {
			t.Errorf("first read %q, want it to end with %q", first, "logged\nprinted\n")
		}
		log.Print("after-log")
		fmt.Println("after-print")
		for _, call := range []string{"second call", "third call"} {
			if text := within(t, call, read); text != first {
				t.Errorf("%s returned %q, want the first call's %q", call, text, first)
			}
		}
		if strings.Contains(first, "after-") {
			t.Errorf("first read %q holds text written after it", first)
		}
		logged, printed := received(t, logs, file)
		if !strings.Contains(logged, "after-log") || strings.Contains(logged, "after-print") {
			t.Errorf("previous logger buffer %q, want after-log only", logged)
		}
		if printed != "after-print\n" {
			t.Errorf("previous file %q, want %q", printed, "after-print\n")
		}
	})

	t.Run("live read after the end", func(t *testing.T) {
		logs, file := previousDestinations(t)
		capture := StartCapture(t)
		log.Print("logged")
		fmt.Println("printed")
		ended := capture.End()
		if !strings.HasSuffix(ended, "logged\nprinted\n") {
			t.Errorf("ending read %q, want it to end with %q", ended, "logged\nprinted\n")
		}
		log.Print("after-log")
		fmt.Println("after-print")
		if live := within(t, "live read after the end", capture.Live); live != ended {
			t.Errorf("live read after the end %q, want the ending read %q", live, ended)
		}
		if again := within(t, "second End", capture.End); again != ended {
			t.Errorf("second End %q, want %q", again, ended)
		}
		assertRestored(t, logs, file)
	})
}

// TC-004 (issue #218)
func TestCaptureOutput_Issue218_TC004_WaitsForPrintedText(t *testing.T) {
	payload := strings.Repeat("0123456789abcdef", 65536) // 1 MiB, larger than a pipe buffer
	want := payload + "marker\n"
	check := func(t *testing.T, text string) {
		t.Helper()
		if text != want {
			tail := text[max(0, len(text)-40):]
			t.Errorf("ending read of %d bytes ending with %q, want the %d bytes printed ending with %q", len(text), tail, len(want), "marker\n")
		}
	}

	t.Run("printed by the test", func(t *testing.T) {
		previousDestinations(t) // the printed text stays out of the test output if it is not captured
		read := CaptureOutput(t)
		fmt.Print(payload)
		fmt.Println("marker")
		check(t, read())
	})

	t.Run("printed by a goroutine that has returned", func(t *testing.T) {
		previousDestinations(t)
		read := CaptureOutput(t)
		var printer sync.WaitGroup
		printer.Add(1)
		go func() {
			defer printer.Done()
			fmt.Print(payload)
			fmt.Println("marker")
		}()
		printer.Wait()
		check(t, read())
	})
}

// TC-005 (issue #218)
func TestCaptureOutput_Issue218_TC005_EndsAtTestEnd(t *testing.T) {
	logs, file := previousDestinations(t)
	var read func() string
	t.Run("capture never read", func(t *testing.T) {
		read = CaptureOutput(t)
		log.Print("logged")
		fmt.Println("printed")
	})
	assertRestored(t, logs, file)
	assertUntouched(t, logs, file)

	log.Print("parent-log")
	fmt.Println("parent-print")
	text := within(t, "the kept function", read)
	if !strings.HasSuffix(text, "logged\nprinted\n") {
		t.Errorf("kept function returned %q, want it to end with %q", text, "logged\nprinted\n")
	}
	if strings.Contains(text, "parent-") {
		t.Errorf("kept function returned %q, holding text written after the subtest", text)
	}
	logged, printed := received(t, logs, file)
	if !strings.Contains(logged, "parent-log") || printed != "parent-print\n" {
		t.Errorf("previous destinations: logged %q, printed %q; want parent-log and %q", logged, printed, "parent-print\n")
	}
}

// TC-006 (issue #218)
func TestCaptureOutput_Issue218_TC006_SequentialCaptures(t *testing.T) {
	logs, file := previousDestinations(t)

	t.Run("one after the other", func(t *testing.T) {
		first := CaptureOutput(t)
		log.Print("one")
		one := first()
		second := CaptureOutput(t)
		fmt.Println("two")
		two := second()
		if !strings.Contains(one, "one") || strings.Contains(one, "two") {
			t.Errorf("first capture %q, want one and not two", one)
		}
		if !strings.Contains(two, "two") || strings.Contains(two, "one") {
			t.Errorf("second capture %q, want two and not one", two)
		}
	})
	assertRestored(t, logs, file)
	assertUntouched(t, logs, file)

	t.Run("second started before the first ends, ended in reverse order", func(t *testing.T) {
		first := CaptureOutput(t)
		log.Print("one")
		second := CaptureOutput(t)
		fmt.Println("two")
		two := second()
		one := first()
		if !strings.Contains(one, "one") {
			t.Errorf("first capture %q, want one", one)
		}
		if !strings.Contains(two, "two") {
			t.Errorf("second capture %q, want two", two)
		}
	})
	assertRestored(t, logs, file)
	assertUntouched(t, logs, file)
}

// TC-007 (issue #218)
func TestCaptureOutput_Issue218_TC007_LiveRead(t *testing.T) {
	logs, file := previousDestinations(t)
	capture := StartCapture(t)
	var reads []string
	live := func() string {
		t.Helper()
		text := within(t, "live read", capture.Live)
		reads = append(reads, text)
		return text
	}

	if text := live(); text != "" {
		t.Errorf("live read before any write %q, want the empty string", text)
	}
	log.Print("logged")
	if text := live(); !strings.Contains(text, "logged") {
		t.Errorf("live read after log.Print %q, want logged", text)
	}
	if again := live(); again != reads[len(reads)-2] {
		t.Errorf("live read with no write in between %q, want %q", again, reads[len(reads)-2])
	}
	fmt.Println("printed")
	for deadline := time.Now().Add(5 * time.Second); !strings.Contains(live(), "printed"); {
		if time.Now().After(deadline) {
			t.Fatalf("no live read within 5 s holds printed; last %q", reads[len(reads)-1])
		}
		time.Sleep(10 * time.Millisecond)
	}
	if again := live(); again != reads[len(reads)-2] {
		t.Errorf("live read with no write in between %q, want %q", again, reads[len(reads)-2])
	}
	if os.Stdout == file {
		t.Errorf("os.Stdout is the previous file after the live reads: the capture ended")
	}
	assertUntouched(t, logs, file)

	log.Print("later")
	ended := capture.End()
	for _, word := range []string{"logged", "printed", "later"} {
		if !strings.Contains(ended, word) {
			t.Errorf("ending read %q, want %s", ended, word)
		}
	}
	endedLines := map[string]bool{}
	for _, line := range completeLines(ended) {
		endedLines[line] = true
	}
	for _, text := range reads {
		for _, line := range completeLines(text) {
			if line != "" && !endedLines[line] {
				t.Errorf("live read line %q is not in the ending read %q", line, ended)
			}
		}
	}
	assertRestored(t, logs, file)
}

// TC-008 (issue #218)
func TestCaptureOutput_Issue218_TC008_ConcurrentWritesNoRace(t *testing.T) {
	const writers, lines = 8, 100
	previousDestinations(t)
	capture := StartCapture(t)
	done, readerStopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(readerStopped)
		for {
			select {
			case <-done:
				return
			default:
				capture.Live()
			}
		}
	}()
	var writing sync.WaitGroup
	for writer := range writers {
		writing.Add(1)
		go func() {
			defer writing.Done()
			for line := range lines {
				if writer%2 == 0 {
					log.Printf("log-%d-%03d", writer, line)
				} else {
					fmt.Printf("print-%d-%03d\n", writer, line)
				}
			}
		}()
	}
	writing.Wait()
	close(done)
	<-readerStopped
	text := capture.End()

	seen := map[string]int{}
	lastLog, firstPrint := -1, -1
	for index, line := range completeLines(text) {
		if strings.HasPrefix(line, "print-") {
			seen[line]++
			if firstPrint < 0 {
				firstPrint = index
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[len(fields)-1], "log-") {
			t.Errorf("unexpected line %q", line)
			continue
		}
		seen[fields[len(fields)-1]]++
		lastLog = index
	}
	for writer := range writers {
		kind := "print"
		if writer%2 == 0 {
			kind = "log"
		}
		for line := range lines {
			if name := fmt.Sprintf("%s-%d-%03d", kind, writer, line); seen[name] != 1 {
				t.Errorf("line %s appears %d times in the ending read, want once", name, seen[name])
			}
		}
	}
	if lastLog < 0 || firstPrint < 0 || lastLog > firstPrint {
		t.Errorf("last logged line at %d, first printed line at %d: want every logged line before the printed lines", lastLog, firstPrint)
	}
}

// TC-009 (issue #218)
func TestCaptureOutput_Issue218_TC009_SetupFailureFailsTheTest(t *testing.T) {
	if os.Getenv(setupFailureEnv) == "1" {
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
			t.Fatal(err)
		}
		limit.Cur = min(limit.Max, 64)
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
			t.Fatal(err)
		}
		exhausted := false
		for range 64 { // the files stay open until the subprocess exits
			if _, err := os.Open(os.DevNull); errors.Is(err, syscall.EMFILE) {
				exhausted = true
				break
			} else if err != nil {
				t.Fatal(err)
			}
		}
		if !exhausted {
			t.Fatal("the file descriptors were not used up")
		}
		CaptureOutput(t)
		t.Log("capture set up")
		return
	}

	// #nosec G204 -- test binary
	command := exec.Command(os.Args[0], "-test.run=^TestCaptureOutput_Issue218_TC009_SetupFailureFailsTheTest$", "-test.count=1", "-test.v")
	command.Env = append(os.Environ(), setupFailureEnv+"=1")
	output, err := command.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the subprocess did not fail (%v):\n%s", err, output)
	}
	for _, want := range []string{"too many open files", "--- FAIL"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("subprocess output lacks %q:\n%s", want, output)
		}
	}
	if strings.Contains(string(output), "capture set up") {
		t.Errorf("the subprocess went on after the failed setup:\n%s", output)
	}
}
