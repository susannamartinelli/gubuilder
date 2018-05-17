package main

import (
	"fmt"
	"os"

	"geouniq.com/gubuilder/subcommands"
	"github.com/fatih/color"
)

var (
	Version = "undefined"
)
var pres = color.New(color.Bold, color.FgHiGreen)
var errp = color.New(color.Bold, color.FgHiRed)
func main() {
	subcommands.RootCmd.Long = pres.Sprintf(`
================================================================================
         - GEOUNIQ MICROSERVICE GENERATOR Ver: %s -                              
================================================================================
  This utility helps you to build, build-image and upload a new microservice 
  image on AWS your repository.
`, Version)
	subcommands.RootCmd.Short = fmt.Sprintf("Geouniq MICORSEVICE generator version: %s", Version)

	if err := subcommands.RootCmd.Execute(); err != nil {
		errp.Print(err)
		os.Exit(1)
	}
}
