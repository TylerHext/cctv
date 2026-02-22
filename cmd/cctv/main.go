package main

import (
	"fmt"
	"os"

	"github.com/tylerhext/cctv/internal/config"
	"github.com/tylerhext/cctv/internal/session"
	"github.com/tylerhext/cctv/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cctv: load config: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]

	if len(args) == 0 {
		if err := tui.Start(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "cctv: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "new":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: cctv new <name>")
			os.Exit(1)
		}
		if err := session.New(args[1], cfg); err != nil {
			fmt.Fprintf(os.Stderr, "cctv: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("created session %q\n", args[1])

	case "kill":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: cctv kill <name>")
			os.Exit(1)
		}
		if err := session.Kill(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "cctv: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("killed session %q\n", args[1])

	case "attach":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: cctv attach <name>")
			os.Exit(1)
		}
		if err := session.Attach(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "cctv: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "cctv: unknown command %q\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: cctv [new|kill|attach] [name]")
		os.Exit(1)
	}
}
