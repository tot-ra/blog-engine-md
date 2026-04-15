package i18n

import "testing"

func TestNormalizeLanguage_Aliases(t *testing.T) {
	tests := map[string]string{
		"ru":       "ru",
		"rus":      "ru",
		"ru-RU":    "ru",
		"et":       "et",
		"est":      "et",
		"et_EE":    "et",
		"en":       "en",
		"unknown":  "en",
		"":         "en",
	}

	for input, want := range tests {
		if got := NormalizeLanguage(input); got != want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUI_EstonianSidebarLabels(t *testing.T) {
	ui := UI("est")

	if ui.Categories != "Kategooriad" {
		t.Fatalf("expected Estonian categories label, got %q", ui.Categories)
	}
	if ui.Time != "Aeg" {
		t.Fatalf("expected Estonian time label, got %q", ui.Time)
	}
	if ui.Graph != "Graaf" {
		t.Fatalf("expected Estonian graph label, got %q", ui.Graph)
	}
	if ui.Languages != "Keeled" {
		t.Fatalf("expected Estonian languages label, got %q", ui.Languages)
	}
	if ui.Breadcrumb != "Leivapururada" {
		t.Fatalf("expected Estonian breadcrumb label, got %q", ui.Breadcrumb)
	}
	if ui.PageNavigation != "Lehe navigeerimine" {
		t.Fatalf("expected Estonian page navigation label, got %q", ui.PageNavigation)
	}
	if ui.Chat != "Vestlus" {
		t.Fatalf("expected Estonian chat label, got %q", ui.Chat)
	}
}
