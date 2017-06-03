package pubsub

import "testing"

func TestManagerSimple(t *testing.T) {
	m := NewManager()
	defer m.Stop()

	sub, err := m.Sub("hello@world.com", "printers")
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	update := `{"printers":{"w01":{"online":true}}}`
	if err := m.Pub("zzz", update); err != nil {
		t.Fatalf("Pub: %v", err)
	}
	select {
	case msg := <-sub.C():
		t.Errorf("Unexpected update: %v", msg)
	default:
		// No update delivered, as it was published to a different node.
	}
	if err := m.Pub("hello@world.com", update); err != nil {
		t.Fatalf("Pub2: %v", err)
	}
	select {
	case msg := <-sub.C():
		if msg != update {
			t.Errorf("Unexpected update message: %q, want: %q", msg, update)
		}
	default:
		t.Errorf("Expected update not received")
	}
	m.Unsub(sub)
}
