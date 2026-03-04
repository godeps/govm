package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/godeps/govm/pkg/client"
)

func main() {
	ctx := context.Background()

	images, err := client.ListOfflineImages()
	if err != nil {
		log.Fatal(err)
	}
	meta, err := client.ListOfflineImageMetadata()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("offline images: %v\n", images)
	fmt.Printf("offline metadata: %d bundles\n", len(meta))
	for _, m := range meta {
		fmt.Printf("- %s size=%d sha256=%s\n", m.Name, m.SizeBytes, short(m.SHA256))
	}

	homeDir := filepath.Join(os.TempDir(), "govm-all-api-home")
	rt, err := client.NewRuntime(&client.RuntimeOptions{
		HomeDir:         homeDir,
		ImageRegistries: []string{"docker.io"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close()

	name := fmt.Sprintf("all-api-%d", time.Now().UnixNano())
	defer func() { _ = rt.RemoveBox(ctx, name, true) }()

	missing, err := rt.GetBox(ctx, "definitely-not-exist")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("get missing box: nil=%v\n", missing == nil)

	box, err := rt.CreateBox(ctx, name, client.BoxOptions{
		OfflineImage: "py312-alpine",
		CPUs:         1,
		MemoryMB:     1024,
		Env: map[string]string{
			"DEMO_MODE": "all-api",
		},
		WorkingDir: "/",
		Network: &client.NetworkConfig{
			Enabled: true,
			Mode:    client.NetworkNAT,
			Policy: &client.NetworkPolicy{
				Mode: client.PolicyBlockAll,
			},
			PortForwards: []client.PortForward{
				{HostIP: "127.0.0.1", HostPort: 18080, GuestPort: 8080, Protocol: client.ProtoTCP},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer box.Close()
	fmt.Printf("created: id=%s name=%s\n", box.ID(), box.Name())

	if err := box.Start(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("started")

	info, err := box.Info()
	if err != nil {
		fmt.Printf("info error (known compatibility issue): %v\n", err)
	} else {
		fmt.Printf("info: id=%s state=%s image=%s\n", info.ID, info.State, info.Image)
	}

	res, err := box.Exec("/bin/sh", &client.ExecOptions{
		Args:       []string{"-lc", "echo DEMO_MODE=$DEMO_MODE; command -v python3; python3 -V"},
		Env:        map[string]string{"EXTRA_ENV": "1"},
		WorkingDir: "/",
		Timeout:    15 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("exec: exit=%d stdout=%q stderr=%q\n", res.ExitCode, strings.Join(res.Stdout, " | "), strings.Join(res.Stderr, " | "))

	got, err := rt.GetBox(ctx, box.ID())
	if err != nil {
		log.Fatal(err)
	}
	if got == nil {
		log.Fatal("expected box from GetBox")
	}
	defer got.Close()
	fmt.Printf("get by id: id=%s\n", got.ID())

	boxes, err := rt.ListBoxes(ctx)
	if err != nil {
		fmt.Printf("list boxes error (known compatibility issue): %v\n", err)
	} else {
		fmt.Printf("list boxes: %d\n", len(boxes))
	}

	if err := box.Stop(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("stopped")

	if err := rt.RemoveBox(ctx, box.ID(), true); err != nil {
		log.Fatal(err)
	}
	fmt.Println("removed")
}

func short(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}
