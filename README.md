# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](https://user-images.githubusercontent.com/4519234/104145615-b6f9f880-5395-11eb-812c-c6597a7aed0f.gif)

This client fetches the list of available relay servers provided by vpngate.net, and allows you to filter and connect to a server of your liking.

You can check out your current IP address and region at <https://ipinfo.io>, or run the following:

```shell
curl ipinfo.io
```

## Requirements

- OpenVPN <= 2.5 (configurations on vpngate.net do not seem work on OpenVPN 2.6+)
- macOS, Linux, or Windows

## Install

You can install vpngate in a few different ways, and it will differ slightly depending on your OS.

### Homebrew

vpngate can be installed with [homebrew](https://brew.sh/) (ensure that xcode is installed before installing homebrew by running `xcode-select --install`).

```shell
brew install openvpn davegallant/public/vpngate
```

## Windows

On Windows, install OpenVPN 2.5.x from the [official website](https://openvpn.net/community-downloads/).

You also need to download an extract the Windows release from the relevant Github release.

Open Command Prompt(cmd.exe) *as Administrator*, and run vpngate.exe

### from source

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

### List available servers

```shell
vpngate list
```

### Connect to a server

Because openvpn creates a network interface, run the connect command with `sudo` or a user with escalated privileges.

```shell
sudo vpngate connect
```

#### Reconnect

To continually attempt to reconnect (this can be combined with `--random`):

```shell
sudo vpngate connect --reconnect
```

#### Random

If the country doesn't matter, a random server can be selected:

```shell
sudo vpngate connect --random
```

#### Proxy

In some cases, anonymity is necessary to populate the list of available VPN servers.

A proxy is a way to bypass restrictions and in some cases, internet censorship.

##### HTTP/HTTPS

Use the specified HTTP/HTTPS proxy to fetch the server list.

```shell
sudo vpngate connect --proxy "http://localhost:8080"
```

##### SOCKS5

Use the specified SOCKS5 proxy to fetch the server list.

```shell
sudo vpngate connect --socks5 "127.0.0.1:1080"
```

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks
