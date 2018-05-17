package subcommands

import (
	"github.com/spf13/cobra"
	"github.com/docker/docker/client"
	"context"
	"fmt"
	"github.com/fatih/color"
	"os"
	"github.com/docker/docker/api/types"
	"io/ioutil"
	"path/filepath"
	"strings"
	"encoding/json"
	"github.com/jhoonb/archivex"
	"github.com/c-bata/go-prompt"
)

var excludeDirs = []string{
	".idea",
	".git",
	"vendor"}


var tagARG, imageARG string
var imageBuilder = &ImageBuilder{}
var imageBuilderArguments = &GuArgumentSlice{}

var imageCmd = &cobra.Command{
	Use:   "image-build",
	Short: "Build the docker image  (you must run with ./Dockerfile file )",
	Long:  `Build the docker image by starting from the build output`,
	PreRunE: preRunImageBuild,
	RunE: imageBuilder.RunCommand,
}
func init() {
	imageCmd.Flags().BoolVarP(&interactiveARG, "interactive", "i", false, "Interactive or not")
	imageBuilderArguments.AddGuArgument(imageTagArgName,"",&imageARG, infos.Sprintf("The image tag: in the form vX.X.X..."))
	imageBuilderArguments.AddArgumentsToCobraCommand(imageCmd)
}

func notInteractiveImageBuild(cmd *cobra.Command) (err error) {
	err = CheckArguments(cmd.Flags(), imageBuilderArguments.Names())
	imageBuilder.ImageName, err = GetPackageName()
	if err != nil {
		return
	}
	err = CheckDockerfile()
	if err != nil {
		return
	}
	imageBuilder.ImageTag = cmd.Flag(imageTagArgName).Value.String()
	return
}
func interactiveImageBuild() (err error) {
	var input string
	var choices []string
	imageBuilder.ImageName, err = GetPackageName()
	if err != nil {
		return
	}
	err = CheckDockerfile()
	if err != nil {
		return
	}

	infos.Println("1 - Please insert a docker image TAG (must start with 'v' eg: v1.1.1")
	input = prompt.Choose("> ", choices)
	imageBuilder.ImageTag = input

	choices = yesOrNoChoices()
	infos.Println("RECAP: Your commands are:")
	color.Cyan(fmt.Sprintf("  docker build -t %s:%s %s", imageBuilder.ImageName, imageBuilder.ImageTag, "."))
	infos.Println("=> Execute?")
	input = prompt.Choose("> ", choices)
	if strings.ToLower(input) != "yes" {
		infos.Println("Exiting...")
		os.Exit(0)
	}
	return
}

func preRunImageBuild(cmd *cobra.Command, args []string) (err error) {
	if  interactiveARG {
		return interactiveImageBuild()
	} else {
		return notInteractiveImageBuild(cmd)
	}
}


type ImageBuilder struct {
	ImageTag        string
	ImageName       string

}
func (i *ImageBuilder) RunCommand(cmd *cobra.Command, args []string) error {
	infos.Printf("1. Building docker image: \r\n\t%s:%s\n", i.ImageName, i.ImageTag)

	infos.Println()

	err := i.checkImageName()
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}
	err = i.checkImageTag()
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}
	builOptions, buildCtx, err := i.createBuildOptionsAndContext()
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}

	buildResponse, err := cli.ImageBuild(ctx, buildCtx, *builOptions)
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}

	err = i.analyzeDockerEngineResponse(buildResponse)
	if err != nil {
		color.Red(fmt.Sprintf("Building image error: %s", err.Error()))
		return err
	}
	return nil
}

func (i ImageBuilder) createBuildOptionsAndContext() (*types.ImageBuildOptions, *os.File, error) {
	bo := types.ImageBuildOptions{}
	// create a temp zip file
	tarFile, err := i.createTar()
	if err != nil {
		return nil, nil, err
	}
	//read and stream zip file
	dockerFileTarReader, err := os.Open(tarFile)
	bo.Context = dockerFileTarReader
	//Path within the build context to the Dockerfile
	bo.Dockerfile = filepath.Base("./Dockerfile")
	//Remove intermediate containers after a successful build
	bo.Remove = true
	//Always remove intermediate containers
	bo.ForceRemove = true
	bo.Tags = []string{fmt.Sprintf("%s:%s", i.ImageName, i.ImageTag)}
	return &bo, dockerFileTarReader, nil
}

func (i ImageBuilder) createTar() (tarfile string, err error) {
	tarfile = "/tmp/dockerfile.tar"
	dirOfDockerfile := filepath.Dir(".")
	tar := new(archivex.TarFile)
	tar.Create(tarfile)
	files, err := ioutil.ReadDir(dirOfDockerfile)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			if !inExcludedDirs(f.Name()) {
				err = tar.AddAll(filepath.Join(dirOfDockerfile, f.Name()), true)
			}
		} else {
			err = tar.AddFile(filepath.Join(dirOfDockerfile, f.Name()))
		}
	}
	if err != nil {
		return
	}
	tar.Close()
	return
}

func (i ImageBuilder) checkImageName() (err error) {
	return CheckDockerImageNameForGeouniq(i.ImageName)
}
func (i ImageBuilder) checkImageTag() (err error) {
	return CheckDockerImageTagForGeouniq(i.ImageTag)
}

func (i ImageBuilder) analyzeDockerEngineResponse(buildResponse types.ImageBuildResponse) (err error) {
	defer buildResponse.Body.Close()
	if b, err := ioutil.ReadAll(buildResponse.Body); err == nil {
		output := string(b)
		lines := strings.Split(output, "\n")

		for _, l := range lines {
			if len(strings.TrimSpace(l)) > 0 {
				stream := make(map[string]string)
				json.Unmarshal([]byte(l), &stream)
				if val, ok := stream["stream"]; ok {
					color.White(val)
					continue
				}
				if val, ok := stream["error"]; ok {
					err = fmt.Errorf(val)
					return err
				}
			}
		}
	}
	return err
}

func inExcludedDirs(name string) bool {
	for _, n := range excludeDirs {
		if n == name {
			return true
		}
	}
	return false
}

