package pubsub

import "testing"

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
