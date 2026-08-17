package parser

import (
	"reflect"
	"testing"
)

func TestParseFrontmatterRelatedControls(t *testing.T) {
	fm, _, err := ParseFrontmatter("---\nrelated: [first-post, blog/second.md]\nhideRelated: true\n---\nBody")
	if err != nil {
		t.Fatal(err)
	}
	if !fm.HideRelated {
		t.Fatal("HideRelated = false")
	}
	if !reflect.DeepEqual(fm.Related, []string{"first-post", "blog/second.md"}) {
		t.Fatalf("Related = %#v", fm.Related)
	}
	if _, ok := fm.Params["related"]; !ok {
		t.Fatal("related missing from raw params")
	}
	if _, ok := fm.Params["hideRelated"]; !ok {
		t.Fatal("hideRelated missing from raw params")
	}
}

func TestParseFrontmatterEmbedding(t *testing.T) {
	fm, _, err := ParseFrontmatter("---\nembedding:\n  version: 1\n  model: text-embedding-3-small\n  dimensions: 2\n  hash: sha256:abc\n  vector: AQI=\n  scale: 0.01\n---\nBody")
	if err != nil {
		t.Fatal(err)
	}
	if fm.Embedding == nil || fm.Embedding.Version != 1 || fm.Embedding.Dimensions != 2 || fm.Embedding.Vector != "AQI=" || fm.Embedding.Hash != "sha256:abc" {
		t.Fatalf("Embedding = %#v", fm.Embedding)
	}
}
