// Copyright 2022 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

import (
	"os"
	"strings"
)

// ConfigPath (here for golint)
const dataPath = "/data/"

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
