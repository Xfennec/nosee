package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (run *Run) readStdout(std io.Reader, exitStatus chan int) {
	scanner := bufio.NewScanner(std)

	for scanner.Scan() {
		text := scanner.Text()
		result := run.currentTaskResult()

		//~ fmt.Printf("stdout=%s (%s)\n", text, run.Host.Name)

		if len(text) > 2 && text[0:2] == "__" {
			parts := strings.Split(text, "=")
			switch parts[0] {
			case "__EXIT":
				if len(parts) != 2 {
					run.addError(fmt.Errorf("Invalid __EXIT: %s\n", text))
					continue
				}
				status, err := strconv.Atoi(parts[1])
				if err != nil {
					run.addError(fmt.Errorf("Invalid __EXIT value: %s\n", text))
					continue
				}
				//~ fmt.Printf("EXIT detected: %s (status %d)\n", text, status)
				exitStatus <- status
			default:
				run.addError(fmt.Errorf("Unknown keyword: %s\n", text))
			}
			continue
		}

		if len(text) > 1 && text[0:1] == "#" {
			result.addLog(text)
			continue
		}

		sep := strings.Index(text, ":")

		if sep == -1 || sep == 0 {
			result.addError(fmt.Errorf("invalid script output: '%s'", text))
			continue
		}

		paramName := strings.TrimSpace(text[0:sep])
		if !IsValidTokenName(paramName) {
			result.addError(fmt.Errorf("invalid parameter name: '%s' (not a valid token name): '%s'", paramName, text))
			continue
		}
		if !IsAllUpper(paramName) {
			result.addError(fmt.Errorf("invalid parameter name: '%s' (upper case needed): '%s'", paramName, text))
			continue
		}

		if _, exists := result.Values[paramName]; exists == true {
			result.addError(fmt.Errorf("parameter '%s' defined multiple times", paramName))
			continue
		}

		value := strings.TrimSpace(text[sep+1:])
		if len(value) == 0 {
			result.addError(fmt.Errorf("empty value for parameter '%s'", paramName))
			continue
		}

		result.Values[paramName] = value
	}

	if err := scanner.Err(); err != nil {
		run.addError(fmt.Errorf("Error reading stdout: %s\n", err))
	}
}

func (run *Run) readStderr(std io.Reader) {
	scanner := bufio.NewScanner(std)

	for scanner.Scan() {
		text := scanner.Text()
		file := filepath.Base(run.currentTaskResult().Task.Probe.Script)
		//~ fmt.Printf("stderr=%s\n", text)
		run.currentTaskResult().addError(fmt.Errorf("%s, stderr: %s", file, text))
	}

	if err := scanner.Err(); err != nil {
		run.addError(fmt.Errorf("Error reading stderr: %s\n", err))
		return // !!!
	}
}

// scripts -> ssh
func (run *Run) stdinInject(out io.WriteCloser, exitStatus chan int) {

	defer out.Close()

	// "pkill" dependency or Linux "ps"? (ie: not Cygwin)
	_, err := out.Write([]byte("export __MAIN_PID=$$\nfunction __kill_subshells() { pkill -TERM -P $__MAIN_PID cat; }\nexport -f __kill_subshells\n"))
	if err != nil {
		run.addError(fmt.Errorf("Error writing (setup parent bash): %s\n", err))
		return
	}

	for num, task := range run.Tasks {

		var result TaskResult
		run.TaskResults = append(run.TaskResults, &result)
		result.StartTime = time.Now()
		result.Task = task
		result.ExitStatus = -1
		result.Values = make(map[string]string)

		var scanner *bufio.Scanner

		if task.Probe.ScriptCache == nil {
			file, err := os.Open(task.Probe.Script)
			if err != nil {
				result.addError(fmt.Errorf("Failed to open script: %s\n", err))
				continue
			}
			defer file.Close()

			scanner = bufio.NewScanner(file)
		} else {
			task.Probe.ScriptCache.Seek(0, io.SeekStart)
			scanner = bufio.NewScanner(task.Probe.ScriptCache)
		}

		args := task.Probe.Arguments

		// cat is needed to "focus" stdin only on the child bash
		str := fmt.Sprintf("cat | __SCRIPT_ID=%d bash -s -- %s ; echo __EXIT=$?\n", num, args)

		_, err = out.Write([]byte(str))
		if err != nil {
			run.addError(fmt.Errorf("Error writing (starting child bash): %s\n", err))
			return
		}

		// no newline so we dont change line numbers
		_, err = out.Write([]byte("trap __kill_subshells EXIT ; "))
		if err != nil {
			run.addError(fmt.Errorf("Error writing (init child bash): %s\n", err))
			return
		}

		for scanner.Scan() {
			text := scanner.Text()
			//fmt.Printf("stdin=%s\n", text)
			_, err := out.Write([]byte(text + "\n"))
			if err != nil {
				run.addError(fmt.Errorf("Error writing: %s\n", err))
				return
			}
		}

		_, err = out.Write([]byte("__kill_subshells\n"))
		if err != nil {
			run.addError(fmt.Errorf("Error writing (bash instance): %s\n", err))
			return
		}

		if err := scanner.Err(); err != nil {
			run.addError(fmt.Errorf("Error wrtiting: %s\n", err))
			return
		}

		status := <-exitStatus
		result.ExitStatus = status
		if status != 0 {
			result.addError(fmt.Errorf("detected non-zero exit status: %d", status))
		}

		result.Duration = time.Now().Sub(result.StartTime)
		if result.Duration > result.Task.Probe.Timeout {
			result.addError(fmt.Errorf("task duration was too long (%s, timeout is %s)", result.Duration, result.Task.Probe.Timeout))
		}
	}
}

func (run *Run) preparePipes() error {
	exitStatus := make(chan int)
	session := run.Host.Connection.Session

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdin for session: %v", err)
	}
	go run.stdinInject(stdin, exitStatus)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go run.readStdout(stdout, exitStatus)

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go run.readStderr(stderr)

	return nil
}
