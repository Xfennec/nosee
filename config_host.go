package main

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

type tomlNetwork struct {
	Host string
	Port int
}

type tomlAuth struct {
	User          string
	Password      string
	Key           string
	KeyPassphrase string `toml:"key_passphrase"`
	SSHAgent      bool   `toml:"ssh_agent"`
}

type tomlHost struct {
	Disabled bool
	Name     string
	Network  tomlNetwork
	Auth     tomlAuth
	Classes  []string
}

func tomlHostToHost(tHost *tomlHost) (*Host, error) {
	var (
		connection Connection
		host       Host
	)

	host.Connection = &connection
	connection.ParentHost = &host

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

	if tHost.Network.Host == "" {
		return nil, errors.New("[network] section, invalid or missing 'host'")
	}
	connection.Host = tHost.Network.Host

	if tHost.Network.Port == 0 {
		return nil, errors.New("[network] section, invalid or missing 'port'")
	}
	connection.Port = tHost.Network.Port

	if tHost.Auth.User == "" {
		return nil, errors.New("[auth] section, invalid or missing 'user'")
	}
	connection.User = tHost.Auth.User

	methodCount := 0

	if tHost.Auth.Password != "" {
		methodCount++
	}

	if tHost.Auth.Key != "" {
		methodCount++
	}

	if tHost.Auth.SSHAgent == true {
		methodCount++
	}

	if methodCount > 1 {
		return nil, errors.New("[auth] section, only one auth method is allowed at a time (password, key or ssh_agent)")
	}

	if methodCount == 0 {
		return nil, errors.New("[auth] section, at least one auth method is needed (password, key or ssh_agent)")
	}

	if tHost.Auth.Password != "" {
		connection.Auths = []ssh.AuthMethod{
			ssh.Password(tHost.Auth.Password),
		}
		return &host, nil
	}

	if tHost.Auth.SSHAgent == true {
		agent, err := SSHAgent()
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

	return nil, errors.New("[auth] section, weird (and invalid) configuration")
}