//go:build cgo && govm_native && darwin && arm64

package binding

/*
#cgo LDFLAGS: -L${SRCDIR}/../native/darwin_arm64 -lgovm_boxlite_bridge -framework CoreFoundation -framework Security -framework IOKit -framework DiskArbitration -Wl,-rpath,${SRCDIR}/../native/darwin_arm64
*/
import "C"
