package vpn

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jszwec/csvutil"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"

	"github.com/davegallant/vpngate/pkg/util"
	"github.com/juju/errors"
)

const (
	vpnList            = "https://www.vpngate.net/api/iphone/"
	httpClientTimeout  = 30 * time.Second
	dialTimeout        = 10 * time.Second
)

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
		return nil, errors.Annotate(err, "Unable to read stream")
	}

	// Trim known invalid rows
	serverList = bytes.TrimPrefix(serverList, []byte("*vpn_servers\r\n"))
	serverList = bytes.TrimSuffix(serverList, []byte("*\r\n"))
	serverList = bytes.ReplaceAll(serverList, []byte(`"`), []byte{})

	if err := csvutil.Unmarshal(serverList, &servers); err != nil {
		return nil, errors.Annotatef(err, "Unable to parse CSV")
	}

	return &servers, nil
}

// createHTTPClient creates an HTTP client with optional proxy configuration
func createHTTPClient(httpProxy string, socks5Proxy string) (*http.Client, error) {
	if httpProxy != "" {
		proxyURL, err := url.Parse(httpProxy)
		if err != nil {
			return nil, errors.Annotatef(err, "Error parsing HTTP proxy: %s", httpProxy)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		return &http.Client{
			Transport: transport,
			Timeout:   httpClientTimeout,
		}, nil
	}

	if socks5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, errors.Annotatef(err, "Error creating SOCKS5 dialer: %v", err)
		}

		// Create a DialContext function from the SOCKS5 dialer
		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Check if context is already done
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// Use the dialer with a timeout
			conn, err := dialer.Dial(network, addr)
			if err != nil {
				return nil, err
			}

			// Respect context cancellation after connection
			go func() {
				<-ctx.Done()
				_ = conn.Close()
			}()

			return conn, nil
		}

		httpTransport := &http.Transport{
			DialContext: dialContext,
		}
		return &http.Client{
			Transport: httpTransport,
			Timeout:   httpClientTimeout,
		}, nil
	}

	return &http.Client{
		Timeout: httpClientTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: dialTimeout,
			}).DialContext,
		},
	}, nil
}

// GetList returns a list of vpn servers
func GetList(httpProxy string, socks5Proxy string) (*[]Server, error) {
	cacheExpired := vpnListCacheIsExpired()

	// Try to use cached list if not expired
	if !cacheExpired {
		servers, err := getVpnListCache()
		if err == nil {
			return servers, nil
		}
		log.Info().Msg("Unable to retrieve vpn list from cache")
	} else {
		log.Info().Msg("The vpn server list cache has expired")
	}

	log.Info().Msg("Fetching the latest server list")

	client, err := createHTTPClient(httpProxy, socks5Proxy)
	if err != nil {
		return nil, err
	}

	var servers *[]Server

	err = util.Retry(5, 1, func() error {
		resp, err := client.Get(vpnList)
		if err != nil {
			return err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return errors.Annotatef(err, "Unexpected status code when retrieving vpn list: %d", resp.StatusCode)
		}

		parsedServers, err := parseVpnList(resp.Body)
		if err != nil {
			return err
		}

		servers = parsedServers

		// Cache the servers for future use
		cacheErr := writeVpnListToCache(*servers)
		if cacheErr != nil {
			log.Warn().Msgf("Unable to write servers to cache: %s", cacheErr)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return servers, nil
}