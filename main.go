package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli"
)

var myRand *rand.Rand

func schedule(hosts []*Host) {
	for {
		time.Sleep(time.Minute * 1)
	}
}

func configurationDirList(inpath string, dirPath string) ([]string, error) {
	configPath := path.Clean(dirPath + "/" + inpath)

	stat, err := os.Stat(configPath)

	if err != nil {
		return nil, fmt.Errorf("invalid directory '%s': %s", configPath, err)
	}

	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("is not a directory '%s'", configPath)
	}

	list, err := filepath.Glob(configPath + "/*.toml")
	if err != nil {
		return nil, fmt.Errorf("error listing '%s' directory: %s", configPath, err)
	}

	return list, nil
}

func mainDefault(ctx *cli.Context) error {

	configPath := ctx.String("config-path")

	config, err := GlobalConfigRead(configPath, "nosee.toml")
	if err != nil {
		return fmt.Errorf("Config error (nosee.toml): %s", err)
	}
	config.configPath = configPath

	hostsdFiles, err := configurationDirList("hosts.d", configPath)
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}

	var hosts []*Host
	hNames := make(map[string]string)

	for _, file := range hostsdFiles {
		var tHost tomlHost

		// defaults
		tHost.Network.SshConnTimeWarn.Duration = config.SshConnTimeWarn

		if _, err := toml.DecodeFile(file, &tHost); err != nil {
			return fmt.Errorf("Error decoding %s: %s", file, err)
		}

		host, err := tomlHostToHost(&tHost, config)
		if err != nil {
			return fmt.Errorf("Error using %s: %s", file, err)
		}

		if host != nil {
			if f, exists := hNames[host.Name]; exists == true {
				return fmt.Errorf("Config error: duplicate name '%s' (%s, %s)", host.Name, f, file)
			}

			hosts = append(hosts, host)
			hNames[host.Name] = file
		}
	}
	fmt.Printf("host count = %d\n", len(hosts))

	probesdFiles, err := configurationDirList("probes.d", configPath)
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}

	var probes []*Probe
	pNames := make(map[string]string)

	for _, file := range probesdFiles {
		var tProbe tomlProbe

		if _, err := toml.DecodeFile(file, &tProbe); err != nil {
			return fmt.Errorf("Error decoding %s: %s", file, err)
		}

		probe, err := tomlProbeToProbe(&tProbe, config)
		if err != nil {
			return fmt.Errorf("Error using %s: %s", file, err)
		}

		if probe != nil {
			if f, exists := pNames[probe.Name]; exists == true {
				return fmt.Errorf("Config error: duplicate name '%s' (%s, %s)", probe.Name, f, file)
			}

			probes = append(probes, probe)
			pNames[probe.Name] = file
		}

	}

	fmt.Printf("probe count = %d\n", len(probes))

	var taskCount int
	for _, host := range hosts {
		for _, probe := range probes {
			if host.MatchProbeTargets(probe) {
				//~ fmt.Printf("Match: %s | %s\n", host.Name, probe.Script)
				var task Task
				task.Probe = probe
				task.NextRun = time.Now()
				host.Tasks = append(host.Tasks, &task)
				taskCount++
			}
		}
	}

	fmt.Printf("task count = %d\n", taskCount)

	var hostGroup sync.WaitGroup
	for i, host := range hosts {
		hostGroup.Add(1)
		go func(i int, host *Host) {
			defer hostGroup.Done()
			if config.StartTimeSpreadSeconds > 0 {
				// Sleep here, to ease global load
				fact := float32(i) / float32(len(hosts)) * 1000 * float32(config.StartTimeSpreadSeconds)
				wait := time.Duration(fact) * time.Millisecond
				time.Sleep(wait)
			}
			host.Schedule()
		}(i, host)
	}

	hostGroup.Wait()
	return fmt.Errorf("QUIT: empty wait group, everyone died :(")
}

func main() {

	source := rand.NewSource(time.Now().UnixNano())
	myRand = rand.New(source)

	app := cli.NewApp()
	app.Usage = "A nosey, agentless, easy monitoring tool over SSH"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config-path, c",
			Value:  "/etc/nosee/",
			Usage:  "configuration directory `PATH`",
			EnvVar: "NOSEE_CONFIG",
		},
	}

	app.Action = mainDefault
	app.Run(os.Args)
}
