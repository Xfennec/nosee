package main

import (
	"encoding/json"
	"os"
	"path"
	"sync"
	"time"

	"github.com/satori/go.uuid"
)

// CurrentFail type hold informations about a failure currently detected
// and not resolved yet
type CurrentFail struct {
	FailStart time.Time
	FailCount int
	OkCount   int
	UniqueID  string

	// optional "payload"
	RelatedTask  *Task // for Checks (!!)
	RelatedHost  *Host // for Runs
	RelatedTTask *Task // for Tasks
}

var (
	currentFails      map[string]*CurrentFail
	currentFailsMutex sync.Mutex
)

const statusFile string = "nosee-fails.json"

// CurrentFailsCreate initialize the global currentFails variable
func CurrentFailsCreate() {
	currentFails = make(map[string]*CurrentFail)
}

// CurrentFailsSave dumps current alerts to disk
func CurrentFailsSave() {
	// doing this in a go routine allows this function to be called
	// by functions that are already locking the mutex
	go func() {
		currentFailsMutex.Lock()
		defer currentFailsMutex.Unlock()

		path := path.Clean(GlobalConfig.SavePath + "/" + statusFile)
		f, err := os.Create(path)
		if err != nil {
			Error.Printf("can't save fails in '%s': %s (see save_path param?)", path, err)
			return
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		err = enc.Encode(&currentFails)
		if err != nil {
			Error.Printf("fails json encode: %s", err)
			return
		}
		Info.Printf("current fails successfully saved to '%s'", path)
	}()
}

// CurrentFailsLoad will load from disk previous "fails"
func CurrentFailsLoad() {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()

	path := path.Clean(GlobalConfig.SavePath + "/" + statusFile)
	f, err := os.Open(path)
	if err != nil {
		Warning.Printf("can't read previous status: %s, no fails loaded", err)
		return
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(&currentFails)
	if err != nil {
		Error.Printf("'%s' json decode: %s", path, err)
	}
	Info.Printf("'%s' loaded: %d fail(s)", path, len(currentFails))
}

// CurrentFailDelete deleted the CurrentFail with the given hash of the global currentFails
func CurrentFailDelete(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	delete(currentFails, hash)
	CurrentFailsSave()
}

// CurrentFailAdd adds a CurrentFail to the global currentFails using given hash
func CurrentFailAdd(hash string, failedCheck *CurrentFail) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash] = failedCheck
	CurrentFailsSave()
}

// CurrentFailInc increments FailCount of the CurrentFail with the given hash
func CurrentFailInc(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].FailCount++
	currentFails[hash].OkCount = 0
	CurrentFailsSave()
}

// CurrentFailDec increments OkCount of the CurrentFail with the given hash
func CurrentFailDec(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].OkCount++
	CurrentFailsSave()
}

// CurrentFailGetAndInc returns the CurrentFail with the given hash and
// increments its FailCount. The CurrentFail is created if it does not
// already exists.
func CurrentFailGetAndInc(hash string) *CurrentFail {
	cf, ok := currentFails[hash]
	if !ok {
		var cf CurrentFail
		uuid, _ := uuid.NewV4()
		cf.FailCount = 1
		cf.OkCount = 0
		cf.FailStart = time.Now()
		cf.UniqueID = uuid.String()
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
