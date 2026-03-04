# govm

Go SDK wrapper for BoxLite VM runtime.

## What this repo contains

- `pkg/client`: public Go API
- `internal/binding`: cgo bridge to native static library
- `rust-bridge`: Rust staticlib exporting `govm_*` C ABI, implemented on top of `github.com/boxlite-ai/boxlite`
- `internal/native/<os>_<arch>`: vendored native artifacts committed in this repository
- `internal/offline/images/<name>/rootfs.tar.gz`: embedded offline OCI image layout bundles

## Build modes

- Default mode (no tags): stub binding, useful for unit tests and CI compile checks without native libs.
- Native mode: build with `-tags govm_native` to link vendored native libs.

## Native mode quick start

```bash
# 1) ensure native libs exist in internal/native/<platform>
./scripts/verify-native.sh

# 2) run tests in native mode
go test -tags govm_native ./...

# 3) run example
go run -tags govm_native ./examples/basic
go run -tags govm_native ./examples/all-api
```

Health checks:

```bash
go run ./cmd/govm-doctor
go run ./cmd/govm-doctor --json
make platform-check
```

Strict cross-platform gate (fails when expected platforms are missing):

```bash
GOVM_DOCTOR_STRICT=1 go run ./cmd/govm-doctor
STRICT=1 make platform-check
```

## Sync native artifacts from local boxlite checkout

```bash
./scripts/vendor-boxlite.sh /path/to/boxlite
```

## Rebuild Rust bridge staticlib

```bash
./scripts/build-bridge.sh release
```

## Public API sample

```go
rt, err := client.NewRuntime(nil)
if err != nil { panic(err) }
defer rt.Close()

box, err := rt.CreateBox(context.Background(), "demo", client.BoxOptions{Image: "alpine:latest"})
if err != nil { panic(err) }
defer box.Close()

_ = box.Start()
res, _ := box.Exec("echo", &client.ExecOptions{Args: []string{"hello"}})
fmt.Println(res.ExitCode, res.Stdout)
```

## Network API

`govm` exposes network controls in `pkg/client`:

- `NetworkMode`: `disabled`, `nat`, `bridged` (`bridged` currently unsupported by backend)
- `NetworkPolicy`: `block_all` / `allow_all` policy intent
- `PortForwards`: host -> guest TCP/UDP forwarding

Example with runtime default profile and per-box override:

```go
rt, err := client.NewRuntime(&client.RuntimeOptions{
	NetworkDefaults: &client.RuntimeNetworkDefaults{Profile: "strict"},
})
if err != nil { panic(err) }
defer rt.Close()

box, err := rt.CreateBox(context.Background(), "net-demo", client.BoxOptions{
	OfflineImage: "py312-alpine",
	Network: &client.NetworkConfig{
		Enabled: true,
		Mode:    client.NetworkNAT,
		Policy:  &client.NetworkPolicy{Mode: client.PolicyBlockAll},
		PortForwards: []client.PortForward{
			{HostIP: "127.0.0.1", HostPort: 18080, GuestPort: 8080, Protocol: client.ProtoTCP},
		},
	},
})
if err != nil { panic(err) }
_ = box
```

Detailed design and capability matrix:

- `docs/networking.md`

## Embedded Offline Images

`govm` can ship offline OCI layout bundles directly in the repository.

- Place bundles under `internal/offline/images/<name>/rootfs.tar.gz`
- Use `client.BoxOptions{OfflineImage: "<name>"}` in `CreateBox`
- At runtime, the bundle is extracted to `~/.govm/offline-rootfs/...` and used as `LocalBundlePath`

`BoxOptions` path fields:

- `LocalBundlePath`: preferred field, points to local OCI layout directory
- `RootfsPath`: backward-compatible alias (deprecated)

Example:

```bash
go run -tags govm_native ./examples/offline
```

Full API walkthrough:

```bash
go run -tags govm_native ./examples/all-api
```

## Garbage Collection

Remove stopped boxes:

```bash
go run ./cmd/govm-gc
```

Remove all boxes forcefully:

```bash
go run ./cmd/govm-gc --all --force
```
