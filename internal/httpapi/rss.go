package httpapi

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// rssFeed is the subset of rss-parser's output the client uses. Firefish returns
// the full parser object; the widgets read `items[].title` and `items[].link`,
// so the endpoint keeps those and the feed-level title/link/description.
type rssFeed struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Items       []rssItem `json:"items"`
}

type rssItem struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	PubDate string `json:"pubDate"`
	GUID    string `json:"guid"`
}

// handleFetchRSS implements GET/POST /api/fetch-rss.
//
// Firefish fetches the URL server-side (browsers cannot read cross-origin
// feeds) with a 5 second timeout and returns rss-parser's JSON. The same SSRF
// guard as drive/files/upload-from-url applies to the URL and every redirect.
func (s *Server) handleFetchRSS(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("url"))
	if target == "" {
		var body struct {
			URL string `json:"url"`
		}
		_ = decodeOptionalJSON(r, &body)
		target = strings.TrimSpace(body.URL)
	}
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		writeAPIError(w, invalidParamError("url must be an http(s) URL"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return checkFetchTarget(next.Context(), next.URL)
		},
	}
	if err := checkFetchTarget(ctx, parsed); err != nil {
		writeAPIError(w, invalidParamError("url is not allowed: "+err.Error()))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		writeAPIError(w, invalidParamError("url is invalid"))
		return
	}
	req.Header.Set("User-Agent", "Cinderfish/0.1 (+fetch-rss)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml;q=0.9, */*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		writeAPIError(w, feedError("Could not fetch the feed."))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeAPIError(w, feedError("The feed answered with an error status."))
		return
	}
	// 1 MiB is far beyond any real feed and keeps a hostile endpoint bounded.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		writeAPIError(w, feedError("The feed is too large."))
		return
	}

	feed, err := parseFeed(data)
	if err != nil {
		writeAPIError(w, feedError("The response is not an RSS or Atom feed."))
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

// firstNonEmpty returns the first trimmed value that is not empty, which skips
// the attribute-only atom:link elements an RSS channel usually carries.
func firstNonEmpty(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func feedError(message string) *APIError {
	return &APIError{
		Message: message, Code: "FETCH_FAILED",
		ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
	}
}

// parseFeed decodes RSS 2.0 and Atom 1.0. A feed with no items is still valid:
// the widget renders an empty list rather than an error.
func parseFeed(data []byte) (rssFeed, error) {
	var root struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &root); err != nil {
		return rssFeed{}, err
	}
	switch root.XMLName.Local {
	case "rss":
		var doc struct {
			Channel struct {
				Title string `xml:"title"`
				// A feed may carry several <link> elements (RSS channel plus
				// atom:link self); encoding/xml matches them all by local name,
				// so keep the list and take the first with a value.
				Links       []string `xml:"link"`
				Description string   `xml:"description"`
				Items       []struct {
					Title   string   `xml:"title"`
					Links   []string `xml:"link"`
					GUID    string   `xml:"guid"`
					PubDate string   `xml:"pubDate"`
				} `xml:"item"`
			} `xml:"channel"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return rssFeed{}, err
		}
		feed := rssFeed{
			Title:       strings.TrimSpace(doc.Channel.Title),
			Link:        firstNonEmpty(doc.Channel.Links),
			Description: strings.TrimSpace(doc.Channel.Description),
			Items:       make([]rssItem, 0, len(doc.Channel.Items)),
		}
		for _, item := range doc.Channel.Items {
			feed.Items = append(feed.Items, rssItem{
				Title:   strings.TrimSpace(item.Title),
				Link:    firstNonEmpty(item.Links),
				PubDate: strings.TrimSpace(item.PubDate),
				GUID:    strings.TrimSpace(item.GUID),
			})
		}
		return feed, nil
	case "feed":
		var doc struct {
			Title   string `xml:"title"`
			Updated string `xml:"updated"`
			Links   []struct {
				Href string `xml:"href,attr"`
				Rel  string `xml:"rel,attr"`
			} `xml:"link"`
			Entries []struct {
				Title     string `xml:"title"`
				ID        string `xml:"id"`
				Updated   string `xml:"updated"`
				Published string `xml:"published"`
				Links     []struct {
					Href string `xml:"href,attr"`
					Rel  string `xml:"rel,attr"`
				} `xml:"link"`
			} `xml:"entry"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			return rssFeed{}, err
		}
		feed := rssFeed{
			Title: strings.TrimSpace(doc.Title),
			Items: make([]rssItem, 0, len(doc.Entries)),
		}
		for _, link := range doc.Links {
			if link.Rel == "" || link.Rel == "alternate" {
				feed.Link = strings.TrimSpace(link.Href)
				break
			}
		}
		for _, entry := range doc.Entries {
			item := rssItem{
				Title:   strings.TrimSpace(entry.Title),
				PubDate: strings.TrimSpace(entry.Published),
				GUID:    strings.TrimSpace(entry.ID),
			}
			if item.PubDate == "" {
				item.PubDate = strings.TrimSpace(entry.Updated)
			}
			for _, link := range entry.Links {
				if link.Rel == "" || link.Rel == "alternate" {
					item.Link = strings.TrimSpace(link.Href)
					break
				}
			}
			feed.Items = append(feed.Items, item)
		}
		return feed, nil
	default:
		return rssFeed{}, errors.New("not a feed")
	}
}
