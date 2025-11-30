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
						&FileAssetItemSpec{},
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
	Name         string
	ValueType    string
	IsRequired   bool
	DefaultValue any
}

func RequiredAttribute(name, valueType string) AttributeSpec {
	return AttributeSpec{name, valueType, true, nil}
}

func OptionalAttribute(name, valueType string, defaultValue any) AttributeSpec {
	return AttributeSpec{name, valueType, false, defaultValue}
}

type ManifestItemSpec interface {
	Type() string
	Attributes() []AttributeSpec
}

func GetDefaultItemAttributes() []AttributeSpec {
	return []AttributeSpec{
		OptionalAttribute("name", "string", ""),
	}
}

type LocalLocationItemSpec struct {
	Path string
}

func GetDefaultLocationItemAttributes() []AttributeSpec {
	return []AttributeSpec{
		RequiredAttribute("name", "string"),
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
			RequiredAttribute("server", "string"),
			RequiredAttribute("username", "string"),
			RequiredAttribute("key_file", "string"),
			OptionalAttribute("key_file_passphrase", "string", ""),
			OptionalAttribute("run_elevated", "bool", false), // TODO: specify default value?
		}...,
	)
}

type S3TransportItemSpec struct{}

func (s *S3TransportItemSpec) Type() string { return "s3" }

func (s *S3TransportItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			RequiredAttribute("bucket_url", "string"),
		}...,
	)
}

type FileAssetItemSpec struct{}

func GetDefaultAssetItemAttributes() []AttributeSpec {
	return append(
		GetDefaultItemAttributes(),
		[]AttributeSpec{
			RequiredAttribute("src", "string"),
			RequiredAttribute("dst", "string"),
			OptionalAttribute("post_command", "[]object", []map[string]any{}),
		}...,
	)
}

func (s *FileAssetItemSpec) Type() string { return "file" }

func (s *FileAssetItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultAssetItemAttributes(),
		[]AttributeSpec{
			RequiredAttribute("src_path", "string"),
			RequiredAttribute("dst_path", "string"),
			OptionalAttribute("recursive", "bool", false),
		}...,
	)
}

type DockerImageAssetItemSpec struct{}

func (s *DockerImageAssetItemSpec) Type() string { return "docker_image" }

func (s *DockerImageAssetItemSpec) Attributes() []AttributeSpec {
	return append(
		GetDefaultAssetItemAttributes(),
		[]AttributeSpec{
			RequiredAttribute("repository", "string|[]string"),
			OptionalAttribute("compare_label", "string", ""),
		}...,
	)
}
