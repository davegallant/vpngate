# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](https://user-images.githubusercontent.com/4519234/104145615-b6f9f880-5395-11eb-812c-c6597a7aed0f.gif)

This client fetches the list of available relay servers provided by vpngate.net, and allows you to filter and connect to a server of your liking.

You can check out your current IP address and region at https://nordvpn.com/what-is-my-ip/, or simply run the following command in a terminal:

```shell
curl ipinfo.io
```

## Requirements

- [openvpn](https://github.com/OpenVPN/openvpn)
- macOS or Linux

## Install

The simplest method of installation is using homebrew. You can also build from source.

### from homebrew

vpngate can be installed with [homebrew](https://brew.sh/) (ensure that xcode is installed before installing homebrew by running `xcode-select --install`).

```shell

brew install openvpn davegallant/public/vpngate
```

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

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks
