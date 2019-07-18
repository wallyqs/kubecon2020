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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

func usage() {
	log.Printf("Usage: chat-access [-s server] [-acc acc-jwt-file] [-sk signing-key-file]\n")
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

const (
	reqSubj    = "chat.req.access"
	reqGroup   = "oscon"
	maxNameLen = 8
)

func main() {
	var server = flag.String("s", "connect.ngs.global", "NATS System")
	var accFile = flag.String("acc", "", "Account JWT File")
	var skFile = flag.String("sk", "", "Account Signing Key")
	var appCreds = flag.String("creds", "", "App Credentials File")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *accFile == "" || *skFile == "" {
		showUsageAndExit(1)
	}

	opts := []nats.Option{nats.Name("OSCON Chat-Access")}
	opts = setupConnOptions(opts)
	if *appCreds != "" {
		opts = append(opts, nats.UserCredentials(*appCreds))
	}

	// Connect to NATS
	nc, err := nats.Connect(*server, opts...)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.LstdFlags)
	log.Print("Connected to NATS System")

	// Load account JWT and signing key
	acc, sk := loadAccountAndSigningKey(*accFile, *skFile)

	// Subscribe to Requests. QueueSubscriber means we can scale
	// up and down as needed.
	_, err = nc.QueueSubscribe(reqSubj, reqGroup, func(m *nats.Msg) {
		if len(m.Data) == 0 {
			m.Respond([]byte("-ERR 'Name can not be empty'"))
		}
		reqName := simpleName(m.Data)
		log.Printf("Registered %q [%q]\n", reqName, m.Data)
		creds := generateUserCreds(acc, sk, reqName)
		m.Respond([]byte(creds))
	})

	if err != nil {
		log.Fatal(err)
	}

	// Setup the interrupt handler to drain so we don't
	// drop requests when scaling down.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Println()
	log.Printf("Draining...")
	nc.Drain()
	log.Fatalf("Exiting")

}

// Some limits for our auto-provisioned users.
const (
	maxMsgSize = 1024
	validFor   = 24 * time.Hour

	// Should match ngs-chat versions.
	audience  = "OSCON-DEMO"
	preSub    = "chat.OSCON2019."
	onlineSub = preSub + "online"
	postsSub  = preSub + "posts.*"
	dmsPub    = preSub + "dms.*"
	dmsSub    = preSub + "dms.%s"
	inboxSub  = "_INBOX.>"
	usagePub  = "ngs.usage"

	credsT = `
-----BEGIN NGS CHAT DEMO USER JWT-----
%s
------END NGS CHAT DEMO USER JWT------

************************* IMPORTANT *************************
Private NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER PRIVATE KEY-----
%s
------END USER PRIVATE KEY------

*************************************************************
`
)

func createNewUserKeys() (string, []byte) {
	kp, _ := nkeys.CreateUser()
	pub, _ := kp.PublicKey()
	priv, _ := kp.Seed()
	return pub, priv
}

func generateUserCreds(acc *jwt.AccountClaims, akp nkeys.KeyPair, name string) string {
	pub, priv := createNewUserKeys()
	nuc := jwt.NewUserClaims(pub)
	nuc.Name = name
	nuc.Expires = time.Now().Add(validFor).Unix()
	nuc.Limits.Payload = maxMsgSize

	// Can listen for DMs, but only to ones to ourselves.
	pubAllow := jwt.StringList{onlineSub, postsSub, dmsPub, usagePub}
	subAllow := jwt.StringList{onlineSub, postsSub, fmt.Sprintf(dmsSub, pub), inboxSub}

	nuc.Permissions.Pub.Allow = pubAllow
	nuc.Permissions.Sub.Allow = subAllow

	nuc.IssuerAccount = acc.Subject

	ujwt, err := nuc.Encode(akp)
	if err != nil {
		log.Printf("Error generating user JWT: %v", err)
		return "-ERR 'Internal Error'"
	}
	return fmt.Sprintf(credsT, ujwt, priv)
}

// For demo, first name, max 8 chars and all lower case.
func simpleName(name []byte) string {
	reqName := string(name)
	reqName = strings.Split(strings.ToLower(reqName), " ")[0]
	if len(reqName) > maxNameLen {
		reqName = reqName[:maxNameLen]
	}
	return reqName
}

func loadAccountAndSigningKey(accFile, skFile string) (*jwt.AccountClaims, nkeys.KeyPair) {
	contents, err := ioutil.ReadFile(accFile)
	if err != nil {
		log.Fatalf("Could not load account file: %v", err)
	}
	acc, err := jwt.DecodeAccountClaims(string(contents))
	if err != nil {
		log.Fatalf("Could not decode account: %v", err)
	}
	seed, err := ioutil.ReadFile(skFile)
	if err != nil {
		log.Fatalf("Could not load signing key file: %v", err)
	}
	kp, err := nkeys.FromSeed(seed)
	if err != nil {
		log.Fatalf("Could not decode signing key: %v", err)
	}
	return acc, kp
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := 5 * time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Printf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatalf("Exiting: %v", nc.LastError())
	}))
	return opts
}
