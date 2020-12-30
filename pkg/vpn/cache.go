package vpn

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

const serverCachefile = "servers.json"

func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error().Msgf("Failed to get user's home directory: %s ", err)
		return ""
	}
	cacheDir := path.Join(homeDir, ".vpngate", "cache")
	return cacheDir
}

func createCacheDir() error {
	cacheDir := getCacheDir()
	var AppFs = afero.NewOsFs()
	return AppFs.MkdirAll(cacheDir, 0700)
}

func getVpnListCache() (*[]Server, error) {
	cacheFile := path.Join(getCacheDir(), serverCachefile)
	serversFile, err := os.Open(cacheFile)

	if err != nil {
		return nil, err
	}

	byteValue, err := ioutil.ReadAll(serversFile)

	if err != nil {
		return nil, err
	}

	var servers []Server

	json.Unmarshal(byteValue, &servers)

	return &servers, nil
}

func writeVpnListToCache(servers []Server) error {

	err := createCacheDir()

	if err != nil {
		return err
	}

	f, err := json.MarshalIndent(servers, "", " ")

	if err != nil {
		return err
	}

	cacheFile := path.Join(getCacheDir(), serverCachefile)

	err = ioutil.WriteFile(cacheFile, f, 0644)

	return err

}

func vpnListCacheIsExpired() bool {
	file, err := os.Stat(path.Join(getCacheDir(), serverCachefile))

	if err != nil {
		return true
	}

	lastModified := file.ModTime()

	return (time.Now().Sub(lastModified)) > time.Duration(24*time.Hour)
}
