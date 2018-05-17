package test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"geouniq.com/gubuilder/subcommands"
)

func TestGoBuilder_runCommand(t *testing.T) {
	g := subcommands.GoBuilder{
		MainPath:    "./cmd",
		OsType:      "linux",
		Version:     "v1.0",
	}

	err := g.RunCommand(nil, []string{})
	assert.Nil(t, err)
}
