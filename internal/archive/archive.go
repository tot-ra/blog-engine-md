package archive

import (
	"sort"
	"time"
)

// PageSummary represents a page for archive grouping
type PageSummary struct {
	Title       string
	URL         string
	Date        time.Time
	Description string
	Tags        []string
	Type        string
}

// ArchiveYear groups pages by year
type ArchiveYear struct {
	Year   int
	Months []ArchiveMonth
	Count  int
}

// ArchiveMonth groups pages by month within a year
type ArchiveMonth struct {
	Month time.Month
	Year  int
	Pages []PageSummary
	Count int
}

// BuildArchive creates an archive structure from pages, sorted newest first
func BuildArchive(pages []PageSummary) []ArchiveYear {
	if len(pages) == 0 {
		return nil
	}

	// Group by year -> month
	yearMap := make(map[int]map[time.Month][]PageSummary)

	for _, p := range pages {
		if p.Date.IsZero() {
			continue
		}
		year := p.Date.Year()
		month := p.Date.Month()

		if yearMap[year] == nil {
			yearMap[year] = make(map[time.Month][]PageSummary)
		}
		yearMap[year][month] = append(yearMap[year][month], p)
	}

	// Build sorted structure
	var years []ArchiveYear
	for year, months := range yearMap {
		ay := ArchiveYear{Year: year}

		var monthList []ArchiveMonth
		for month, monthPages := range months {
			// Sort pages within month by date descending
			sort.Slice(monthPages, func(i, j int) bool {
				return monthPages[i].Date.After(monthPages[j].Date)
			})
			monthList = append(monthList, ArchiveMonth{
				Month: month,
				Year:  year,
				Pages: monthPages,
				Count: len(monthPages),
			})
		}

		// Sort months descending
		sort.Slice(monthList, func(i, j int) bool {
			return monthList[i].Month > monthList[j].Month
		})

		ay.Months = monthList
		for _, m := range monthList {
			ay.Count += m.Count
		}
		years = append(years, ay)
	}

	// Sort years descending (newest first)
	sort.Slice(years, func(i, j int) bool {
		return years[i].Year > years[j].Year
	})

	return years
}
