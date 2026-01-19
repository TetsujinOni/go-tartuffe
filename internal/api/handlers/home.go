package handlers

import (
	"net/http"

	"github.com/TetsujinOni/go-tartuffe/internal/response"
	"github.com/TetsujinOni/go-tartuffe/internal/web"
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
	// Content negotiation: HTML for browsers, JSON for API clients
	if web.AcceptsHTML(r) {
		data := web.HomePageData{
			PageData: web.PageData{
				Title:       "over the wire test doubles",
				Description: "Placeholder description for the home page.",
			},
			Notices: []web.Notice{},
		}
		web.Render(w, "index.html", data)
		return
	}

	baseURL := buildBaseURL(r)

	resp := HomeResponse{
		Links: HomeLinks{
			Imposters: Link{Href: baseURL + "/imposters"},
			Config:    Link{Href: baseURL + "/config"},
			Logs:      Link{Href: baseURL + "/logs"},
		},
	}

	response.WriteJSON(w, http.StatusOK, resp)
}
