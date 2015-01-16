package main

import (
	"encoding/json"
	"fmt"
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/github.com/elazarl/go-bindata-assetfs"
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/github.com/googollee/go-socket.io"
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/github.com/skratchdot/open-golang/open"
	"log"
	"net/http"
)

var (
	debug = false
)

//{"status": 200, "message": "ok", "data":
type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SocketIOResponse struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type PlaylistType struct {
	Playlist Playlist `json:"playlist"`
	Name     string   `json:"name"`
}

type PlaylistLengthType struct {
	Length int `json:"length"`
}

type TrackType struct {
	SpotifyTrackUri string `json:"spotify_track_uri"`
	Name            string `json:"name"`
	Cover           string `json:"cover"`
}

type AddedType struct {
	SpotifyTrackUri  string `json:"spotify_track_uri"`
	SpotifyTrackName string `json:"spotify_track_name"`
	Found            bool   `json:"found"`
	Karaoke          bool   `json:"karaoke"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Server struct {
	goog *Google
	sp   *Spotify
	sios *socketio.Server
	sio  socketio.Socket
}

func newServer() (*Server, error) {
	goog := NewGoogle()
	sp, err := NewSpotify()
	if err != nil {
		return nil, fmt.Errorf("Error initializting spotify: %s", err)
	}

	ioServer, err := socketio.NewServer(nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating socketio server: %s", err)
	}
	server := &Server{goog: goog, sp: sp, sios: ioServer}

	ioServer.On("connection", func(so socketio.Socket) {
		so.On("test", func(msg string) {
			fmt.Println(msg)
		})
		server.sio = so
	})
	ioServer.On("error", func(so socketio.Socket, err error) {
		fmt.Printf("socketio error: %s", err)
	})

	return server, nil
}

func main() {

	server, err := newServer()
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/socket.io/", server.sios)
	http.HandleFunc("/google/login", server.googleLogin)
	http.HandleFunc("/spotify/login", server.spotifyLogin)
	http.HandleFunc("/spotify/playlists", server.spotifyPlaylists)
	http.HandleFunc("/portify/transfer/start", server.transferStart)

	fs := http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "static"})
	http.Handle("/", fs)

	open.Run("http://localhost:3132/")
	panic(http.ListenAndServe(":3132", nil))
}

func (s *Server) googleLogin(w http.ResponseWriter, r *http.Request) {
	var response *Response
	var login LoginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&login)
	if err != nil {
		http.Error(w, "You must supply a login and password", http.StatusBadRequest)
		return
	}

	err = s.goog.Login(login.Email, login.Password)

	if err != nil {
		response = &Response{Status: 400, Message: "login failed."}
	} else {
		response = &Response{Status: 200, Message: "login successful."}
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) spotifyLogin(w http.ResponseWriter, r *http.Request) {
	var response *Response
	var login LoginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&login)
	if err != nil {
		http.Error(w, "You must supply a login and password", http.StatusBadRequest)
		return
	}

	err = s.sp.Login(login.Username, login.Password)
	if err != nil {
		response = &Response{Status: 400, Message: "login failed."}
	} else {
		response = &Response{Status: 200, Message: "login successful."}
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) spotifyPlaylists(w http.ResponseWriter, r *http.Request) {
	var response *Response
	if !s.sp.LoggedIn() {
		response = &Response{Status: 402, Message: "Spotify: not logged in"}
	} else {
		spPlaylists := s.sp.AllPlaylists()
		response = &Response{Status: 200, Message: "ok", Data: spPlaylists}
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) transferStart(w http.ResponseWriter, r *http.Request) {
	var response *Response

	var playlists []Playlist
	err := json.NewDecoder(r.Body).Decode(&playlists)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid playlists specified: %s", err), http.StatusBadRequest)
		return
	}

	if !s.goog.LoggedIn() {
		response = &Response{Status: 401, Message: "Google: not logged in."}
	} else if !s.sp.LoggedIn() {
		response = &Response{Status: 402, Message: "Spotify: not logged in"}
	} else if len(playlists) == 0 {
		response = &Response{Status: 403, Message: "Please select at least one playlist."}
	}

	if response == nil {
		go s.doTransferStart(playlists)
		response = &Response{Status: 200, Message: "transfer will start."}
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (s *Server) doTransferStart(playlists []Playlist) {
	// Convert to map to check for playlist
	playlistMap := make(map[string]bool)
	for _, playlist := range playlists {
		playlistMap[playlist.Uri] = true
	}

	// Iterate over all spotify playlists (should be cached anyway)
	spPlaylists := s.sp.AllPlaylists()
	for i, spPlaylist := range spPlaylists {
		trackChan, count := s.sp.PlaylistTracks(&spPlaylist)
		s.sio.Emit("portify", &SocketIOResponse{"playlist_length", PlaylistLengthType{count}})

		if _, ok := playlistMap[spPlaylist.Uri]; ok {
			s.sio.Emit("portify", &SocketIOResponse{"playlist_started", PlaylistType{spPlaylist, spPlaylist.Name}})
			err := s.createFullPlaylist(spPlaylist.Name, trackChan, count, i)
			s.sio.Emit("portify", &SocketIOResponse{"playlist_done", PlaylistType{spPlaylist, spPlaylist.Name}})
			if err != nil {
				fmt.Printf("Error creating playlist %s: %v", spPlaylist.Name, err)
			}
		}
	}

	s.sio.Emit("portify", &SocketIOResponse{"all_done", nil})
	fmt.Printf("Complete\n")

}

func (s *Server) createFullPlaylist(playlistName string, trackChan chan BasicTrack, trackCount int, playlistNum int) error {
	fmt.Printf("Processing playlist '%s'\n", playlistName)
	googSongNids := []string{}

	done := make(chan int)
	for i := 0; i < trackCount; i++ {
		go func(i int) {
			prefix := fmt.Sprintf("(%d:%d/%d)", playlistNum, i+1, trackCount)

			track := <-trackChan
			bestTrack, err := s.goog.FindBestTrack(track.Name)
			if err != nil {
				fmt.Printf("%s: Couldn't find track: %s\n", prefix, err)
				s.sio.Emit("gmusic", &SocketIOResponse{"not_added",
					AddedType{
						Found:            false,
						SpotifyTrackUri:  track.Uri,
						SpotifyTrackName: track.Name,
					},
				},
				)
			} else {
				googSongNids = append(googSongNids, bestTrack.Nid)
				fmt.Printf("%s: '%s' -> '%s - %s'\n", prefix, track.Name, bestTrack.Artist, bestTrack.Title)
				s.sio.Emit("gmusic", &SocketIOResponse{"added",
					AddedType{
						Found:            true,
						SpotifyTrackUri:  track.Uri,
						SpotifyTrackName: track.Name,
					},
				},
				)
			}
			done <- i
		}(i)
	}

	for i := 0; i < trackCount; i++ {
		<-done
	}

	fmt.Printf("Creating '%s' in Google Music\n", playlistName)
	playlistId, err := s.goog.CreatePlaylist(playlistName, false)
	if err != nil {
		return fmt.Errorf("Error creating playlist: %v", err)
	}
	s.goog.AddTracks(playlistId, googSongNids)
	if err != nil {
		return fmt.Errorf("Error adding tracks to playlist: %v", err)
	}
	return nil
}
