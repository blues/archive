// Copyright 2020 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/blues/note-go/note"
	"github.com/google/uuid"
)

// Event indicating that something happened
var archiveIncoming *Event

// Handler that performs archiving in a way that's serialized on the archive.  In theory
// we could parallelize this quite easily by using a goroutine, however we might consume
// quite a bit of memory so we'll just keep it serialized for now.
func archiveHandler() {

	// Initialize the queue
	archiveIncoming = EventNew()

	// Loop, performing archives
	for {

		// Read all archive IDs
		dataDir, _ := os.Open(configDataPath(""))
		archiveIDFiles, err := dataDir.ReadDir(0)
		dataDir.Close()
		if err != nil {
			fmt.Printf("data directory read error: %s\n", err)
		} else {
			for _, archiveIDFile := range archiveIDFiles {
				performArchive(archiveIDFile.Name())
			}
		}

		// Wait until something comes in
		archiveIncoming.Wait(time.Duration(1) * time.Hour)

	}

}

// Process a single archive, by ID
func performArchive(archiveID string) {
	var rc RouteConfig

	// First, to save memory because file descriptors are large, gather directory
	// entries incrementally as a string array, and then sort the array.
	dataDir, err := os.Open(configDataPath(archiveID + instanceIncomingEvents))
	if err != nil {
		fmt.Printf("can't open incoming events for %s: %s\n", archiveID, err)
		return
	}
	filenames := []string{}
	for {
		files, err := dataDir.ReadDir(1)
		if err != nil {
			break
		}
		for _, file := range files {
			filename := file.Name()
			if !strings.HasPrefix(filename, ".") {
				filenames = append(filenames, filename)
			}
		}
	}
	dataDir.Close()
	sort.Strings(filenames)
	if len(filenames) == 0 {
		return
	}

	// Append a special filename to ensure that we terminate cleanly
	filenames = append(filenames, "completed -1")

	// Next, iterate over the sorted filenames
	prevFolder := ""
	prevFiles := []string{}
	prevTime := int64(0)
	lastTime := int64(0)

	// Accumulate filenames for a given folder
	for _, filename := range filenames {

		// Parse the filename into folder and time
		index := strings.LastIndex(filename, " ")
		if index == -1 {
			continue
		}
		thisFolder := filename[:index]
		thisTime, _ := strconv.ParseInt(filename[index+1:], 10, 0)
		if thisTime == 0 {
			continue
		}

		// Special case for the first folder
		if prevFolder == "" {
			prevFolder = thisFolder
			prevTime = thisTime
			lastTime = thisTime
		}

		// If this is the same as the previous folder, just add the filename to the list
		if prevFolder == thisFolder {
			prevFiles = append(prevFiles, configDataPath(archiveID+instanceIncomingEvents)+filename)
			lastTime = thisTime
			continue
		}

		// Read the route config if it hasn't yet been read
		if rc.ArchiveID == "" {
			rcJSON, err := os.ReadFile(configDataPath(archiveID) + instanceRouteConfigFile)
			if err != nil {
				fmt.Printf("can't read %s config file: %s\n", archiveID, err)
				continue
			}
			err = note.JSONUnmarshal(rcJSON, &rc)
			if err != nil {
				continue
			}
		}

		// If the time has expired OR the count is excessive, do it
		nowUs := time.Now().UnixNano() / 1000
		elapsedMins := ((nowUs - prevTime) / 1000000) / 60
		if elapsedMins < int64(rc.ArchiveEveryMins) && len(prevFiles) < rc.ArchiveCountExceeds {
			fmt.Printf("archive: %s folder '%s' is %d mins old and has %d events (will archive at %d mins or %d events)\n",
				rc.ArchiveID, prevFolder, elapsedMins, len(prevFiles), rc.ArchiveEveryMins, rc.ArchiveCountExceeds)
			continue
		}

		// Upload the archive, and either set or delete the error file
		archiveBucketKey := fmt.Sprintf("%s/%d-%d-%d.json", strings.ReplaceAll(prevFolder, " ", "/"), prevTime, lastTime, len(prevFiles))
		err = uploadArchive(rc, archiveBucketKey, prevFiles)
		errFilePath := configDataPath(rc.ArchiveID) + instanceRouteErrorFile
		if err != nil {
			fmt.Printf("error uploading to %s: %s\n", rc.ArchiveID, err)
			errBytes := []byte(err.Error())
			tempFile := uuid.New().String() + ".temp"
			tempPath := configDataPath(rc.ArchiveID) + tempFile
			err = os.WriteFile(tempPath, errBytes, 0644)
			if err == nil {
				os.Rename(tempPath, errFilePath)
			}
		} else {

			// Remove the error file
			os.Remove(errFilePath)

			// Remove the successfully-archived files
			for _, filepath := range prevFiles {
				os.Remove(filepath)
			}

			fmt.Printf("archive: %s folder '%s' (%d events) archived\n", rc.ArchiveID, prevFolder, len(prevFiles))

		}

		// Move on to the next folder
		if thisTime == -1 {
			break
		}
		prevFolder = thisFolder
		prevTime = thisTime
		lastTime = int64(0)
		prevFiles = []string{filename}

	}

}

// Upload an archive
func uploadArchive(rc RouteConfig, bucketKey string, filepaths []string) (err error) {

	// Generate the archive object in-memory
	useArray := false
	outArray := []interface{}{}
	useMap := false
	outMap := map[string][]interface{}{}
	outBytes := []byte{}
	for _, filepath := range filepaths {
		eventJSON, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("error reading %s: %s\n", filepath, err)
			continue
		}
		var event map[string]interface{}
		err = note.JSONUnmarshal(eventJSON, &event)
		if err != nil {
			fmt.Printf("error unmarshaling %s: %s\n", filepath, err)
			continue
		}

		// Generate the structure based upon array format
		if rc.FileFormat == "array" {
			useArray = true
			outArray = append(outArray, event)
		} else if strings.HasPrefix(rc.FileFormat, "object:") {
			useMap = true
			fieldName := strings.TrimPrefix(rc.FileFormat, "object:")
			outMap[fieldName] = append(outMap[fieldName], event)
		} else if rc.FileFormat == "ndjson" {
			outBytes = append(outBytes, eventJSON...)
			outBytes = append(outBytes, []byte("\n")...)
		} else {
			return fmt.Errorf("invalid file format: %s", rc.FileFormat)
		}

	}

	// Unpack into bytes if appropriate
	if useArray {
		outBytes, err = note.JSONMarshal(outArray)
		if err != nil {
			return fmt.Errorf("can't marshal array: %s", err)
		}
	} else if useMap {
		outBytes, err = note.JSONMarshal(outMap)
		if err != nil {
			return fmt.Errorf("can't marshal map: %s", err)
		}
	}

	// Upload bytes to S3
	bucket := aws.String(rc.BucketName)
	key := aws.String(bucketKey)
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(rc.KeyID, rc.KeySecret, ""),
		Endpoint:         aws.String(rc.BucketEndpoint),
		Region:           aws.String(rc.BucketRegion),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return fmt.Errorf("error creating session: %s", err)
	}
	s3Client := s3.New(newSession)
	cparams := &s3.CreateBucketInput{
		Bucket: bucket,
	}
	s3Client.CreateBucket(cparams)
	puparams := &s3.PutObjectInput{
		Body:   strings.NewReader(string(outBytes)),
		Bucket: bucket,
		Key:    key,
	}
	_, err = s3Client.PutObject(puparams)
	if err != nil {
		return fmt.Errorf("err uploading object: %s", err)
	}

	// Done
	return

}
