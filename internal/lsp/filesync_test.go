package lsp

import (
	"testing"
	"time"
)

func TestFileSyncRefCounting(t *testing.T) {
	var opens, closes int
	fs := NewFileSync(func(string) error { opens++; return nil }, func(string) error { closes++; return nil })
	fs.SetTTL(20 * time.Millisecond)

	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if opens != 1 || fs.RefCount("a.go") != 2 {
		t.Fatalf("open refs = %d/%d, want 1/2", opens, fs.RefCount("a.go"))
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	if closes != 0 || fs.RefCount("a.go") != 1 {
		t.Fatalf("close refs = %d/%d, want 0/1", closes, fs.RefCount("a.go"))
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	if closes != 0 || fs.RefCount("a.go") != 0 {
		t.Fatalf("close refs = %d/%d, want 0/0 before TTL", closes, fs.RefCount("a.go"))
	}
	waitForClose(t, &closes, 1)
}

func TestFileSyncKeepsFileOpenUntilTTLExpires(t *testing.T) {
	var opens, closes int
	fs := NewFileSync(func(string) error { opens++; return nil }, func(string) error { closes++; return nil })
	fs.SetTTL(50 * time.Millisecond)

	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	if opens != 1 || closes != 0 {
		t.Fatalf("opens/closes = %d/%d, want 1/0 before TTL", opens, closes)
	}
}

func TestFileSyncClosesAfterTTLExpires(t *testing.T) {
	var closes int
	fs := NewFileSync(nil, func(string) error { closes++; return nil })
	fs.SetTTL(10 * time.Millisecond)

	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	waitForClose(t, &closes, 1)
}

func TestFileSyncReopenBeforeTTLCancelsClose(t *testing.T) {
	var opens, closes int
	fs := NewFileSync(func(string) error { opens++; return nil }, func(string) error { closes++; return nil })
	fs.SetTTL(30 * time.Millisecond)

	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if opens != 1 || closes != 0 || fs.RefCount("a.go") != 1 {
		t.Fatalf("opens/closes/refs = %d/%d/%d, want 1/0/1", opens, closes, fs.RefCount("a.go"))
	}
}

func TestFileSyncCloseAllClosesImmediately(t *testing.T) {
	var closes int
	fs := NewFileSync(nil, func(string) error { closes++; return nil })
	fs.SetTTL(time.Hour)

	if err := fs.Open("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Open("b.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Close("a.go"); err != nil {
		t.Fatal(err)
	}
	if err := fs.CloseAll(); err != nil {
		t.Fatal(err)
	}
	if closes != 2 || fs.RefCount("a.go") != 0 || fs.RefCount("b.go") != 0 {
		t.Fatalf("closes/refs = %d/%d/%d, want 2/0/0", closes, fs.RefCount("a.go"), fs.RefCount("b.go"))
	}
}

func waitForClose(t *testing.T, closes *int, want int) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if *closes == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("closes = %d, want %d", *closes, want)
}
