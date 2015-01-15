package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// My excuse is that portify stored it in global state too
var (
	debug = false
	sp *Spotify
	goog *Google
)

//{"status": 200, "message": "ok", "data":
type Response struct {
	Status int `json:"status"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}

type LoginRequest struct {
	Email string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
  

	http.HandleFunc("/google/login", googleLogin)
	http.HandleFunc("/spotify/login", spotifyLogin)
	http.HandleFunc("/spotify/playlists", spotifyPlaylists)
	http.HandleFunc("/portify/transfer/start", transferStart)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	var err error
	goog = NewGoogle()
	sp, err = NewSpotify()
	if err != nil {
		log.Fatal("Error initializing spotify: %v", err)
	}

	fmt.Printf("Starting server on port 3132\n")
  	panic(http.ListenAndServe(":3132", nil))
}

func googleLogin(w http.ResponseWriter, r *http.Request) {
	var response *Response
	var login LoginRequest
  	decoder := json.NewDecoder(r.Body)
  	err := decoder.Decode(&login)
  	if err != nil {
		http.Error(w, "You must supply a login and password", http.StatusBadRequest)
		return  		
  	}

  	err = goog.Login(login.Email, login.Password)
  	
  	if err != nil {
	  	response = &Response{ Status: 400, Message: "login failed.", }  		
  	} else {
  		response = &Response{ Status: 200, Message: "login successful.", } 
  	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func spotifyLogin(w http.ResponseWriter, r *http.Request) {
	var response *Response
	var login LoginRequest
  	decoder := json.NewDecoder(r.Body)
  	err := decoder.Decode(&login)
  	if err != nil {
		http.Error(w, "You must supply a login and password", http.StatusBadRequest)
		return  		
  	}

  	err = sp.Login(login.Username, login.Password)
  	if err != nil {
	  	response = &Response{ Status: 400, Message: "login failed.", }  		
  	} else {
  		response = &Response{ Status: 200, Message: "login successful.", } 
  	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func spotifyPlaylists(w http.ResponseWriter, r *http.Request) {
	var response *Response
	if !sp.LoggedIn() {
		response = &Response{ Status: 402, Message: "Spotify: not logged in", }
	} else {
	  	spPlaylists := sp.AllPlaylists()
	  	response = &Response{ Status: 200, Message: "ok", Data: spPlaylists }
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func transferStart(w http.ResponseWriter, r *http.Request) {
	var response *Response

	var playlists []Playlist
  	err := json.NewDecoder(r.Body).Decode(&playlists)
  	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid playlists specified: %s", err), http.StatusBadRequest)
		return  		
  	}

  	if !goog.LoggedIn() {
		response = &Response{ Status: 401, Message: "Google: not logged in.", }
  	} else if !sp.LoggedIn() {
  		response = &Response{ Status: 402, Message: "Spotify: not logged in", }
  	} else if len(playlists) == 0 {
  		response = &Response{ Status: 403, Message: "Please select at least one playlist.", }
  	}

  	if response == nil {
	  	go doTransferStart(playlists)
  		response = &Response{ Status: 200, Message: "transfer will start.", } 
  	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func doTransferStart(playlists []Playlist) {
	// Convert to map to check for playlist
	playlistMap := make(map[string]bool)
	for _, playlist := range playlists {
		playlistMap[playlist.Uri] = true
	}

	// Iterate over all spotify playlists (should be cached anyway)
	spPlaylists := sp.AllPlaylists()
	done := make(chan int)
	playlistCount := 0
	for _, spPlaylist := range spPlaylists {
		trackChan, count := sp.PlaylistTracks(&spPlaylist)

		if _, ok := playlistMap[spPlaylist.Uri]; ok {
			playlistCount += 1
			go func(i int, name string) {
				err := goog.CreateFullPlaylist(name, trackChan, count, i)
				if err != nil {
					fmt.Printf("Error creating playlist %s: %v", name, err)
				}
				done <- i
			}(playlistCount, spPlaylist.Name)
		}
	}

	for i := 0; i < playlistCount; i++ {
		<- done
	}
	fmt.Printf("Complete\n")

}
