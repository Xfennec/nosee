package main

import (
	"sync"
	"time"
)

// CurrentFail type hold informations about a failure currently detected
// and not resolved yet
type CurrentFail struct {
	FailStart time.Time
	FailCount int
	OkCount   int
	// probably need some sort of uniq ID for the fail (rand?)

	// optional "payload"
	RelatedTask  *Task // for Checks (!!)
	RelatedHost  *Host // for Runs
	RelatedTTask *Task // for Tasks
}

var (
	currentFails      map[string]*CurrentFail
	currentFailsMutex sync.Mutex
)

// CurrentFailsCreate initialize the global currentFails variable
func CurrentFailsCreate() {
	currentFails = make(map[string]*CurrentFail)
}

// CurrentFailDelete deleted the CurrentFail with the given hash of the global currentFails
func CurrentFailDelete(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	delete(currentFails, hash)
}

// CurrentFailAdd adds a CurrentFail to the global currentFails using given hash
func CurrentFailAdd(hash string, failedCheck *CurrentFail) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash] = failedCheck
}

// CurrentFailInc increments FailCount of the CurrentFail with the given hash
func CurrentFailInc(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].FailCount++
	currentFails[hash].OkCount = 0
}

// CurrentFailDec increments OkCount of the CurrentFail with the given hash
func CurrentFailDec(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].OkCount++
}

// CurrentFailGetAndInc returns the CurrentFail with the given hash and
// increments its FailCount. The CurrentFail is created if it does not
// already exists.
func CurrentFailGetAndInc(hash string) *CurrentFail {
	cf, ok := currentFails[hash]
	if !ok {
		var cf CurrentFail
		cf.FailCount = 1
		cf.OkCount = 0
		cf.FailStart = time.Now()
		CurrentFailAdd(hash, &cf)
		return &cf
	}

	CurrentFailInc(hash)
	return cf
}

// CurrentFailGetAndDec returns the CurrentFail with the given hash and
// increments its OkCount
func CurrentFailGetAndDec(hash string) *CurrentFail {
	cf, ok := currentFails[hash]
	if !ok {
		return nil
	}
	CurrentFailDec(hash)
	return cf
}
