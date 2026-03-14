//go:build cgo && govm_native

package binding

/*
#include <stdlib.h>

int call_box_exec_with_go_callback(
    void* handle,
    const char* command,
    const char* opts_json,
    void* user_data,
    int* out_exit_code,
    char** out_err
);
*/
import "C"

import (
	"encoding/json"
	"runtime/cgo"
	"sync"
	"unsafe"
)

type outputCollector struct {
	mu     sync.Mutex
	stdout []string
	stderr []string
	cb     ExecCallbacks
}

//export goOutputCallback
func goOutputCallback(text *C.char, streamType C.int, userData unsafe.Pointer) {
	h := cgo.Handle(userData)
	collector := h.Value().(*outputCollector)
	line := C.GoString(text)

	collector.mu.Lock()
	if int(streamType) == 0 {
		collector.stdout = append(collector.stdout, line)
		cb := collector.cb.OnStdout
		collector.mu.Unlock()
		if cb != nil {
			cb(line)
		}
		return
	}
	collector.stderr = append(collector.stderr, line)
	cb := collector.cb.OnStderr
	collector.mu.Unlock()
	if cb != nil {
		cb(line)
	}
}

func (b *Box) Exec(command string, opts ExecOptions) (ExecResult, error) {
	return b.execWithCallbacks(command, opts, ExecCallbacks{})
}

func (b *Box) ExecStream(command string, opts ExecOptions, cb ExecCallbacks) (ExecResult, error) {
	return b.execWithCallbacks(command, opts, cb)
}

func (b *Box) execWithCallbacks(command string, opts ExecOptions, cb ExecCallbacks) (ExecResult, error) {
	cCommand := C.CString(command)
	defer C.free(unsafe.Pointer(cCommand))

	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return ExecResult{}, err
	}
	cOptsJSON := C.CString(string(optsJSON))
	defer C.free(unsafe.Pointer(cOptsJSON))

	collector := &outputCollector{cb: cb}
	h := cgo.NewHandle(collector)
	defer h.Delete()

	var exitCode C.int
	var outErr *C.char

	res := C.call_box_exec_with_go_callback(
		b.handle,
		cCommand,
		cOptsJSON,
		unsafe.Pointer(h),
		&exitCode,
		&outErr,
	)
	if res < 0 {
		return ExecResult{}, getError(outErr)
	}

	return ExecResult{
		ExitCode: int(exitCode),
		Stdout:   collector.stdout,
		Stderr:   collector.stderr,
	}, nil
}
