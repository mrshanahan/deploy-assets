# deploy-assets

A Go-based tool for copying assets around to various locations.

The tool uses a JSON _manifest_ to determine how & where & what to move. For example, the following manifest will move a single directory from the local machine to a remote server where commands are issued via SSH. (Values in `{{ DOUBLE_BRACKETS }}` are replaced by environment variables):

    {
        "locations": [
            {
                "type": "local",
                "name": "local"
            },
            {
                "type": "ssh",
                "name": "remote",
                "server": "foo.internal:22",
                "username": "{{ SSH_USERNAME }}",
                "key_file": "{{ SSH_KEY_FILE }}"
            }
        ],
        "transport": {
            "type": "s3",
            "bucket_url": "s3://foo-deploy-assets"
        },
        "assets": [
            {
                "type": "dir",
                "src": "local",
                "src_path": "./package",
                "dst": "remote",
                "dst_path": "/etc/foo-service/package"
            }
        ]
    }

Then run the tool pointing to this manifest:

    $ SSH_USERNAME=foo SSH_KEY_FILE=~/.ssh/foo.pem deploy-assets -manifest ./foo-manifest.json

For more options use the `-help` flag:

    $ deploy-assets -help

## Authentication

### AWS (`s3` transport)

The `s3` transport type will be the most common way to upload files to an AWS-based instance. To authenticate you can use the `aws configure sso` command, which will have you create and name an AWS _profile_.

You can control the default AWS CLI profile using the `AWS_DEFAULT_PROFILE` environment variable. Set that either in a profile script or on the command line to control which profile you are using:

    $ AWS_DEFAULT_PROFILE=aws-deploy-assets deploy-assets -manifest ./foo-manifest.json

See [this page](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html) for information on configuring the IAM Identity Center authentication with the AWS CLI.

### SSH (`ssh` executor)

The `ssh` executor allows you to specify a remote user (`username`), private key (`key_file`), and private key passphrase (`key_file_passphrase`) for this executor. The passhprase is optional, and `deploy-assets` will treat an empty value for `key_file_passphrase` as an indicator that the key file is unencrypted.

Thus, for the following location block:

    {
        "locations": [
            {
                "type": "ssh",
                "name": "remote",
                "server": "foo.internal:22",
                "username": "{{ SSH_USERNAME }}",
                "key_file": "{{ SSH_KEY_FILE }}",
                "key_file_passphrase": "{{ SSH_KEY_FILE_PASSPHRASE }}"
            },
            ...
        ],
        ...
    }

The following will work for an unencrypted key `id_rsa_unencrypted`:

    $ SSH_USERNAME=ubuntu SSH_KEY_FILE=~/.ssh/id_rsa_unencrypted deploy-assets -manifest ./foo-manifest.json

And the following will work for an encrypted key `id_rsa_encrypted`:

    $ read -p "Enter passphrase for ssh key file: " -s ssh_key_file_passphrase
    $ SSH_USERNAME=ubuntu SSH_KEY_FILE=~/.ssh/id_rsa_encrypted SSH_KEY_FILE_PASSPHRASE=$ssh_key_file_passphrase deploy-assets -manifest ./foo-manifest.json

## Manifest

The _manifest_ is a JSON file that defines what assets need to be copied, where they are going, and how they are getting there. It is fundamentally a single object with three major subsections: `locations`, `transport`, and `assets`.

### `locations`

All location items have a required `name` property used to reference them in the rest of the manifest.
- `*` (all location types):
    - `name` (**required**, `string`): Name used to refer to this location. Unlike the other sections this must be provided as it will be used as a reference within the manifest.
- `local`: Targets the local environment where the tool is running. Commands are issued by subprocesses.
- `ssh`: Targets a remote environment over SSH.
    - `server` (**required**, `string`): Hostname plus port, e.g. `foo.com:22`
    - `username` (**required**, `string`): Username to use to connect to the server
    - `key_file` (**required**, `string`): Local path to the key file used to authenticate as given user
    - `run_elevated` (`bool`): If true, use `sudo` for all commmands. Defaults to `false`.

### `transport`

This is currently a single object rather than a collection. All transport types support an optional `name` (`string`) attribute; otherwise, their name will be generated based on their type.

- `*` (all transport types):
    - `name` (`string`): Name used to refer to this transport. If not provided it will be generated based on the type.
- `s3`: Use an S3 bucket to faciliate transfers between environments.
    - `bucket_url` (**required**, `string`): S3 URL to the bucket to use as the temporary cache for files, e.g. `s3://test-bucket`. Files will be cleaned up to the extent possible.


### `assets`

- `*` (all asset types):
    - `name` (`string`): Name used to refer to this asset. If not provided it will be generated based on the type.
    - `src` (**required**, `string`): Name of the location where the asset lives.
    - `dst` (**required**, `string`): Name of the location(s) where the asset should be transferred. Currently this must either be the name of a location or `*`, in which case the asset will be transferred to every other location than the source.
- `dir`: Transfer the contents of a directory.
    - `src_path` (**required**, `string`): Path to the directory in the source location.
    - `dst_path` (**required**, `string`): Path to the directory in the destination location.
        - Note that the asset will **_replace_** the given directory, not be copied into it.
        - E.g. if `src_path` is `foo` and contains `bar.txt` and `baz.zip` and `dst_path` is `/etc/foo`, then after the transfer `/etc/foo` will contain `bar.txt` and `baz.zip` , not a directory named `foo` with those files.
- `docker_image`: Package & transfer Docker container images.
    - `repository` (**required**, `string` or `string[]`): Names of images to package & transfer.
        - Wildcards are not accepted here; they will be treated literally.

### Variable expansion

Using `{{ VARIABLE_NAME }}` within a string in the manifest will cause that block to be replaced by the value of the environment variable `VARIABLE_NAME` at runtime. If the environment variable is empty or not defined, the replacement will be empty.

For example, say we have the value of `key_file` in an `ssh` location has the value `"{{ SSH_KEY_FILE }}"`.

- If we run `SSH_KEY_FILE=./id_rsa deploy-assets -manifest foo-manifest.json`, then `key_file` will have the value `"./id_rsa"`.
- If we run `deploy-assets -manifest foo-manifest.json` (and `SSH_KEY_FILE` is not set elsewhere), then `key_file` will have the value `""`.

Note that **path expansion is not supported (currently) in the manifest!** We assume all paths are absolute paths, so using `SSH_KEY_FILE=~/.ssh/id_rsa` would cause an error when setting up the SSH connection. Ensure paths are expanded in the shell before passing them to `deploy-assets`.

## Caveats

- This is not a super sophisticated tool - there are no retries nor lots of flexible options. It serves my own needs specifically.
- SSH interactions currently use `bash` by default. There is currently no way to specify another shell; you will have to modify it yourself.
- Not a lot of sophisticated command-line quoting or escaping is done, and command lines are build slightly differently between local & SSH executors. Proceed at your own risk.

## Development

This tool uses the standard Go project layout, so just `go run` or `go build` the main file in [`cmd/`](./cmd):

    $ go run ./cmd/deploy-assets.go
    $ go build ./cmd/deploy-assets.go

A Makefile is provided that will run basic commands, including installing the binary in your home directory's `bin`. (This is not system-agnosotic.)

    $ make              # Builds executable
    $ make build        # Same as above
    $ make run          # Default go run
    $ make test         # Runs all known tests
    $ make install      # Builds binary to ~/.local/bin/deploy-assets
    $ make install \    # Override default install directory
        INSTALL_DIR=/usr/local/bin
