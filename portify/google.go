package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const SJURL = "https://mclients.googleapis.com/sj/v1.10/"
const LOGINURL = "https://www.google.com/accounts/ClientLogin"

type Google struct {
	client *http.Client
	auth   string
}

// A subset of the SearchResult track containing only the data we need
type RelevantTrack struct {
	Nid    string
	Artist string
	Title  string
}

func NewGoogle() *Google {
	client := &http.Client{}
	return &Google{client: client}
}

func (g *Google) LoggedIn() bool {
	return g.auth != ""
}

func (g *Google) Login(email string, password string) error {

	resp, err := g.client.PostForm(LOGINURL,
		url.Values{
			"Email":       {email},
			"Passwd":      {password},
			"accountType": {"HOSTED_OR_GOOGLE"},
			"source":      {"goportify"},
			"service":     {"sj"},
		})

	if err != nil {
		return fmt.Errorf("HTTP Error logging in: %s", err)
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("Couldn't login")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Error reading login body: %s", err)
	}

	authResponse, err := parseAuthResponse(string(body))
	if err != nil {
		return fmt.Errorf("Error parsing auth response: %s", err)
	}
	if authResponse["Auth"] == "" {
		return fmt.Errorf("Didn't receive an auth response")
	}

	g.auth = authResponse["Auth"]

	return nil
}

func (g *Google) Search(query string, maxResults int) (*SearchResult, error) {
	url := fmt.Sprintf("%squery?q=%s&max-items=%d", SJURL, url.QueryEscape(query), maxResults)
	// url := SJURL + "query?q=Katy%20perry&max-items=2"
	body, err := g.execute("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (g *Google) FindBestTrack(query string) (*RelevantTrack, error) {
	sResult, err := g.Search(query, 2)
	if err != nil {
		return nil, fmt.Errorf("Couldn't execute search: %s\n", err)
	}

	for _, entry := range sResult.Entries {
		// Filter only tracks
		if entry.Type == "1" {
			return &RelevantTrack{
				Nid:    entry.Track.Nid,
				Artist: entry.Track.Artist,
				Title:  entry.Track.Title,
			}, nil
		}
	}
	return nil, fmt.Errorf("No tracks for %s", query)
}

func (g *Google) CreatePlaylist(name string, public bool) (string, error) {
	mutations := buildCreatePlaylist(name, public)
	content := &DataPlaylistItem{mutations}

	body, err := g.execute("POST", SJURL+"playlistbatch?alt=json", content)
	if err != nil {
		return "", fmt.Errorf("Couldn't execute playlistbatch: %v", err)
	}

	var response MutateResponseContainer
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("Unable to unmarshal json: %v", err)
	}
	return response.MutateResponse[0].ID, nil
	// return res['mutate_response'][0]['id']
}

func (g *Google) AddTracks(playlistId string, songIds []string) error {
	mutations := buildAddTracks(playlistId, songIds...)
	content := &DataTrackItem{mutations}

	_, err := g.execute("POST", SJURL+"plentriesbatch?alt=json", content)
	if err != nil {
		return fmt.Errorf("Couldn't execute http query: %v", err)
	}
	return nil
}

func (g *Google) execute(method string, url string, content interface{}) ([]byte, error) {
	var req *http.Request
	var err error

	if method == "POST" {
		bytes, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("Error marshalling track post data: %v", err)
		}
		jsonContent := string(bytes)
		postContent := strings.NewReader(jsonContent)
		// fmt.Printf("Executing %s request to %s with content:\n %s\n", method, url, jsonContent)
		req, err = http.NewRequest(method, url, postContent)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("Error creating track post request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("GoogleLogin auth=%s", g.auth))
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error posting batch: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Returned status code %s", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading login body: %s", err)
	}

	return body, nil
}
