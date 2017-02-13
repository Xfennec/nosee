package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Connection struct {
	User    string
	Auths   []ssh.AuthMethod
	Host    string
	Port    int
	Session *ssh.Session
	Client  *ssh.Client
}

func (connection *Connection) Close() error {
    var (
	sessionError error
	clientError error
    )

    if connection.Session != nil {
	sessionError = connection.Session.Close()
    }
    if connection.Client != nil {
	clientError = connection.Client.Close()
    }

    if clientError != nil {
	return clientError
    }

    return sessionError
}

func (connection *Connection) Connect() error {
	sshConfig := &ssh.ClientConfig{
		User: connection.User,
		Auth: connection.Auths,
	}

	dial, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", connection.Host, connection.Port), sshConfig)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	connection.Client = dial

	session, err := dial.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}
	connection.Session = session

	return nil
}

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func PublicKeyFilePassPhrase(file, passphrase string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	block, _ := pem.Decode(buffer)
	private, err := x509.DecryptPEMBlock(block, []byte(passphrase))
	if err != nil {
		return nil
	}
	block.Headers = nil
	block.Bytes = private
	key, err := ssh.ParsePrivateKey(pem.EncodeToMemory(block))
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func SSHAgent() (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
	}
	return nil, fmt.Errorf("SSH agent: %v (check SSH_AUTH_SOCK?)", err)
}
