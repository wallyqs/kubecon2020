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
	"time"

	"github.com/marcusolsson/tui-go"
)

func (s *state) setupUI() tui.UI {
	s.channels = tui.NewList()
	s.channels.AddItems(dName("OSCON"), dName("NATS"), dName("General"))

	s.direct = tui.NewList()
	for name, _ := range s.dms {
		s.direct.AddItems(dName(name))
	}

	sidebar := tui.NewVBox(
		tui.NewLabel(" CHANNELS"),
		s.channels,
		tui.NewLabel(""),
		tui.NewLabel(" DIRECT MESSAGES "),
		s.direct,
		tui.NewSpacer(),
	)
	sidebar.SetBorder(true)

	s.msgs = tui.NewGrid(4, 0)

	msgsScroll := tui.NewScrollArea(s.msgs)
	msgsScroll.SetAutoscrollToBottom(true)
	msgsBox := tui.NewVBox(msgsScroll)
	msgsBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(msgsBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		if m := e.Text(); m != "" {
			p := &post{user: s.name, msg: m, time: time.Now().Format("15:04")}
			s.Lock()
			defer s.Unlock()
			s.addPostToCurrent(p)
			s.sendPost(p)
			s.msgs.AppendRow(postEntry(p))
			e.SetText("")
		}
	})

	root := tui.NewHBox(sidebar, chat)

	ui, err := tui.New(root)
	if err != nil {
		log.Fatal(err)
	}

	input.SetFocused(true)

	s.channels.OnItemActivated(func(l *tui.List) {
		s.channels.SetFocused(false)
		input.SetFocused(true)
	})
	s.channels.OnSelectionChanged(func(l *tui.List) {
		s.Lock()
		defer s.Unlock()
		if s.sameChannel() {
			if s.cur.index > 0 {
				s.channels.SetFocused(false)
				s.direct.SetFocused(true)
			}
			return
		}
		if s.channels.Selected() >= 0 {
			s.setPostsDisplay(s.chSel())
			s.direct.SetSelected(-1)
		}
	})

	s.direct.OnItemActivated(func(l *tui.List) {
		s.direct.SetFocused(false)
		input.SetFocused(true)
	})
	s.direct.OnSelectionChanged(func(l *tui.List) {
		s.Lock()
		defer s.Unlock()
		if s.sameDirect() {
			if s.cur.index == 0 {
				s.direct.SetSelected(-1)
				s.direct.SetFocused(false)
				s.channels.SetFocused(true)
				s.channels.SetSelected(s.channels.Length() - 1)
				s.setPostsDisplay(s.chSel())
			}
			return
		}
		if s.direct.Selected() >= 0 {
			s.setPostsDisplay(s.dmSel())
			s.channels.SetSelected(-1)
		}
	})

	s.selectFirstChannel()

	// Navigation
	ui.SetKeybinding("TAB", func() {
		s.Lock()
		defer s.Unlock()
		if input.IsFocused() {
			input.SetFocused(false)
			if s.cur == nil || s.cur.kind == channel {
				s.direct.SetFocused(false)
				s.channels.SetFocused(true)
			} else {
				s.channels.SetFocused(false)
				s.direct.SetFocused(true)
			}
		} else {
			s.channels.SetFocused(false)
			s.direct.SetFocused(false)
			input.SetFocused(true)
		}
	})

	s.ui = ui
	return ui
}

func postUser(u string) string {
	return fmt.Sprintf("%-8s", "<"+u+">")
}
func postEntry(p *post) tui.Widget {
	return tui.NewHBox(
		tui.NewLabel(p.time),
		tui.NewPadder(1, 0, tui.NewLabel(postUser(p.user))),
		tui.NewLabel(p.msg),
		tui.NewSpacer(),
	)
}
