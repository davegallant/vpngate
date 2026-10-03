package vpn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jszwec/csvutil"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"

	"github.com/davegallant/vpngate/pkg/util"
)

const (
	httpClientTimeout = 30 * time.Second
	dialTimeout       = 10 * time.Second
	fetchRetryDelay   = time.Second
	fetchRetryCount   = 5
)

// vpnList is the URL of the vpngate server list API. It is a var so tests
// can point it at a local httptest.Server instead of the real endpoint.
var vpnList = "https://www.vpngate.net/api/iphone/"

// Server holds information about a vpn relay server
type Server struct {
	HostName          string `csv:"#HostName"`
	CountryLong       string `csv:"CountryLong"`
	CountryShort      string `csv:"CountryShort"`
	Score             int    `csv:"Score"`
	IPAddr            string `csv:"IP"`
	OpenVpnConfigData string `csv:"OpenVPN_ConfigData_Base64"`
	Ping              string `csv:"Ping"`
}

// parseVpnList parses the VPN server list from CSV format
func parseVpnList(r io.Reader) (*[]Server, error) {
	var servers []Server

	serverList, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read stream: %w", err)
	}

	// Trim known invalid rows
	serverList = bytes.TrimPrefix(serverList, []byte("*vpn_servers\r\n"))
	serverList = bytes.TrimSuffix(serverList, []byte("*\r\n"))
	serverList = bytes.ReplaceAll(serverList, []byte(`\"`), []byte{})

	if err := csvutil.Unmarshal(serverList, &servers); err != nil {
		return nil, fmt.Errorf("unable to parse CSV: %w", err)
	}

	for i := range servers {
		if alias, ok := countryAliases[servers[i].CountryLong]; ok {
			servers[i].CountryLong = alias
		}
	}

	return &servers, nil
}

// countryAliases maps vpngate.net's CountryLong values to more familiar
// country names.
var countryAliases = map[string]string{
	"Korea Republic of":  "South Korea",
	"Russian Federation": "Russia",
}

// baseTransport returns a copy of http.DefaultTransport so proxy
// configurations keep sane timeouts (TLS handshake, response header,
// idle connections) instead of a zero-value transport.
func baseTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}

// createHTTPClient creates an HTTP client with optional proxy configuration
func createHTTPClient(httpProxy string, socks5Proxy string) (*http.Client, error) {
	if httpProxy != "" {
		proxyURL, err := url.Parse(httpProxy)
		if err != nil {
			return nil, fmt.Errorf("error parsing HTTP proxy %q: %w", httpProxy, err)
		}
		transport := baseTransport()
		transport.Proxy = http.ProxyURL(proxyURL)
		return &http.Client{
			Transport: transport,
			Timeout:   httpClientTimeout,
		}, nil
	}

	if socks5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("error creating SOCKS5 dialer: %w", err)
		}

		// proxy.SOCKS5's dialer has no context-aware Dial, so run it in
		// a goroutine and select on a timeout-bound context. The
		// buffered channel guarantees the goroutine never blocks on
		// send, so no goroutine outlives the dial either way.
		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
			defer cancel()

			type dialResult struct {
				conn net.Conn
				err  error
			}
			resultCh := make(chan dialResult, 1)
			go func() {
				conn, err := dialer.Dial(network, addr)
				resultCh <- dialResult{conn: conn, err: err}
			}()

			select {
			case <-dialCtx.Done():
				return nil, dialCtx.Err()
			case res := <-resultCh:
				return res.conn, res.err
			}
		}

		transport := baseTransport()
		transport.DialContext = dialContext
		return &http.Client{
			Transport: transport,
			Timeout:   httpClientTimeout,
		}, nil
	}

	transport := baseTransport()
	transport.DialContext = (&net.Dialer{
		Timeout: dialTimeout,
	}).DialContext
	return &http.Client{
		Timeout:   httpClientTimeout,
		Transport: transport,
	}, nil
}

// ListOptions controls how the vpn server list is fetched.
type ListOptions struct {
	Refresh bool
	NoCache bool
}

// GetList returns a list of vpn servers.
func GetList(httpProxy string, socks5Proxy string) (*[]Server, error) {
	return GetListWithOptions(httpProxy, socks5Proxy, ListOptions{})
}

// GetListWithOptions returns a list of vpn servers with cache controls.
func GetListWithOptions(httpProxy string, socks5Proxy string, opts ListOptions) (*[]Server, error) {
	cacheExpired := vpnListCacheIsExpired()

	// Try to use cached list if not expired, unless explicitly bypassed.
	if !opts.Refresh && !opts.NoCache && !cacheExpired {
		servers, err := getVpnListCache()
		if err == nil {
			return servers, nil
		}
		log.Info().Msg("Unable to retrieve vpn list from cache")
	} else if opts.Refresh {
		log.Info().Msg("Refreshing the vpn server list")
	} else if opts.NoCache {
		log.Info().Msg("Bypassing the vpn server list cache")
	} else {
		log.Info().Msg("The vpn server list cache has expired")
	}

	log.Info().Msg("Fetching the latest server list")

	client, err := createHTTPClient(httpProxy, socks5Proxy)
	if err != nil {
		return nil, err
	}

	var servers *[]Server

	err = util.Retry(fetchRetryCount, fetchRetryDelay, func() error {
		resp, err := client.Get(vpnList)
		if err != nil {
			return err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code when retrieving vpn list: %d", resp.StatusCode)
		}

		parsedServers, err := parseVpnList(resp.Body)
		if err != nil {
			return err
		}

		servers = parsedServers

		// Cache the servers for future use, unless caching is disabled.
		if !opts.NoCache {
			cacheErr := writeVpnListToCache(*servers)
			if cacheErr != nil {
				log.Warn().Msgf("Unable to write servers to cache: %s", cacheErr)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return servers, nil
}
