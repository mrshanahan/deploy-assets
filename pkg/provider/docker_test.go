package provider

import (
	"testing"

	"github.com/mrshanahan/deploy-assets/internal/util"
)

func TestDockerYamlDefault(t *testing.T) {
	p := NewDockerProvider("floop", []string{"flim/blam:latest", "florp:v1"}, "borp")
	expected :=
		`docker:
    name: floop
    repositories:
        - flim/blam:latest
        - florp:v1
    compare_label: borp`
	actual := p.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestDockerYamlDeep(t *testing.T) {
	p := NewDockerProvider("floop", []string{"flim/blam:latest", "florp:v1"}, "borp")
	expected :=
		`        docker:
            name: floop
            repositories:
                - flim/blam:latest
                - florp:v1
            compare_label: borp`
	actual := p.Yaml(util.TabsToIndent(2))
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}

func TestDockerYamlNoLabel(t *testing.T) {
	p := NewDockerProvider("floop", []string{"flim/blam:latest", "florp:v1"}, "")
	expected :=
		`docker:
    name: floop
    repositories:
        - flim/blam:latest
        - florp:v1
    compare_label: `
	actual := p.Yaml(0)
	if expected != actual {
		t.Errorf("yaml contents not equal:\nexpected:\n=======\n%s\n=======\ngot:\n=======\n%s\n=======", expected, actual)
	}
}
