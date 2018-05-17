package subcommands

import (
	"github.com/spf13/cobra"
	"github.com/fatih/color"

)
var Version string
var infos = color.New(color.Bold, color.FgHiWhite)
var interactiveARG bool
var RootCmd = &cobra.Command{
	Use:   "gubuilder",
	}
