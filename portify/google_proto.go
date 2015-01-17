package main

import (
	"github.com/rckclmbr/goportify/Godeps/_workspace/src/code.google.com/p/go-uuid/uuid"
	"strings"
)

type DataPlaylistItem struct {
	Mutations []MutationPlaylistItem `json:"mutations"`
}

type DataTrackItem struct {
	Mutations []MutationTrackItem `json:"mutations"`
}

type MutationTrackItem struct {
	Create CreateTrackItem `json:"create"`
}

type MutationPlaylistItem struct {
	Create CreatePlaylistItem `json:"create"`
}

type CreateTrackItem struct {
	ClientId              string `json:"clientId"`
	CreationTimestamp     string `json:"creationTimestamp"`
	Deleted               bool   `json:"deleted"`
	LastModifiedTimestamp string `json:"lastModifiedTimestamp"`
	PlaylistId            string `json:"playlistId"`
	Source                int    `json:"source"`
	TrackId               string `json:"trackId"`
	PrecedingEntryId      string `json:"precendingEntryId"`
	FollowingEntryId      string `json:"followingEntryId"`
}

type CreatePlaylistItem struct {
	CreationTimestamp     string `json:"creationTimestamp"`
	Deleted               bool   `json:"deleted"`
	LastModifiedTimestamp string `json:"lastModifiedTimestamp"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	AccessControlled      bool   `json:"accessControlled"`
}

type SearchResult struct {
	Entries []struct {
		Album struct {
			AlbumArtRef            string   `json:"albumArtRef"`
			AlbumArtist            string   `json:"albumArtist"`
			AlbumId                string   `json:"albumId"`
			Artist                 string   `json:"artist"`
			ArtistId               []string `json:"artistId"`
			DescriptionAttribution struct {
				Kind         string `json:"kind"`
				LicenseTitle string `json:"license_title"`
				LicenseURL   string `json:"license_url"`
				SourceTitle  string `json:"source_title"`
				SourceURL    string `json:"source_url"`
			} `json:"description_attribution"`
			Kind string `json:"kind"`
			Name string `json:"name"`
			Year int    `json:"year"`
		} `json:"album"`
		Artist struct {
			ArtistArtRef         string `json:"artistArtRef"`
			ArtistId             string `json:"artistId"`
			ArtistBioAttribution struct {
				Kind         string `json:"kind"`
				LicenseTitle string `json:"license_title"`
				LicenseURL   string `json:"license_url"`
				SourceTitle  string `json:"source_title"`
				SourceURL    string `json:"source_url"`
			} `json:"artist_bio_attribution"`
			Kind string `json:"kind"`
			Name string `json:"name"`
		} `json:"artist"`
		BestResult             bool    `json:"best_result"`
		NavigationalConfidence float64 `json:"navigational_confidence"`
		NavigationalResult     bool    `json:"navigational_result"`
		Playlist               struct {
			AlbumArtRef []struct {
				URL string `json:"url"`
			} `json:"albumArtRef"`
			Description          string `json:"description"`
			Kind                 string `json:"kind"`
			Name                 string `json:"name"`
			OwnerName            string `json:"ownerName"`
			OwnerProfilePhotoUrl string `json:"ownerProfilePhotoUrl"`
			ShareToken           string `json:"shareToken"`
			Type                 string `json:"type"`
		} `json:"playlist"`
		Score float64 `json:"score"`
		Track struct {
			Album       string `json:"album"`
			AlbumArtRef []struct {
				URL string `json:"url"`
			} `json:"albumArtRef"`
			AlbumArtist               string   `json:"albumArtist"`
			AlbumAvailableForPurchase bool     `json:"albumAvailableForPurchase"`
			AlbumId                   string   `json:"albumId"`
			Artist                    string   `json:"artist"`
			ArtistId                  []string `json:"artistId"`
			Composer                  string   `json:"composer"`
			ContentType               string   `json:"contentType"`
			DiscNumber                int      `json:"discNumber"`
			DurationMillis            string   `json:"durationMillis"`
			EstimatedSize             string   `json:"estimatedSize"`
			Genre                     string   `json:"genre"`
			Kind                      string   `json:"kind"`
			Nid                       string   `json:"nid"`
			PlayCount                 int      `json:"playCount"`
			PrimaryVideo              struct {
				ID         string `json:"id"`
				Kind       string `json:"kind"`
				Thumbnails []struct {
					Height int    `json:"height"`
					URL    string `json:"url"`
					Width  int    `json:"width"`
				} `json:"thumbnails"`
			} `json:"primaryVideo"`
			StoreId                       string `json:"storeId"`
			Title                         string `json:"title"`
			TrackAvailableForPurchase     bool   `json:"trackAvailableForPurchase"`
			TrackAvailableForSubscription bool   `json:"trackAvailableForSubscription"`
			TrackNumber                   int    `json:"trackNumber"`
			TrackType                     string `json:"trackType"`
			Year                          int    `json:"year"`
		} `json:"track"`
		Type string `json:"type"`
	} `json:"entries"`
	Kind string `json:"kind"`
}

type MutateResponseContainer struct {
	MutateResponse []struct {
		ClientID     string `json:"client_id"`
		ID           string `json:"id"`
		ResponseCode string `json:"response_code"`
	} `json:"mutate_response"`
}

func buildAddTracks(playlistId string, songIds ...string) []MutationTrackItem {
	mutations := make([]MutationTrackItem, len(songIds))

	prevId := ""
	curId := uuid.New()
	nextId := uuid.New()

	for i, songId := range songIds {
		details := MutationTrackItem{
			Create: CreateTrackItem{
				ClientId:              curId,
				CreationTimestamp:     "-1",
				Deleted:               false,
				LastModifiedTimestamp: "0",
				PlaylistId:            playlistId,
				Source:                1,
				TrackId:               songId,
			},
		}
		if strings.HasPrefix(songId, "T") {
			details.Create.Source = 2 // AA track
		}

		if i > 0 {
			details.Create.PrecedingEntryId = prevId
		}
		if i < len(songIds)-1 {
			details.Create.FollowingEntryId = nextId
		}

		mutations[i] = details

		prevId = curId
		curId = nextId
		nextId = uuid.New()
	}
	return mutations
}

func buildCreatePlaylist(name string, public bool) []MutationPlaylistItem {
	mutations := make([]MutationPlaylistItem, 1)
	mutations[0] = MutationPlaylistItem{
		Create: CreatePlaylistItem{
			CreationTimestamp:     "-1",
			Deleted:               false,
			LastModifiedTimestamp: "0",
			Name:             name,
			Type:             "USER_GENERATED",
			AccessControlled: public,
		},
	}
	return mutations
}

// Parses the SID, LSID, and Auth of a login
func parseAuthResponse(response string) (map[string]string, error) {
	//   SID=DQAAAGgA...7Zg8CTN
	//   LSID=DQAAAGsA...lk8BBbG
	//   Auth=DQAAAGgA...dk3fA5N
	res := make(map[string]string)
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		p := strings.SplitN(line, "=", 2)
		if len(p) == 2 {
			res[p[0]] = p[1]
		}
	}
	return res, nil
}
