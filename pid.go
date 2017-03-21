package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type PIDFile struct {
	Path string
}

func checkPIDFileExists(path string) error {
	if pidByte, err := ioutil.ReadFile(path); err == nil {
		pidString := strings.TrimSpace(string(pidByte))
		if pid, err := strconv.Atoi(pidString); err == nil {
			if pidIsRunning(pid) {
				return fmt.Errorf("pid file '%s' already exists", path)
			}
		}
	}
	return nil
}

func NewPIDFile(path string) (*PIDFile, error) {
	if err := checkPIDFileExists(path); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755)); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return nil, err
	}

	return &PIDFile{Path: path}, nil
}

func (file PIDFile) Remove() error {
	return os.Remove(file.Path)
}

func pidIsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))

	if err != nil && err.Error() == "no such process" {
		return false
	}

	if err != nil && err.Error() == "os: process already finished" {
		return false
	}

	return true
}
