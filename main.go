// Copyright 2022 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package main

// Main service entry point
func main() {

	// Init our archive task, which periodically files requests into folders.
	// Note that this must be initialized before HTTP handlers because of
	// event queue.
	go archiveHandler()

	// Init our web request server, which files requests in Incoming
	go HTTPInboundHandler(":80")

	// Handle console input
	inputHandler()

}
