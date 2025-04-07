package transport

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/util"
)

func NewS3Transport(name string, bucketUrl string) config.Transport {
	return &s3Transport{name, bucketUrl}
}

type s3Transport struct {
	name      string
	bucketUrl string
}

func (t *s3Transport) Validate(config config.Config) error {
	errs := []error{}
	src := config.SrcExecutor
	if err := ValidateAWSCLIInstallation(src); err != nil {
		errs = append(errs, fmt.Errorf("Source '%s' does not have AWS CLI ('aws') available on PATH; install or update PATH & try again", src.Name()))
	} else if err := ValidateAWSCLILogin(src); err != nil {
		errs = append(errs, fmt.Errorf("Source '%s' is not authenticated with S3; authenticate & try again", src.Name()))
	}

	for _, dst := range config.DstExecutors {
		if err := ValidateAWSCLIInstallation(dst); err != nil {
			errs = append(errs, fmt.Errorf("Destination '%s' does not have AWS CLI ('aws') available on PATH; install or update PATH & try again", dst.Name()))
		} else if err := ValidateAWSCLILogin(dst); err != nil {
			errs = append(errs, fmt.Errorf("Destination '%s' is not authenticated with S3; authenticate & try again", dst.Name()))
		}
	}

	return errors.Join(errs...)
}

func ValidateAWSCLIInstallation(e config.Executor) error {
	_, _, err := e.ExecuteShell("which aws")
	return err
}

func ValidateAWSCLILogin(e config.Executor) error {
	_, _, err := e.ExecuteShell("aws sts caller-identity ")
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
