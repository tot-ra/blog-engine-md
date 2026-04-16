package builder

import (
	"testing"

	"github.com/tot-ra/blog-engine/internal/config"
)

func TestPrevNextConfigForPage_InheritsNearestSectionRule(t *testing.T) {
	disabled := false
	enabled := true
	sameCategoryOnly := true

	builder := &SiteBuilder{
		config: &config.SiteConfig{
			Navigation: config.Navigation{
				PrevNext: config.PrevNextConfig{
					Enabled:          false,
					SameCategoryOnly: false,
					Sections: map[string]config.PrevNextSectionConfig{
						"/study/": {
							Enabled: &disabled,
						},
						"/study/mahtra_pohikool/": {
							Enabled:          &enabled,
							SameCategoryOnly: &sameCategoryOnly,
						},
					},
				},
			},
		},
		languages: map[string]struct{}{
			"est": {},
			"rus": {},
		},
	}

	cfg := builder.prevNextConfigForPage(&Page{
		URL: "/est/study/mahtra_pohikool/2klass/sissejuhatus_oppeainesse/",
	})

	if !cfg.Enabled {
		t.Fatal("expected prev/next to be enabled by inherited section rule")
	}
	if !cfg.SameCategoryOnly {
		t.Fatal("expected sameCategoryOnly to be enabled by inherited section rule")
	}
}

func TestPrevNextConfigForPage_LeavesOtherSectionsDisabled(t *testing.T) {
	enabled := true

	builder := &SiteBuilder{
		config: &config.SiteConfig{
			Navigation: config.Navigation{
				PrevNext: config.PrevNextConfig{
					Enabled:          false,
					SameCategoryOnly: false,
					Sections: map[string]config.PrevNextSectionConfig{
						"/study/mahtra_pohikool/": {
							Enabled: &enabled,
						},
					},
				},
			},
		},
		languages: map[string]struct{}{
			"est": {},
		},
	}

	cfg := builder.prevNextConfigForPage(&Page{
		URL: "/est/study/EG/4klass/example/",
	})

	if cfg.Enabled {
		t.Fatal("expected prev/next to stay disabled outside configured section")
	}
}
