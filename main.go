package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Knetic/govaluate"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

var myRand *rand.Rand
var globalAlerts []*Alert

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

func createHosts(ctx *cli.Context, config *Config) ([]*Host, error) {

	hostsdFiles, err := configurationDirList("hosts.d", config.configPath)
	if err != nil {
		return nil, fmt.Errorf("Error: %s", err)
	}

	var hosts []*Host
	hNames := make(map[string]string)

	for _, file := range hostsdFiles {
		var tHost tomlHost

		// defaults
		tHost.Network.SSHConnTimeWarn.Duration = config.SSHConnTimeWarn

		if _, err := toml.DecodeFile(file, &tHost); err != nil {
			return nil, fmt.Errorf("Error decoding %s: %s", file, err)
		}

		host, err := tomlHostToHost(&tHost, config)
		if err != nil {
			return nil, fmt.Errorf("Error using %s: %s", file, err)
		}

		if host != nil {
			if f, exists := hNames[host.Name]; exists == true {
				return nil, fmt.Errorf("Config error: duplicate name '%s' (%s, %s)", host.Name, f, file)
			}

			hosts = append(hosts, host)
			hNames[host.Name] = file
		}
	}
	Info.Printf("host count = %d\n", len(hosts))

	Info.Print("Testing connections…")
	errors := make(chan error, len(hosts))
	for _, host := range hosts {
		go func(host *Host) {
			if err := host.TestConnection(); err != nil {
				errors <- fmt.Errorf("Error connecting %s: %s", host.Name, err)
			} else {
				errors <- nil
			}
		}(host)
	}
	for i := 0; i < len(hosts); i++ {
		select {
		case err := <-errors:
			if err != nil {
				return nil, err
			}
		}
	}

	probesdFiles, err := configurationDirList("probes.d", config.configPath)
	if err != nil {
		return nil, fmt.Errorf("Error: %s", err)
	}

	var probes []*Probe
	pNames := make(map[string]string)

	for _, file := range probesdFiles {
		var tProbe tomlProbe

		if _, err := toml.DecodeFile(file, &tProbe); err != nil {
			return nil, fmt.Errorf("Error decoding %s: %s", file, err)
		}

		probe, err := tomlProbeToProbe(&tProbe, config)
		if err != nil {
			return nil, fmt.Errorf("Error using %s: %s", file, err)
		}

		if probe != nil {
			if f, exists := pNames[probe.Name]; exists == true {
				return nil, fmt.Errorf("Config error: duplicate name '%s' (%s, %s)", probe.Name, f, file)
			}

			probes = append(probes, probe)
			pNames[probe.Name] = file
		}
	}
	Info.Printf("probe count = %d\n", len(probes))

	alertdFiles, err := configurationDirList("alerts.d", config.configPath)
	if err != nil {
		return nil, fmt.Errorf("Error: %s", err)
	}

	var alerts []*Alert
	aNames := make(map[string]string)
	for _, file := range alertdFiles {
		var tAlert tomlAlert

		if _, err := toml.DecodeFile(file, &tAlert); err != nil {
			return nil, fmt.Errorf("Error decoding %s: %s", file, err)
		}

		alert, err := tomlAlertToAlert(&tAlert, config)
		if err != nil {
			return nil, fmt.Errorf("Error using %s: %s", file, err)
		}

		if alert != nil {
			if f, exists := aNames[alert.Name]; exists == true {
				return nil, fmt.Errorf("Config error: duplicate name '%s' (%s, %s)", alert.Name, f, file)
			}

			alerts = append(alerts, alert)
			aNames[alert.Name] = file
		}
	}
	globalAlerts = alerts
	Info.Printf("alert count = %d\n", len(alerts))

	// check if we have at least one "general" alert receiver
	generalReceivers := 0
	for _, alert := range alerts {
		for _, target := range alert.Targets {
			if target == GeneralClass || target == "*" {
				generalReceivers++
			}
		}
	}
	if generalReceivers == 0 {
		return nil, fmt.Errorf("Config error: at least one alert must match the 'general' class")
	}

	// update hosts with tasks
	var taskCount int
	for _, host := range hosts {
		for _, probe := range probes {
			if host.MatchProbeTargets(probe) {
				var task Task
				task.Probe = probe
				task.PrevRun = time.Now()
				task.NextRun = time.Now()
				host.Tasks = append(host.Tasks, &task)
				taskCount++
			}
		}
	}
	Info.Printf("task count = %d\n", taskCount)

	return hosts, nil
}

func scheduleHosts(hosts []*Host, config *Config) error {
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

func mainDefault(ctx *cli.Context) error {
	LogInit(ctx)

	config, err := GlobalConfigRead(ctx.String("config-path"), "nosee.toml")
	if err != nil {
		Error.Printf("Config (nosee.toml): %s", err)
		return cli.NewExitError("", 1)
	}

	hosts, err := createHosts(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}

	CurrentFailsCreate()

	if pidPath := ctx.String("pid-file"); pidPath != "" {
		pid, err := NewPIDFile(pidPath)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("Error with pid file: %s", err), 100)
		}
		defer pid.Remove()
	}

	if err := scheduleHosts(hosts, config); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func mainCheck(ctx *cli.Context) error {
	LogInit(ctx.Parent())

	fmt.Printf("Checking configuration…\n")

	config, err := GlobalConfigRead(ctx.Parent().String("config-path"), "nosee.toml")
	if err != nil {
		Error.Printf("Config (nosee.toml): %s", err)
		return cli.NewExitError("", 1)
	}

	_, err = createHosts(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}
	fmt.Println("OK")
	return nil
}

func mainRecap(ctx *cli.Context) error {
	LogInit(ctx.Parent())

	config, err := GlobalConfigRead(ctx.Parent().String("config-path"), "nosee.toml")
	if err != nil {
		Error.Printf("Config (nosee.toml): %s", err)
		return cli.NewExitError("", 1)
	}

	hosts, err := createHosts(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}

	if ctx.Bool("no-color") == true {
		color.NoColor = true
	}

	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	for _, host := range hosts {
		fmt.Printf("%s: %s\n", cyan("Host"), host.Name)
		for _, task := range host.Tasks {
			fmt.Printf("  %s: %s (%dm)\n", green("Probe"), task.Probe.Name, int(task.Probe.Delay.Minutes()))
			for _, check := range task.Probe.Checks {
				fmt.Printf("    %s: %s (%s)\n", yellow("Check"), check.Desc, strings.Join(check.Classes, ", "))
				var msg AlertMessage
				msg.Classes = check.Classes
				alertCount := 0
				for _, alert := range globalAlerts {
					if msg.MatchAlertTargets(alert) {
						alertCount++
						fmt.Printf("      %s: %s\n", red("Alert"), alert.Name)
					}
				}
				if alertCount == 0 {
					fmt.Println(red("      No valid alert for this check!"))
				}
			}
		}
	}

	return nil
}

func mainExpr(ctx *cli.Context) error {
	LogInit(ctx.Parent())
	if ctx.NArg() == 0 {
		err := fmt.Errorf("Error, you must provide a govaluate expression parameter.\nSee https://github.com/Knetic/govaluate for syntax and features.")
		return cli.NewExitError(err, 1)
	}
	exprString := ctx.Args().Get(0)

	expr, err := govaluate.NewEvaluableExpressionWithFunctions(exprString, CheckFunctions)
	if err != nil {
		return cli.NewExitError(err, 2)
	}

	if vars := expr.Vars(); len(vars) > 0 {
		err := fmt.Errorf("Undefined variables: %s", strings.Join(vars, ", "))
		return cli.NewExitError(err, 11)
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return cli.NewExitError(err, 3)
	}

	fmt.Println(InterfaceValueToString(result))
	return nil
}

func main() {
	source := rand.NewSource(time.Now().UnixNano())
	myRand = rand.New(source)

	// generic (aka "not cli command specific") inits
	CheckFunctionsInit()

	app := cli.NewApp()
	app.Usage = "Nosee: a nosey, agentless, easy monitoring tool over SSH"
	app.Version = "0.1"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config-path, c",
			Value:  "/etc/nosee/",
			Usage:  "configuration directory `PATH`",
			EnvVar: "NOSEE_CONFIG",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "warning",
			Usage: "log `level` verbosity (trace, info, warning)",
		},
		cli.StringFlag{
			Name:  "log-file, f",
			Usage: "log file to `FILE` (append)",
		},
		cli.BoolFlag{
			Name:  "log-timestamp, t",
			Usage: "add timestamp to log output",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "no stdout/err output (except launch errors)",
		},
		cli.StringFlag{
			Name:  "pid-file, p",
			Usage: "create pid `FILE`",
		},
	}

	app.Action = mainDefault

	app.Commands = []cli.Command{
		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check configuration files and connections",
			Action:  mainCheck,
		},
		{
			Name:    "recap",
			Aliases: []string{"r"},
			Usage:   "Recap configuration",
			Action:  mainRecap,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "no-color",
					Usage: "disable color output ",
				},
			},
		},
		{
			Name:    "expr",
			Aliases: []string{"e"},
			Usage:   "Test 'govaluate' expression (See Checks 'If')",
			Action:  mainExpr,
		},
	}

	app.Run(os.Args)
}
