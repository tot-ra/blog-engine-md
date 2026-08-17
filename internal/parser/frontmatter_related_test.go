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
