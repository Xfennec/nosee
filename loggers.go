package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func loggersList(config *Config) ([]string, error) {
	lgDirPath := path.Clean(config.configPath + "/scripts/loggers/")
	stat, err := os.Stat(lgDirPath)

	if err != nil {
		return nil, fmt.Errorf("invalid 'loggers' directory '%s': %s", lgDirPath, err)
	}

	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("is not a directory '%s'", lgDirPath)
	}

	scripts, err := filepath.Glob(lgDirPath + "/*")
	if err != nil {
		return nil, fmt.Errorf("error listing '%s' directory: %s", lgDirPath, err)
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

func loggersExec(run *Run) {
	varMap := make(map[string]interface{})
	varMap["NOSEE_SRV"] = GlobalConfig.Name
	varMap["VERSION"] = NoseeVersion
	varMap["HOST_NAME"] = run.Host.Name
	varMap["HOST_FILE"] = run.Host.Filename
	varMap["CLASSES"] = strings.Join(run.Host.Classes, ",")

	var valuesBuff bytes.Buffer
	for _, result := range run.TaskResults {
		for key, val := range result.Values {
			// df.toml;DISK_FULLEST_PERC;27
			str := fmt.Sprintf("%s;%s;%s\n", result.Task.Probe.Filename, key, val)
			valuesBuff.WriteString(str)
		}
	}

	go func() {
		for _, script := range globalLogers {
			cmd := exec.Command(script)

			// we inject Values thru stdin:
			cmd.Stdin = strings.NewReader(valuesBuff.String())

			env := os.Environ()
			for key, val := range varMap {
				env = append(env, fmt.Sprintf("%s=%s", key, InterfaceValueToString(val)))
			}
			cmd.Env = env

			if cmdOut, err := cmd.CombinedOutput(); err != nil {
				Warning.Printf("error running logger '%s': %s: %s", script, err, bytes.TrimSpace(cmdOut))
			} else {
				Trace.Printf("logger '%s' OK: %s", script, bytes.TrimSpace(cmdOut))
			}
		}
	}()
}
