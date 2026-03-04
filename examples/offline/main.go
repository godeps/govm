package main

import (
	"context"
	"fmt"
	"log"

	"github.com/godeps/govm/pkg/client"
)

func main() {
	imgs, err := client.ListOfflineImages()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("embedded offline images: %v\n", imgs)

	rt, err := client.NewRuntime(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close()

	_ = rt.RemoveBox(context.Background(), "offline-demo", true)

	box, err := rt.CreateBox(context.Background(), "offline-demo", client.BoxOptions{
		OfflineImage: "py312-alpine",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer box.Close()

	if err := box.Start(); err != nil {
		log.Fatal(err)
	}
	defer box.Stop()

	res, err := box.Exec("/bin/sh", &client.ExecOptions{
		Args: []string{"-lc", "echo offline-ok; command -v python3; python3 -V"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("exit=%d stdout=%v stderr=%v\n", res.ExitCode, res.Stdout, res.Stderr)
}
