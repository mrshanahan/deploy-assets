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

## Manifest

The following types are available for each section:

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

## Caveats

- This is not a super sophisticated tool - there are no retries nor lots of flexible options. It serves my own needs specifically.
- SSH interactions currently use `bash` by default. There is currently no way to specify another shell; you will have to modify it yourself.
- Not a lot of sophisticated command-line quoting or escaping is done, and command lines are build slightly differently between local & SSH executors. Proceed at your own risk.


