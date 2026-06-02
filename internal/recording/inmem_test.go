package recording

import (
	"errors"
	"testing"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

func TestInMemoryRecorderInitializeStoresBacklog(t *testing.T) {
	r := NewInMemoryRecorder()
	backlog := []session.Execution{
		{Stdout: "a\n"},
		{Stdout: "b\n"},
	}
	if err := r.Initialize("cmd", time.Second, backlog); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := r.Frames(); len(got) != 2 || got[0].Stdout != "a\n" || got[1].Stdout != "b\n" {
		t.Errorf("Frames after Initialize: %+v", got)
	}
	h := r.Header()
	if h.Command != "cmd" || h.Interval != time.Second {
		t.Errorf("Header after Initialize: %+v", h)
	}
}

func TestInMemoryRecorderWriteFrameAppends(t *testing.T) {
	r := NewInMemoryRecorder()
	_ = r.Initialize("x", time.Second, nil)
	if err := r.WriteFrame(session.Execution{Stdout: "c\n"}); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	if got := r.Frames(); len(got) != 1 || got[0].Stdout != "c\n" {
		t.Errorf("Frames after WriteFrame: %+v", got)
	}
}

func TestInMemoryRecorderCloseIdempotent(t *testing.T) {
	r := NewInMemoryRecorder()
	_ = r.Initialize("x", time.Second, nil)
	if err := r.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if !r.Closed() {
		t.Errorf("Closed() should be true after Close")
	}
}

func TestInMemoryRecorderFailWriteAfter(t *testing.T) {
	r := NewInMemoryRecorder()
	r.FailWriteAfter(1)
	_ = r.Initialize("x", time.Second, nil)
	if err := r.WriteFrame(session.Execution{Stdout: "ok\n"}); err != nil {
		t.Fatalf("first WriteFrame should succeed: %v", err)
	}
	err := r.WriteFrame(session.Execution{Stdout: "boom\n"})
	if err == nil {
		t.Fatal("second WriteFrame should fail")
	}
	if !errors.Is(err, ErrInjectedWriteFailure) {
		t.Errorf("err = %v want ErrInjectedWriteFailure", err)
	}
}
