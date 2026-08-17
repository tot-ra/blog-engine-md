package related

import (
	"reflect"
	"testing"
)

func TestComputeRelatedOrderThresholdAndCount(t *testing.T) {
	entries := []Entry{
		{Path: "en/blog/q.md", Language: "en", TranslationPath: "blog/q.md", Vector: []float32{1, 0}},
		{Path: "en/blog/a.md", URL: "/a/", Title: "A", Language: "en", TranslationPath: "blog/a.md", Vector: []float32{0.9, 0.4358899}},
		{Path: "en/blog/b.md", URL: "/b/", Title: "B", Language: "en", TranslationPath: "blog/b.md", Vector: []float32{0.8, 0.6}},
		{Path: "en/blog/c.md", Language: "en", TranslationPath: "blog/c.md", Vector: []float32{0.2, 0.9799}},
	}
	got := ComputeRelated(entries, Config{Count: 2, MinScore: 0.5, Diversity: 1})[entries[0].Path]
	if len(got) != 2 || got[0].Path != entries[1].Path || got[1].Path != entries[2].Path {
		t.Fatalf("matches = %#v", got)
	}
}

func TestComputeRelatedFiltersLanguageAndTranslation(t *testing.T) {
	entries := []Entry{
		{Path: "en/blog/q.md", Language: "en", TranslationPath: "blog/q.md", Vector: []float32{1, 0}},
		{Path: "ru/blog/q.md", Language: "ru", TranslationPath: "blog/q.md", Vector: []float32{1, 0}},
		{Path: "ru/blog/other.md", Language: "ru", TranslationPath: "blog/other.md", Vector: []float32{1, 0}},
		{Path: "en/blog/ok.md", Language: "en", TranslationPath: "blog/ok.md", Vector: []float32{0.9, 0.4358899}},
	}
	got := ComputeRelated(entries, Config{Count: 5, MinScore: 0, Diversity: 1})[entries[0].Path]
	if len(got) != 1 || got[0].Path != entries[3].Path {
		t.Fatalf("same-language matches = %#v", got)
	}
	got = ComputeRelated(entries, Config{Count: 5, MinScore: 0, Diversity: 1, CrossLanguage: true})[entries[0].Path]
	if len(got) != 2 || got[0].Path != entries[2].Path {
		t.Fatalf("cross-language matches = %#v", got)
	}
}

func TestComputeRelatedMMRDiversifiesNearDuplicates(t *testing.T) {
	entries := []Entry{
		{Path: "q", Language: "en", TranslationPath: "q", Vector: []float32{1, 0}},
		{Path: "a", Language: "en", TranslationPath: "a", Vector: []float32{0.99, 0.141067}},
		{Path: "b", Language: "en", TranslationPath: "b", Vector: []float32{0.98, 0.198997}},
		{Path: "diverse", Language: "en", TranslationPath: "d", Vector: []float32{0.8, -0.6}},
	}
	got := ComputeRelated(entries, Config{Count: 2, MinScore: 0, Diversity: 0.5})["q"]
	if len(got) != 2 || got[0].Path != "a" || got[1].Path != "diverse" {
		t.Fatalf("MMR matches = %#v", got)
	}
}

func TestTagBonusBreaksTie(t *testing.T) {
	entries := []Entry{
		{Path: "q", Language: "en", TranslationPath: "q", Tags: []string{"go"}, Vector: []float32{1, 0}},
		{Path: "a", Language: "en", TranslationPath: "a", Vector: []float32{0.8, 0.6}},
		{Path: "z", Language: "en", TranslationPath: "z", Tags: []string{"go"}, Vector: []float32{0.8, -0.6}},
	}
	got := ComputeRelated(entries, Config{Count: 1, MinScore: 0, Diversity: 1})["q"]
	if len(got) != 1 || got[0].Path != "z" {
		t.Fatalf("matches = %#v", got)
	}
}

func TestResolveManualPreservesOrder(t *testing.T) {
	entries := []Entry{{Path: "en/blog/a.md", URL: "/en/blog/a/", Title: "A"}, {Path: "en/blog/b.md", URL: "/en/blog/b/", Title: "B"}}
	got := ResolveManual([]string{"b", "en/blog/a.md"}, entries)
	want := []string{"en/blog/b.md", "en/blog/a.md"}
	paths := []string{got[0].Path, got[1].Path}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v", paths)
	}
}
