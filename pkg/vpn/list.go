package vpn

import (
	"net/http"

	"bytes"
	"io"

	"github.com/jszwec/csvutil"
	"github.com/rs/zerolog/log"

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

	if err := csvutil.Unmarshal(serverList, &servers); err != nil {
		return nil, errors.Annotatef(err, "Unable to parse CSV")
	}

	return &servers, nil

}

// GetList returns a list of vpn servers
func GetList() (*[]Server, error) {

	cacheExpired := vpnListCacheIsExpired()

	var servers *[]Server

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

	r, err := http.Get(vpnList)

	if err != nil {
		return nil, errors.Annotate(err, "Unable to retrieve vpn list")
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, errors.Annotatef(err, "Unexpected status code when retrieving vpn list: %d", r.StatusCode)
	}

	servers, err = parseVpnList(r.Body)

	if err != nil {
		return nil, errors.Annotate(err, "unable to parse vpn list")
	}

	err = writeVpnListToCache(*servers)

	if err != nil {
		log.Warn().Msgf("Unable to write servers to cache: %s", err)
	}

	return servers, nil
}
