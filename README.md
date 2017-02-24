# Nosee
A nosey, agentless, easy monitoring tool over SSH.

**Warning: Heavy WIP!**

What is it?
-----------

It's an answer when you found usual monitoring systems too heavy and complex.

Nosee uses SSH protocol to execute scripts on monitored systems, checking
for whatever you want. The result is checked and Nosee will ring an alert
of your choice if anything is wrong.

In short : SSH, no agent, simple configuration, usual scripting.

Currently, Nosee requires bash on monitored hosts. It was tested successfully
tested with Linux (of course) but using Cygwin on Windows hosts too.

The Nosee daemon itself can virtually run under any Go supported platform.


Here's a general figure of how Nosee works:

![Nosee general configuration structure](https://raw.github.com/Xfennec/nosee/master/doc/images/img_general.png)

How do you build it?
--------------------

If you have Go installed:

	go get github.com/Xfennec/nosee

You will then be able to launch the binary located in you Go "bin" directory.
(since Go 1.8, `~/go/bin` if you haven't defined any `$GOPATH`)


How do you use it?
------------------

Configuration is mainly done by simple text file using
the [TOML](https://github.com/toml-lang/toml) syntax.

Let's monitor CPU temperature of one of our Web servers.

### Step1. Create a *Host* (SSH connection)

Create a file in the `hosts.d` directory. (ex: `hosts.d/web_myapp.toml`).

```toml
name = "MyApp Webserver"
classes = ["linux", "web", "myapp"]

[network]
host = "192.168.0.100"
port = 22

[auth]
user = "test5"
password = "test5"
```

The `classes` parameter is completely free, you may chose anything that
fits your infrastructure. It will determine what checks will be done on
this host (see below).

Authentication by password is extremely bad, of course, as writing down
a password in a configuration file. Nosee supports other (preferred) options
as passphrases and ssh-agent.

### Step2. Create a *Probe*

Create a file in the `probes.d` directory. (ex: `probes.d/port_80.toml`).

```toml
name="CPU temperature"
targets = ["linux"]

script = "cpu_temp.sh"

delay = "1m"

# Checks

[[check]]
desc = "critical CPU temperature"
if = "CPU_TEMP > 85"
classes = ["critical"]
```

The `targets` parameter will match the `classes` of our host. Targets can
be more precise with things like `linux & web`. (both `linux` and `web` classes
must exist in host)

The `delay` explains that this probe must be run every minute. This is
the lowest delay available.

Then we have a check. You can have multiple checks in a probe. This check
will look at the `CPU_TEMP` value returned by the `cpu_temp.sh`
script (see below) and evaluate the `if` expression. You can have a look
at [govaluate](https://github.com/Knetic/govaluate) for details about
expression's syntax.

If this expression becomes true, the probe will ring a `warning` alert. Here
again, you are free to use any class of your choice to create your own
error typology. (ex: `["critical", "hardware_guys"]` to ring a specific group
of users in charge of critical failures of the hardware)

### Step3. Create a *script* (or use a provided one)

Scripts are hosted in the `scripts/probes/` directory.

```bash
#!/bin/bash

val=$(cat /sys/class/thermal/thermal_zone0/temp)
temp=$(awk "BEGIN {print $val/1000}")
echo "CPU_TEMP:" $temp
```

This script will run on monitored hosts (so… stay light). Here, we read
the first thermal zone and divide it by 1000 to get Celsius value.

Scripts must print `KEY: val` lines to feed checks, as seen above. That's it.

### Step4. Create an *Alert*

Create a file in the `alerts.d` directory. (ex: `alerts.d/mail_julien.toml`).

```toml
name = "Mail Julien"

targets = ["julien", "warning", "critical", "general"]

command = "mail"

arguments = [
    "-s",
    "Nosee: $SUBJECT",
    "julien@mycompagny.com"
]
```

This simple alert will use the usual `mail` command when an alert matches
one (or more) of the given targets. It works exactly the same as classes/targets
for Hosts/Probes to let you create your own vocabulary.
(ex: `"web & production & critical"` is a valid target)

As you may have seen, some variables are available for arguments, like `$SUBJECT`.

There's a special class `general` for very important general messages. At
least one alert must listen permanently at this class.

### Step5. Run Nosee!

	./nosee -c path/to/config/dir

You are now ready to burn your Web server CPU to get your alert mail.
You can use `-l info` to get more information about what Nosee is doing.

	./nosee help

… will tell you more.

Anything else ? (WIP)
---------------------

 - global `nosee.toml` configuration
 - SSH runs (group of probes)
 - "*" targets
 - needed_failures / needed_successes
 - alert scripts
 - alert limits
 - alert env and stdin
 - timeouts
 - rescheduling
 - GOOD and BAD alerts
 - extensive configuration validation
 - alert examples (pushover, SMS, …)
 - script caching

What is the future of Nosee? (WIP)
----------------------------

 - configuration "summary" command
 - graphs (RRD - Round-Robin database)
 - remote Nosee interconnections
