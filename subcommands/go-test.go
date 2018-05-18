package subcommands

import (
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"bufio"
	"fmt"
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"

	"github.com/c-bata/go-prompt"

	"github.com/fatih/color"
	"strings"
)

var goTester = &GoTester{}
var goTesterArguments = &GuArgumentSlice{}
var testCmd = &cobra.Command{
	Use:     "go-test",
	Short:   "Test the golang application",
	Long:    `Test the golang application`,
	PreRunE: preRunGoTest,
	RunE:    goTester.RunCommand,
}

func init() {
	testCmd.Flags().BoolVarP(&interactiveARG, "interactive", "i", false, "Interactive or not")
	goTesterArguments.AddArgumentsToCobraCommand(testCmd)
}

func notInteractiveGoTest(cmd *cobra.Command) (err error) {
	err = CheckArguments(cmd.Flags(), goBuilderArguments.Names())
	goTester.Packagename, err = GetPackageName()
	if err!= nil {
		return
	}
	return
}

func interactiveGoTest() (err error) {
	goTester.Packagename, err = GetPackageName()
	if err!= nil {
		return
	}
	infos.Println("RECAP:")
	color.Cyan(fmt.Sprintf("Testing go PACKAGE:%s", goTester.Packagename))
	infos.Println("=> Execute?")
	input := prompt.Choose("> ", yesOrNoChoices())
	if strings.ToLower(input) != "yes" {
		infos.Println("Exiting...")
		os.Exit(0)
	}
	return
}

func preRunGoTest(cmd *cobra.Command, args []string) (err error) {
	if interactiveARG {
		return interactiveGoTest()
	} else {
		return notInteractiveGoTest(cmd)
	}
}

type GoTester struct {
	Packagename string
}

func (g *GoTester) RunCommand(cmd *cobra.Command, args []string) error {
	pwd, _ := os.Getwd()
	cc := TestContainerManager{
		PackageName:     g.Packagename,
		HostPackagePath: pwd,
		HostSSHPath:     filepath.Join(os.Getenv("HOME"), ".ssh"),
	}

	conf, hostConf, err := cc.createContainerStartConfiguration()
	if err != nil {
		return err
	}
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	//retrieve image from repository
	infos.Println("1. Pulling Docker image \r\n\t", imageForBuildname)
	_, err = cli.ImagePull(ctx, imageForBuildname, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	infos.Println("2. Command to test \r\n\t", strings.Join(conf.Cmd, " "))
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
	failing := false
	for scanner.Scan() {
		txt := scanner.Text()
		if strings.HasPrefix(txt, "FAIL") {
			failing = true
			color.Red(txt)
		} else {
			color.White(txt)
		}
	}
	if failing {
		return fmt.Errorf("failing tests exiting")
	}
	defer  cli.ContainerRemove(ctx,resp.ID, types.ContainerRemoveOptions{RemoveLinks:true, RemoveVolumes:true, Force: true} )
	return nil
}
