package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/urfave/cli.v1"
)

// Loggers for trace, info, warning and error severity
var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func writerCreate(std io.Writer, fd *os.File, quiet bool) io.Writer {
	if quiet {
		if fd != nil {
			if std != ioutil.Discard {
				return fd
			}
		}
		return ioutil.Discard
	}

	// no log at all for this stream (no std, no file)
	if std == ioutil.Discard {
		return ioutil.Discard
	}
	// both
	if fd != nil {
		return io.MultiWriter(fd, std)
	}
	return std
}

// LogInit will initialize loggers
func LogInit(ctx *cli.Context) {
	var (
		traceHandle   io.Writer
		infoHandle    io.Writer
		warningHandle io.Writer
		errorHandle   io.Writer
	)

	level := ctx.String("log-level")
	file := ctx.String("log-file")
	quiet := ctx.Bool("quiet")
	timestamp := ctx.Bool("log-timestamp")

	var (
		err error
		fd  *os.File
	)
	if file != "" {
		fd, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create log file '%s' (%s)\n", file, err)
			os.Exit(1)
		}
	} else {
		fd = nil
	}

	switch level {
	case "trace":
		traceHandle = writerCreate(os.Stdout, fd, quiet)
		infoHandle = writerCreate(os.Stdout, fd, quiet)
		warningHandle = writerCreate(os.Stdout, fd, quiet)
		errorHandle = writerCreate(os.Stderr, fd, quiet)
	case "info":
		traceHandle = writerCreate(ioutil.Discard, fd, quiet)
		infoHandle = writerCreate(os.Stdout, fd, quiet)
		warningHandle = writerCreate(os.Stdout, fd, quiet)
		errorHandle = writerCreate(os.Stderr, fd, quiet)
	case "warning":
		traceHandle = writerCreate(ioutil.Discard, fd, quiet)
		infoHandle = writerCreate(ioutil.Discard, fd, quiet)
		warningHandle = writerCreate(os.Stdout, fd, quiet)
		errorHandle = writerCreate(os.Stderr, fd, quiet)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: invalid log level '%s'\n", level)
		os.Exit(1)
	}

	var flags = 0
	if timestamp {
		flags = log.Ldate | log.Ltime
	}

	Trace = log.New(traceHandle,
		"TRACE: ",
		flags|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		flags)

	Warning = log.New(warningHandle,
		"WARNING: ",
		flags)

	Error = log.New(errorHandle,
		"ERROR: ",
		flags)

	Trace.Println("Log init")
}
