package embeddings

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheSaveIsDeterministicAndRoundTrips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	cache := NewCache("text-embedding-3-small", 2)
	cache.Entries["z.md"] = Entry{Hash: "sha256:z", Vec: "eno=", Scale: 0.1, Lang: "en", URL: "/z/"}
	cache.Entries["a.md"] = Entry{Hash: "sha256:a", Vec: "YWE=", Scale: 0.2, Lang: "ru", URL: "/a/"}
	if err := cache.Save(path); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Index(first, []byte(`"a.md"`)) > bytes.Index(first, []byte(`"z.md"`)) {
		t.Fatalf("cache keys are not sorted:\n%s", first)
	}
	loaded, err := Load(path, "ignored", 9)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Model != cache.Model || loaded.Dims != cache.Dims || len(loaded.Entries) != 2 {
		t.Fatalf("Load() = %#v", loaded)
	}
	if err := loaded.Save(path); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("serialization changed:\nfirst=%s\nsecond=%s", first, second)
	}
}
