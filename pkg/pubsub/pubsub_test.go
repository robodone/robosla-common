package pubsub

import (
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	sub, err := node.Sub("hello")
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if err := node.Pub(`{"world":1}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		t.Errorf("Unexpected update: %v", msg)
	default:
		// Nothing happened, as expected
	}
	if err := node.Pub(`{"hello":1}`); err != nil {
		t.Fatalf("Pub2: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"hello":1}`
		if msg != want {
			t.Errorf("Unexpected update message: %q, want: %q", msg, want)
		}
	default:
		t.Errorf("Expected update not received")
	}
	node.Unsub(sub)
}

func TestDeep(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	sub, err := node.Sub("hello.world")
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if err := node.Pub(`{"world":1}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		t.Errorf("Unexpected update: %v", msg)
	default:
		// Nothing happened, as expected
	}
	if err := node.Pub(`{"hello":{"world":"lala", "zzz": 3}}`); err != nil {
		t.Fatalf("Pub2: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"hello":{"world":"lala"}}`
		if msg != want {
			t.Errorf("Unexpected update message: %q, want: %q", msg, want)
		}
	default:
		t.Errorf("Expected update not received")
	}
	node.Unsub(sub)
}

func TestInitialUpdate(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	if err := node.Pub(`{"hello":{"lala":"bb","world":4}}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	if err := node.Pub(`{"hello":{"world":5}}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	sub, err := node.Sub("hello")
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"hello":{"lala":"bb","world":5}}`
		if msg != want {
			t.Errorf("Unexpected update message: %q, want: %q", msg, want)
		}
	default:
		t.Errorf("Expected update not received")
	}
	node.Unsub(sub)
}

func TestString(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	sub, err := node.SubString("hello.world")
	if err != nil {
		t.Fatalf("SubString: %v", err)
	}
	if err := node.Pub(`{"hello":{"world":"lala"}}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := "lala"
		if msg != want {
			t.Errorf("Unexpected update message: %q, want: %q", msg, want)
		}
		// We don't expect immediate response, but we do expect a quick response.
	case <-time.Tick(time.Second):
		t.Errorf("Expected update not received")
	}
	sub.Unsub()
}

// Test that subscribing to upper-level paths
func TestUpperLevel(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	sub, err := node.Sub("hello")
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if err := node.Pub(`{"world":1}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		t.Errorf("Unexpected update: %v", msg)
	default:
		// Nothing happened, as expected
	}
	if err := node.Pub(`{"hello":{"world":"lala", "zzz": 3}}`); err != nil {
		t.Fatalf("Pub2: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"hello":{"world":"lala","zzz":3}}`
		if msg != want {
			t.Errorf("Unexpected update message:\n%s\nwant:\n%s\n", msg, want)
		}
	default:
		t.Errorf("Expected update not received")
	}
	node.Unsub(sub)
}

func TestSeries(t *testing.T) {
	node := NewNode()
	defer node.Stop()

	sub, err := node.Sub("ts.terminalOutput.123")
	if err != nil {
		t.Fatalf("SubString: %v", err)
	}

	// First update: just two values
	if err := node.Pub(`{"ts":{"terminalOutput":{"123":[{"ts":1,"out":"hello"},{"ts":2,"out":"world"}]}}}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"ts":{"terminalOutput":{"123":[{"out":"hello","ts":1},{"out":"world","ts":2}]}}}`
		if msg != want {
			t.Errorf("Unexpected update message:\n%s\nwant:\n%s\n", msg, want)
		}
		// We don't expect immediate response, but we do expect a quick response.
	case <-time.Tick(time.Second):
		t.Errorf("Expected update not received")
	}

	// Second update: one more value
	if err := node.Pub(`{"ts":{"terminalOutput":{"123":[{"ts":3,"out":"lala"}]}}}`); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		want := `{"ts":{"terminalOutput":{"123":[{"out":"lala","ts":3}]}}}`
		if msg != want {
			t.Errorf("Unexpected update message:\n%s\nwant:\n%s\n", msg, want)
		}
		// We don't expect immediate response, but we do expect a quick response.
	case <-time.Tick(time.Second):
		t.Errorf("Expected update not received")
	}

	node.Unsub(sub)
}
