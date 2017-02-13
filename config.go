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
}

type Config struct {
	StartTimeSpreadSeconds int
}

func GlobalConfigRead(dir, file string) (*Config, error) {
	var config Config
	var tConfig tomlConfig

	// defaults:
    // config.xxx -> default if config file not exists
    // tConfig.xxx -> default if parameter's not provided in config file
	config.StartTimeSpreadSeconds = 15
    tConfig.StartTimeSpread.Duration = 15 * time.Second


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

	return &config, nil
}
