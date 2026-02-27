package pagination

import (
	"fmt"
	"strings"
)

// PaginatedResult holds the result of paginating a slice of items
type PaginatedResult[T any] struct {
	Items       []T
	CurrentPage int
	TotalPages  int
	TotalItems  int
	HasPrev     bool
	HasNext     bool
	PrevURL     string
	NextURL     string
}

// Paginate splits items into pages and returns the result for the requested page.
// Page numbering starts at 1.
// URL pattern: baseURL for page 1, baseURL + "page/2/" for page 2, etc.
func Paginate[T any](items []T, pageSize int, currentPage int, baseURL string) *PaginatedResult[T] {
	totalItems := len(items)
	if pageSize <= 0 {
		pageSize = 10
	}

	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	if currentPage < 1 {
		currentPage = 1
	}
	if currentPage > totalPages {
		currentPage = totalPages
	}

	start := (currentPage - 1) * pageSize
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	var pageItems []T
	if start < totalItems {
		pageItems = items[start:end]
	}

	// Ensure baseURL ends with /
	baseURL = strings.TrimSuffix(baseURL, "/") + "/"

	result := &PaginatedResult[T]{
		Items:       pageItems,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		HasPrev:     currentPage > 1,
		HasNext:     currentPage < totalPages,
	}

	if result.HasPrev {
		result.PrevURL = pageURL(baseURL, currentPage-1)
	}
	if result.HasNext {
		result.NextURL = pageURL(baseURL, currentPage+1)
	}

	return result
}

// pageURL generates the URL for a given page number
func pageURL(baseURL string, page int) string {
	if page <= 1 {
		return baseURL
	}
	return fmt.Sprintf("%spage/%d/", baseURL, page)
}
