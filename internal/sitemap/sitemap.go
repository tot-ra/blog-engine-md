package sitemap

import (
	"encoding/xml"
	"fmt"
	"time"
)

// SitemapEntry represents a single URL in the sitemap
type SitemapEntry struct {
	URL        string
	LastMod    time.Time
	ChangeFreq string  // weekly, monthly, yearly, never
	Priority   float64 // 0.0-1.0
}

// XML structures for sitemap

type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	NS      string    `xml:"xmlns,attr"`
	URLs    []urlItem `xml:"url"`
}

type urlItem struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority"`
}

// Generate creates a sitemap XML string from entries
func Generate(entries []SitemapEntry) (string, error) {
	var urls []urlItem
	for _, e := range entries {
		lastMod := ""
		if !e.LastMod.IsZero() {
			lastMod = e.LastMod.Format("2006-01-02")
		}
		urls = append(urls, urlItem{
			Loc:        e.URL,
			LastMod:    lastMod,
			ChangeFreq: e.ChangeFreq,
			Priority:   e.Priority,
		})
	}

	us := urlSet{
		NS:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: urls,
	}

	output, err := xml.MarshalIndent(us, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sitemap: %w", err)
	}

	return xml.Header + string(output), nil
}
