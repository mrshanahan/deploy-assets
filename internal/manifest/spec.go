package manifest

func NewManifestSpec() *ManifestSpec {
	return &ManifestSpec{
		Kinds: []ManifestKindSpec{
			&ServerKindSpec{
				GenericKindSpec: GenericKindSpec{
					itemSpecs: []ManifestItemSpec{
						&LocalServerItemSpec{},
						&SSHServerItemSpec{},
					},
				},
			},
			&TransportKindSpec{
				GenericKindSpec: GenericKindSpec{
					itemSpecs: []ManifestItemSpec{
						&S3TransportItemSpec{},
					},
				},
			},
			&AssetsKindSpec{
				GenericKindSpec: GenericKindSpec{
					itemSpecs: []ManifestItemSpec{
						&DirAssetItemSpec{},
						&DockerImageAssetItemSpec{},
					},
				},
			},
		},
	}
}

type ManifestSpec struct {
	Kinds []ManifestKindSpec
}

type ManifestKindSpec interface {
	Name() string
	IsCollection() bool
	ItemSpecs() map[string]ManifestItemSpec
}

type GenericKindSpec struct {
	itemSpecs []ManifestItemSpec
}

func (s *GenericKindSpec) ItemSpecs() map[string]ManifestItemSpec {
	m := make(map[string]ManifestItemSpec)
	for _, s := range s.itemSpecs {
		m[s.Type()] = s
	}
	return m
}

type ServerKindSpec struct {
	GenericKindSpec
}

func (s *ServerKindSpec) Name() string { return "servers" }

func (s *ServerKindSpec) IsCollection() bool { return true }

type TransportKindSpec struct {
	GenericKindSpec
}

func (s *TransportKindSpec) Name() string { return "transport" }

func (s *TransportKindSpec) IsCollection() bool { return false }

type AssetsKindSpec struct {
	GenericKindSpec
}

func (s *AssetsKindSpec) Name() string { return "assets" }

func (s *AssetsKindSpec) IsCollection() bool { return true }

type AttributeSpec struct {
	Name       string
	ValueType  string
	IsRequired bool
}

type ManifestItemSpec interface {
	Type() string
	Attributes() []AttributeSpec
}

func GetDefaultItemAttributes() []AttributeSpec {
	return []AttributeSpec{
		AttributeSpec{"name", "string", false},
	}
}

type LocalServerItemSpec struct {
	Path string
}

func (s *LocalServerItemSpec) Type() string { return "local" }

func (s *LocalServerItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"path", "string", true},
		}...,
	)
}

type SSHServerItemSpec struct{}

func (s *SSHServerItemSpec) Type() string { return "ssh" }

func (s *SSHServerItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"server", "string", true},
			AttributeSpec{"username", "string", true},
			AttributeSpec{"key_path", "string", true},
			AttributeSpec{"run_elevated", "bool", false}, // TODO: specify default value?
		}...,
	)
}

type S3TransportItemSpec struct{}

func (s *S3TransportItemSpec) Type() string { return "s3" }

func (s *S3TransportItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"bucket_url", "string", true},
		}...,
	)
}

type DirAssetItemSpec struct{}

func (s *DirAssetItemSpec) Type() string { return "dir" }

func (s *DirAssetItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"src_path", "string", true},
			AttributeSpec{"dst_path", "string", true},
		}...,
	)
}

type DockerImageAssetItemSpec struct{}

func (s *DockerImageAssetItemSpec) Type() string { return "docker_image" }

func (s *DockerImageAssetItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"repository", "string|[]string", true},
		}...,
	)
}
