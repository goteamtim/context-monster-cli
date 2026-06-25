package skills

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestMain doubles as a slow-subprocess helper. When the test binary is
// re-spawned by TestExecute_contextCancelledDuringRun with the
// SKILLS_TEST_SLOW env var set, it sleeps indefinitely so the test can
// cancel the context and observe that Execute returns promptly.
func TestMain(m *testing.M) {
	if os.Getenv("SKILLS_TEST_SLOW") == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestExecute_cancelledContext verifies that a pre-cancelled context causes
// Execute to return an error immediately rather than attempting to run the skill.
func TestExecute_cancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	skill := Skill{
		Manifest: Manifest{Name: "test", Command: "./nonexistent"},
		Dir:      t.TempDir(),
	}

	_, err := Execute(ctx, skill, json.RawMessage(`{}`), nil)
	if err == nil {
		t.Fatal("expected error for pre-cancelled context, got nil")
	}
}

// TestExecute_contextCancelledDuringRun verifies that cancelling the context
// while a skill subprocess is running causes Execute to return an error
// promptly — this is the Ctrl+C case.
func TestExecute_contextCancelledDuringRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SKILLS_TEST_SLOW=1 causes the re-spawned test binary to sleep for 30s,
	// acting as a stand-in for a slow skill. Execute inherits os.Environ() so
	// the subprocess sees the variable automatically.
	t.Setenv("SKILLS_TEST_SLOW", "1")

	skill := Skill{
		Manifest: Manifest{Name: "slow", Command: os.Args[0]},
		Dir:      t.TempDir(),
	}

	// Cancel the context after 200ms while Execute is blocked on the subprocess.
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := Execute(ctx, skill, json.RawMessage(`{}`), nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error when context is cancelled mid-execution")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Execute took %v; expected cancellation within 5s", elapsed)
	}
}
