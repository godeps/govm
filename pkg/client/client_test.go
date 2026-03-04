package client

import "testing"

func TestRuntimeCloseIdempotent(t *testing.T) {
	m := newMockRuntimeProvider()
	r := newRuntimeWith(m)
	r.Close()
	r.Close()
	if !m.freed {
		t.Fatal("expected runtime free")
	}
}

func TestCreateAndGetBox(t *testing.T) {
	m := newMockRuntimeProvider()
	r := newRuntimeWith(m)
	box, err := r.CreateBox(t.Context(), "demo", BoxOptions{Image: "alpine:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if box == nil || box.ID() == "" {
		t.Fatal("expected created box")
	}
	box2, err := r.GetBox(t.Context(), box.ID())
	if err != nil {
		t.Fatal(err)
	}
	if box2 == nil {
		t.Fatal("expected existing box")
	}
}

func TestRemoveBox(t *testing.T) {
	m := newMockRuntimeProvider()
	m.AddBox("box-1", "alpine:latest")
	r := newRuntimeWith(m)
	if err := r.RemoveBox(t.Context(), "box-1", false); err != nil {
		t.Fatal(err)
	}
	box, err := r.GetBox(t.Context(), "box-1")
	if err != nil {
		t.Fatal(err)
	}
	if box != nil {
		t.Fatal("expected removed box")
	}
}
