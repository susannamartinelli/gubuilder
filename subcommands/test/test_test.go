package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"geouniq.com/gubuilder/subcommands"
)

func TestGoTester_runCommand(t *testing.T) {
	g := subcommands.GoTester{
	}

	err := g.RunCommand(nil, []string{})
	assert.Nil(t, err)
}

