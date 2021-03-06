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
	Name            string
	StartTimeSpread Duration `toml:"start_time_spread"`
	SSHConnTimeWarn Duration `toml:"ssh_connection_time_warn"`
	SSHBlindTrust   bool     `toml:"ssh_blindtrust_fingerprints"`
	SavePath        string   `toml:"save_path"`
	HeartbeatDelay  Duration `toml:"heartbeat_delay"`
}

// Config is the final form of the nosee.toml config file
type Config struct {
	configPath   string
	loadDisabled bool
	doConnTest   bool

	Name                   string
	StartTimeSpreadSeconds int
	SSHConnTimeWarn        time.Duration
	SSHBlindTrust          bool
	SavePath               string
	HeartbeatDelay         time.Duration
}

// GlobalConfig exports the Nosee server configuration
var GlobalConfig *Config

// GlobalConfigRead reads given file and returns a Config
func GlobalConfigRead(dir, file string) (*Config, error) {
	var config Config
	var tConfig tomlConfig

	// defaults:
	// config.xxx -> default if config file not exists
	// tConfig.xxx -> default if parameter's not provided in config file
	config.Name = ""
	tConfig.Name = ""

	config.StartTimeSpreadSeconds = 15
	tConfig.StartTimeSpread.Duration = 15 * time.Second

	config.SSHConnTimeWarn = 10 * time.Second
	tConfig.SSHConnTimeWarn.Duration = config.SSHConnTimeWarn

	config.SSHBlindTrust = false
	tConfig.SSHBlindTrust = false

	config.SavePath = "./"
	tConfig.SavePath = config.SavePath

	config.HeartbeatDelay = 30 * time.Second
	tConfig.HeartbeatDelay.Duration = config.HeartbeatDelay

	config.configPath = dir
	config.loadDisabled = false
	config.doConnTest = true

	if stat, err := os.Stat(config.configPath); err != nil || !stat.Mode().IsDir() {
		return nil, fmt.Errorf("configuration directory not found: %s (%s)", err, config.configPath)
	}

	configPath := path.Clean(dir + "/" + file)

	if stat, err := os.Stat(configPath); err != nil || !stat.Mode().IsRegular() {
		Warning.Printf("no %s file, using defaults\n", configPath)
		return &config, nil
	}

	if _, err := toml.DecodeFile(configPath, &tConfig); err != nil {
		return nil, fmt.Errorf("decoding %s: %s", file, err)
	}

	if tConfig.Name != "" {
		config.Name = tConfig.Name
	}

	if tConfig.StartTimeSpread.Duration > (1 * time.Minute) {
		return nil, errors.New("'start_time_spread' can't be more than a minute")
	}
	config.StartTimeSpreadSeconds = int(tConfig.StartTimeSpread.Duration.Seconds())

	if tConfig.SSHConnTimeWarn.Duration < (1 * time.Second) {
		return nil, errors.New("'ssh_connection_time_warn' can't be less than a second")
	}
	config.SSHConnTimeWarn = tConfig.SSHConnTimeWarn.Duration

	config.SSHBlindTrust = tConfig.SSHBlindTrust

	// should check if writable
	config.SavePath = tConfig.SavePath

	if tConfig.HeartbeatDelay.Duration < (5 * time.Second) {
		return nil, errors.New("'heartbeat_delay' can't be less than 5 seconds")
	}
	config.HeartbeatDelay = tConfig.HeartbeatDelay.Duration

	return &config, nil
}
