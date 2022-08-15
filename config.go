// Copyright 2022 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ServiceConfig is the service configuration file format
type ServiceConfig struct {

	// Host URL
	HostURL string `json:"host_url,omitempty"`
}

// ConfigPath (here for golint)
const configFilePath = "/config/config.json"
const dataPath = "/data/"

// Config is our configuration, read out of a file for security reasons
var Config ServiceConfig

// Retrieve the data directory
func configDataPath(folder string) string {
	homedir, _ := os.UserHomeDir()
	path := homedir + dataPath
	if folder != "" {
		path += folder
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		os.MkdirAll(path, 0777)
	}
	return path
}

// configLoad gets the current value of the service config
func configLoad() {

	// Read the file and unmarshall if no error
	homedir, _ := os.UserHomeDir()
	path := homedir + configFilePath
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("can't load config from %s: %s\n", path, err)
		os.Exit(-1)
	}

	err = json.Unmarshal(contents, &Config)
	if err != nil {
		fmt.Printf("Can't parse config JSON from: %s: %s\n", path, err)
		os.Exit(-1)
	}

}
