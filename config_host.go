package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type tomlNetwork struct {
	Host            string
	Port            int
	SshConnTimeWarn Duration `toml:"ssh_connection_time_warn"`
}

type tomlAuth struct {
	User          string
	Password      string
	Key           string
	KeyPassphrase string `toml:"key_passphrase"`
	SSHAgent      bool   `toml:"ssh_agent"`
	Pubkey        string
}

type tomlHost struct {
	Disabled bool
	Name     string
	Network  tomlNetwork
	Auth     tomlAuth
	Classes  []string
	Default  []tomlDefault
}

func tomlHostToHost(tHost *tomlHost, config *Config) (*Host, error) {
	var (
		connection Connection
		host       Host
	)

	host.Connection = &connection

	if tHost.Disabled == true {
		return nil, nil
	}

	if tHost.Name == "" {
		return nil, errors.New("invalid or missing 'name'")
	}
	host.Name = tHost.Name

	if tHost.Classes == nil {
		return nil, errors.New("no valid 'classes' parameter found")
	}

	if len(tHost.Classes) == 0 {
		return nil, errors.New("empty classes")
	}
	for _, class := range tHost.Classes {
		if !IsValidTokenName(class) {
			return nil, fmt.Errorf("invalid class name '%s'", class)
		}
	}
	host.Classes = tHost.Classes

	host.Defaults = make(map[string]interface{})
	if err := checkTomlDefault(host.Defaults, tHost.Default); err != nil {
		return nil, err
	}

	if tHost.Network.Host == "" {
		return nil, errors.New("[network] section, invalid or missing 'host'")
	}
	connection.Host = tHost.Network.Host

	if tHost.Network.Port == 0 {
		return nil, errors.New("[network] section, invalid or missing 'port'")
	}
	connection.Port = tHost.Network.Port

	if tHost.Network.SshConnTimeWarn.Duration < (1 * time.Second) {
		return nil, errors.New("'ssh_connection_time_warn' can't be less than a second")
	}
	connection.SshConnTimeWarn = tHost.Network.SshConnTimeWarn.Duration

	if tHost.Auth.User == "" {
		return nil, errors.New("[auth] section, invalid or missing 'user'")
	}
	connection.User = tHost.Auth.User

	if tHost.Auth.Key != "" && tHost.Auth.Password != "" {
		return nil, errors.New("[auth] section, can't use key and password at the same time (see key_passphrase parameter, perhaps?)")
	}
	if tHost.Auth.KeyPassphrase != "" && tHost.Auth.Password != "" {
		return nil, errors.New("[auth] section, can't use key_passphrase and password at the same time")
	}
	if tHost.Auth.SSHAgent == true && tHost.Auth.Password != "" {
		return nil, errors.New("[auth] section, can't use SSH agent and password at the same time")
	}
	if tHost.Auth.SSHAgent == true && tHost.Auth.KeyPassphrase != "" {
		return nil, errors.New("[auth] section, can't use SSH agent and key_passphrase at the same time")
	}
	if tHost.Auth.SSHAgent == true && tHost.Auth.Key != "" {
		return nil, errors.New("[auth] section, can't use SSH agent and key at the same time (see pubkey parameter, perhaps?)")
	}

	if tHost.Auth.Key != "" {
		if fd, err := os.Open(tHost.Auth.Key); err != nil {
			return nil, fmt.Errorf("can't access to key '%s': %s", tHost.Auth.Key, err)
		} else {
			fd.Close()
		}
	}

	// !!! there's many returns following this line, be careful

	if tHost.Auth.Password != "" {
		connection.Auths = []ssh.AuthMethod{
			ssh.Password(tHost.Auth.Password),
		}
		return &host, nil
	}

	if tHost.Auth.SSHAgent == true {
		agent, err := SSHAgent(tHost.Auth.Pubkey)
		if err != nil {
			return nil, err
		}
		connection.Auths = []ssh.AuthMethod{
			agent,
		}
		return &host, nil
	}

	if tHost.Auth.Key != "" && tHost.Auth.KeyPassphrase == "" {
		connection.Auths = []ssh.AuthMethod{
			PublicKeyFile(tHost.Auth.Key),
		}
		return &host, nil
	}

	if tHost.Auth.Key != "" && tHost.Auth.KeyPassphrase != "" {
		connection.Auths = []ssh.AuthMethod{
			PublicKeyFilePassPhrase(tHost.Auth.Key, tHost.Auth.KeyPassphrase),
		}
		return &host, nil
	}

	return nil, errors.New("[auth] section, at least one auth method is needed (password, key or ssh_agent)")
}
