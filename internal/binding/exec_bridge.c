//go:build cgo && govm_native
// +build cgo,govm_native

#include <stdlib.h>
#include "_cgo_export.h"

typedef void (*govm_output_callback)(const char* text, int stream_type, void* user_data);

int govm_box_exec(
    void* handle,
    const char* command,
    const char* opts_json,
    govm_output_callback callback,
    void* user_data,
    int* out_exit_code,
    char** out_err
);

int call_box_exec_with_go_callback(
    void* handle,
    const char* command,
    const char* opts_json,
    void* user_data,
    int* out_exit_code,
    char** out_err
) {
    return govm_box_exec(handle, command, opts_json,
        (govm_output_callback)goOutputCallback, user_data,
        out_exit_code, out_err);
}
