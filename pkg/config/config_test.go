package config

import (
	"fmt"
	"testing"

	"github.com/mrshanahan/deploy-assets/internal/util"
)

type testProvider struct {
	name    string
	doodads int
}

func (p *testProvider) Name() string {
	return p.name
}

func (p *testProvider) Yaml(indent int) string {
	propIndent := util.YamlIndentString(indent + util.TabsToIndent(1))
	return fmt.Sprintf(
		`%sfile:
%sname: %s
%sdoodads: %d`,
		util.YamlIndentString(indent),
		propIndent, p.name,
		propIndent, p.doodads)

}

func (p *testProvider) Sync(config SyncConfig) (SyncResult, error) {
	return SYNC_RESULT_NOCHANGE, nil
}

func TestProviderConfigYamlDefault(t *testing.T) {
	c := &ProviderConfig{
		Provider: &testProvider{"foobar", 5},
		Src:      "hither",
		Dst:      "thither",
		PostCommands: []*PostCommand{
			{
				Command: "systemctl restart foobar.service",
				Trigger: "on_changed",
			},
			{
				Command: "rm /tmp/foobar.tmp",
				Trigger: "always",
			},
		},
	}

	expected :=
		`- src: hither
  dst: thither
  provider:
      file:
          name: foobar
          doodads: 5
  post_commands:
      - command: "systemctl restart foobar.service"
        trigger: on_changed
      - command: "rm /tmp/foobar.tmp"
        trigger: always`

	actual := c.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestProviderConfigYamlDeep(t *testing.T) {
	c := &ProviderConfig{
		Provider: &testProvider{"foobar", 5},
		Src:      "hither",
		Dst:      "thither",
		PostCommands: []*PostCommand{
			{
				Command: "systemctl restart foobar.service",
				Trigger: "on_changed",
			},
			{
				Command: "rm /tmp/foobar.tmp",
				Trigger: "always",
			},
		},
	}

	expected :=
		`        - src: hither
          dst: thither
          provider:
              file:
                  name: foobar
                  doodads: 5
          post_commands:
              - command: "systemctl restart foobar.service"
                trigger: on_changed
              - command: "rm /tmp/foobar.tmp"
                trigger: always`

	actual := c.Yaml(util.TabsToIndent(2))
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}
