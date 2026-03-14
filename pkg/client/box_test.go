package client

import (
	"reflect"
	"testing"
)

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

func TestBoxExecStream(t *testing.T) {
	m := newMockRuntimeProvider()
	b := m.AddBox("box-1", "alpine:latest")
	b.execResult = struct {
		ExitCode int
		Stdout   []string
		Stderr   []string
	}{
		ExitCode: 0,
		Stdout:   []string{"hello", "govm"},
		Stderr:   []string{"warn"},
	}

	r := newRuntimeWith(m)
	box, err := r.GetBox(t.Context(), "box-1")
	if err != nil {
		t.Fatal(err)
	}

	var stdout []string
	var stderr []string
	res, err := box.ExecStream("echo", &ExecOptions{Args: []string{"hello", "govm"}}, ExecStreamCallbacks{
		OnStdout: func(line string) { stdout = append(stdout, line) },
		OnStderr: func(line string) { stderr = append(stderr, line) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stdout, []string{"hello", "govm"}) {
		t.Fatalf("unexpected stdout callbacks: %+v", stdout)
	}
	if !reflect.DeepEqual(stderr, []string{"warn"}) {
		t.Fatalf("unexpected stderr callbacks: %+v", stderr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", res.ExitCode)
	}
	if !reflect.DeepEqual(res.Stdout, []string{"hello", "govm"}) {
		t.Fatalf("unexpected stdout result: %+v", res.Stdout)
	}
	if !reflect.DeepEqual(res.Stderr, []string{"warn"}) {
		t.Fatalf("unexpected stderr result: %+v", res.Stderr)
	}
}
