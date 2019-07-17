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
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/marcusolsson/tui-go"
	"github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type state struct {
	sync.Mutex
	nc    *nats.Conn
	me    *jwt.UserClaims
	skp   nkeys.KeyPair
	name  string
	posts map[string][]*post
	dms   map[string]*user
	users map[string]*user
	cur   *selection
	ui    tui.UI

	// UI Items
	msgs     *tui.Grid
	channels *tui.List
	direct   *tui.List
}

type user struct {
	name  string
	nkey  string
	posts []*post
	last  time.Time
}

type post struct {
	user string
	msg  string
	time string
}

type pkind int

const (
	channel = pkind(iota)
	direct
)

type selection struct {
	index int
	name  string
	kind  pkind
}

// For testing
func (s *state) pre() {
	now := time.Now()
	s.posts["OSCON"] = []*post{
		{user: "derek", msg: "Hello Portland!", time: now.Add(-time.Hour).Format("15:04")},
		{user: "wally", msg: "Welcome to OSCON 2019", time: now.Format("15:04")},
	}
	s.posts["NATS"] = []*post{}
	s.posts["General"] = []*post{}
}

func newState() *state {
	s := &state{posts: make(map[string][]*post), dms: make(map[string]*user), users: make(map[string]*user)}
	s.pre()
	return s
}

func (s *state) selectFirstChannel() {
	s.channels.Select(0)
	s.setPostsDisplay(s.chSel())
}

const lpre = " - "

func dName(name string) string {
	return lpre + name
}

func sName(name string) string {
	return name[len(lpre):]
}

func (s *state) chSel() *selection {
	return &selection{
		index: s.channels.Selected(),
		name:  sName(s.channels.SelectedItem()),
		kind:  channel,
	}
}
func (s *state) dmSel() *selection {
	return &selection{
		index: s.direct.Selected(),
		name:  sName(s.direct.SelectedItem()),
		kind:  direct,
	}
}

func (s *state) addPostToCurrent(p *post) {
	switch s.cur.kind {
	case channel:
		s.posts[s.cur.name] = append(s.posts[s.cur.name], p)
	case direct:
		u := s.dms[s.cur.name]
		u.posts = append(u.posts, p)
	}
}

func (s *state) setPostsDisplay(sel *selection) {
	s.cur = sel
	s.msgs.RemoveRows()
	var posts []*post
	switch sel.kind {
	case channel:
		posts = s.posts[sel.name]
		s.direct.SetSelected(-1)
	case direct:
		if u := s.dms[sel.name]; u != nil {
			posts = u.posts
		}
		s.channels.SetSelected(-1)
	}
	for _, p := range posts {
		s.msgs.AppendRow(postEntry(p))
	}
}

func (s *state) sameChannel() bool {
	if s.cur == nil || s.cur.kind != channel || s.cur.index != s.channels.Selected() {
		return false
	}
	return true
}

func (s *state) sameDirect() bool {
	if s.cur == nil || s.cur.kind != direct || s.cur.index != s.direct.Selected() {
		return false
	}
	return true
}

func (s *state) addNewUser(name, nkey string) *user {
	u := &user{name, nkey, nil, time.Now()}
	s.users[nkey] = u

	du := s.dms[u.name]
	if du == nil {
		s.dms[u.name] = u
	} else {
		// We have a collision here. e.g. chose the same simple name.
		// Attempt to find a new one in form name(2), name(3).
		for i := 2; i < 10002; i++ {
			u.name = fmt.Sprintf("%s(%d)", name, i)
			du := s.dms[u.name]
			if du == nil {
				s.dms[u.name] = u
				return u
			}
		}
		log.Fatalf("Name collision error, alternatives exhausted")
	}
	return u
}
