package main

import (
	"fmt"
	"strings"
	"time"
)

// Host is the final form of hosts.d files
type Host struct {
	Name       string
	Filename   string
	Disabled   bool
	Classes    []string
	Connection *Connection
	Defaults   map[string]interface{}
	Tasks      []*Task
}

// HasClass returns true if this Host has this class
func (host *Host) HasClass(class string) bool {
	if class == "*" {
		return true
	}

	for _, hClass := range host.Classes {
		if hClass == class {
			return true
		}
	}
	return false
}

// MatchProbeTargets returns true if this Host matches probe's classes
func (host *Host) MatchProbeTargets(probe *Probe) bool {
	for _, pTargets := range probe.Targets {
		tokens := strings.Split(pTargets, "&")
		matched := 0
		mustMatch := len(tokens)
		for _, token := range tokens {
			ttoken := strings.TrimSpace(token)
			if host.HasClass(ttoken) {
				matched++
			}
		}
		if matched == mustMatch {
			return true
		}
	}
	return false
}

// Schedule will loop forever, creating and executing runs for this host
func (host *Host) Schedule() {
	for {
		start := time.Now()

		var run Run
		run.Host = host
		run.StartTime = start

		for _, task := range host.Tasks {
			if start.After(task.NextRun) || start.Equal(task.NextRun) {
				taskable, err := task.Taskable()
				if err != nil {
					Trace.Printf("Taskable() failed: %s", err)
					run.addError(err)
					continue
				}
				if taskable == false {
					Info.Printf("host '%s', paused task '%s'\n", host.Name, task.Probe.Name)
					continue
				}

				task.ReSchedule(start.Add(task.Probe.Delay))
				Info.Printf("host '%s', running task '%s'\n", host.Name, task.Probe.Name)
				run.Tasks = append(run.Tasks, task)
			}
		}

		if len(run.Tasks) > 0 {
			run.Go()
			run.Alerts()
			Trace.Printf("currentFails count = %d\n", len(currentFails))
			loggersExec(&run)
		}
		Info.Printf("host '%s', run ended", host.Name)

		end := time.Now()
		dur := end.Sub(start)

		if dur < time.Minute {
			remains := time.Minute - dur
			time.Sleep(remains)
		} else {
			run.addError(fmt.Errorf("run duration was too long (%s)", run.Duration))
		}
		Trace.Printf("(loop %s)\n", host.Name)
	}
}

// TestConnection will return nil if connection to the host was successful
func (host *Host) TestConnection() error {

	//const bootstrap = "bash -s --"

	startTime := time.Now()

	channel := make(chan error, 1)
	go func() {
		if err := host.Connection.Connect(); err != nil {
			channel <- err
		}
		defer host.Connection.Close()
		channel <- nil
	}()

	connTimeout := host.Connection.SSHConnTimeWarn * 2

	select {
	case err := <-channel:
		if err != nil {
			return err
		}
	case <-time.After(connTimeout):
		return fmt.Errorf("SSH connection timeout (after %s)", connTimeout)
	}

	dialDuration := time.Now().Sub(startTime)

	if dialDuration > host.Connection.SSHConnTimeWarn {
		return fmt.Errorf("SSH connection time was too long: %s (ssh_connection_time_warn = %s)", dialDuration, host.Connection.SSHConnTimeWarn)
	}

	/*if err := run.prepareTestPipes(); err != nil {
		return err
	}*/

	/*if err := host.TestRun(bootstrap); err != nil {
		return err
	}*/
	Info.Printf("Connection to '%s' OK (%s)", host.Name, dialDuration)

	return nil
}
