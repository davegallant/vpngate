package vpn

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetListWithOptions fetches and parses a local fixture served over
// HTTP, exercising the same code path as a real fetch without depending on
// vpngate.net being reachable.
func TestGetListWithOptions(t *testing.T) {
	dat, err := os.ReadFile("../../test_data/vpn_list.csv")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(dat)
	}))
	defer server.Close()

	originalVpnList := vpnList
	vpnList = server.URL
	defer func() { vpnList = originalVpnList }()

	servers, err := GetListWithOptions("", "", ListOptions{NoCache: true})
	assert.NoError(t, err)
	assert.Equal(t, 98, len(*servers))
}

// TestParseVpnList parses a local copy of vpn list csv
func TestParseVpnList(t *testing.T) {
	dat, err := os.Open("../../test_data/vpn_list.csv")
	assert.NoError(t, err)

	servers, err := parseVpnList(dat)
	assert.NoError(t, err)

	assert.Equal(t, len(*servers), 98)

	assert.Equal(t, (*servers)[0].CountryLong, "Japan")
	assert.Equal(t, (*servers)[0].CountryShort, "jp")
	assert.Equal(t, (*servers)[0].HostName, "public-vpn-227")
	assert.Equal(t, (*servers)[0].Ping, "13")
	assert.Equal(t, (*servers)[0].Score, 2086924)
}
