package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/clementd64/proxy64/internal/http2https"
	"github.com/clementd64/proxy64/internal/nat64"
	"github.com/clementd64/proxy64/internal/sni"
	"github.com/clementd64/proxy64/internal/utils"
)

var cmds = map[string]func(args []string) error{
	"http2https": func(args []string) error {
		cmd := flag.NewFlagSet("http2https", flag.ExitOnError)
		addr := cmd.String("addr", ":80", "address to listen on")
		cmd.Parse(args)

		return http2https.Listen(*addr)
	},

	"nat64": func(args []string) error {
		cmd := flag.NewFlagSet("nat64", flag.ExitOnError)
		port := cmd.Int("port", 1337, "port to listen on")
		cmd.Parse(args)

		return nat64.Listen(*port)
	},

	"snid": func(args []string) error {
		cmd := flag.NewFlagSet("snid", flag.ExitOnError)
		addr := cmd.String("addr", "0.0.0.0:443", "port to listen on")
		var allowed utils.IPv6List
		cmd.Var(&allowed, "allow", "comma-separated list of allowed target ranges")
		cmd.Parse(args)

		return sni.Listen(*addr, allowed)
	},
}

func run(args []string) error {
	if len(args) < 1 {
		return errors.New("no command provided")
	}

	if cmd, ok := cmds[args[0]]; ok {
		return cmd(args[1:])
	}
	return errors.New("unknown command")
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
