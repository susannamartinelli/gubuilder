package subcommands

import (
	"github.com/spf13/cobra"
	"github.com/c-bata/go-prompt"
	"strings"
	"fmt"
	"github.com/fatih/color"
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/aws"
	"os"
	"sort"
	"encoding/base64"
	"time"
	"text/template"
	"bytes"
	"os/exec"
)

var remoteImageARG string
var imageDeployer = &ImageDeployer{}
var imageDeployerArguments = &GuArgumentSlice{}
var deployCmd = &cobra.Command{
	Use:     "image-deploy",
	Short:   "Deploy the docker image ",
	Long:    `Deploy the docker image on AWS (TIP: use 'tab-key' to autocomplete suggestions)`,
	PreRunE: preRunImageDeploy,
	RunE:    imageDeployer.RunCommand,
}
func init() {
	deployCmd.Flags().BoolVarP(&interactiveARG, "interactive", "i", false, "Interactive or not")
	imageDeployerArguments.AddGuArgument(imageNameArgName,"",&imageARG, infos.Sprintf("The image name: in the form geouniq.com/..."))
	imageDeployerArguments.AddGuArgument(ecsImageTagArgName,"",&tagARG, infos.Sprintf("The image tag: in the form vX.X.X..."))
	imageDeployerArguments.AddGuArgument(ecsImageNameArgName,"",&remoteImageARG, infos.Sprintf("The image tag: in the form vX.X.X..."))
	imageDeployerArguments.AddArgumentsToCobraCommand(deployCmd)
}
func notInteractiveImageDeploy(cmd *cobra.Command) (err error) {
	err = CheckArguments(cmd.Flags(), imageDeployerArguments.Names())
	imageDeployer.LocalImage = cmd.Flag(imageNameArgName).Value.String()
	imageDeployer.RemoteImageTag = cmd.Flag(ecsImageTagArgName).Value.String()
	imageDeployer.RemoteImage = cmd.Flag(ecsImageNameArgName).Value.String()
	return
}

func interactiveImageDeploy() (err error) {
	var input string
	var choices []string
	var ecrSession *ecr.ECR
	var repositoryURI string
	//aws session
	ecrSession, err = GetECRSession()
	if err!= nil {
		color.Red("Unable to get local repositories list: %s", err)
		return err
	}
	//local repository
	choices, err = localRepositoriesChoices()
	if err!= nil {
		color.Red("Unable to get local repositories list: %s", err)
		return err
	}
	infos.Println("1 - Please select a LOCAL docker image REPOSITORY.")
	input = prompt.Choose("> ", choices)
	imageDeployer.LocalImage = input

	//remote repository
	choices, repositoryURI, _, err = remoteRepositoriesChoices(ecrSession)
	if err!= nil {
		color.Red("Unable to get remote repositories list: %s", err)
		return err
	}
	infos.Println("2 - Please select a AWS docker REPOSITORY ("+repositoryURI+").")
	input = prompt.Choose("> ", choices)
	imageDeployer.RemoteImage = input
	imageDeployer.RemoteRepositoryUri = repositoryURI

	//remote image
	choices, err = remoteImageChoices(ecrSession, imageDeployer.RemoteImage)
	if err!= nil {
		color.Red("Unable to get remote repositories list: %s", err)
		return err
	}
	infos.Println("3 - Please select a AWS docker IMAGE.")
	input = prompt.Choose("> ", choices)
	imageDeployer.RemoteImageTag = input

	//confirming
	choices = yesOrNoChoices()
	infos.Println("RECAP: Your commands are:")
	color.Cyan(fmt.Sprintf("  docker tag %s %s", imageDeployer.LocalImage,imageDeployer.getRemoteImageUrl()))
	color.Cyan(fmt.Sprintf("  docker push %s", imageDeployer.getRemoteImageUrl()))
	infos.Println("=> Execute?")
	input = prompt.Choose("> ", choices)
	if strings.ToLower(input) != "yes" {
		infos.Println("Exiting...")
		os.Exit(0)
	}
	return nil
}



func preRunImageDeploy(cmd *cobra.Command, args []string) (err error) {
	if interactiveARG {
		return interactiveImageDeploy()
	} else {
		return notInteractiveImageDeploy(cmd)
	}

}

func yesOrNoChoices ()[]string {
	return []string{"yes", "no"}
}

func localRepositoriesChoices() (choices []string, err error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	defer cli.Close()
	args := filters.NewArgs()
	if err != nil {
		return
	}
	opts := types.ImageListOptions{
		All: true,
		Filters: args,
	}
	images, err := cli.ImageList(ctx, opts)
	if err != nil {
		return
	}
	for _,i := range images {
		for _, v := range i.RepoTags {
			choices = append(choices, v)
		}
	}
	sort.Strings(choices)
	return
}

func remoteRepositoriesChoices(svc *ecr.ECR) (choices []string, repositoryURI string, registryID string, err error) {
	//get the repository name

	repos, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{})
	if err != nil {
		return
	}
	for _, r := range repos.Repositories {
		choices = append(choices, *r.RepositoryName)
		registryID = *r.RegistryId
		repositoryURI = strings.TrimSuffix(*r.RepositoryUri, *r.RepositoryName)

	}

	sort.Strings(choices)
	return
}

func remoteImageChoices(svc *ecr.ECR, remoteRepositoryUri string)  (choices []string, err error) {
	images, err := svc.ListImages(&ecr.ListImagesInput{
		RepositoryName: aws.String(remoteRepositoryUri),
	})
	if err != nil {
		return
	}
	for _, i := range images.ImageIds {
		if i.ImageTag!=nil {
			choices = append(choices, *i.ImageTag)
		}
	}
	sort.Strings(choices)
	return
}

type ImageDeployer struct {
	RemoteRepositoryUri string
	LocalImage string
	RemoteImage string
	RemoteImageTag string

}
func (i *ImageDeployer) getRemoteImageUrl() string {
	return fmt.Sprintf("%s%s:%s", i.RemoteRepositoryUri, i.RemoteImage, i.RemoteImageTag)
}
func (i *ImageDeployer) RunCommand(cmd *cobra.Command, args []string) error {

	type Auth struct {
		Token         string
		User          string
		Pass          string
		ProxyEndpoint string
		ExpiresAt     time.Time
	}
	type Image struct {
		Local string
		Remote string
	}
	infos.Println()
	err := i.checkImageName()
	if err != nil {
		color.Red(fmt.Sprintf("Check image name error: %s", err.Error()))
		return err
	}
	err = i.checkImageTag()
	if err != nil {
		color.Red(fmt.Sprintf("Check image tag error: %s", err.Error()))
		return err
	}
	ecrSession, err := GetECRSession()
	if err != nil {
		color.Red(fmt.Sprintf("Getting ECR credentials: %s", err.Error()))
		return err
	}

	authToken, err  := ecrSession.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		color.Red(fmt.Sprintf("Getting ECR auth token: %s", err.Error()))
		return err
	}
	fields := make([]Auth, len(authToken.AuthorizationData))
	for i, auth := range authToken.AuthorizationData {
		// extract base64 token
		data, err := base64.StdEncoding.DecodeString(*auth.AuthorizationToken)
		if err != nil {
			color.Red(fmt.Sprintf("Getting ECR auth token: %s", err.Error()))
			return err
		}
		// extract username and password
		token := strings.SplitN(string(data), ":", 2)

		// object to pass to template
		fields[i] = Auth{
			Token:         *auth.AuthorizationToken,
			User:          token[0],
			Pass:          token[1],
			ProxyEndpoint: *(auth.ProxyEndpoint),
			ExpiresAt:     *(auth.ExpiresAt),
		}
	}
	image := []Image{
		{
			Local: i.LocalImage,
			Remote: i.getRemoteImageUrl(),
		},
	}
	const LOGIN_TEMPLATE = `{{range .}}login -u {{.User}} -p {{.Pass}} {{.ProxyEndpoint}}{{end}}`
	const TAG_TEMPLATE = `{{range .}}tag {{.Local}} {{.Remote}}{{end}}`
	const PUSH_TEMPLATE = `{{range .}}push {{.Remote}}{{end}}`
	var tpl bytes.Buffer

	loginTemplate, err := template.New("login").Parse(LOGIN_TEMPLATE)
	err = loginTemplate.Execute(&tpl, fields)
	if err != nil {
		color.Red(fmt.Sprintf("Registry Login error: %s", err.Error()))
		return err
	}
	infos.Println("1. Login to ECR repository")
	command := tpl.String()
	argumts := strings.Split(command, " ")
	outCmd := exec.Command("docker", argumts...)
	output, err   := outCmd.CombinedOutput()
	if err != nil {
		color.Red(fmt.Sprintf("Registry Login error: %s", string(output)))
		return err
	} else {
		color.White(string(output))
	}

	tpl.Reset()
	tagTemplate, err := template.New("tag").Parse(TAG_TEMPLATE)
	err = tagTemplate.Execute(&tpl, image)
	if err != nil {
		color.Red(fmt.Sprintf("Tag image error: %s", err.Error()))
		return err
	}
	infos.Println("2. Tag image")
	command = tpl.String()
	argumts = strings.Split(command, " ")
	outCmd = exec.Command("docker", argumts...)
	output, err   = outCmd.CombinedOutput()
	if err != nil {
		color.Red(fmt.Sprintf("Tag image error: %s",  string(output)))
		return err
	} else {
		color.White(string(output))
	}

	tpl.Reset()
	pushTemplate, err := template.New("push").Parse(PUSH_TEMPLATE)
	err = pushTemplate.Execute(&tpl, image)
	if err != nil {
		color.Red(fmt.Sprintf("Push image error: %s", err.Error()))
		return err
	}
	infos.Println("3. Push image")
	command = tpl.String()
	argumts = strings.Split(command, " ")
	outCmd = exec.Command("docker", argumts...)
	output, err   = outCmd.CombinedOutput()

	if err != nil {
		color.Red(fmt.Sprintf("Push image error: %s", string(output)))
		return err
	} else {
		color.White(string(output))
	}

	return nil
}

func (i ImageDeployer) checkImageName()  (err error){
	return CheckDockerImageNameForGeouniq(i.LocalImage)
}
func (i ImageDeployer) checkImageTag()  (err error){
	return CheckDockerImageTagForGeouniq(i.RemoteImageTag)
}

