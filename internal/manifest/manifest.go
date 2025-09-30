package manifest

import (
	"errors"
	"fmt"

	"github.com/mrshanahan/deploy-assets/internal/config"
	"github.com/mrshanahan/deploy-assets/internal/executor"
	"github.com/mrshanahan/deploy-assets/internal/provider"
	"github.com/mrshanahan/deploy-assets/internal/transport"
)

type Manifest struct {
	Executors map[string]config.Executor
	Transport config.Transport
	Providers []*config.ProviderConfig
}

type defaultNameTracker struct {
	counter map[string]int
}

func newDefaultNameTracker() *defaultNameTracker {
	return &defaultNameTracker{make(map[string]int)}
}

func (t *defaultNameTracker) GetName(kind string, typ string) string {
	key := fmt.Sprintf("%s/%s", kind, typ)
	val := t.counter[key] + 1
	t.counter[key] = val
	return fmt.Sprintf("%s%d", typ, val)
}

func BuildManifest(root *ManifestNode) (*Manifest, error) {
	manifest := &Manifest{
		Executors: map[string]config.Executor{},
		Transport: nil,
		Providers: []*config.ProviderConfig{},
	}

	errs := buildExecutors(root, manifest)
	errs = append(errs, buildTransport(root, manifest)...)
	errs = append(errs, buildProviders(root, manifest)...)

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	} else {
		return manifest, nil
	}
}

func buildExecutors(root *ManifestNode, manifest *Manifest) []error {
	locationsNode := root.Kinds["locations"]
	errs := []error{}
	defaultNames := newDefaultNameTracker()
	for _, l := range locationsNode.Items {
		var name string
		nameAttr, prs := l.Attributes["name"]
		if !prs || !nameAttr.Present {
			name = defaultNames.GetName("locations", l.Type)
		} else {
			name = nameAttr.GetValue().(string)
		}
		switch l.Type {
		case "local":
			manifest.Executors[name] = executor.NewLocalExecutor(name)
		case "ssh":
			addr := l.Attributes["server"].GetValue().(string)
			user := l.Attributes["username"].GetValue().(string)
			keyPath := l.Attributes["key_file"].GetValue().(string)
			keyPassphrase := l.Attributes["key_file_passphrase"].GetValue().(string)
			runElevated := l.Attributes["run_elevated"].GetValue().(bool)
			exec, err := executor.NewSSHExecutor(name, addr, user, keyPath, keyPassphrase, runElevated)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to initialize executor for location '%s': %v", name, err))
			} else {
				manifest.Executors[name] = exec
			}
		default:
			errs = append(errs, fmt.Errorf("unknown executor type: %s", l.Type))
		}
	}
	return errs
}

func buildTransport(root *ManifestNode, manifest *Manifest) []error {
	transportNode := root.Kinds["transport"]
	errs := []error{}
	defaultNames := newDefaultNameTracker()

	t := transportNode.Items[0]
	var name string
	nameAttr, prs := t.Attributes["name"]
	if !prs || !nameAttr.Present {
		name = defaultNames.GetName("transport", t.Type)
	} else {
		name = nameAttr.GetValue().(string)
	}
	switch t.Type {
	case "s3":
		bucketUrl := t.Attributes["bucket_url"].GetValue().(string)
		manifest.Transport = transport.NewS3Transport(name, bucketUrl)
	default:
		errs = append(errs, fmt.Errorf("unknown transport type: %s", t.Type))
	}

	return errs
}

func buildProviders(root *ManifestNode, manifest *Manifest) []error {
	assetsNode := root.Kinds["assets"]
	errs := []error{}
	defaultNames := newDefaultNameTracker()
	for _, a := range assetsNode.Items {
		var name string
		nameAttr, prs := a.Attributes["name"]
		if !prs || !nameAttr.Present {
			name = defaultNames.GetName("assets", a.Type)
		} else {
			name = nameAttr.GetValue().(string)
		}

		src := a.Attributes["src"].GetValue().(string)
		dst := a.Attributes["dst"].GetValue().(string)

		if _, prs := manifest.Executors[src]; !prs {
			errs = append(errs, fmt.Errorf("no such location: %s", src))
			continue
		}
		if _, prs := manifest.Executors[dst]; !prs && dst != "*" {
			errs = append(errs, fmt.Errorf("no such location: %s", dst))
			continue
		}

		postCommand := a.Attributes["post_command"].GetValue().(string)
		runPostCommand := a.Attributes["run_post_command"].GetValue().(string)
		// TODO: move this validation elsewhere/introduce an "enum" type
		if runPostCommand != "always" && runPostCommand != "on_changed" {
			errs = append(errs, fmt.Errorf("invalid post-command condition: %s", runPostCommand))
			continue
		}

		providerConfig := &config.ProviderConfig{
			Src:            src,
			Dst:            dst,
			PostCommand:    postCommand,
			RunPostCommand: runPostCommand,
		}

		switch a.Type {
		case "file":
			srcPath := a.Attributes["src_path"].GetValue().(string)
			dstPath := a.Attributes["dst_path"].GetValue().(string)
			recursive := a.Attributes["recursive"].GetValue().(bool)
			providerConfig.Provider = provider.NewFileProvider(name, srcPath, dstPath, recursive)
		case "docker_image":
			compareLabel := a.Attributes["compare_label"].GetValue().(string)
			repositoryAttr := a.Attributes["repository"]
			var repositories []string
			if repositoryAttr.MatchingValueType == "string" {
				repositories = []string{repositoryAttr.GetValue().(string)}
			} else {
				repositories = repositoryAttr.GetValue().([]string)
			}
			providerConfig.Provider = provider.NewDockerProvider(name, repositories, compareLabel)
		default:
			errs = append(errs, fmt.Errorf("unknown provider type: %s", a.Type))
			continue
		}

		manifest.Providers = append(manifest.Providers, providerConfig)
	}
	return errs
}
