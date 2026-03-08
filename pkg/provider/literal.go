package provider

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mrshanahan/deploy-assets/internal/util"
	"github.com/mrshanahan/deploy-assets/pkg/config"
)

func NewLiteralProvider(name, value, dstPath string) config.Provider {
	return &literalProvider{
		name:    name,
		value:   value,
		dstPath: dstPath,
	}
}

type literalProvider struct {
	name    string
	value   string
	dstPath string
}

func (p *literalProvider) Name() string { return p.name }

func (p *literalProvider) Yaml(indent int) string {
	propIndent := util.YamlIndentString(indent + util.TabsToIndent(1))
	return fmt.Sprintf(
		`%sliteral:
%sname: %s
%svalue: |
%s
%sdst_path: %s`,
		util.YamlIndentString(indent),
		propIndent, p.name,
		propIndent,
		util.IndentLines(p.value, indent+util.TabsToIndent(2)),
		propIndent, p.dstPath,
	)
}

func (p *literalProvider) Sync(cfg config.SyncConfig) (config.SyncResult, error) {
	b64Value := base64.StdEncoding.EncodeToString([]byte(p.value))
	stdoutRaw, _, err := cfg.DstExecutor.ExecuteShell(fmt.Sprintf("(test -e '%s' && echo 'exists') || echo 'not-exists'", p.dstPath))
	if err != nil {
		return config.SYNC_RESULT_NOCHANGE, fmt.Errorf("failed to check for existing target file '%s': %w", p.dstPath, err)
	}
	stdout := strings.Trim(stdoutRaw, " \n")
	var successResult config.SyncResult
	if stdout == "exists" {
		successResult = config.SYNC_RESULT_UPDATED
	} else {
		successResult = config.SYNC_RESULT_CREATED
	}

	if _, _, err := cfg.DstExecutor.ExecuteShell(fmt.Sprintf("echo '%s' | base64 -d > '%s'", b64Value, p.dstPath)); err != nil {
		return config.SYNC_RESULT_NOCHANGE, fmt.Errorf("failed to write value to %s: %w", p.dstPath, err)
	}

	return successResult, nil
}
