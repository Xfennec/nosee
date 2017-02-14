package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	StartTimeSpread Duration `toml:"start_time_spread"`
	SshConnTimeWarn Duration `toml:"ssh_connection_time_warn"`
	CacheScripts    bool     `toml:"cache_scripts"`
}

type Config struct {
	configPath string

	StartTimeSpreadSeconds int
	SshConnTimeWarn        time.Duration
	CacheScripts           bool
}

func GlobalConfigRead(dir, file string) (*Config, error) {
	var config Config
	var tConfig tomlConfig

	// defaults:
	// config.xxx -> default if config file not exists
	// tConfig.xxx -> default if parameter's not provided in config file
	config.StartTimeSpreadSeconds = 15
	tConfig.StartTimeSpread.Duration = 15 * time.Second

	config.SshConnTimeWarn = 6 * time.Second
	tConfig.SshConnTimeWarn.Duration = config.SshConnTimeWarn

	config.CacheScripts = true
	tConfig.CacheScripts = config.CacheScripts

	configPath := path.Clean(dir + "/" + file)
	stat, err := os.Stat(configPath)

	if err != nil || !stat.Mode().IsRegular() {
		fmt.Printf("Warning: no %s file, using defaults\n", configPath)
		return &config, nil
	}

	if _, err := toml.DecodeFile(configPath, &tConfig); err != nil {
		return nil, fmt.Errorf("decoding %s: %s", file, err)
	}

	if tConfig.StartTimeSpread.Duration > (1 * time.Minute) {
		return nil, errors.New("'start_time_spread' can't be more than a minute")
	}
	config.StartTimeSpreadSeconds = int(tConfig.StartTimeSpread.Duration.Seconds())

	/*if tConfig.SshConnTimeWarn.Duration < (1 * time.Second) {
		return nil, errors.New("'ssh_connection_time_warn' can't be less than a second")
	}*/
	config.SshConnTimeWarn = tConfig.SshConnTimeWarn.Duration

	config.CacheScripts = tConfig.CacheScripts

	return &config, nil
}
