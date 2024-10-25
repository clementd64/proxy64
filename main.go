package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/clementd64/proxy64/internal/http2https"
	"github.com/clementd64/proxy64/internal/nat64"
)

func http2httpsCmd(args []string) error {
	cmd := flag.NewFlagSet("http2https", flag.ExitOnError)
	port := cmd.Int("port", 80, "port to listen on")
	cmd.Parse(args)

	return http2https.Listen(*port)
}

func nat64Cmd(args []string) error {
	cmd := flag.NewFlagSet("nat64", flag.ExitOnError)
	port := cmd.Int("port", 1337, "port to listen on")
	cmd.Parse(args)

	return nat64.Listen(*port)
}

func run(args []string) error {
	if len(args) < 1 {
		return errors.New("no command provided")
	}

	switch args[0] {
	case "http2https":
		return http2httpsCmd(args[1:])
	case "nat64":
		return nat64Cmd(args[1:])
	default:
		return errors.New("unknown command")
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
