package main

import (
	"bytes"
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

// NoseeVersion in X.Y string format
const NoseeVersion = "0.1"

var myRand *rand.Rand
var globalAlerts []*Alert
var globalLogers []string
var appStartTime time.Time

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

func createProbes(ctx *cli.Context, config *Config) ([]*Probe, error) {
	probesdFiles, errd := configurationDirList("probes.d", config.configPath)
	if errd != nil {
		return nil, fmt.Errorf("Error: %s", errd)
	}

	var probes []*Probe
	pNames := make(map[string]string)

	for _, file := range probesdFiles {
		var tProbe tomlProbe

		if _, err := toml.DecodeFile(file, &tProbe); err != nil {
			return nil, fmt.Errorf("Error decoding %s: %s", file, err)
		}

		_, filename := path.Split(file)
		probe, err := tomlProbeToProbe(&tProbe, config, filename)
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
	return probes, nil
}

func createAlerts(ctx *cli.Context, config *Config) ([]*Alert, error) {
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
	//  = alerts
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
	return alerts, nil
}

func createHosts(ctx *cli.Context, config *Config) ([]*Host, error) {
	hostsdFiles, errc := configurationDirList("hosts.d", config.configPath)
	if errc != nil {
		return nil, fmt.Errorf("Error: %s", errc)
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

		_, filename := path.Split(file)
		host, err := tomlHostToHost(&tHost, config, filename)
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

	if config.doConnTest == true {
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
	}

	probes, err := createProbes(ctx, config)
	if err != nil {
		return nil, err
	}

	globalAlerts, err = createAlerts(ctx, config)
	if err != nil {
		return nil, err
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
	GlobalConfig = config

	heartbeats, err := heartbeatsList(config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 2)
	}

	globalLogers, err = loggersList(config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 2)
	}

	hosts, err := createHosts(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}

	CurrentFailsCreate()
	CurrentFailsLoad()

	if pidPath := ctx.String("pid-file"); pidPath != "" {
		pid, err := NewPIDFile(pidPath)
		if err != nil {
			return cli.NewExitError(fmt.Errorf("Error with pid file: %s", err), 100)
		}
		defer pid.Remove()
	}

	heartbeatsSchedule(heartbeats, config.HeartbeatDelay)

	if err := scheduleHosts(hosts, config); err != nil {
		return cli.NewExitError(err, 1)
	}

	return nil
}

func mainCheck(ctx *cli.Context) error {
	LogInit(ctx.Parent())

	fmt.Printf("Checking configuration and connections…\n")

	config, err := GlobalConfigRead(ctx.Parent().String("config-path"), "nosee.toml")
	if err != nil {
		Error.Printf("Config (nosee.toml): %s", err)
		return cli.NewExitError("", 1)
	}
	GlobalConfig = config

	_, err = heartbeatsList(config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 2)
	}

	_, err = loggersList(config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 2)
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
	GlobalConfig = config

	// TODO: should probably display heartbeats/loggers in the recap, then?
	_, err = heartbeatsList(config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 2)
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
		err := fmt.Errorf("Error, you must provide a govaluate expression parameter, see https://github.com/Knetic/govaluate for syntax and features")
		return cli.NewExitError(err, 1)
	}
	exprString := ctx.Args().Get(0)

	expr, err := govaluate.NewEvaluableExpressionWithFunctions(exprString, CheckFunctions)
	if err != nil {
		return cli.NewExitError(err, 2)
	}

	if vars := expr.Vars(); len(vars) > 0 {
		errv := fmt.Errorf("Undefined variables: %s", strings.Join(vars, ", "))
		return cli.NewExitError(errv, 11)
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return cli.NewExitError(err, 3)
	}

	fmt.Println(InterfaceValueToString(result))
	return nil
}

func mainTest(ctx *cli.Context) error {
	LogInit(ctx.Parent())

	config, err := GlobalConfigRead(ctx.Parent().String("config-path"), "nosee.toml")
	if err != nil {
		Error.Printf("Config (nosee.toml): %s", err)
		return cli.NewExitError("", 1)
	}
	config.loadDisabled = true // WARNING!
	config.doConnTest = false  // WARNING!
	GlobalConfig = config

	hosts, err := createHosts(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}

	// createHosts already load probes, but we need the full list
	// and not only probes targeting our host
	probes, err := createProbes(ctx, config)
	if err != nil {
		Error.Println(err)
		return cli.NewExitError("", 10)
	}

	requestedHost := ctx.Args().Get(0)
	requestedProbe := ctx.Args().Get(1)

	if requestedHost == "" {
		var list bytes.Buffer
		for _, host := range hosts {
			list.WriteString(fmt.Sprintf("- %s (%s)\n", host.Filename, host.Name))
		}
		Error.Printf("you must give a host Name or hosts.d/ filename:\n%s", list.String())
		return cli.NewExitError("", 1)
	}

	if requestedProbe == "" {
		var list bytes.Buffer
		for _, probe := range probes {
			list.WriteString(fmt.Sprintf("- %s (%s)\n", probe.Filename, probe.Name))
		}
		Error.Printf("you must give a probe Name or probes.d/ filename:\n%s", list.String())
		return cli.NewExitError("", 1)
	}

	// Locate requested host and probe…
	var foundHost *Host
	for _, host := range hosts {
		if host.Name == requestedHost || host.Filename == requestedHost {
			foundHost = host
			break
		}
	}
	if foundHost == nil {
		Error.Printf("can't find '%s' host", requestedHost)
		return cli.NewExitError("", 1)
	}

	var foundProbe *Probe
	for _, probe := range probes {
		if probe.Name == requestedProbe || probe.Filename == requestedProbe {
			foundProbe = probe
			break
		}
	}
	if foundProbe == nil {
		Error.Printf("can't find '%s' probe", requestedProbe)
		return cli.NewExitError("", 1)
	}

	if ctx.Bool("no-color") == true {
		color.NoColor = true
	}

	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	magenta := color.New(color.FgMagenta).SprintFunc()
	magentaS := color.New(color.FgMagenta).Add(color.CrossedOut).SprintFunc()

	_, scriptName := path.Split(foundProbe.Script)
	fmt.Printf("Testing: host '%s' with probe '%s' (%s, %s) using script '%s'\n", cyan(foundHost.Name), green(foundProbe.Name), foundHost.Filename, foundProbe.Filename, magenta(scriptName))
	if foundHost.Disabled == true {
		fmt.Printf("Note: the host '%s' is currently %s\n", red(foundHost.Name), red("disabled"))
	}
	if foundProbe.Disabled == true {
		fmt.Printf("Note: the probe '%s' is currently %s\n", red(foundProbe.Name), red("disabled"))
	}
	if foundHost.MatchProbeTargets(foundProbe) == false {
		fmt.Printf("Note: the probe '%s' does %s match host '%s' (see classes and targets)\n", red(foundProbe.Name), red("not"), red(foundHost.Name))
	}

	// print defaults
	for key, val := range foundProbe.Defaults {
		if _, ok := foundHost.Defaults[key]; ok == true {
			fmt.Printf("default: %s = %s -> %s (host override)\n",
				magenta(key),
				magentaS(InterfaceValueToString(val)),
				magenta(foundHost.Defaults[key]))
		} else {
			fmt.Printf("default: %s = %s\n", magenta(key), magenta(InterfaceValueToString(val)))
		}
	}

	var run Run
	run.StartTime = time.Now()
	run.Host = foundHost

	var task Task
	task.Probe = foundProbe
	task.PrevRun = time.Now()
	task.NextRun = time.Now()

	run.Tasks = append(run.Tasks, &task)
	run.Go()

	if len(run.Errors) > 0 {
		for _, err := range run.Errors {
			fmt.Printf("run error: %s\n", red(err))
		}
		return nil
	}

	result := run.TaskResults[0]

	for key, val := range result.Values {
		fmt.Printf("value: %s = %s\n", yellow(key), yellow(val))
	}

	for _, err := range result.Logs {
		fmt.Printf("log: %s\n", cyan(err))
	}

	if result.ExitStatus == 0 {
		fmt.Printf("script exit status: %s (success)\n", green(result.ExitStatus))
	} else {
		fmt.Printf("script exit status: %s (error)\n", red(result.ExitStatus))
	}
	fmt.Printf("script duration: %s (+ ssh dial duration: %s)\n", result.Duration, run.DialDuration)

	if run.totalErrorCount() > 0 {
		for _, err := range result.Errors {
			fmt.Printf("error: %s\n", red(err))
		}
		return nil
	}

	result.DoChecks()

	// DoChecks may add its own errors
	for _, err := range result.Errors {
		fmt.Printf("error: %s\n", red(err))
	}

	for _, check := range result.SuccessfulChecks {
		fmt.Printf("check %s: %s: false (no alert)\n", green("GOOD"), green(check.Desc))
	}
	for _, check := range result.FailedChecks {
		fmt.Printf("check %s: %s: true (alert)\n", red("BAD"), red(check.Desc))
	}

	return nil
}

func main() {
	// generic (aka "not cli command specific") inits
	source := rand.NewSource(time.Now().UnixNano())
	myRand = rand.New(source)
	CheckFunctionsInit()
	appStartTime = time.Now()

	app := cli.NewApp()
	app.Usage = "Nosee: a nosey, agentless, easy monitoring tool over SSH"
	app.Version = NoseeVersion

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
			Name:      "check",
			Aliases:   []string{"c"},
			Usage:     "Check configuration files and connections",
			ArgsUsage: " ",
			Action:    mainCheck,
		},
		{
			Name:      "recap",
			Aliases:   []string{"r"},
			Usage:     "Recap configuration",
			ArgsUsage: " ",
			Action:    mainRecap,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "no-color",
					Usage: "disable color output ",
				},
			},
		},
		{
			Name:      "expr",
			Aliases:   []string{"e"},
			Usage:     "Test 'govaluate' expression (See Checks 'If')",
			ArgsUsage: "expression",
			Action:    mainExpr,
		},
		{
			Name:        "test",
			Aliases:     []string{"t"},
			Usage:       "Test any Probe on a any Host",
			ArgsUsage:   "host probe",
			Description: "use Name or filename.toml (without path) for host and probe (disabled or not, targeted or not)",
			Action:      mainTest,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "no-color",
					Usage: "disable color output ",
				},
			},
		},
	}

	app.Run(os.Args)
}
