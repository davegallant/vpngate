# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![vpngate](https://user-images.githubusercontent.com/4519234/103447878-41887c80-4c5f-11eb-8681-add9717dbb88.gif)

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a server, speed tests kick off to determine latency, upload speed and download speed.

You can check out your current IP address and region at https://nordvpn.com/what-is-my-ip/, or simply run the following command in a terminal:

```sh
$ curl ipinfo.io
```

## Requirements

- [openvpn](https://github.com/OpenVPN/openvpn)
- macOS or Linux

## Install

Ensure that [go](https://golang.org/doc/install) is installed.

```sh
$ go get github.com/davegallant/vpngate
```

### MacOS

OpenVPN can be installed with [homebrew](https://brew.sh/).

```sh
$ brew install openvpn
```

## Usage

### List available servers

```sh
$ vpngate list
```

### Connect to a server

On macOS, `openvpn` may not be within your PATH. To fix this, run:

```sh
$ export PATH=$(brew --prefix openvpn)/sbin:$PATH
```

The above command can also be added to a bash/zsh profile for future use.

Because openvpn creates a network interface, run the connect command with `sudo` or a user with escalated privileges.

```sh
$ sudo vpngate connect
```

#### Random

If the country doesn't matter, a random server can be selected:

```sh
$ sudo vpngate connect --random
```

## Notes

- I do not maintain any of the servers on vpngate.net (connect to these servers at your own discretion)
- Many of the listed servers claim to have a logging policy of 2 weeks


## Todo

- Allow for servers to be cycled periodically (--cycle)
