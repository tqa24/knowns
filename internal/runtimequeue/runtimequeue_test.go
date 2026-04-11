package runtimequeue

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestEnqueueCoalescesDuplicateJobs(t *testing.T) {
	SetTestBypass(true)
	defer SetTestBypass(false)
	t.Setenv("HOME", t.TempDir())
	storeRoot := filepath.Join(t.TempDir(), ".knowns")

	job1, err := Enqueue(storeRoot, JobIndexTask, "abc")
	if err != nil {
		t.Fatalf("enqueue first job: %v", err)
	}
	job2, err := Enqueue(storeRoot, JobIndexTask, "abc")
	if err != nil {
		t.Fatalf("enqueue second job: %v", err)
	}
	queue, err := LoadQueue(storeRoot)
	if err != nil {
		t.Fatalf("load queue: %v", err)
	}
	if len(queue.Jobs) != 1 {
		t.Fatalf("expected 1 coalesced job, got %d", len(queue.Jobs))
	}
	if queue.Jobs[0].ID != job1.ID {
		t.Fatalf("expected coalesced job to keep original id %s, got %s", job1.ID, queue.Jobs[0].ID)
	}
	if job2.ID != job1.ID {
		t.Fatalf("expected returned coalesced job id %s, got %s", job1.ID, job2.ID)
	}
}

func TestAcquireClientTracksIndependentLeases(t *testing.T) {
	SetTestBypass(true)
	defer SetTestBypass(false)
	t.Setenv("HOME", t.TempDir())
	storeRoot := filepath.Join(t.TempDir(), ".knowns")

	first, err := AcquireClient("mcp", storeRoot, false)
	if err != nil {
		t.Fatalf("acquire first client: %v", err)
	}
	defer first.Release()
	second, err := AcquireClient("opencode", storeRoot, true)
	if err != nil {
		t.Fatalf("acquire second client: %v", err)
	}
	defer second.Release()

	leases, err := ActiveLeases()
	if err != nil {
		t.Fatalf("active leases: %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("expected 2 leases, got %d", len(leases))
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release first lease: %v", err)
	}
	leases, err = ActiveLeases()
	if err != nil {
		t.Fatalf("active leases after release: %v", err)
	}
	if len(leases) != 1 {
		t.Fatalf("expected second lease to remain, got %d leases", len(leases))
	}
}

func TestRunDaemonStopsAfterIdle(t *testing.T) {
	SetTestBypass(true)
	defer SetTestBypass(false)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KNOWNS_RUNTIME_IDLE_TIMEOUT_MS", "100")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	started := time.Now()
	err := RunDaemon(ctx, func(storeRoot string, job Job) error {
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("run daemon: %v", err)
	}
	if time.Since(started) < 100*time.Millisecond {
		t.Fatalf("daemon exited before idle timeout elapsed")
	}
}
