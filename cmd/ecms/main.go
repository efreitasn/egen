package main

import (
	"fmt"
	"os"

	"github.com/efreitasn/cfop"
	"github.com/efreitasn/ecms/cmd/ecms/internal/cmds"
)

func main() {
	set := cfop.NewSubcmdsSet()

	set.Add(
		"build",
		"Builds the website",
		cfop.NewCmd(cfop.CmdConfig{
			Fn: cmds.Build,
		}),
	)

	set.Add(
		"version",
		"Prints the version",
		cfop.NewCmd(cfop.CmdConfig{
			Fn: cmds.Version,
		}),
	)

	err := cfop.Init(
		"ecms",
		"A CMS",
		os.Args,
		set,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
