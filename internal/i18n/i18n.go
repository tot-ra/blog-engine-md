package i18n

import (
	"fmt"
	"strings"
	"time"
)

// UIStrings contains localized labels used by templates and UI components.
type UIStrings struct {
	Home            string
	Blog            string
	Docs            string
	Tags            string
	Categories      string
	Time            string
	Graph           string
	OnThisPage      string
	Previous        string
	Next            string
	Listen          string
	Stop            string
	Rewind          string
	PlaybackPos     string
	ToggleTheme     string
	AskMyAgent      string
	Projects        string
	BlogNavigation  string
	BlogViewMode    string
	BlogGraphView   string
	ToggleSectionOf string
}

var ruMonthsGenitive = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}

var ruMonthsStandalone = map[time.Month]string{
	time.January:   "Январь",
	time.February:  "Февраль",
	time.March:     "Март",
	time.April:     "Апрель",
	time.May:       "Май",
	time.June:      "Июнь",
	time.July:      "Июль",
	time.August:    "Август",
	time.September: "Сентябрь",
	time.October:   "Октябрь",
	time.November:  "Ноябрь",
	time.December:  "Декабрь",
}

var enMonthsStandalone = map[time.Month]string{
	time.January:   "January",
	time.February:  "February",
	time.March:     "March",
	time.April:     "April",
	time.May:       "May",
	time.June:      "June",
	time.July:      "July",
	time.August:    "August",
	time.September: "September",
	time.October:   "October",
	time.November:  "November",
	time.December:  "December",
}

// NormalizeLanguage normalizes language code and defaults to "en".
func NormalizeLanguage(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	if l == "" {
		return "en"
	}
	if idx := strings.IndexByte(l, '-'); idx > 0 {
		l = l[:idx]
	}
	if idx := strings.IndexByte(l, '_'); idx > 0 {
		l = l[:idx]
	}
	switch l {
	case "ru":
		return "ru"
	default:
		return "en"
	}
}

// UI returns localized UI strings for a language code.
func UI(lang string) UIStrings {
	switch NormalizeLanguage(lang) {
	case "ru":
		return UIStrings{
			Home:            "Главная",
			Blog:            "Блог",
			Docs:            "Документы",
			Tags:            "Теги",
			Categories:      "Категории",
			Time:            "Время",
			Graph:           "Граф",
			OnThisPage:      "На этой странице",
			Previous:        "Предыдущая",
			Next:            "Следующая",
			Listen:          "Слушать",
			Stop:            "Стоп",
			Rewind:          "В начало",
			PlaybackPos:     "Позиция воспроизведения",
			ToggleTheme:     "Сменить тему",
			AskMyAgent:      "Спросить моего агента",
			Projects:        "Проекты",
			BlogNavigation:  "Навигация по блогу",
			BlogViewMode:    "Режим просмотра блога",
			BlogGraphView:   "Граф блога",
			ToggleSectionOf: "Переключить раздел",
		}
	default:
		return UIStrings{
			Home:            "Home",
			Blog:            "Blog",
			Docs:            "Docs",
			Tags:            "Tags",
			Categories:      "Categories",
			Time:            "Time",
			Graph:           "Graph",
			OnThisPage:      "On this page",
			Previous:        "Previous",
			Next:            "Next",
			Listen:          "Listen",
			Stop:            "Stop",
			Rewind:          "Rewind",
			PlaybackPos:     "Playback position",
			ToggleTheme:     "Toggle theme",
			AskMyAgent:      "Ask my agent",
			Projects:        "Projects",
			BlogNavigation:  "Blog navigation",
			BlogViewMode:    "Blog view mode",
			BlogGraphView:   "Blog graph view",
			ToggleSectionOf: "Toggle section",
		}
	}
}

// MonthName returns a standalone month label.
func MonthName(lang string, m time.Month) string {
	if NormalizeLanguage(lang) == "ru" {
		if s, ok := ruMonthsStandalone[m]; ok {
			return s
		}
		return m.String()
	}
	if s, ok := enMonthsStandalone[m]; ok {
		return s
	}
	return m.String()
}

// FormatDateLong formats a date for article headers.
func FormatDateLong(t time.Time, lang string) string {
	if t.IsZero() {
		return ""
	}
	if NormalizeLanguage(lang) == "ru" {
		month := ruMonthsGenitive[t.Month()]
		if month == "" {
			month = strings.ToLower(t.Month().String())
		}
		return fmt.Sprintf("%d %s %d", t.Day(), month, t.Year())
	}
	return t.Format("January 2, 2006")
}

// FormatDateShort formats a compact date for archive lists.
func FormatDateShort(t time.Time, lang string) string {
	if t.IsZero() {
		return ""
	}
	if NormalizeLanguage(lang) == "ru" {
		month := ruMonthsGenitive[t.Month()]
		if month == "" {
			month = strings.ToLower(t.Month().String())
		}
		return fmt.Sprintf("%02d %s", t.Day(), month)
	}
	return t.Format("Jan 02")
}

// SegmentLabel returns translated labels for known route segments.
func SegmentLabel(lang, segment string) string {
	seg := strings.ToLower(strings.TrimSpace(segment))
	switch NormalizeLanguage(lang) {
	case "ru":
		switch seg {
		case "blog":
			return "Блог"
		case "docs":
			return "Документы"
		case "tags":
			return "Теги"
		case "archive":
			return "Архив"
		case "graph":
			return "Граф"
		}
	default:
		switch seg {
		case "blog":
			return "Blog"
		case "docs":
			return "Docs"
		case "tags":
			return "Tags"
		case "archive":
			return "Archive"
		case "graph":
			return "Graph"
		}
	}
	return ""
}
