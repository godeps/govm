//go:build cgo && govm_native && linux && amd64

package binding

/*
#cgo LDFLAGS: -L${SRCDIR}/../native/linux_amd64 -lgovm_boxlite_bridge -lpthread -ldl -lm -Wl,-rpath,${SRCDIR}/../native/linux_amd64
*/
import "C"
