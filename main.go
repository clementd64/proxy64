package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/clementd64/proxy64/internal/http2https"
	"github.com/clementd64/proxy64/internal/nat64"
	"github.com/clementd64/proxy64/internal/sni"
)

func http2httpsCmd(args []string) error {
	cmd := flag.NewFlagSet("http2https", flag.ExitOnError)
	addr := cmd.String("addr", ":80", "address to listen on")
	cmd.Parse(args)

	return http2https.Listen(*addr)
}

func nat64Cmd(args []string) error {
	cmd := flag.NewFlagSet("nat64", flag.ExitOnError)
	port := cmd.Int("port", 1337, "port to listen on")
	cmd.Parse(args)

	return nat64.Listen(*port)
}

func snidCmd(args []string) error {
	cmd := flag.NewFlagSet("snid", flag.ExitOnError)
	addr := cmd.String("addr", "0.0.0.0:443", "port to listen on")
	cmd.Parse(args)

	return sni.Listen(*addr)
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
	case "snid":
		return snidCmd(args[1:])
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
