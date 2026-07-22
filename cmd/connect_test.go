package cmd

import (
	"testing"

	"github.com/davegallant/vpngate/pkg/daemon"
	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/stretchr/testify/assert"
)

func TestExtractHostname(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain hostname", "public-vpn-227", "public-vpn-227"},
		{"formatted selection", "public-vpn-227  Japan  1.2.3.4  ping 13", "public-vpn-227"},
		{"legacy pipe format", "public-vpn-227 | Japan (1.2.3.4)", "public-vpn-227"},
		{"trims whitespace", "  public-vpn-227  ", "public-vpn-227"},
		{"empty string", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractHostname(tc.input))
		})
	}
}

func TestBuildServerSelection(t *testing.T) {
	servers := []vpn.Server{
		{HostName: "public-vpn-1", CountryLong: "Japan", IPAddr: "1.2.3.4", Ping: "10"},
		{HostName: "public-vpn-2222", CountryLong: "United States", IPAddr: "5.6.7.8", Ping: "200"},
	}

	labels, serverMap := buildServerSelection(servers)
	assert.Len(t, labels, 2)
	// each server is indexed by both its formatted label and its hostname
	assert.Len(t, serverMap, 4)

	for _, s := range servers {
		got, ok := serverMap[s.HostName]
		assert.True(t, ok)
		assert.Equal(t, s, got)
	}
}

func TestForwardableConnectArgs(t *testing.T) {
	flagReconnect = true
	flagRandom = false
	flagProxy = "http://127.0.0.1:8080"
	flagSocks5Proxy = ""
	flagCountry = "Japan"
	flagMaxPing = 100
	flagMinScore = 0
	flagRefresh = true
	flagNoCache = false
	t.Cleanup(func() {
		flagReconnect = false
		flagProxy = ""
		flagCountry = ""
		flagMaxPing = 0
		flagRefresh = false
	})

	args := forwardableConnectArgs()
	assert.Equal(t, []string{
		"--reconnect",
		"--proxy", "http://127.0.0.1:8080",
		"--country", "Japan",
		"--max-ping", "100",
		"--refresh",
	}, args)
}

func TestTailLogMissingFile(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())
	assert.Equal(t, "", tailLog())
}
