// Copyright 2020 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/blues/note-go/note"
	"github.com/google/uuid"
)

// File folders/names
const instanceRouteConfigFile = "route.json"
const instanceIncomingEvents = "incoming/"

// Configuration object
type RouteConfig struct {
	ArchiveID      string `json:"archive_id"`
	ArchiveMins    int    `json:"archive_mins"`
	BucketEndpoint string `json:"bucket_endpoint"`
	BucketName     string `json:"bucket_name"`
	BucketRegion   string `json:"bucket_region"`
	FileAccess     string `json:"file_access"`
	FileFormat     string `json:"file_format"`
	FileName       string `json:"file_name"`
	KeyID          string `json:"key_id"`
	KeySecret      string `json:"key_secret"`
}

// Root handler
func inboundWebRootHandler(w http.ResponseWriter, r *http.Request) {

	// Get parameters from the request header, validating as we go along
	parsedURL, _ := url.Parse(r.RequestURI)
	target := path.Base(parsedURL.Path)
	if target == "favicon.ico" {
		return
	}

	eventJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeErr(w, err.Error())
		return
	}
	if len(eventJSON) == 0 {
		writeErr(w, "event is blank")
		return
	}
	var event note.Event
	err = note.JSONUnmarshal(eventJSON, &event)
	if err != nil {
		writeErr(w, err.Error())
		return
	}

	var exists bool
	var rc RouteConfig
	rc.ArchiveID, exists = headerField(r, "archive_id")
	if !exists {
		writeErr(w, "archive_id not specified")
		return
	}
	s, _ := headerField(r, "archive_mins")
	rc.ArchiveMins, _ = strconv.Atoi(s)
	if rc.ArchiveMins <= 0 {
		rc.ArchiveMins = 1440
	}

	rc.BucketEndpoint, _ = headerField(r, "bucket_endpoint")

	rc.BucketName, exists = headerField(r, "bucket_name")
	if !exists {
		writeErr(w, "bucket_name not specified")
		return
	}

	rc.BucketRegion, exists = headerField(r, "bucket_region")
	if !exists {
		writeErr(w, "bucket_region not specified")
		return
	}

	rc.FileAccess, exists = headerField(r, "file_access")
	if !exists {
		writeErr(w, "file_access not specified")
		return
	}

	rc.FileFormat, exists = headerField(r, "file_format")
	if !exists {
		rc.FileFormat = "[id]/[year]-[month]/[device]/[when]"
	}

	rc.FileName, exists = headerField(r, "file_name")
	if !exists {
		writeErr(w, "file_name not specified")
		return
	}

	rc.KeyID, exists = headerField(r, "key_id")
	if !exists {
		writeErr(w, "key_id not specified")
		return
	}
	rc.KeySecret, exists = headerField(r, "key_secret")
	if !exists {
		writeErr(w, "key_secret not specified")
		return
	}

	// Atomically write configuration to a config file
	rcJSON, err := note.JSONMarshal(rc)
	if err != nil {
		fmt.Printf("error marshaling route config: %s\n", err)
	} else {
		tempFile := uuid.New().String() + ".temp"
		tempPath := configDataPath(rc.ArchiveID) + tempFile
		err := os.WriteFile(tempPath, rcJSON, 0644)
		if err != nil {
			fmt.Printf("error writing route config to %s: %s\n", tempPath, err)
		} else {
			filePath := configDataPath(rc.ArchiveID) + instanceRouteConfigFile
			err = os.Rename(tempPath, filePath)
			if err != nil {
				fmt.Printf("error renaming %s to %s\n", tempPath, filePath)
			}
		}
	}

	// Write the event in an atomic way
	filePath := fmt.Sprintf("%s%s%d", configDataPath(rc.ArchiveID), instanceIncomingEvents, time.Now().UnixNano())
	err = os.WriteFile(filePath, eventJSON, 0644)
	if err != nil {
		fmt.Printf("error writing %s: %s\n", filePath, err)
	}

	// Done
	w.Write([]byte("{}"))

}

// Clean comments out of the specified field
func headerField(r *http.Request, fieldName string) (out string, exists bool) {
	s1 := r.Header.Get(fieldName)
	s2 := strings.TrimSpace(strings.Split(s1, " ")[0])
	return s2, s2 != ""
}

// Write an error message as a JSON object
func writeErr(w http.ResponseWriter, message string) {
	w.Write([]byte(fmt.Sprintf("{\"err\":\"%s\"}", message)))
}
