package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/godeps/govm/pkg/client"
)

func main() {
	var (
		all   bool
		force bool
		name  string
	)
	flag.BoolVar(&all, "all", false, "remove all boxes (including running when -force is set)")
	flag.BoolVar(&force, "force", true, "force remove")
	flag.StringVar(&name, "name", "", "only remove this box name/id")
	flag.Parse()

	rt, err := client.NewRuntime(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create runtime failed: %v\n", err)
		os.Exit(1)
	}
	defer rt.Close()

	ctx := context.Background()

	if strings.TrimSpace(name) != "" {
		if err := rt.RemoveBox(ctx, name, force); err != nil {
			fmt.Fprintf(os.Stderr, "remove %q failed: %v\n", name, err)
			os.Exit(2)
		}
		fmt.Printf("removed: %s\n", name)
		return
	}

	boxes, err := rt.ListBoxes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list boxes failed: %v\n", err)
		os.Exit(1)
	}

	removed := 0
	skipped := 0
	for _, b := range boxes {
		state := strings.ToLower(strings.TrimSpace(b.State))
		if !all && (state == "running" || state == "starting") {
			skipped++
			continue
		}
		if err := rt.RemoveBox(ctx, b.ID, force); err != nil {
			fmt.Fprintf(os.Stderr, "remove %s (%s) failed: %v\n", b.ID, b.Name, err)
			continue
		}
		removed++
		fmt.Printf("removed: id=%s name=%s state=%s\n", b.ID, b.Name, b.State)
	}

	fmt.Printf("gc done: removed=%d skipped=%d total=%d\n", removed, skipped, len(boxes))
}
