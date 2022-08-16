// Copyright 2020 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"os"
	"time"
)

// Event indicating that something happened
var archiveIncoming *Event

// Handler that performs archiving in a way that's serialized on the archive.  In theory
// we could parallelize this quite easily by using a goroutine, however we might consume
// quite a bit of memory so we'll just keep it serialized for now.
func archiveHandler() {

	// Initialize the queue
	archiveIncoming := EventNew()

	// Read all archive IDs
	dataDir, _ := os.Open(configDataPath(""))
	archiveIDFiles, err := dataDir.ReadDir(0)
	dataDir.Close()
	if err == nil {
		for _, archiveIDFile := range archiveIDFiles {
			performArchive(archiveIDFile.Name())
		}
	}

	// Wait until something comes in
	archiveIncoming.Wait(time.Duration(1) * time.Hour)

}

// Process a single archive, by ID
func performArchive(archiveID string) {

	// Read all events pending for the archive
	dataDir, _ := os.Open(configDataPath(""))
	archiveEventFiles, err := dataDir.ReadDir(0)
	dataDir.Close()
	if err == nil {
		for _, archiveEventFile := range archiveEventFiles {
			performArchive(archiveEventFile.Name())
		}
	}

}
