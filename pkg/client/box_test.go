package client

import "testing"

func TestBoxExec(t *testing.T) {
	m := newMockRuntimeProvider()
	b := m.AddBox("box-1", "alpine:latest")
	b.execResult = struct {
		ExitCode int
		Stdout   []string
		Stderr   []string
	}{ExitCode: 0, Stdout: []string{"hello govm"}}

	r := newRuntimeWith(m)
	box, err := r.GetBox(t.Context(), "box-1")
	if err != nil {
		t.Fatal(err)
	}
	res, err := box.Exec("echo", &ExecOptions{Args: []string{"hello", "govm"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || len(res.Stdout) == 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
