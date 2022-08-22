package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Connection is the final form of connection informations of hosts.d files
type Connection struct {
	User            string
	Auths           []ssh.AuthMethod
	Host            string
	Port            int
	Ciphers         []string
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

// knownHostHash hash hostname using salt64 like ssh is
// doing for "hashed" .ssh/known_hosts files
func knownHostHash(hostname string, salt64 string) string {
	buffer, err := base64.StdEncoding.DecodeString(salt64)
	if err != nil {
		return ""
	}
	h := hmac.New(sha1.New, buffer)
	h.Write([]byte(hostname))
	res := h.Sum(nil)

	hash := base64.StdEncoding.EncodeToString(res)
	return hash
}

// Implements ssh.HostKeyCallback which is now required due to CVE-2017-3204
// see https://github.com/golang/go/issues/29286 for the ecdsa-sha2-nistp256 part
// ("If ClientConfig.HostKeyAlgorithms is not set, a reasonable default is set for acceptable host key type")
func hostKeyChecker(hostname string, remote net.Addr, key ssh.PublicKey) error {
	path := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	hostKeyCallback, err := knownhosts.New(path)
	if err != nil {
		return err
	}

	err = hostKeyCallback(hostname, remote, key)
	if err != nil {
		return fmt.Errorf("%s, use ssh client to manually connect to %s (you may have to specify algo: ssh -o HostKeyAlgorithms=ecdsa-sha2-nistp256 …)", err, hostname)
	}
	return nil
}

// Old ssh.HostKeyCallback implementation
// We parse $HOME/.ssh/known_hosts and check for a matching key + hostname
// Supported : Hashed hostnames, revoked keys (or any other marker), non-standard ports
// Unsupported yet: patterns (*? wildcards)
// This code is temporary, x/crypto/ssh will probably provide something similar. One day.
func _hostKeyChecker(hostname string, remote net.Addr, key ssh.PublicKey) error {
	path := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening '%s': %s", path, err)
	}
	defer file.Close()

	// remove standard port if given, add square brackets for non-standard ones
	hp := strings.Split(hostname, ":")
	if len(hp) == 2 {
		if hp[1] == "22" {
			hostname = hp[0]
		} else {
			hostname = "[" + hp[0] + "]:" + hp[1]
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		marker, hosts, hostKey, _, _, err := ssh.ParseKnownHosts(scanner.Bytes())
		if err == io.EOF {
			continue
		}
		if err != nil {
			return fmt.Errorf("parsing '%s': %s", path, err)
		}
		if marker != "" {
			continue // @cert-authority or @revoked
		}
		fmt.Printf("%s VS %s", key.Marshal(), hostKey.Marshal())
		if bytes.Equal(key.Marshal(), hostKey.Marshal()) {
			for _, host := range hosts {
				if len(host) > 1 && host[0:1] == "|" {
					parts := strings.Split(host, "|")
					if parts[1] != "1" {
						Trace.Printf("'%s': only type 1 is supported for hashed hosts", path)
						continue
					}
					if knownHostHash(hostname, parts[2]) == parts[3] {
						Trace.Printf("successfully found a matching key in '%s' for (hashed) '%s'", path, hostname)
						return nil
					}
				} else {
					if host == hostname {
						Trace.Printf("successfully found a matching key in '%s' for '%s'", path, hostname)
						return nil
					}
				}
			}
			Info.Printf("searching '%s' in '%s': found a matching key, but not with exact hostname(s): %s (patterns are not supported yet)", hostname, path, strings.Join(hosts, ", "))
		}
	}

	return fmt.Errorf("can't find matching key in '%s' for '%s' (try 'ssh %s' to add it?)", path, hostname, hostname)
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

	if len(connection.Ciphers) > 0 {
		sshConfig.Config = ssh.Config{
			Ciphers: connection.Ciphers,
		}
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
				Trace.Printf("successfully found %s key in the SSH agent (%s)", pubkeyFile, fields[2])
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
