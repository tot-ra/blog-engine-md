package tags

import (
	"sort"
	"time"
)

// PageSummary represents a page with minimal data for tag/archive indexing
type PageSummary struct {
	Title       string
	URL         string
	Date        time.Time
	Description string
	Tags        []string
	Type        string
}

// TagIndex maps tag names to pages that have that tag
type TagIndex map[string][]PageSummary

// BuildTagIndex builds a tag index from a slice of pages
func BuildTagIndex(pages []PageSummary) TagIndex {
	idx := make(TagIndex)
	for _, p := range pages {
		for _, tag := range p.Tags {
			if tag == "" {
				continue
			}
			idx[tag] = append(idx[tag], p)
		}
	}

	// Sort pages within each tag by date descending
	for tag := range idx {
		sort.Slice(idx[tag], func(i, j int) bool {
			return idx[tag][i].Date.After(idx[tag][j].Date)
		})
	}

	return idx
}

// Tags returns all tags sorted alphabetically
func (idx TagIndex) Tags() []string {
	tags := make([]string, 0, len(idx))
	for tag := range idx {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// Count returns the number of pages with a given tag
func (idx TagIndex) Count(tag string) int {
	return len(idx[tag])
}
