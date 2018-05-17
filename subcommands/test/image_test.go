package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"geouniq.com/gubuilder/subcommands"
)

func Test_runCommand(t *testing.T) {
	i := subcommands.ImageBuilder{
		ImageName: "geouniq.com/ost",
		ImageTag: "latest",
	}

	err := i.RunCommand(nil, []string{"-i", "false"})
	assert.Nil(t, err)

}
