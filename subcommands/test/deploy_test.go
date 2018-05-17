package test

import (
	"testing"

	"geouniq.com/gubuilder/subcommands"
	"github.com/stretchr/testify/assert"
)

func Test_prepareRun(t *testing.T) {
	i := subcommands.ImageDeployer{
		RemoteRepositoryUri:"216314997889.dkr.ecr.eu-west-1.amazonaws.com",
		LocalImage:"geouniq.com/hengin:0.2.3",
		RemoteImage: "216314997889.dkr.ecr.eu-west-1.amazonaws.com/geouniq/hengin",
		RemoteImageTag: "v1.0.0-test",


	}
	err := i.RunCommand(nil, []string{})
	//err := PreRunImageDeploy(nil, []string {})
	assert.NoError(t, err)
}
