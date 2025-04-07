package manifest

func NewManifestSpec() *ManifestSpec {
	return &ManifestSpec{
		Kinds: []ManifestKindSpec{
			&LocationKindSpec{
				GenericKindSpec: GenericKindSpec{
					itemSpecs: []ManifestItemSpec{
						&LocalLocationItemSpec{},
						&SSHLocationItemSpec{},
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

type LocationKindSpec struct {
	GenericKindSpec
}

func (s *LocationKindSpec) Name() string { return "locations" }

func (s *LocationKindSpec) IsCollection() bool { return true }

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

type LocalLocationItemSpec struct {
	Path string
}

func GetDefaultLocationItemAttributes() []AttributeSpec {
	return []AttributeSpec{
		AttributeSpec{"name", "string", true},
	}
}

func (s *LocalLocationItemSpec) Type() string { return "local" }

func (s *LocalLocationItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultLocationItemAttributes(),
		[]AttributeSpec{}...,
	)
}

type SSHLocationItemSpec struct{}

func (s *SSHLocationItemSpec) Type() string { return "ssh" }

func (s *SSHLocationItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultLocationItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"server", "string", true},
			AttributeSpec{"username", "string", true},
			AttributeSpec{"key_file", "string", true},
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

func GetDefaultAssetItemAttributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"src", "string", true},
			AttributeSpec{"dst", "string", true},
		}...,
	)
}

func (s *DirAssetItemSpec) Type() string { return "dir" }

func (s *DirAssetItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultAssetItemAttributes(),
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
		GetDefaultAssetItemAttributes(),
		[]AttributeSpec{
			AttributeSpec{"repository", "string|[]string", true},
		}...,
	)
}
