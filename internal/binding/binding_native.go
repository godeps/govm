//go:build cgo && govm_native

package binding

/*
#include <stdlib.h>
#include <stdbool.h>

void govm_free_string(char* s);
void* govm_runtime_new(const char* config_json, char** out_err);
void govm_runtime_free(void* runtime);
char* govm_create_box(void* runtime, const char* opts_json, const char* name, char** out_err);
void* govm_get_box(void* runtime, const char* id_or_name, char** out_err);
int govm_list_boxes(void* runtime, char** out_json, char** out_err);
int govm_remove_box(void* runtime, const char* id_or_name, bool force, char** out_err);
int govm_box_start(void* handle, char** out_err);
int govm_box_stop(void* handle, char** out_err);
int govm_box_info(void* handle, char** out_json, char** out_err);
char* govm_box_id(void* handle);
void govm_box_free(void* handle);
*/
import "C"

import (
	"encoding/json"
	"errors"
	"time"
	"unsafe"
)

type BoxOptions struct {
	Image               string            `json:"image"`
	RootfsPath          string            `json:"rootfs_path,omitempty"`
	CPUs                int               `json:"cpus,omitempty"`
	MemoryMB            int               `json:"memory_mb,omitempty"`
	Env                 map[string]string `json:"env,omitempty"`
	WorkingDir          string            `json:"working_dir,omitempty"`
	NetworkMode         string            `json:"network_mode,omitempty"`
	NetworkPolicyMode   string            `json:"network_policy_mode,omitempty"`
	PortForwards        []PortForward     `json:"port_forwards,omitempty"`
	MacOSNetworkEnabled bool              `json:"macos_network_enabled,omitempty"`
}

type PortForward struct {
	HostIP    string `json:"host_ip,omitempty"`
	HostPort  int    `json:"host_port,omitempty"`
	GuestPort int    `json:"guest_port,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type BoxInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Image     string    `json:"image"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type RuntimeOptions struct {
	HomeDir         string   `json:"home_dir,omitempty"`
	ImageRegistries []string `json:"image_registries,omitempty"`
}

type ExecOptions struct {
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TTY        bool              `json:"tty,omitempty"`
	User       string            `json:"user,omitempty"`
	TimeoutSec float64           `json:"timeout_secs,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
}

type ExecResult struct {
	ExitCode int
	Stdout   []string
	Stderr   []string
}

func freeString(s *C.char) {
	if s != nil {
		C.govm_free_string(s)
	}
}

func getError(errPtr *C.char) error {
	if errPtr == nil {
		return errors.New("unknown error")
	}
	msg := C.GoString(errPtr)
	freeString(errPtr)
	return errors.New(msg)
}

type Box struct{ handle unsafe.Pointer }

func (b *Box) Start() error {
	var outErr *C.char
	if res := C.govm_box_start(b.handle, &outErr); res < 0 {
		return getError(outErr)
	}
	return nil
}

func (b *Box) Stop() error {
	var outErr *C.char
	if res := C.govm_box_stop(b.handle, &outErr); res < 0 {
		return getError(outErr)
	}
	return nil
}

func (b *Box) Info() (BoxInfo, error) {
	var outJSON, outErr *C.char
	if res := C.govm_box_info(b.handle, &outJSON, &outErr); res < 0 {
		return BoxInfo{}, getError(outErr)
	}
	defer freeString(outJSON)
	var info BoxInfo
	if err := json.Unmarshal([]byte(C.GoString(outJSON)), &info); err != nil {
		return BoxInfo{}, err
	}
	return info, nil
}

func (b *Box) Free() {
	if b.handle != nil {
		C.govm_box_free(b.handle)
		b.handle = nil
	}
}

type Runtime struct{ handle unsafe.Pointer }

func NewRuntime(opts *RuntimeOptions) (*Runtime, error) {
	var cCfg *C.char
	if opts != nil {
		data, err := json.Marshal(opts)
		if err != nil {
			return nil, err
		}
		cCfg = C.CString(string(data))
		defer C.free(unsafe.Pointer(cCfg))
	}
	var outErr *C.char
	h := C.govm_runtime_new(cCfg, &outErr)
	if h == nil {
		return nil, getError(outErr)
	}
	return &Runtime{handle: h}, nil
}

func (r *Runtime) CreateBox(name string, opts BoxOptions) (string, error) {
	data, err := json.Marshal(opts)
	if err != nil {
		return "", err
	}
	cOpts := C.CString(string(data))
	defer C.free(unsafe.Pointer(cOpts))
	var cName *C.char
	if name != "" {
		cName = C.CString(name)
		defer C.free(unsafe.Pointer(cName))
	}
	var outErr *C.char
	id := C.govm_create_box(r.handle, cOpts, cName, &outErr)
	if id == nil {
		return "", getError(outErr)
	}
	defer freeString(id)
	return C.GoString(id), nil
}

func (r *Runtime) GetBox(idOrName string) (*Box, string, error) {
	cID := C.CString(idOrName)
	defer C.free(unsafe.Pointer(cID))
	var outErr *C.char
	h := C.govm_get_box(r.handle, cID, &outErr)
	if h == nil {
		if outErr != nil {
			return nil, "", getError(outErr)
		}
		return nil, "", nil
	}
	cBoxID := C.govm_box_id(h)
	boxID := ""
	if cBoxID != nil {
		boxID = C.GoString(cBoxID)
		freeString(cBoxID)
	}
	return &Box{handle: h}, boxID, nil
}

func (r *Runtime) ListBoxes() ([]BoxInfo, error) {
	var outJSON, outErr *C.char
	if res := C.govm_list_boxes(r.handle, &outJSON, &outErr); res < 0 {
		return nil, getError(outErr)
	}
	defer freeString(outJSON)
	var infos []BoxInfo
	if err := json.Unmarshal([]byte(C.GoString(outJSON)), &infos); err != nil {
		return nil, err
	}
	return infos, nil
}

func (r *Runtime) RemoveBox(idOrName string, force bool) error {
	cID := C.CString(idOrName)
	defer C.free(unsafe.Pointer(cID))
	var outErr *C.char
	if res := C.govm_remove_box(r.handle, cID, C.bool(force), &outErr); res < 0 {
		return getError(outErr)
	}
	return nil
}

func (r *Runtime) Free() {
	if r.handle != nil {
		C.govm_runtime_free(r.handle)
		r.handle = nil
	}
}
