package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
)

func TestBuildHeaderSocialResolvesHandlesAndURLs(t *testing.T) {
	b := &SiteBuilder{
		config: &config.SiteConfig{
			Author: config.Author{
				Social: map[string]string{
					"linkedin": "kurapov",
					"github":   "https://github.com/tot-ra",
					"twitter":  "tot_ra",
				},
			},
		},
	}

	got := b.buildHeaderSocial()
	if len(got) != 2 {
		t.Fatalf("expected only github+linkedin icon links, got %#v", got)
	}
	if got[0].Icon != "github" || got[0].URL != "https://github.com/tot-ra" || got[0].Label != "GitHub" {
		t.Fatalf("unexpected github link: %#v", got[0])
	}
	if got[1].Icon != "linkedin" || got[1].URL != "https://www.linkedin.com/in/kurapov" || got[1].Label != "LinkedIn" {
		t.Fatalf("unexpected linkedin link: %#v", got[1])
	}
}

func TestResolveAuthorSocialURL(t *testing.T) {
	cases := []struct {
		network string
		value   string
		want    string
	}{
		{"github", "tot-ra", "https://github.com/tot-ra"},
		{"github", "@tot-ra", "https://github.com/tot-ra"},
		{"linkedin", "in/kurapov/", "https://www.linkedin.com/in/kurapov/"},
		{"linkedin", "https://www.linkedin.com/in/kurapov/", "https://www.linkedin.com/in/kurapov/"},
		{"github", "   ", ""},
	}
	for _, tc := range cases {
		if got := resolveAuthorSocialURL(tc.network, tc.value); got != tc.want {
			t.Fatalf("resolveAuthorSocialURL(%q, %q)=%q, want %q", tc.network, tc.value, got, tc.want)
		}
	}
}
