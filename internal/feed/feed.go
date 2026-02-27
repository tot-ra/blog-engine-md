package feed

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// FeedItem represents a single item in a feed
type FeedItem struct {
	Title       string
	URL         string    // absolute URL
	Date        time.Time
	Description string    // HTML content or excerpt
	Categories  []string  // tags/categories
	GUID        string    // permalink
}

// FeedGenerator generates RSS and Atom feeds
type FeedGenerator struct {
	siteTitle   string
	siteURL     string
	siteLang    string
	authorName  string
	authorEmail string
}

// NewFeedGenerator creates a new feed generator
func NewFeedGenerator(siteTitle, siteURL, siteLang, authorName, authorEmail string) *FeedGenerator {
	return &FeedGenerator{
		siteTitle:   siteTitle,
		siteURL:     strings.TrimSuffix(siteURL, "/"),
		siteLang:    siteLang,
		authorName:  authorName,
		authorEmail: authorEmail,
	}
}

// --- RSS 2.0 XML structures ---

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string      `xml:"title"`
	Link          string      `xml:"link"`
	Description   string      `xml:"description"`
	Language      string      `xml:"language"`
	LastBuildDate string      `xml:"lastBuildDate"`
	AtomLink      rssAtomLink `xml:"atom:link"`
	Items         []rssItem   `xml:"item"`
}

type rssAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	GUID        rssGUID       `xml:"guid"`
	PubDate     string        `xml:"pubDate"`
	Description rssCDATA      `xml:"description"`
	Categories  []string      `xml:"category,omitempty"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type rssCDATA struct {
	Value string `xml:",cdata"`
}

// GenerateRSS generates an RSS 2.0 feed
func (g *FeedGenerator) GenerateRSS(items []FeedItem, feedPath string) (string, error) {
	feedURL := g.siteURL + "/" + strings.TrimPrefix(feedPath, "/")

	var rssItems []rssItem
	for _, item := range items {
		guid := item.GUID
		if guid == "" {
			guid = item.URL
		}
		rssItems = append(rssItems, rssItem{
			Title: item.Title,
			Link:  item.URL,
			GUID: rssGUID{
				IsPermaLink: "true",
				Value:       guid,
			},
			PubDate:     item.Date.UTC().Format(time.RFC1123Z),
			Description: rssCDATA{Value: item.Description},
			Categories:  item.Categories,
		})
	}

	now := time.Now().UTC().Format(time.RFC1123Z)

	root := rssRoot{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:         g.siteTitle,
			Link:          g.siteURL,
			Description:   g.siteTitle,
			Language:      g.siteLang,
			LastBuildDate: now,
			AtomLink: rssAtomLink{
				Href: feedURL,
				Rel:  "self",
				Type: "application/rss+xml",
			},
			Items: rssItems,
		},
	}

	output, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal RSS: %w", err)
	}

	return xml.Header + string(output), nil
}

// --- Atom 1.0 XML structures ---

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	NS      string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Author  *atomAuthor `xml:"author,omitempty"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name  string `xml:"name,omitempty"`
	Email string `xml:"email,omitempty"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	ID         string         `xml:"id"`
	Updated    string         `xml:"updated"`
	Published  string         `xml:"published"`
	Links      []atomLink     `xml:"link"`
	Summary    atomContent    `xml:"summary"`
	Categories []atomCategory `xml:"category,omitempty"`
}

type atomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

// GenerateAtom generates an Atom 1.0 feed
func (g *FeedGenerator) GenerateAtom(items []FeedItem, feedPath string) (string, error) {
	feedURL := g.siteURL + "/" + strings.TrimPrefix(feedPath, "/")

	var entries []atomEntry
	for _, item := range items {
		id := item.GUID
		if id == "" {
			id = item.URL
		}
		var cats []atomCategory
		for _, c := range item.Categories {
			cats = append(cats, atomCategory{Term: c})
		}
		entries = append(entries, atomEntry{
			Title:     item.Title,
			ID:        id,
			Updated:   item.Date.UTC().Format(time.RFC3339),
			Published: item.Date.UTC().Format(time.RFC3339),
			Links: []atomLink{
				{Href: item.URL, Rel: "alternate", Type: "text/html"},
			},
			Summary: atomContent{
				Type:  "html",
				Value: item.Description,
			},
			Categories: cats,
		})
	}

	var author *atomAuthor
	if g.authorName != "" || g.authorEmail != "" {
		author = &atomAuthor{
			Name:  g.authorName,
			Email: g.authorEmail,
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	f := atomFeed{
		NS:      "http://www.w3.org/2005/Atom",
		Title:   g.siteTitle,
		ID:      g.siteURL + "/",
		Updated: now,
		Links: []atomLink{
			{Href: g.siteURL + "/", Rel: "alternate", Type: "text/html"},
			{Href: feedURL, Rel: "self", Type: "application/atom+xml"},
		},
		Author:  author,
		Entries: entries,
	}

	output, err := xml.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Atom: %w", err)
	}

	return xml.Header + string(output), nil
}
