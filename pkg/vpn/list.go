package vpn

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/jszwec/csvutil"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"

	"github.com/davegallant/vpngate/pkg/util"
	"github.com/juju/errors"
)

const (
	vpnList = "https://www.vpngate.net/api/iphone/"
)

// Server holds in formation about a vpn relay server
type Server struct {
	HostName          string `csv:"#HostName"`
	CountryLong       string `csv:"CountryLong"`
	CountryShort      string `csv:"CountryShort"`
	Score             int    `csv:"Score"`
	IPAddr            string `csv:"IP"`
	OpenVpnConfigData string `csv:"OpenVPN_ConfigData_Base64"`
	Ping              string `csv:"Ping"`
}

func streamToBytes(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(stream)
	if err != nil {
		log.Error().Msg("Unable to stream bytes")
	}
	return buf.Bytes()
}

// parse csv
func parseVpnList(r io.Reader) (*[]Server, error) {
	var servers []Server

	serverList := streamToBytes(r)

	// Trim known invalid rows
	serverList = bytes.TrimPrefix(serverList, []byte("*vpn_servers\r\n"))
	serverList = bytes.TrimSuffix(serverList, []byte("*\r\n"))
	serverList = bytes.ReplaceAll(serverList, []byte(`"`), []byte{})

	if err := csvutil.Unmarshal(serverList, &servers); err != nil {
		return nil, errors.Annotatef(err, "Unable to parse CSV")
	}

	return &servers, nil
}

// GetList returns a list of vpn servers
func GetList(httpProxy string, socks5Proxy string) (*[]Server, error) {
	cacheExpired := vpnListCacheIsExpired()

	var servers *[]Server
	var client *http.Client

	if !cacheExpired {
		servers, err := getVpnListCache()

		if err != nil {
			log.Info().Msg("Unable to retrieve vpn list from cache")
		} else {
			return servers, nil
		}

	} else {
		log.Info().Msg("The vpn server list cache has expired")
	}

	log.Info().Msg("Fetching the latest server list")

	if httpProxy != "" {
		proxyURL, err := url.Parse(httpProxy)
		if err != nil {
			log.Error().Msgf("Error parsing proxy: %s", err)
			os.Exit(1)
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}

		client = &http.Client{
			Transport: transport,
		}

	} else if socks5Proxy != "" {
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			log.Error().Msgf("Error creating SOCKS5 dialer: %v", err)
			os.Exit(1)
		}

		httpTransport := &http.Transport{
			Dial: dialer.Dial,
		}

		client = &http.Client{
			Transport: httpTransport,
		}
	} else {
		client = &http.Client{}
	}

	var r *http.Response

	err := util.Retry(5, 1, func() error {
		var err error
		r, err = client.Get(vpnList)
		if err != nil {
			return err
		}
		defer func() {
			_ = r.Body.Close()
		}()

		if r.StatusCode != 200 {
			return errors.Annotatef(err, "Unexpected status code when retrieving vpn list: %d", r.StatusCode)
		}

		servers, err = parseVpnList(r.Body)

		if err != nil {
			return err
		}

		err = writeVpnListToCache(*servers)

		if err != nil {
			log.Warn().Msgf("Unable to write servers to cache: %s", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return servers, nil
}
