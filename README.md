# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a relay server, speed tests kick off frequently to determine latency, upload speed and download speed.

Once connected, you can check out your IP address: https://nordvpn.com/what-is-my-ip/

## requirements

- openvpn
- macOS or Linux (Windows may work but untested)

## usage

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

## TODO

- Cache list of vpn servers in a json file (merge cache so old configs can be still used?)
- Allow for servers to be cycled periodically (--cycle)
- Allow for a specific country to be selected (--country)
