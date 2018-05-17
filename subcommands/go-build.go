package subcommands

import (
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"github.com/fatih/color"
	"path/filepath"
	"strings"
	"github.com/c-bata/go-prompt"
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"

	"bufio"
	)

var osbuildARG, packagePathARG, versionARG string
var ostypes = []string{"darwin", "linux"}
var goBuilder = &GoBuilder{}
var goBuilderArguments = &GuArgumentSlice{}
const (
	imageForBuildname = "susannam/golang-glide:latest"
)

var buildCmd = &cobra.Command{
	Use:     "go-build",
	Short:   "Build the golang application (you must run with ./glide.yaml file )",
	Long:    `Build the golang application located the current package into ./bin/main`,
	PreRunE: preRunGoBuild,
	RunE:    goBuilder.RunCommand,
}

func init() {
	buildCmd.Flags().BoolVarP(&interactiveARG, "interactive", "i", false, "Interactive or not")
	goBuilderArguments.AddGuArgument(osArgName, "", &osbuildARG, infos.Sprintf("The GOOS format (Operating System) you want to build, availables: %v", ostypes))
	goBuilderArguments.AddGuArgument(mainPathArgName, "", &packagePathARG, infos.Sprintf("The path to a 'main.go' file or to a dir which contains 'main.go' and other go files"))
	goBuilderArguments.AddGuArgument(versionArgName, "", &versionARG, infos.Sprintf("The package version (in the form vX.X.X...)"))
	goBuilderArguments.AddArgumentsToCobraCommand(buildCmd)
}
func notInteractiveGoBuild(cmd *cobra.Command) (err error) {
	err = CheckArguments(cmd.Flags(), goBuilderArguments.Names())
	goBuilder.Packagename, err  = GetPackageName()
	if err != nil {
		return
	}
	goBuilder.OsType = 	 cmd.Flag(osArgName).Value.String()
	goBuilder.MainPath = cmd.Flag(mainPathArgName).Value.String()
	goBuilder.Version = cmd.Flag(versionArgName).Value.String()
	return
}
func interactiveGoBuild() (err error) {
	var input string
	goBuilder.Packagename, err  = GetPackageName()
	if err != nil {
		return err
	}

	infos.Println("1 - Please select an OS.")
	input = prompt.Choose("> ", ostypes)
	goBuilder.OsType = input

	infos.Println("2 - Please insert relative path to main.go.")
	input = prompt.Choose(">", []string{"./cmd"})
	goBuilder.MainPath = input

	infos.Println("3 - Please insert version")
	input = prompt.Choose(">", []string{})
	goBuilder.Version = input

	infos.Println("RECAP:")
	color.Cyan(fmt.Sprintf("  Building go PACKAGE: %s PATH:%s for PLATFORM:%s VERSION:%s", goBuilder.Packagename, goBuilder.MainPath, goBuilder.OsType, goBuilder.Version))

	infos.Println("=> Execute?")
	input = prompt.Choose("> ", yesOrNoChoices())
	if strings.ToLower(input) != "yes" {
		infos.Println("Exiting...")
		os.Exit(0)
	}
	return
}

func preRunGoBuild(cmd *cobra.Command, args []string) (err error) {
	if interactiveARG {
		return interactiveGoBuild()
	} else {
		return notInteractiveGoBuild(cmd)
	}
}


// GoBuilder conatins informations in order to compile the main() function
type GoBuilder struct {
	Packagename	string
	MainPath    string
	OsType      string
	Version     string
}
func (g GoBuilder) Checks() (err error) {
	if len(g.Packagename) == 0 {
		err = fmt.Errorf("package name cannot be empty")
	}
	if len(g.MainPath) == 0 {
		err = fmt.Errorf("main path cannot be empty")
	}
	if len(g.OsType) == 0 {
		err = fmt.Errorf("OS type cannot be empty")
	}
	if len(g.Version) == 0 {
		err = fmt.Errorf("version cannot be empty")
	}
	if err != nil {
		return
	}
	err = ParseSemanticVersion(g.Version)
	if err != nil {
		return fmt.Errorf("wrong version %s: %s", g.Version, err.Error())
	}
	osmatch := false
	for _,t := range ostypes {
		if t == g.OsType {
			osmatch = true
		}
	}
	if !osmatch {
		return fmt.Errorf("wrong OS type %s", g.OsType)
	}
	return
}
func (g *GoBuilder) RunCommand(cmd *cobra.Command, args []string) (err error) {
	err = g.Checks()
	if err != nil {
		return err
	}
	pwd, _ := os.Getwd()
	cc := BuildContainerManager{
		TestContainerManager: TestContainerManager{
			PackageName:      g.Packagename,
			HostPackagePath:  pwd,
			HostSSHPath:      filepath.Join(os.Getenv("HOME"), ".ssh"),
		},
		PackageVersion:   g.Version,
		Main:             g.MainPath,
		OsType:			  g.OsType,
	}
	conf, hostConf, err := cc.createContainerStartConfiguration()
	if err != nil {
		return
	}

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return
	}
	defer cli.Close()

	//retrieve image from repository
	infos.Println("1. Pulling Docker image \r\n\t", imageForBuildname)
	_, err = cli.ImagePull(ctx, imageForBuildname, types.ImagePullOptions{})
	if err != nil {
		return err
	}


	infos.Println("2. Command to build \r\n\t", strings.Join(conf.Cmd, " "))
	resp, err := cli.ContainerCreate(ctx, conf, hostConf, nil, "")
	if err != nil {
		return err
	}
	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	reader, err := cli.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return err
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		color.White(scanner.Text())
	}
    defer  cli.ContainerRemove(ctx,resp.ID, types.ContainerRemoveOptions{RemoveLinks:true, RemoveVolumes:true, Force: true} )
	return nil
}
