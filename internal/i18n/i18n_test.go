package i18n

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalizeLanguage_Aliases(t *testing.T) {
	tests := map[string]string{
		"ru":      "ru",
		"rus":     "ru",
		"ru-RU":   "ru",
		" RU_ru ": "ru",
		"et":      "et",
		"est":     "et",
		"et_EE":   "et",
		"en":      "en",
		"unknown": "en",
		"":        "en",
	}

	for input, want := range tests {
		if got := NormalizeLanguage(input); got != want {
			t.Fatalf("NormalizeLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUI_LocalizedStrings(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want UIStrings
	}{
		{
			name: "English default",
			lang: "unknown",
			want: UIStrings{
				Home:            "Home",
				Blog:            "Blog",
				Docs:            "Docs",
				Languages:       "Languages",
				Navigation:      "Navigation",
				ViewMode:        "View mode",
				Tags:            "Tags",
				Categories:      "Categories",
				Time:            "Time",
				Graph:           "Graph",
				OnThisPage:      "On this page",
				Breadcrumb:      "Breadcrumb",
				PageNavigation:  "Page navigation",
				Previous:        "Previous",
				Next:            "Next",
				Listen:          "Listen",
				Stop:            "Stop",
				Rewind:          "Rewind",
				PlaybackPos:     "Playback position",
				ToggleTheme:     "Toggle theme",
				AskMyAgent:      "Ask my agent",
				Projects:        "Projects",
				Chat:            "Chat",
				HeroVideo:       "Hero video",
				BlogNavigation:  "Blog navigation",
				BlogViewMode:    "Blog view mode",
				BlogGraphView:   "Blog graph view",
				ToggleSectionOf: "Toggle section",
				LogIn:           "Log in",
			},
		},
		{
			name: "Russian",
			lang: "rus",
			want: UIStrings{
				Home:            "Главная",
				Blog:            "Блог",
				Docs:            "Документы",
				Languages:       "Языки",
				Navigation:      "Навигация",
				ViewMode:        "Режим просмотра",
				Tags:            "Теги",
				Categories:      "Категории",
				Time:            "Время",
				Graph:           "Граф",
				OnThisPage:      "На этой странице",
				Breadcrumb:      "Хлебные крошки",
				PageNavigation:  "Навигация по странице",
				Previous:        "Предыдущая",
				Next:            "Следующая",
				Listen:          "Слушать",
				Stop:            "Стоп",
				Rewind:          "В начало",
				PlaybackPos:     "Позиция воспроизведения",
				ToggleTheme:     "Сменить тему",
				AskMyAgent:      "Спросить моего агента",
				Projects:        "Проекты",
				Chat:            "Чат",
				HeroVideo:       "Видео",
				BlogNavigation:  "Навигация по блогу",
				BlogViewMode:    "Режим просмотра блога",
				BlogGraphView:   "Граф блога",
				ToggleSectionOf: "Переключить раздел",
				LogIn:           "Войти",
			},
		},
		{
			name: "Estonian",
			lang: "est",
			want: UIStrings{
				Home:            "Avaleht",
				Blog:            "Blogi",
				Docs:            "Dokumendid",
				Languages:       "Keeled",
				Navigation:      "Navigeerimine",
				ViewMode:        "Vaade",
				Tags:            "Sildid",
				Categories:      "Kategooriad",
				Time:            "Aeg",
				Graph:           "Graaf",
				OnThisPage:      "Sellel lehel",
				Breadcrumb:      "Leivapururada",
				PageNavigation:  "Lehe navigeerimine",
				Previous:        "Eelmine",
				Next:            "Järgmine",
				Listen:          "Kuula",
				Stop:            "Peata",
				Rewind:          "Algusesse",
				PlaybackPos:     "Taasesituse asukoht",
				ToggleTheme:     "Vaheta teemat",
				AskMyAgent:      "Küsi minu agendilt",
				Projects:        "Projektid",
				Chat:            "Vestlus",
				HeroVideo:       "Tutvustusvideo",
				BlogNavigation:  "Blogi navigeerimine",
				BlogViewMode:    "Blogi vaade",
				BlogGraphView:   "Blogi graaf",
				ToggleSectionOf: "Lülita jaotis",
				LogIn:           "Logi sisse",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UI(tt.lang); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("UI(%q) mismatch\ngot:  %#v\nwant: %#v", tt.lang, got, tt.want)
			}
		})
	}
}

func TestMonthName(t *testing.T) {
	tests := []struct {
		lang  string
		month time.Month
		want  string
	}{
		{lang: "en", month: time.January, want: "January"},
		{lang: "ru", month: time.January, want: "Январь"},
		{lang: "et", month: time.January, want: "Jaanuar"},
		{lang: "en", month: time.Month(13), want: "%!Month(13)"},
		{lang: "ru", month: time.Month(13), want: "%!Month(13)"},
		{lang: "et", month: time.Month(13), want: "%!Month(13)"},
	}

	for _, tt := range tests {
		if got := MonthName(tt.lang, tt.month); got != tt.want {
			t.Fatalf("MonthName(%q, %v) = %q, want %q", tt.lang, tt.month, got, tt.want)
		}
	}
}

func TestFormatDateLong(t *testing.T) {
	date := time.Date(2024, time.March, 5, 14, 30, 0, 0, time.UTC)
	tests := map[string]string{
		"en": "March 5, 2024",
		"ru": "5 марта 2024",
		"et": "5. marts 2024",
	}

	for lang, want := range tests {
		if got := FormatDateLong(date, lang); got != want {
			t.Fatalf("FormatDateLong(%q) = %q, want %q", lang, got, want)
		}
	}
	if got := FormatDateLong(time.Time{}, "en"); got != "" {
		t.Fatalf("FormatDateLong(zero) = %q, want empty string", got)
	}
}

func TestFormatDateShort(t *testing.T) {
	date := time.Date(2024, time.March, 5, 14, 30, 0, 0, time.UTC)
	tests := map[string]string{
		"en": "Mar 05",
		"ru": "05 марта",
		"et": "05 Mar",
	}

	for lang, want := range tests {
		if got := FormatDateShort(date, lang); got != want {
			t.Fatalf("FormatDateShort(%q) = %q, want %q", lang, got, want)
		}
	}
	if got := FormatDateShort(time.Time{}, "en"); got != "" {
		t.Fatalf("FormatDateShort(zero) = %q, want empty string", got)
	}
}

func TestSegmentLabel(t *testing.T) {
	tests := []struct {
		lang    string
		segment string
		want    string
	}{
		{lang: "en", segment: " blog ", want: "Blog"},
		{lang: "en", segment: "docs", want: "Docs"},
		{lang: "en", segment: "tags", want: "Tags"},
		{lang: "en", segment: "archive", want: "Archive"},
		{lang: "en", segment: "graph", want: "Graph"},
		{lang: "en", segment: "events", want: "Events"},
		{lang: "ru", segment: "blog", want: "Блог"},
		{lang: "ru", segment: "docs", want: "Документы"},
		{lang: "ru", segment: "tags", want: "Теги"},
		{lang: "ru", segment: "archive", want: "Архив"},
		{lang: "ru", segment: "graph", want: "Граф"},
		{lang: "ru", segment: "events", want: "События"},
		{lang: "et", segment: "blog", want: "Blogi"},
		{lang: "et", segment: "docs", want: "Dokumendid"},
		{lang: "et", segment: "tags", want: "Sildid"},
		{lang: "et", segment: "archive", want: "Arhiiv"},
		{lang: "et", segment: "graph", want: "Graaf"},
		{lang: "en", segment: "missing", want: ""},
		{lang: "ru", segment: "missing", want: ""},
		{lang: "et", segment: "missing", want: ""},
	}

	for _, tt := range tests {
		if got := SegmentLabel(tt.lang, tt.segment); got != tt.want {
			t.Fatalf("SegmentLabel(%q, %q) = %q, want %q", tt.lang, tt.segment, got, tt.want)
		}
	}
}
