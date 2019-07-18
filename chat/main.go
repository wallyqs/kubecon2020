// Copyright 2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func usage() {
	log.Printf("Usage: chat [-s server] [-creds file] [-n name]\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	var server = flag.String("s", "connect.ngs.global", "NATS System")
	var name = flag.String("n", "", "Override Chat Name")
	var userCreds = flag.String("creds", "", "User Credentials File")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	// Use UserCredentials
	if *userCreds == "" {
		showUsageAndExit(1)
	}

	// Initialize our state
	s := newState(*userCreds)

	// Connect to NATS system
	log.Print("Connecting to NATS system")
	opts := []nats.Option{nats.Name("OSCON Chat")}
	opts = setupConnOptions(opts)
	opts = append(opts, nats.UserCredentials(*userCreds))

	// Connect to NATS
	nc, err := nats.Connect(*server, opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Setup NATS and announce ourselves.
	s.setupNATS(nc, *userCreds, *name)

	// Setup terminal UI
	ui := s.setupUI()

	// Ctrl-C to exit.
	ui.SetKeybinding("Ctrl+C", func() { ui.Quit() })

	// Setup expiration timer if the user expires.
	if s.me.Expires > 0 {
		expiresInSecs := time.Duration(s.me.Expires - time.Now().Unix())
		time.AfterFunc(time.Second*expiresInSecs, func() {
			ui.Quit()
			log.Fatalf("Your credentials have expired.")
		})
	}

	// Loop on UI.
	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
