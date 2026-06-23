# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](https://github.com/user-attachments/assets/dafa2702-8d68-4b5f-badb-2c53ddd68991)

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

On Windows, install OpenVPN from the [official website](https://openvpn.net/community-downloads/).

As there is no installer at the moment, you will need to download and extract the Windows release from the relevant Github release.

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

> If on macOS, you may need to add openvpn to your PATH (if you installed it with brew): `export PATH=$(brew --prefix openvpn)/sbin:$PATH`

### Examples

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

<!-- cobra:docs:start -->
### vpngate

vpngate is a client for vpngate.net

```
vpngate [flags]
```

#### Options

```
  -h, --help   help for vpngate
```

### vpngate cache

Manage cached vpn server data

#### Options

```
  -h, --help   help for cache
```

### vpngate cache clear

Clear cached vpn server data

```
vpngate cache clear [flags]
```

#### Options

```
  -h, --help   help for clear
```

### vpngate cache path

Print the cache directory path

```
vpngate cache path [flags]
```

#### Options

```
  -h, --help   help for path
```

### vpngate completion

Generate the autocompletion script for the specified shell

#### Synopsis

Generate the autocompletion script for vpngate for the specified shell.
See each sub-command's help for details on how to use the generated script.


#### Options

```
  -h, --help   help for completion
```

### vpngate completion bash

Generate the autocompletion script for bash

#### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(vpngate completion bash)

To load completions for every new session, execute once:

##### Linux:

	vpngate completion bash > /etc/bash_completion.d/vpngate

##### macOS:

	vpngate completion bash > $(brew --prefix)/etc/bash_completion.d/vpngate

You will need to start a new shell for this setup to take effect.


```
vpngate completion bash
```

#### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### vpngate completion fish

Generate the autocompletion script for fish

#### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	vpngate completion fish | source

To load completions for every new session, execute once:

	vpngate completion fish > ~/.config/fish/completions/vpngate.fish

You will need to start a new shell for this setup to take effect.


```
vpngate completion fish [flags]
```

#### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### vpngate completion powershell

Generate the autocompletion script for powershell

#### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	vpngate completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
vpngate completion powershell [flags]
```

#### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### vpngate completion zsh

Generate the autocompletion script for zsh

#### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(vpngate completion zsh)

To load completions for every new session, execute once:

##### Linux:

	vpngate completion zsh > "${fpath[1]}/_vpngate"

##### macOS:

	vpngate completion zsh > $(brew --prefix)/share/zsh/site-functions/_vpngate

You will need to start a new shell for this setup to take effect.


```
vpngate completion zsh [flags]
```

#### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### vpngate connect

Connect to a vpn server (survey selection appears if hostname is not provided)

#### Synopsis

Connect to a vpn from a list of relay servers. Because openvpn creates a network interface, run the connect command with 'sudo' or a user with escalated privileges.

```
vpngate connect [flags]
```

#### Options

```
      --country string   filter by country name or country code (i.e. Japan or jp)
  -h, --help             help for connect
      --max-ping int     filter out servers with ping higher than this value
      --min-score int    filter out servers with score lower than this value
      --no-cache         do not read from or write to the vpn server list cache
  -p, --proxy string     provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)
  -r, --random           connect to a random server
  -t, --reconnect        continually attempt to connect to the server
      --refresh          refresh the vpn server list cache before connecting
  -s, --socks5 string    provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)
```

### vpngate list

List all available vpn servers

```
vpngate list [flags]
```

#### Options

```
      --country string   filter by country name or country code (i.e. Japan or jp)
  -h, --help             help for list
      --max-ping int     filter out servers with ping higher than this value
      --min-score int    filter out servers with score lower than this value
      --no-cache         do not read from or write to the vpn server list cache
  -o, --output string    output format: table, json, csv (default "table")
  -p, --proxy string     provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)
      --refresh          refresh the vpn server list cache before listing
  -s, --socks5 string    provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)
      --sort string      sort by one of none, score, ping, country, hostname (default "none")
```

<!-- cobra:docs:end -->

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks
