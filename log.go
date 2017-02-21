package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func LogInit(level string) {
	var (
		traceHandle   io.Writer
		infoHandle    io.Writer
		warningHandle io.Writer
		errorHandle   io.Writer
	)

	switch level {
	case "trace":
		traceHandle = os.Stdout
		infoHandle = os.Stdout
		warningHandle = os.Stdout
		errorHandle = os.Stderr
	case "info":
		traceHandle = ioutil.Discard
		infoHandle = os.Stdout
		warningHandle = os.Stdout
		errorHandle = os.Stderr
	case "warning":
		traceHandle = ioutil.Discard
		infoHandle = ioutil.Discard
		warningHandle = os.Stdout
		errorHandle = os.Stderr
	default:
		fmt.Fprintf(os.Stderr, "ERROR: invalid log level '%s'\n", level)
		os.Exit(1)
	}

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime)

	Warning = log.New(warningHandle,
		"WARNING: ",
		0)

	Error = log.New(errorHandle,
		"ERROR: ",
		0)

	Trace.Println("Log init")
}
