# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](vpngate.gif)

This client fetches the list of available relay servers provided by vpngate.net, and allows you to filter and connect to a server of your liking.

You can check out your current IP address and region at <https://ipinfo.io/json>, or run the following:

```shell
curl ipinfo.io
```

## Requirements

- OpenVPN
- macOS, Linux, or Windows

## Install

You can install vpngate in a few different ways, and it will differ slightly depending on your OS.

### Homebrew (macOS and linux)

vpngate can be installed with [homebrew](https://brew.sh/) (if on macOS, ensure that xcode is installed before installing homebrew by running `xcode-select --install`).

```shell
brew install openvpn davegallant/public/vpngate
```

### Windows

vpngate can be installed with [winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/), which also installs OpenVPN as a dependency:

```shell
winget install davegallant.vpngate
```

> The winget package is submitted automatically on each release and goes through community review before it becomes searchable, so it may lag a release or two behind. If it's not available yet, use manual install below.

Alternatively, install OpenVPN from the [official website](https://openvpn.net/community-downloads/), then manually download and extract the Windows release from the relevant Github release.

Once the release is extracted, open Command Prompt *as Administrator*, and run vpngate.exe from the location where it was extracted.

<img width="278" alt="image" src="https://github.com/user-attachments/assets/fb47270d-82bb-4790-833a-377b874c8104">

<img width="565" alt="image" src="https://github.com/user-attachments/assets/42287904-6c00-48d1-bff3-9757cf250519">

### Build from source

Ensure that [go](https://golang.org/doc/install) is installed.

```shell
CGO_ENABLED=0 go get github.com/davegallant/vpngate
```

Ensure that the go bin path is discoverable:

```shell
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.profile
source ~/.profile
```

## Usage

For full usage instructions, see [the cli docs](docs/cli/vpngate.md).

> If on macOS, you may need to add openvpn to your PATH (if you installed it with brew): `export PATH=$(brew --prefix openvpn)/sbin:$PATH`

### Examples

Run in the background, then check on it or disconnect later. `status` and
`disconnect` need `sudo` too, since the daemon's state is only readable by
the user that ran `connect -d` (root):

```shell
sudo vpngate connect -d --country Japan
sudo vpngate status
sudo vpngate disconnect
```

List Japanese servers sorted by lowest ping:

```shell
vpngate list --country Japan --sort ping
```

List high-scoring US servers as JSON:

```shell
vpngate list --country us --min-score 1000000 --output json
```

Connect to a random server with quality filters:

```shell
sudo vpngate connect --random --country Japan --max-ping 100 --min-score 500000
```

Refresh the cached server list before listing:

```shell
vpngate list --refresh
```

Bypass the cache entirely:

```shell
vpngate list --no-cache
```

Inspect or clear the cache:

```shell
vpngate cache path
vpngate cache clear
```

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks
