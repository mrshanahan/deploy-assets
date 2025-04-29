package executor

import (
	"encoding/base64"
	"fmt"

	"github.com/mrshanahan/deploy-assets/internal/util"
)

func createCommandScript(cmd string, exec func(string) error) (string, error) {
	scriptPathBase64 := util.GetTempFilePath("deploy-assets-execute-b64")
	scriptContentsBase64 := base64.StdEncoding.EncodeToString([]byte(cmd))
	createScriptCmd := fmt.Sprintf("echo '%s' | base64 -d > \"%s\"", scriptContentsBase64, scriptPathBase64)
	if err := exec(createScriptCmd); err != nil {
		return "", err
	}
	return scriptPathBase64, nil
}
