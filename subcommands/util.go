package subcommands

import (
	"fmt"
	"strings"
	"os"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/spf13/pflag"
	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"path/filepath"
)
const (
	geouniqSuffix = "geouniq"
	geouniqTag = "v"
)


func CheckArguments(flagset *pflag.FlagSet, toCheck []string) (err error) {

	for _, c := range toCheck {
		if !flagset.Changed(c) {
			return fmt.Errorf("missing '--%s' argument", c)
		}
	}
	return
}

func CheckDockerImageNameForGeouniq(name string) (err error) {
	if !strings.HasPrefix(name, geouniqSuffix) {
		err = fmt.Errorf("wrong prefix for image name %s, should start with %s", name, geouniqSuffix)
	}
	return
}

func CheckDockerImageTagForGeouniq(tag string) (err error) {
	if !strings.HasPrefix(tag, geouniqTag) {
		err = fmt.Errorf("wrong prefix for image tag %s, should start with %s", tag, geouniqTag)
	}
	return
}

func GetECRSession() (*ecr.ECR, error) {
	homedir := os.Getenv("HOME")
	awsCredentials := credentials.NewSharedCredentials(filepath.Join(homedir, ".aws", "credentials"), "default")
	config := aws.Config{
		Region:      aws.String("eu-west-1"),
		Credentials: awsCredentials,
	}
	session, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})
	if err != nil {
		return nil, err
	}
	svc := ecr.New(session)
	return svc, nil
}

type GuArgument struct {
	Name        string
	Alias       string
	Pointer     *string
	Description string
}

type GuArgumentSlice []GuArgument

func NewGuArgumentSlice() GuArgumentSlice {
	gs := []GuArgument{}
	return gs
}
func (gs *GuArgumentSlice) AddGuArgument(name string, alias string, pointer *string, descr string) {
	gu := GuArgument{
		Name:        name,
		Alias:       alias,
		Pointer:     pointer,
		Description: descr,
	}
	*gs = append(*gs, gu)
	return
}
func (gs GuArgumentSlice) Names() (names []string) {
	for _, gu := range gs {
		names = append(names, gu.Name)
	}
	return
}
func (gs GuArgumentSlice) AddArgumentsToCobraCommand(cmd *cobra.Command) {
	for _, gu := range gs {
		cmd.Flags().StringVarP(gu.Pointer, gu.Name, gu.Alias, "", gu.Description)
	}
	RootCmd.AddCommand(cmd)
}

func ParseSemanticVersion(toParse string) (err error) {
	if !strings.HasPrefix(toParse, "v") && !strings.HasPrefix(toParse, "V"){
		err = fmt.Errorf("package version must start with 'V' or 'v' ")
		return
	}
	_, err = semver.NewVersion(toParse)
	return
}

func GetPackageName() (string, error) {
	type Glide struct {
		Package string `yaml:"package"`
	}
	g := Glide{}
	yamlFile, err := ioutil.ReadFile("./glide.yaml")

	if err != nil {
		return "", fmt.Errorf("you must run go-build in a directory which contains a 'glide.yaml': %s", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &g)
	if err != nil {
		return "", err
	}
	return g.Package, nil
}

func CheckDockerfile() (err error) {
	_, err = os.Stat("./Dockerfile")
	return
}