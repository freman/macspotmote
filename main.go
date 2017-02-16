/*
 *   macspotmote - Control Spotify on OSX via a web interface
 *   Copyright (c) 2017 Shannon Wynter.
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/davecgh/go-spew/spew"
	"github.com/everdev/mack"
	"github.com/gorilla/websocket"

	log "github.com/Sirupsen/logrus"
)

const (
	seperator       = ":!|!:"
	refreshInterval = 250 * time.Millisecond

	writeWait  = 1 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var (
	version = "Undefined"
	commit  = "Undefined"
)

var status = struct {
	Volume     int
	State      string
	Repeat     bool
	Shuffle    bool
	Artist     string
	Name       string
	Artwork    string
	Position   float64
	Duration   int
	Popularity int
}{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func refreshStatus(command string) error {
	result, err := mack.Tell("Spotify", command, strings.Join([]string{
		"get sound volume",
		"player state as string",
		"repeating as string",
		"shuffling as string",
		"player position as string",
		"artist of current track as string",
		"name of current track as string",
		"artwork url of current track",
		"duration of current track as string",
		"popularity of current track as string",
	}, ` & "`+seperator+`" & `))

	if err != nil {
		spew.Dump(err)
		return err
	}
	fields := strings.Split(result, seperator)

	status.Volume, _ = strconv.Atoi(fields[0])
	status.State = strings.Title(fields[1])
	status.Repeat, _ = strconv.ParseBool(fields[2])
	status.Shuffle, _ = strconv.ParseBool(fields[3])
	status.Position, _ = strconv.ParseFloat(fields[4], 64)
	status.Artist = fields[5]
	status.Name = fields[6]
	status.Artwork = fields[7]
	status.Duration, _ = strconv.Atoi(fields[8])
	status.Popularity, _ = strconv.Atoi(fields[9])

	return nil
}

func setVolume(volume int) error {
	if volume > 100 {
		volume = 100
	}
	if volume < 0 {
		volume = 0
	}

	return refreshStatus(fmt.Sprintf("set sound volume to %d", volume))
}

func pause() error {
	return refreshStatus("pause")
}

func play() error {
	return refreshStatus("play")
}

func playPause() error {
	return refreshStatus("playpause")
}

func nextTrack() error {
	return refreshStatus("next track")
}

func prevTrack() error {
	return refreshStatus("previous track")
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, m, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if mt != websocket.TextMessage {
			continue
		}
		obj := struct {
			Action string
			Value  string
		}{}

		err = json.Unmarshal(m, &obj)
		if err != nil {
			continue
		}

		switch obj.Action {
		case "setVolume":
			vol, _ := strconv.Atoi(obj.Value)
			setVolume(vol)
		case "playPause":
			playPause()
		case "nextTrack":
			nextTrack()
		case "prevTrack":
			prevTrack()
		}

	}
}

func writer(ws *websocket.Conn) {
	updateTicker := time.NewTicker(refreshInterval)
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		updateTicker.Stop()
		ws.Close()
	}()

	bytes, _ := json.Marshal(status)
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := ws.WriteMessage(websocket.TextMessage, bytes); err != nil {
		return
	}

	for {
		select {
		case <-updateTicker.C:
			bytes, _ := json.Marshal(status)
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, bytes); err != nil {
				return
			}
		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	go writer(ws)
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	box, err := rice.FindBox("index")
	if err != nil {
		http.Error(w, "oops", http.StatusInternalServerError)
		return
	}
	f, err := box.Open("index.html")
	if err != nil {
		http.Error(w, "oops", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, f)
}

func main() {
	addr := flag.String("addr", ":8080", "http service address")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("esudp - %s (%s)\n", version, commit)
		fmt.Println("https://github.com/freman/macspotmote")
		return
	}

	refreshStatus("")
	go func() {
		ticker := time.NewTicker(refreshInterval)
		for range ticker.C {
			refreshStatus("")
		}
	}()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}
