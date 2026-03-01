package poller

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockClipboard implements the Clipboard interface for testing.
type mockClipboard struct {
	mu          sync.Mutex
	checkFunc   func() ([]byte, error)
	updateFunc  func(wsl, win string) error
	closeCalled atomic.Bool
}

func (m *mockClipboard) Check() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.checkFunc != nil {
		return m.checkFunc()
	}
	return nil, nil
}

func (m *mockClipboard) UpdateClipboard(wsl, win string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateFunc != nil {
		return m.updateFunc(wsl, win)
	}
	return nil
}

func (m *mockClipboard) Close() error {
	m.closeCalled.Store(true)
	return nil
}

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

// overrideWslPath replaces wslToWinPath for the duration of a test.
func overrideWslPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := wslToWinPath
	wslToWinPath = fn
	t.Cleanup(func() { wslToWinPath = orig })
}

func fakeWslPath(wslPath string) (string, error) {
	return `C:\fake\` + filepath.Base(wslPath), nil
}

// --- hashBytes tests ---

func TestHashBytes(t *testing.T) {
	data := []byte("hello world")
	h := hashBytes(data)
	expected := fmt.Sprintf("%x", sha256.Sum256(data))
	if h != expected {
		t.Errorf("hashBytes(%q) = %q, want %q", data, h, expected)
	}

	// Determinism
	if h2 := hashBytes(data); h2 != h {
		t.Error("hashBytes is not deterministic")
	}

	// Different inputs produce different hashes
	other := hashBytes([]byte("different"))
	if other == h {
		t.Error("different inputs should produce different hashes")
	}
}

// --- poll tests ---

func TestPoll_NoImage(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	mock := &mockClipboard{checkFunc: func() ([]byte, error) { return nil, nil }}

	err := poll(mock, testLogger(), t.TempDir())
	if err != nil {
		t.Fatalf("poll() returned error: %v", err)
	}
}

func TestPoll_NewScreenshot(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	dir := t.TempDir()
	imgData := []byte("fake-png-data")

	var updateWsl, updateWin string
	mock := &mockClipboard{
		checkFunc: func() ([]byte, error) { return imgData, nil },
		updateFunc: func(wsl, win string) error {
			updateWsl = wsl
			updateWin = win
			return nil
		},
	}

	err := poll(mock, testLogger(), dir)
	if err != nil {
		t.Fatalf("poll() returned error: %v", err)
	}

	hash := hashBytes(imgData)
	expectedFile := filepath.Join(dir, hash+".png")

	saved, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
	if string(saved) != string(imgData) {
		t.Error("saved file content mismatch")
	}
	if updateWsl != expectedFile {
		t.Errorf("UpdateClipboard wslPath = %q, want %q", updateWsl, expectedFile)
	}
	if updateWin == "" {
		t.Error("UpdateClipboard winPath should not be empty")
	}
}

func TestPoll_Dedup(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	dir := t.TempDir()
	imgData := []byte("same-image")
	updateCount := 0

	mock := &mockClipboard{
		checkFunc: func() ([]byte, error) { return imgData, nil },
		updateFunc: func(wsl, win string) error {
			updateCount++
			return nil
		},
	}

	if err := poll(mock, testLogger(), dir); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	if err := poll(mock, testLogger(), dir); err != nil {
		t.Fatalf("second poll: %v", err)
	}

	if updateCount != 1 {
		t.Errorf("UpdateClipboard called %d times, want 1 (dedup)", updateCount)
	}
}

func TestPoll_CheckError(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	checkErr := errors.New("powershell died")
	mock := &mockClipboard{checkFunc: func() ([]byte, error) { return nil, checkErr }}

	err := poll(mock, testLogger(), t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, checkErr) {
		t.Errorf("error should wrap checkErr, got: %v", err)
	}
}

func TestPoll_WslPathFailure(t *testing.T) {
	overrideWslPath(t, func(string) (string, error) {
		return "", errors.New("wslpath not found")
	})
	dir := t.TempDir()
	imgData := []byte("some-image")
	mock := &mockClipboard{
		checkFunc: func() ([]byte, error) { return imgData, nil },
	}

	err := poll(mock, testLogger(), dir)
	if err != nil {
		t.Fatalf("poll should not return error on wslpath failure: %v", err)
	}

	// File should still be saved
	hash := hashBytes(imgData)
	if _, err := os.Stat(filepath.Join(dir, hash+".png")); err != nil {
		t.Error("file should still be saved even when wslpath fails")
	}
}

func TestPoll_UpdateFailure(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	dir := t.TempDir()
	imgData := []byte("image-data")
	mock := &mockClipboard{
		checkFunc:  func() ([]byte, error) { return imgData, nil },
		updateFunc: func(wsl, win string) error { return errors.New("update failed") },
	}

	err := poll(mock, testLogger(), dir)
	if err != nil {
		t.Fatalf("poll should not return error on update failure: %v", err)
	}

	hash := hashBytes(imgData)
	if _, err := os.Stat(filepath.Join(dir, hash+".png")); err != nil {
		t.Error("file should still be saved even when update fails")
	}
}

// --- Run tests ---

func TestRun_ShutdownCallsClose(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	mock := &mockClipboard{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, testLogger(), 100, t.TempDir(), func() (Clipboard, error) {
			return mock, nil
		})
	}()

	// Let it run a tick or two
	time.Sleep(250 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit after context cancel")
	}

	if !mock.closeCalled.Load() {
		t.Error("Close() was not called on shutdown")
	}
}

func TestRun_CircuitBreakerRestart(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	factoryCalls := 0
	checkErr := errors.New("persistent error")

	var mu sync.Mutex
	var activeMock *mockClipboard

	factory := func() (Clipboard, error) {
		mu.Lock()
		defer mu.Unlock()
		factoryCalls++
		m := &mockClipboard{
			checkFunc: func() ([]byte, error) { return nil, checkErr },
		}
		activeMock = m
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, testLogger(), 100, t.TempDir(), factory)
	}()

	// Wait for circuit breaker to trigger (5 errors * 100ms interval + margin)
	time.Sleep(800 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit")
	}

	mu.Lock()
	calls := factoryCalls
	mu.Unlock()
	if calls < 2 {
		t.Errorf("factory called %d times, want >= 2 (circuit breaker should restart)", calls)
	}

	_ = activeMock // just verify it was assigned
}

func TestRun_ShutdownClosesLatestClient(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	checkErr := errors.New("persistent error")

	var clients []*mockClipboard
	var mu sync.Mutex

	factory := func() (Clipboard, error) {
		mu.Lock()
		defer mu.Unlock()
		m := &mockClipboard{
			checkFunc: func() ([]byte, error) { return nil, checkErr },
		}
		clients = append(clients, m)
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, testLogger(), 100, t.TempDir(), factory)
	}()

	// Wait for at least one circuit breaker restart
	time.Sleep(800 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(clients) < 2 {
		t.Fatalf("expected >= 2 clients (circuit breaker restart), got %d", len(clients))
	}

	// The LAST client should have Close called (via the deferred func)
	last := clients[len(clients)-1]
	if !last.closeCalled.Load() {
		t.Error("Close() was not called on the latest client after shutdown")
	}
}

// --- Integration test ---

func TestIntegration_SignalCausesCloseAndExit(t *testing.T) {
	overrideWslPath(t, fakeWslPath)
	dir := t.TempDir()

	var pollCount atomic.Int32
	mock := &mockClipboard{
		checkFunc: func() ([]byte, error) {
			pollCount.Add(1)
			return nil, nil // no image
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, testLogger(), 100, dir, func() (Clipboard, error) {
			return mock, nil
		})
	}()

	// Let it tick a few times
	time.Sleep(350 * time.Millisecond)

	if pollCount.Load() < 2 {
		t.Errorf("expected at least 2 polls, got %d", pollCount.Load())
	}

	// Simulate SIGINT/SIGTERM by cancelling context
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit within 5 seconds of context cancel")
	}

	if !mock.closeCalled.Load() {
		t.Error("Close() was not called on the active client after signal")
	}
}
