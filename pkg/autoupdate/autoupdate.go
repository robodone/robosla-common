package autoupdate

import (
	"log"
	"os"
	"sync"
	"time"
)

var (
	updatesMu            sync.Mutex
	updatesDisabledDepth = 0
)

func DisableUpdates() {
	updatesMu.Lock()
	defer updatesMu.Unlock()
	updatesDisabledDepth++
}

func EnableUpdates() {
	updatesMu.Lock()
	defer updatesMu.Unlock()
	if updatesDisabledDepth > 0 {
		updatesDisabledDepth--
	}
}

func areUpdatesEnabled() bool {
	updatesMu.Lock()
	defer updatesMu.Unlock()
	return updatesDisabledDepth == 0
}

func Run(manifestURL, version string, initialDelay, delay time.Duration) {
	// Before we perform the first check, we need to wait for initialDelay.
	// This is useful in case if the agent is started on boot, and the network
	// is not yet connected. It might be beneficial to wait a few minutes before
	// the check.
	time.Sleep(initialDelay)
	for {
		if areUpdatesEnabled() {
			needsRestart, err := UpdateCurrentBinaryIfNeeded(manifestURL, version)
			if err != nil {
				log.Printf("UpdateCurrentBinaryIfNeeded failed: %v", err)
			}
			if needsRestart {
				break
			}
		}
		time.Sleep(delay)
	}
	// Wait for an opportunity to restart.
	for !areUpdatesEnabled() {
		time.Sleep(time.Second)
	}
	// Exit, just to be restarted by systemd.
	log.Printf("New version downloaded. Restarting to start it")
	os.Exit(0)
}
