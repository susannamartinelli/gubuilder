package subcommands

import (
	"github.com/spf13/cobra"
	"github.com/c-bata/go-prompt"
	"fmt"
	"github.com/fatih/color"
)
var notest bool
var allCmd = &cobra.Command{
	Use:     "all",
	Short:   "Execute all steps go build, go test, image build and image deploy (you must run with ./glide.yaml file )",
	Long:    `Execute all steps go build, go test, image build and image deploy`,
	PreRunE: PreRunALL,
	RunE:    RunALL,
}
func init(){
	allCmd.Flags().BoolVar(&notest ,"notest", false, "No test running")
	RootCmd.AddCommand(allCmd)
}

func PreRunALL(cmd *cobra.Command, args []string) (err error) {
	var input string

	packageName, err  := GetPackageName()
	if err != nil {
		return err
	}
	//aws session
	ecrSession, err := GetECRSession()
	if err!= nil {
		color.Red("Unable to get local repositories list: %s", err)
		return err
	}

	goBuilder.Packagename=packageName
	goTester.Packagename=packageName
	imageBuilder.ImageName=packageName

	infos.Println("1 - Please select an OS.")
	input = prompt.Choose("> ", ostypes)
	goBuilder.OsType = input

	infos.Println("2 - Please insert relative path to main.go.")
	input = prompt.Choose(">", []string{"./cmd"})
	goBuilder.MainPath = input

	infos.Println("3 - Please insert version")
	input = prompt.Choose(">", []string{})
	goBuilder.Version = input
	imageBuilder.ImageTag = input
	imageDeployer.LocalImage=fmt.Sprintf("%s:%s", packageName, input)
	imageDeployer.RemoteImageTag = input

	//remote repository
	choices, repositoryURI, _, err := remoteRepositoriesChoices(ecrSession)
	if err!= nil {
		color.Red("Unable to get remote repositories list: %s", err)
		return err
	}
	infos.Println("4 - Please select a AWS docker REPOSITORY ("+repositoryURI+").")
	input = prompt.Choose("> ", choices)
	imageDeployer.RemoteImage = input
	imageDeployer.RemoteRepositoryUri = repositoryURI
	return nil
}

func RunALL(cmd *cobra.Command, args []string) (err error) {
	color.Cyan("Building package: %s %s %s\n", goBuilder.Packagename, goBuilder.OsType, goBuilder.Version)
	err = goBuilder.RunCommand(nil, []string{})
	if err != nil {
		return
	}
	if notest {
		color.Cyan("Skipping test package: %s\n", goTester.Packagename)
	} else {
		color.Cyan("Testing package: %s\n", goTester.Packagename)
		err = goTester.RunCommand(nil, []string{})
		if err != nil {
			return
		}
	}

	color.Cyan("Image build: %s %s\n", imageBuilder.ImageName, imageBuilder.ImageTag)
	err = imageBuilder.RunCommand(nil, []string{})
	if err != nil {
		return
	}
	color.Cyan("Image deploy: %s %s\n", imageBuilder.ImageName, imageBuilder.ImageTag)
	err = imageDeployer.RunCommand(nil, []string{})
	if err != nil {
		return
	}
	return
}