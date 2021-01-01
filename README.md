# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![image](https://user-images.githubusercontent.com/4519234/103308173-ce250780-49df-11eb-9032-ef832e5b9463.png)

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a relay server, speed tests kick off to determine latency, upload speed and download speed.

![image](https://user-images.githubusercontent.com/4519234/103308641-e47f9300-49e0-11eb-8ff2-77c6e3e8cc7b.png)

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

### List Available VPNs

```sh
$ vpngate list
```

### Connect to a VPN

Because openvpn creates a network interface, run the connect command with `sudo` or a user with escalated privileges.

On macOS, export path by using `export PATH=$(brew --prefix openvpn)/sbin:$PATH` (this can be run each time in a terminal or put in a bash/zsh profile)

```sh
$ sudo vpngate connect
```

#### Random

If the country doesn't matter, a random VPN can be selected:

```sh
$ sudo vpngate connect --random
```

## Notes

- I do not maintain any of the VPN servers on vpngate.net. Connect to these VPN servers at your own discretion
- Many of the listed servers claim to have a logging policy of 2 weeks


## Todo

- Allow for servers to be cycled periodically (--cycle)
