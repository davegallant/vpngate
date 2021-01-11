# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](https://user-images.githubusercontent.com/4519234/104145413-df352780-5394-11eb-92c8-ebd37192bca0.gif)

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a server, speed tests kick off to determine latency, upload speed and download speed.

You can check out your current IP address and region at https://nordvpn.com/what-is-my-ip/, or simply run the following command in a terminal:

```console
$ curl ipinfo.io
```

## Requirements

- [openvpn](https://github.com/OpenVPN/openvpn)
- macOS or Linux

## Install from source

Ensure that [go](https://golang.org/doc/install) is installed.

```console
$ CGO_ENABLED=0 go get github.com/davegallant/vpngate
```

Ensure that the go bin path is discoverable:

```console
$ echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.profile
$ source ~/.profile
```

### MacOS

OpenVPN can be installed with [homebrew](https://brew.sh/).

```console
$ brew install openvpn
```

## Usage

### List available servers

```console
$ vpngate list
```

### Connect to a server

On macOS, `openvpn` may not be within your PATH. To fix this, run:

```console
$ export PATH=$(brew --prefix openvpn)/sbin:$PATH
```

The above command can also be added to a bash/zsh profile for future use.

Because openvpn creates a network interface, run the connect command with `sudo` or a user with escalated privileges.

```console
$ sudo vpngate connect
```

#### Random

If the country doesn't matter, a random server can be selected:

```console
$ sudo vpngate connect --random
```

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks


## Todo

- Allow for servers to be cycled periodically (--cycle)
