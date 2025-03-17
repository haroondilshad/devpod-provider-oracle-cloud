# DevPod Provider Oracle Cloud

<!-- markdownlint-disable-next-line MD013 MD034 -->
[![Go Report Card](https://goreportcard.com/badge/github.com/haroondilshad/devpod-provider-oracle-cloud)](https://goreportcard.com/report/github.com/haroondilshad/devpod-provider-oracle-cloud)

DevPod on Oracle Cloud Infrastructure

<!-- toc -->

* [Usage](#usage)
* [Development](#development)
  * [Required environment variables](#required-environment-variables)
  * [Testing independently of DevPod](#testing-independently-of-devpod)
  * [Testing in the DevPod ecosystem](#testing-in-the-devpod-ecosystem)
* [Contributing](#contributing)
  * [Open in a container](#open-in-a-container)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

> Use [this referral code](https://hetzner.cloud/?ref=UWVUhEZNkm6p) to get €20 in
> credits (at time of writing).

[DevPod](https://devpod.sh/) on Oracle Cloud Infrastructure. This project is built on top of the [Hetzner provider](https://github.com/mrsimonemms/devpod-provider-hetzner).

## Usage

To use this provider in your DevPod setup, you will need to do the following steps:

1. See the [DevPod documentation](https://devpod.sh/docs/managing-providers/add-provider)
   for how to add a provider
2. Use the reference `haroondilshad/devpod-provider-oracle-cloud` to download the latest
   release from GitHub
3. Configure your Oracle Cloud Infrastructure credentials using the OCI CLI or by creating a config file at `~/.oci/config`

## Required environment variables

| Variable | Description | Example |
| --- | --- | --- |
| `COMPARTMENT_ID` | Oracle Cloud Infrastructure compartment ID | `ocid1.compartment.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| `REGION` | Oracle Cloud Infrastructure region | `us-ashburn-1` |
| `AVAILABILITY_DOMAIN` | Oracle Cloud Infrastructure availability domain | `AD-1` |
| `DISK_IMAGE` | Oracle Cloud Infrastructure image name | `Oracle-Linux-8.6-2022.05.31-0` |
| `DISK_SIZE` | Disk size in GB | `50` |
| `MACHINE_TYPE` | Oracle Cloud Infrastructure shape | `VM.Standard.E4.Flex` |
| `MACHINE_FOLDER` | Local home folder | `~/.ssh` |
| `MACHINE_ID` | Unique identifier for the machine | `some-machine-id` |
| `OCI_CONFIG_FILE` | Path to OCI config file | `~/.oci/config` |
| `OCI_PROFILE` | Profile to use in OCI config file | `DEFAULT` |

## Development

### Testing independently of DevPod

To test the provider workflow, you can run the CLI commands directly.

| Command | Description | Example |
| --- | --- | --- |
| `command` | Run a command on the instance | `COMMAND="ls -la" go run . command` |
| `create` | Create an instance | `go run . create` |
| `delete` | Delete an instance | `go run . delete` |
| `init` | Initialise an instance | `go run . init` |
| `start` | Start an instance | `go run . start` |
| `status` | Retrieve the status of an instance | `go run . status` |
| `stop` | Stop an instance | `go run . stop` |

### Testing in the DevPod ecosystem

To test the provider within the DevPod ecosystem:

1. Install the latest version of the Oracle Cloud provider
2. Backup the original binary:

   ```shell
   mv ~/.devpod/contexts/default/providers/oracle-cloud/binaries/oracle_provider/devpod-provider-oracle-cloud-linux-amd64 ~/.devpod/contexts/default/providers/oracle-cloud/binaries/oracle_provider/devpod-provider-oracle-cloud-linux-amd64-orig
   ```

3. Build the binary:

   ```shell
   go build .
   ```

4. Move the new binary to the DevPod base:

   ```shell
   mv ./devpod-provider-oracle-cloud ~/.devpod/contexts/default/providers/oracle-cloud/binaries/oracle_provider/devpod-provider-oracle-cloud-linux-amd64
   ```

## Contributing

* Get an [Oracle Cloud Infrastructure](https://www.oracle.com/cloud/) account
* Configure your OCI credentials using the OCI CLI or by creating a config file at `~/.oci/config`

### Open in a container

* [Open in a container](https://code.visualstudio.com/docs/devcontainers/containers)
