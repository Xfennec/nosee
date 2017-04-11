package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Connection is the final form of connection informations of hosts.d files
type Connection struct {
	User            string
	Auths           []ssh.AuthMethod
	Host            string
	Port            int
	SSHConnTimeWarn time.Duration
	Session         *ssh.Session
	Client          *ssh.Client
}

// Close will clone the connection and the session
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

// Implements ssh.HostKeyCallback which is now required due to CVE-2017-3204
// https://github.com/src-d/go-git/pull/329
// This code is temporary and will probably make its way in x/crypto/ssh itself
func hostKeyChecker(hostname string, remote net.Addr, key ssh.PublicKey) error {
	path := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("host key verification with '%s': %s", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], hostname) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				return fmt.Errorf("host key verification with '%s': error parsing %q: %v", path, fields[2], err)
			}
			break
		}
	}

	if hostKey == nil {
		return fmt.Errorf("host key verification with '%s': no hostkey for %s", path, hostname)
	}
	return nil
}

func hostKeyBilndTrustChecker(hostname string, remote net.Addr, key ssh.PublicKey) error {
	return nil
}

// Connect will dial SSH server and open a session
func (connection *Connection) Connect() error {
	sshConfig := &ssh.ClientConfig{
		User: connection.User,
		Auth: connection.Auths,
	}

	if GlobalConfig.SSHBlindTrust == true {
		sshConfig.HostKeyCallback = hostKeyBilndTrustChecker
	} else {
		sshConfig.HostKeyCallback = hostKeyChecker
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

// PublicKeyFile returns an AuthMethod using a private key file
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

// PublicKeyFilePassPhrase returns an AuthMethod using a private key file
// and a passphrase
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

// SSHAgent returns an AuthMethod using SSH agent connection. The pubkeyFile
// params restricts the AuthMethod to only one key, so it wont spam the
// SSH server if the agent holds multiple keys.
func SSHAgent(pubkeyFile string) (ssh.AuthMethod, error) {
	sshAgent, errd := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if errd == nil {
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
	return nil, fmt.Errorf("SSH agent: %v (check SSH_AUTH_SOCK?)", errd)
}
