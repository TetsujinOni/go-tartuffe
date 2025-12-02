package handlers

import (
	"net/http"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
)

// HomeResponse is the hypermedia response for GET /
type HomeResponse struct {
	Links HomeLinks `json:"_links"`
}

// HomeLinks contains the available API links
type HomeLinks struct {
	Imposters Link `json:"imposters"`
	Config    Link `json:"config"`
	Logs      Link `json:"logs"`
}

// Link is a hypermedia link
type Link struct {
	Href string `json:"href"`
}

// Home handles GET /
func Home(w http.ResponseWriter, r *http.Request) {
	resp := HomeResponse{
		Links: HomeLinks{
			Imposters: Link{Href: "/imposters"},
			Config:    Link{Href: "/config"},
			Logs:      Link{Href: "/logs"},
		},
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
