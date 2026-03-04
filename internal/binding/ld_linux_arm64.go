//go:build cgo && govm_native && linux && arm64

package binding

/*
#cgo LDFLAGS: -L${SRCDIR}/../native/linux_arm64 -lgovm_boxlite_bridge -lpthread -ldl -lm -Wl,-rpath,${SRCDIR}/../native/linux_arm64
*/
import "C"
