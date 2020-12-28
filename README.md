# vpngate

This is a client for [vpngate.net](https://www.vpngate.net/).

This client fetches the list of available relay servers provided by vpngate.net. Once connected to a relay server, speed tests kick off frequently to determine latency, upload speed and download speed.

Once connected, you can check out your IP address: https://nordvpn.com/what-is-my-ip/

## requirements

- openvpn
- macOS or Linux (Windows may work but untested)

## TODO

- Allow for random server to be selected (--random)
- Allow for servers to be cycled through when a speedtest fails (--cycle)
- Allow for a specific country to be selected (--country)
