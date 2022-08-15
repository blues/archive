// Copyright 2020 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/blues/note-go/note"
)

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

	archiveID, exists := headerField(r, "archive_id")
	if !exists {
		writeErr(w, "archive_id not specified")
		return
	}
	s, _ := headerField(r, "archive_mins")
	archiveMins, _ := strconv.Atoi(s)
	if archiveMins <= 0 {
		archiveMins = 1440
	}

	bucketEndpoint, _ := headerField(r, "bucket_endpoint")

	bucketName, exists := headerField(r, "bucket_name")
	if !exists {
		writeErr(w, "bucket_name not specified")
		return
	}

	bucketRegion, exists := headerField(r, "bucket_region")
	if !exists {
		writeErr(w, "bucket_region not specified")
		return
	}

	fileAccess, exists := headerField(r, "file_access")
	if !exists {
		writeErr(w, "file_access not specified")
		return
	}

	fileFormat, exists := headerField(r, "file_format")
	if !exists {
		fileFormat = "[id]/[year]-[month]/[device]/[when]"
	}

	fileName, exists := headerField(r, "file_name")
	if !exists {
		writeErr(w, "file_name not specified")
		return
	}

	keyID, exists := headerField(r, "key_id")
	if !exists {
		writeErr(w, "key_id not specified")
		return
	}
	keySecret, exists := headerField(r, "key_secret")
	if !exists {
		writeErr(w, "key_secret not specified")
		return
	}

	// Debug
	fmt.Printf("archiveID:%s archiveMins:%d bucketEndpoint:%s bucketName:%s bucketRegion:%s fileAccess:%s fileFormat:%s fileName:%s keyID:%s keySecret:%s\n%s\n\n", archiveID, archiveMins, bucketEndpoint, bucketName, bucketRegion, fileAccess, fileFormat, fileName, keyID, keySecret, string(eventJSON))

	// Done
	w.Write([]byte("I'm watching you."))

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
