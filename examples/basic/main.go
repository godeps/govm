package main

import (
	"context"
	"fmt"
	"log"

	"github.com/godeps/govm/pkg/client"
)

func main() {
	rt, err := client.NewRuntime(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Close()

	box, err := rt.CreateBox(context.Background(), "demo", client.BoxOptions{Image: "python:3.12-alpine"})
	if err != nil {
		log.Fatal(err)
	}
	defer box.Close()

	if err := box.Start(); err != nil {
		log.Fatal(err)
	}
	defer box.Stop()

	res, err := box.Exec("python", &client.ExecOptions{
		Args: []string{"-c", "print('hello from minimal python image')"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("exit=%d stdout=%v stderr=%v\n", res.ExitCode, res.Stdout, res.Stderr)
}
