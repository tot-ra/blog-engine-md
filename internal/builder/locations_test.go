package builder

import (
	"strings"
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
	"github.com/tot-ra/blog-engine/internal/parser"
)

func TestEnhanceLocationLinksDecoratesFolderLinksAndEmbedsPreview(t *testing.T) {
	lat := 59.454948
	lng := 24.672783
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Locations: config.LocationsConfig{Enabled: true, Section: "locations"},
		},
		languages:  map[string]struct{}{"ru": {}, "en": {}},
		pages:      map[string]*Page{},
		pagesByURL: map[string]*Page{},
	}
	place := &Page{
		ID:      "ru-locations-ankru-10",
		URL:     "/ru/locations/ankru-10/",
		Title:   "Ankru 10",
		Content: `<p>Parking at Põhjala.</p>`,
		Frontmatter: &parser.Frontmatter{
			Title:   "Ankru 10",
			Address: "Ankru 10, Tallinn, Estonia",
			Lat:     lat,
			Lng:     lng,
		},
	}
	article := &Page{
		ID:      "ru-events-festival",
		URL:     "/ru/events/festival/",
		Title:   "Festival",
		Content: `<p>On the lot at <a href="/ru/locations/ankru-10/">Ankru 10</a>.</p>`,
	}
	listing := &Page{
		ID:      "ru-locations",
		URL:     "/ru/locations/",
		Title:   "Locations",
		Content: `<a class="section-index-card" href="/ru/locations/ankru-10/">Ankru 10</a>`,
	}
	other := &Page{
		ID:      "ru-blog-other",
		URL:     "/ru/blog/other/",
		Title:   "Other",
		Content: `<p>See <a href="/ru/events/festival/">the festival</a>.</p>`,
	}
	for _, page := range []*Page{place, article, listing, other} {
		b.pages[page.ID] = page
		b.pagesByURL[page.URL] = page
	}

	b.enhanceLocationLinks()

	if !strings.Contains(article.Content, `class="location-link"`) {
		t.Fatalf("expected article location link to be decorated, got %s", article.Content)
	}
	if !strings.Contains(article.Content, `aria-expanded="false"`) {
		t.Fatalf("expected aria-expanded on location link, got %s", article.Content)
	}
	if !strings.Contains(article.Content, `"html":"\u003cp\u003eParking at Põhjala.\u003c/p\u003e"`) &&
		!strings.Contains(article.Content, `"html":"<p>Parking at Põhjala.</p>"`) {
		t.Fatalf("expected location markdown HTML in preview JSON, got %s", article.Content)
	}
	if !strings.Contains(article.Content, `"address":"Ankru 10, Tallinn, Estonia"`) {
		t.Fatalf("expected address in preview JSON, got %s", article.Content)
	}
	if !strings.Contains(article.Content, `class="location-previews"`) {
		t.Fatalf("expected preview script on article, got %s", article.Content)
	}
	if strings.Contains(listing.Content, "location-link") {
		t.Fatalf("section cards must stay regular links, got %s", listing.Content)
	}
	if strings.Contains(other.Content, "location-link") {
		t.Fatalf("non-location links must stay untouched, got %s", other.Content)
	}
	if !strings.Contains(place.Content, `class="location-page-map"`) || !strings.Contains(place.Content, `class="location-map"`) || !strings.Contains(place.Content, `data-lat="59.454948"`) {
		t.Fatalf("expected standalone location page to include a map, got %s", place.Content)
	}
	if !strings.Contains(place.Content, `class="location-panel-map-btn"`) || !strings.Contains(place.Content, `google.com/maps/search/?api=1&amp;query=`) {
		t.Fatalf("expected standalone location page to include a Google Maps button, got %s", place.Content)
	}
	if strings.Contains(article.Content, "location-page-map") {
		t.Fatalf("inline preview must not embed the standalone page map wrapper")
	}
}

func TestEnhanceLocationLinksDisabledDoesNothing(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Locations: config.LocationsConfig{Enabled: false, Section: "locations"},
		},
		pages:      map[string]*Page{},
		pagesByURL: map[string]*Page{},
	}
	article := &Page{
		URL:     "/ru/events/festival/",
		Content: `<p><a href="/ru/locations/ankru-10/">Ankru 10</a></p>`,
	}
	place := &Page{URL: "/ru/locations/ankru-10/", Title: "Ankru 10", Content: `<p>Here</p>`}
	b.pages["a"] = article
	b.pages["p"] = place
	b.pagesByURL[article.URL] = article
	b.pagesByURL[place.URL] = place

	b.enhanceLocationLinks()
	if strings.Contains(article.Content, "location-link") {
		t.Fatalf("disabled locations must not rewrite links, got %s", article.Content)
	}
}

func TestEnhanceLocationLinksResolvesLanguageLessHref(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Locations: config.LocationsConfig{Enabled: true, Section: "locations"},
		},
		languages:  map[string]struct{}{"ru": {}},
		pages:      map[string]*Page{},
		pagesByURL: map[string]*Page{},
	}
	place := &Page{URL: "/ru/locations/ankru-10/", Title: "Ankru 10", Content: `<p>Here</p>`}
	article := &Page{
		URL:     "/ru/events/festival/",
		Content: `<p><a href="/locations/ankru-10">Ankru 10</a></p>`,
	}
	b.pages["p"] = place
	b.pages["a"] = article
	b.pagesByURL[place.URL] = place
	b.pagesByURL[article.URL] = article

	b.enhanceLocationLinks()
	if !strings.Contains(article.Content, `href="/ru/locations/ankru-10/"`) || !strings.Contains(article.Content, "location-link") {
		t.Fatalf("expected language-less href to resolve to the location page, got %s", article.Content)
	}
}

func TestURLHasLocationPlace(t *testing.T) {
	if !urlHasLocationPlace("/ru/locations/ankru-10/", "locations") {
		t.Fatal("child location URL should match")
	}
	if urlHasLocationPlace("/ru/locations/", "locations") {
		t.Fatal("section root is not a place")
	}
	if urlHasLocationPlace("/ru/blog/locations-are-cool/", "locations") {
		t.Fatal("slug containing the word should not match a path segment")
	}
}

func TestLocationPreviewHTMLStripsNestedEngineChrome(t *testing.T) {
	got := locationPreviewHTML(`<div class="location-page-map"><iframe></iframe></div><p>Body</p><script type="application/json" class="location-previews">{}</script>`)
	if got != "<p>Body</p>" {
		t.Fatalf("got %q", got)
	}
}
