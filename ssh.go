package main

import (
	"bufio"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Command struct {
	ScriptFile string
	Arguments  string // need shell protection here!
	//~ Env []string // same
}

type Connection struct {
	User       string
	Auths      []ssh.AuthMethod
	Host       string
	Port       int
	ParentHost *Host // temporary, I hope (need a better "split" between Host and Connection)
}

func (connection *Connection) RunCommands(cmds []Command) error {
	var (
		session *ssh.Session
		err     error
	)

	const bootstrap = "bash -s --"

	if session, err = connection.newSession(); err != nil {
		return err
	}
	defer session.Close()

	if err = connection.preparePipes(session, cmds); err != nil {
		return err
	}

	err = session.Run(bootstrap)
	return err
}

func (connection *Connection) newSession() (*ssh.Session, error) {

	sshConfig := &ssh.ClientConfig{
		User: connection.User,
		Auth: connection.Auths,
	}

	dial, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", connection.Host, connection.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial: %s (%s)", err, connection.ParentHost.Name)
	}

	session, err := dial.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	return session, nil
}

func (connection *Connection) preparePipes(session *ssh.Session, cmds []Command) error {
	exitStatus := make(chan int)

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdin for session: %v", err)
	}
	go stdinFilter(cmds, stdin, exitStatus)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go readStd(stdout, connection.ParentHost.Name, "stdout", exitStatus)

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go readStd(stderr, connection.ParentHost.Name, "stderr", exitStatus)

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

func readStd(std io.Reader, name string, prefix string, exitStatus chan int) {

	scanner := bufio.NewScanner(std)

	for scanner.Scan() {
		text := scanner.Text()

		if prefix == "stdout" && len(text) > 2 && text[0:2] == "__" {
			parts := strings.Split(text, "=")
			switch parts[0] {
			case "__EXIT":
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "Invalid __EXIT: %s\n", text)
					continue
				}
				status, err := strconv.Atoi(parts[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid __EXIT value: %s\n", text)
					continue
				}
				//~ fmt.Printf("EXIT detected: %s (status %d)\n", text, status)
				exitStatus <- status
			default:
				fmt.Fprintf(os.Stderr, "Unknown keyword: %s\n", text)
			}
			continue
		}

		fmt.Printf("%s=%s (%s)\n", prefix, text, name)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %s\n", err)
		os.Exit(1)
	}
}

// scripts -> ssh
func stdinFilter(cmds []Command, out io.WriteCloser, exitStatus chan int) {

	// "pkill" dependency or Linux "ps"? (ie: not Cygwin)

	//~ _, err := out.Write([]byte("function __kill_subshells() { l=$(ps -C cat -o pid=); if [ -n \"$l\" ]; then kill $l; fi }\nexport -f __kill_subshells\n"))
	_, err := out.Write([]byte("export __MAIN_PID=$$\nfunction __kill_subshells() { pkill -TERM -P $__MAIN_PID cat; }\nexport -f __kill_subshells\n"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing (setup parent bash): %s\n", err)
	}

	for num, cmd := range cmds {

		file, err := os.Open(cmd.ScriptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open script: %s\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		args := cmd.Arguments

		// cat is needed to "focus" stdin on the child bash
		str := fmt.Sprintf("cat | __SCRIPT_ID=%d bash -s -- %s ; echo __EXIT=$?\n", num, args)

		_, err = out.Write([]byte(str))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing (starting child bash): %s\n", err)
		}

		// no newline so we dont change line numbers
		_, err = out.Write([]byte("trap __kill_subshells EXIT ; "))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing (init child bash): %s\n", err)
		}

		for scanner.Scan() {
			text := scanner.Text()
			//fmt.Printf("stdin=%s\n", text)
			_, err := out.Write([]byte(text + "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing: %s\n", err)
			}
		}

		_, err = out.Write([]byte("__kill_subshells\n"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing (bash instance): %s\n", err)
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error wrtiting: %s\n", err)
			os.Exit(1)
		}

		status := <-exitStatus
		if status != 0 {
			fmt.Printf("(detected exit status %d)\n", status)
		}

	}
	out.Close()
}
