package vpn

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const serverCachefile = "servers.json"

func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".vpngate", "cache")
	return cacheDir, nil
}

func createCacheDir() error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(cacheDir, 0o700)
}

func getVpnListCache() (*[]Server, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}
	cacheFile := filepath.Join(cacheDir, serverCachefile)
	serversFile, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = serversFile.Close()
	}()

	byteValue, err := io.ReadAll(serversFile)
	if err != nil {
		return nil, err
	}

	var servers []Server

	err = json.Unmarshal(byteValue, &servers)
	if err != nil {
		return nil, err
	}

	return &servers, nil
}

func writeVpnListToCache(servers []Server) error {
	if err := createCacheDir(); err != nil {
		return err
	}

	f, err := json.MarshalIndent(servers, "", " ")
	if err != nil {
		return err
	}

	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	cacheFile := filepath.Join(cacheDir, serverCachefile)

	return os.WriteFile(cacheFile, f, 0o644)
}

func vpnListCacheIsExpired() bool {
	cacheDir, err := getCacheDir()
	if err != nil {
		return true
	}
	file, err := os.Stat(filepath.Join(cacheDir, serverCachefile))
	if err != nil {
		return true
	}

	lastModified := file.ModTime()

	return time.Since(lastModified) > 24*time.Hour
}