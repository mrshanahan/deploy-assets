package transport

import (
	"fmt"
	"net/url"

	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
)

func NewS3Transport(name string, bucketUrl string) config.Transport {
	return &s3Transport{name, bucketUrl}
}

type s3Transport struct {
	name      string
	bucketUrl string
}

func (t *s3Transport) Yaml(indent int) string {
	propIndent := util.YamlIndentString(indent + util.TabsToIndent(1))
	return fmt.Sprintf(
		`%ss3:
%sname: %s
%sbucket_url: %s`,
		util.YamlIndentString(indent),
		propIndent, t.name,
		propIndent, t.bucketUrl)
}

func (t *s3Transport) Validate(exec config.Executor) error {
	if err := ValidateAWSCLIInstallation(exec); err != nil {
		return fmt.Errorf("location '%s' does not have AWS CLI ('aws') available on PATH; install or update PATH & try again", exec.Name())
	}
	if err := ValidateAWSCLILogin(exec); err != nil {
		return fmt.Errorf("location '%s' is not authenticated with S3; authenticate & try again", exec.Name())
	}

	return nil
}

func ValidateAWSCLIInstallation(e config.Executor) error {
	_, _, err := e.ExecuteShell("which aws")
	return err
}

func ValidateAWSCLILogin(e config.Executor) error {
	_, _, err := e.ExecuteShell("aws sts get-caller-identity")
	return err
}

func (t *s3Transport) TransferFile(src config.Executor, srcPath string, dst config.Executor, dstPath string) error {
	bucketSubpath := util.GetTimestampedFileName("transfer")
	fullBucketPath, err := url.JoinPath(t.bucketUrl, bucketSubpath)
	if err != nil {
		return err
	}
	if _, _, err := src.ExecuteCommand("aws", "s3", "cp", srcPath, fullBucketPath); err != nil {
		return err
	}
	defer src.ExecuteCommand("aws", "s3", "rm", fullBucketPath)

	if _, _, err := dst.ExecuteCommand("aws", "s3", "cp", fullBucketPath, dstPath); err != nil {
		return err
	}

	return nil
}
