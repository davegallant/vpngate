package vpn

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetListReal tests getting the real list of vpn servers
func TestGetListReal(t *testing.T) {
	_, err := GetList("", "")

	assert.NoError(t, err)
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
