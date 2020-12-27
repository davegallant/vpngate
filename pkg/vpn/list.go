package vpn

import (
	"net/http"

	"bytes"
	"fmt"
	"io"

	"github.com/jszwec/csvutil"

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
	IpAddr            string `csv:"IP"`
	OpenVpnConfigData string `csv:"OpenVPN_ConfigData_Base64"`
	Ping              int    `csv:"Ping"`
}

func streamToBytes(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

// parse csv
func parseVpnList(r io.Reader) ([]Server, error) {

	var servers []Server

	serverList := streamToBytes(r)

	// Trim known invalid rows
	serverList = bytes.TrimPrefix(serverList, []byte("*vpn_servers\r\n"))
	serverList = bytes.TrimSuffix(serverList, []byte("*\r\n"))

	if err := csvutil.Unmarshal(serverList, &servers); err != nil {
		fmt.Println(err)
		return nil, errors.Annotate(err, "Unable to parse CSV")
	}

	return servers, nil

}

// GetList returns a list of vpn servers
func GetList() (*[]Server, error) {

	r, err := http.Get(vpnList)

	if err != nil {
		return nil, errors.Annotate(err, "Unable to retrieve vpn list")
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		return nil, errors.Annotatef(err, "Unexpected status code when retrieving vpn list: %s", r.StatusCode)
	}

	vpnServers, err := parseVpnList(r.Body)

	if err != nil {
		return nil, errors.Annotate(err, "unable to parse vpn list")
	}

	return &vpnServers, nil
}
