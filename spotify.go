package main

import (
	"fmt"
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/github.com/op/go-libspotify/spotify"
	"os"
	"time"
)

type Spotify struct {
	session  *spotify.Session
	running  bool
	login    chan error
	loggedIn bool
}

type Playlist struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

type BasicTrack struct {
	Uri  string `json:"uri"`
	Name string `json:"name"`
}

func NewSpotify() (*Spotify, error) {
	appKey, err := Asset("spotify_appkey.key")
	// appKey, err := ioutil.ReadFile("spotify_appkey.key")
	if err != nil {
		return nil, fmt.Errorf("You must have spotify_appkey.key")
	}

	session, err := spotify.NewSession(&spotify.Config{
		ApplicationKey:   appKey,
		ApplicationName:  "goportify",
		CacheLocation:    "tmp",
		SettingsLocation: "tmp",
	})

	sp := &Spotify{
		session,
		true,
		make(chan error),
		false,
	}
	go sp.loop()
	if err != nil {
		return nil, fmt.Errorf("Error creating session: %v", err)
	}
	return sp, nil
}

func (sp *Spotify) loop() {
	session := sp.session
	for sp.running {
		if debug {
			println("waiting for connection state change", session.ConnectionState())
		}

		select {
		case err := <-session.LoggedInUpdates():
			sp.login <- err
		case <-session.LoggedOutUpdates():
			if debug {
				println("!! logout updated")
			}
		case err := <-session.ConnectionErrorUpdates():
			if debug {
				println("!! connection error", err.Error())
			}
		case msg := <-session.MessagesToUser():
			println("!! message to user", msg)
		case message := <-session.LogMessages():
			if debug {
				println("!! log message", message.String())
			}
		case _ = <-session.CredentialsBlobUpdates():
			if debug {
				println("!! blob updated")
			}
		case <-session.ConnectionStateUpdates():
			if debug {
				println("!! connstate", session.ConnectionState())
			}
		case <-time.After(5 * time.Second):
			if debug {
				println("state change timeout")
			}
		}
	}

	session.Close()
	os.Exit(32)
}

func (sp *Spotify) LoggedIn() bool {
	return sp.loggedIn
}

func (sp *Spotify) Login(username string, password string) error {
	creds := spotify.Credentials{
		Username: username,
		Password: password,
	}
	sp.session.Login(creds, true)
	select {
	case l := <-sp.login:
		sp.loggedIn = true
		return l
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Timeout")
	}
}

func (sp *Spotify) AllPlaylists() []Playlist {
	playlistContainer, err := sp.session.Playlists()
	if err != nil {
		fmt.Printf("Couldn't get playlist container: %v", err)
	}
	playlistContainer.Wait()

	playlists := []Playlist{Playlist{ "starred", "Starred Tracks", },}

	for i := 0; i < playlistContainer.Playlists(); i++ {
		switch playlistContainer.PlaylistType(i) {
		case spotify.PlaylistTypeStartFolder:
		case spotify.PlaylistTypeEndFolder:
			// TODO: ?
			continue
		case spotify.PlaylistTypePlaylist:
			playlist := playlistContainer.Playlist(i)
			playlist.Wait()
			p := Playlist{
				playlist.Link().String(),
				playlist.Name(),
			}
			playlists = append(playlists, p)
		}
	}

	return playlists
}

func (sp *Spotify) PlaylistTracks(wantedPlaylist *Playlist) (chan BasicTrack, int) {

	playlistContainer, err := sp.session.Playlists()
	if err != nil {
		fmt.Printf("Couldn't get playlist container: %v", err)
	}
	playlistContainer.Wait()
	var selectedPlaylist *spotify.Playlist
	if wantedPlaylist.Uri == "starred" {
		selectedPlaylist = sp.session.Starred()
	} else {
		for i := 0; i < playlistContainer.Playlists(); i++ {
			switch playlistContainer.PlaylistType(i) {
			case spotify.PlaylistTypeStartFolder:
			case spotify.PlaylistTypeEndFolder:
				// TODO: ?
				continue
			case spotify.PlaylistTypePlaylist:
				playlist := playlistContainer.Playlist(i)
				playlist.Wait()
				if playlist.Link().String() == wantedPlaylist.Uri {
					selectedPlaylist = playlist
					break
				}
			}
		}
	}

	ret := make(chan BasicTrack)
	go func() {
		for j := 0; j < selectedPlaylist.Tracks(); j++ {
			track := selectedPlaylist.Track(j).Track()
			track.Wait()
			artist := track.Artist(0)
			artist.Wait()
			ret <- BasicTrack{
				Uri:  track.Link().String(),
				Name: fmt.Sprintf("%s - %s", artist.Name(), track.Name()),
			}
		}
		close(ret)
	}()
	return ret, selectedPlaylist.Tracks()
}
