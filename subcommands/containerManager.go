package subcommands

import (
	"os"
	"fmt"
	"time"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"path/filepath"

)

type BuildContainerManager struct {
	TestContainerManager
	PackageVersion string
	Main           string
	OsType		   string
}

func (cc BuildContainerManager) getContainerCommand() (string, error) {
	glideCommands := "glide i -force;"
	ldFlags := fmt.Sprintf("-s -X main.Version=%s -X main.BuildTime=%s -X main.GitHash=$(git rev-parse HEAD)", cc.PackageVersion, time.Now().UTC().Format(time.RFC3339))
	return fmt.Sprintf("%s CGO_ENABLED=0 GOOS=%s go build -v -a -installsuffix cgo -o bin/main -ldflags \"%s\" %s", glideCommands, cc.OsType, ldFlags, cc.Main), nil
}

func (cc BuildContainerManager) checkVersion() error {
	return ParseSemanticVersion(cc.PackageVersion)
}

func (cc BuildContainerManager) checkMain() error {
	if _, err := os.Stat(filepath.Join(cc.HostPackagePath, cc.Main)); err!= nil {
		return err
	}
	return nil
}
func (cc BuildContainerManager) checks() error {
	err := cc.TestContainerManager.checks()
	if err != nil {
		return err
	}
	err = cc.checkVersion()
	if err != nil {
		return err
	}
	err = cc.checkMain()
	if err != nil {
		return  err
	}
	return nil
}
func (cc BuildContainerManager) createContainerStartConfiguration() (*container.Config, *container.HostConfig, error) {
	err := cc.checks()
	if err != nil {
		return nil, nil, err
	}

	dockerCmd, err := cc.getContainerCommand()
	if err != nil {
		return nil, nil, err
	}
	c, h := cc.configs(dockerCmd)
	return c, h, nil
}

type TestContainerManager struct {
	PackageName     string
	HostPackagePath string
	HostSSHPath     string
}

func (cc TestContainerManager) getContainerCommand() (string, error) {
	glideCommands := "glide i -force"
	return fmt.Sprintf("%s go test %s/...", glideCommands, cc.PackageName), nil
}

func (cc TestContainerManager) getContainerPackagePath() string {
	return filepath.Join("/go/src", cc.PackageName)
}
func (cc TestContainerManager) checks () error {
	err := cc.checkLocalSSHDirectory()
	if err != nil {
		return  err
	}
	err = cc.checkLocalPackageDirectory()
	if err != nil {
		return err
	}
	return nil
}
func (cc TestContainerManager) createContainerStartConfiguration() (*container.Config, *container.HostConfig, error) {
	err := cc.checks()
	if err != nil {
		return nil, nil, err
	}

	dockerCmd, err := cc.getContainerCommand()
	if err != nil {
		return nil, nil, err
	}
	c, h := cc.configs(dockerCmd)
	return c, h, nil

}
func (cc TestContainerManager) configs(cmd string) (*container.Config, *container.HostConfig){
	conf := container.Config{
		Image:        imageForBuildname,
		Cmd:          []string{"/bin/bash", "-c", cmd},
		WorkingDir:   cc.getContainerPackagePath(),
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		ArgsEscaped:  true,
	}

	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: cc.HostPackagePath,
				Target: cc.getContainerPackagePath(),
			},
			{
				Type:   mount.TypeBind,
				Source: cc.HostSSHPath,
				Target: "/root/.ssh",
			},
		},
	}
	return &conf, &hostConfig
}

func (cc TestContainerManager) checkLocalSSHDirectory() error {
	if info, err := os.Stat(cc.HostSSHPath); os.IsNotExist(err) {
		return fmt.Errorf("ssh dir %s not found", cc.HostSSHPath)
	} else if !info.IsDir() {
		return fmt.Errorf(".ssh %s is not a directory", cc.HostSSHPath)
	}
	return nil
}

func (cc *TestContainerManager) checkLocalPackageDirectory() error {
	if info, err := os.Stat(cc.HostPackagePath); os.IsNotExist(err) {
		return fmt.Errorf("local package dir %s not found", cc.HostPackagePath)
	} else if !info.IsDir() {
		return fmt.Errorf("package dir %s is not a directory", cc.HostPackagePath)
	} else {
		if cc.HostPackagePath, err = filepath.Abs(cc.HostPackagePath); err!= nil {
			return err
		}

	}

	return nil
}
