package main

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Connection struct {
	User            string
	Auths           []ssh.AuthMethod
	Host            string
	Port            int
	SSHConnTimeWarn time.Duration
	Session         *ssh.Session
	Client          *ssh.Client
}

func (connection *Connection) Close() error {
	var (
		sessionError error
		clientError  error
	)

	Trace.Printf("SSH closing connection (%s)\n", connection.Host)

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
	Trace.Printf("SSH connection to %s@%s:%d\n", connection.User, connection.Host, connection.Port)
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

func SSHAgent(pubkeyFile string) (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err == nil {
		agent := agent.NewClient(sshAgent)

		// we'll try every key, then
		if pubkeyFile == "" {
			return ssh.PublicKeysCallback(agent.Signers), nil
		}

		agentSigners, err := agent.Signers()
		if err != nil {
			return nil, fmt.Errorf("requesting SSH agent key/signer list: %s", err)
		}

		buffer, err := ioutil.ReadFile(pubkeyFile)
		if err != nil {
			return nil, fmt.Errorf("reading public key '%s': %s", pubkeyFile, err)
		}

		fields := strings.Fields(string(buffer))

		if len(fields) < 3 {
			return nil, fmt.Errorf("invalid field count for public key '%s'", pubkeyFile)
		}

		buffer2, err := base64.StdEncoding.DecodeString(fields[1])
		if err != nil {
			return nil, fmt.Errorf("decoding public key '%s': %s", pubkeyFile, err)
		}

		key, err := ssh.ParsePublicKey(buffer2)
		if err != nil {
			return nil, fmt.Errorf("parsing public key '%s': %s", pubkeyFile, err)
		}

		for _, potentialSigner := range agentSigners {
			if bytes.Compare(key.Marshal(), potentialSigner.PublicKey().Marshal()) == 0 {
				Info.Printf("successfully found %s key in the SSH agent (%s)", pubkeyFile, fields[2])
				cb := func() ([]ssh.Signer, error) {
					signers := []ssh.Signer{potentialSigner}
					return signers, nil
				}
				return ssh.PublicKeysCallback(cb), nil
			}
		}
		return nil, fmt.Errorf("can't find '%s' key in the SSH agent", pubkeyFile)
	}
	return nil, fmt.Errorf("SSH agent: %v (check SSH_AUTH_SOCK?)", err)
}
