package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func hearthbeatsList(config *Config) ([]string, error) {
	hbDirPath := path.Clean(config.configPath + "/scripts/hearthbeats/")
	stat, err := os.Stat(hbDirPath)

	if err != nil {
		return nil, fmt.Errorf("invalid 'hearthbeats' directory '%s': %s", hbDirPath, err)
	}

	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("is not a directory '%s'", hbDirPath)
	}

	scripts, err := filepath.Glob(hbDirPath + "/*")
	if err != nil {
		return nil, fmt.Errorf("error listing '%s' directory: %s", hbDirPath, err)
	}

	for _, scriptPath := range scripts {
		stat, err := os.Stat(scriptPath)

		if err != nil {
			return nil, fmt.Errorf("invalid 'script' file '%s': %s", scriptPath, err)
		}

		if !stat.Mode().IsRegular() {
			return nil, fmt.Errorf("is not a regular 'script' file '%s'", scriptPath)
		}

		_, err = ioutil.ReadFile(scriptPath)
		if err != nil {
			return nil, fmt.Errorf("error reading script file '%s': %s", scriptPath, err)
		}
	}

	return scripts, nil
}

func hearthbeatExecute(script string) {
	varMap := make(map[string]interface{})
	varMap["NOSEE_SRV"] = GlobalConfig.Name
	varMap["VERSION"] = NoseeVersion
	varMap["DATETIME"] = time.Now().Format(time.RFC3339)
	varMap["STARTTIME"] = appStartTime.Format(time.RFC3339)
	varMap["UPTIME"] = (int)(time.Since(appStartTime).Seconds())

	cmd := exec.Command(script)

	env := os.Environ()
	for key, val := range varMap {
		env = append(env, fmt.Sprintf("%s=%s", key, InterfaceValueToString(val)))
	}
	cmd.Env = env

	if cmdOut, err := cmd.CombinedOutput(); err != nil {
		Warning.Printf("error running hearthbeat '%s': %s: %s", script, err, bytes.TrimSpace(cmdOut))
	}
	Trace.Printf("hearthbeat '%s' OK", script)
}

func hearthbeatsExecute(scripts []string) {
	for _, script := range scripts {
		hearthbeatExecute(script)
	}
}

func hearthbeatsSchedule(scripts []string, delay time.Duration) {
	go func() {
		for {
			hearthbeatsExecute(scripts)
			Info.Printf("hearthbeat, %d scripts", len(scripts))
			// should check total exec duration and compare to delay, here!
			time.Sleep(delay)
		}
	}()
}
