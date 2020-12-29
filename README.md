# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

![image](https://user-images.githubusercontent.com/4519234/103308173-ce250780-49df-11eb-9032-ef832e5b9463.png)

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a relay server, speed tests kick off to determine latency, upload speed and download speed.

![image](https://user-images.githubusercontent.com/4519234/103308641-e47f9300-49e0-11eb-8ff2-77c6e3e8cc7b.png)

Once connected, you can check out your IP address: https://nordvpn.com/what-is-my-ip/

## Requirements

- [openvpn](https://github.com/OpenVPN/openvpn)
- macOS or Linux (might work on Windows)

## Usage

Because openvpn creates a network interface, run the connect command with `sudo` or an account with escalated privileges.

```sh
vpngate is a client for vpngate.net

Usage:
  vpngate [flags]
  vpngate [command]

Available Commands:
  connect     Connect to a vpn server
  help        Help about any command
  list        List all available vpn servers

Flags:
  -h, --help   help for vpngate

Use "vpngate [command] --help" for more information about a command.
```

## Notes

- I do not maintain any of the VPN servers on vpngate.net. Connect to these VPN servers at your own discretion
- Many of the listed servers claim to have a logging policy of 2 weeks


## Todo

- Cache list of vpn servers in a json file (merge cache so old configs can be still used?)
- Allow for servers to be cycled periodically (--cycle)
- Allow for a specific country to be selected (--country)

