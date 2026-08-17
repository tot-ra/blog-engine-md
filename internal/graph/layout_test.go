package graph

import (
	"math"
	"testing"
)

func TestProjectPCA3DSeparatesOrthogonalVectors(t *testing.T) {
	vectors := [][]float32{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{-1, 0, 0, 0},
	}
	got := projectPCA3D(vectors)
	if len(got) != 4 {
		t.Fatalf("expected 4 points, got %d", len(got))
	}
	// Opposite vectors along the first axis should land far apart after scaling.
	dist := math.Hypot(math.Hypot(got[0][0]-got[3][0], got[0][1]-got[3][1]), got[0][2]-got[3][2])
	if dist < layoutExtent {
		t.Fatalf("expected opposite vectors to be well separated, dist=%v positions=%v", dist, got)
	}
}

func TestAssignEmbeddingLayoutUsesVectorsAndTagCentroids(t *testing.T) {
	pages := []PageInfo{
		{ID: "a", Title: "A", URL: "/a/", Type: "blog", Tags: []string{"go"}, Vector: []float32{1, 0, 0}},
		{ID: "b", Title: "B", URL: "/b/", Type: "blog", Tags: []string{"go"}, Vector: []float32{-1, 0, 0}},
	}
	g := BuildGraph(pages)

	byID := map[string]GraphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	a, okA := byID["a"]
	b, okB := byID["b"]
	tag, okTag := byID["tag-go"]
	if !okA || !okB || !okTag {
		t.Fatalf("missing nodes: %#v", byID)
	}
	if a.X == 0 && a.Y == 0 && a.Z == 0 && b.X == 0 && b.Y == 0 && b.Z == 0 {
		t.Fatal("expected embedding-backed articles to leave the origin")
	}
	// Tag should sit between its articles (centroid), not at a random sphere point.
	midX := (a.X + b.X) / 2
	midY := (a.Y + b.Y) / 2
	midZ := (a.Z + b.Z) / 2
	if math.Abs(tag.X-midX) > 0.01 || math.Abs(tag.Y-midY) > 0.01 || math.Abs(tag.Z-midZ) > 0.01 {
		t.Fatalf("tag centroid mismatch: tag=%v mid=(%v,%v,%v)", tag, midX, midY, midZ)
	}
}

func TestAssignEmbeddingLayoutIsDeterministicWithoutVectors(t *testing.T) {
	pages := []PageInfo{
		{ID: "a", Title: "A", URL: "/a/", Type: "page"},
		{ID: "b", Title: "B", URL: "/b/", Type: "page"},
	}
	g1 := BuildGraph(pages)
	g2 := BuildGraph(pages)
	if len(g1.Nodes) != len(g2.Nodes) {
		t.Fatalf("node count mismatch")
	}
	for i := range g1.Nodes {
		if g1.Nodes[i].X != g2.Nodes[i].X || g1.Nodes[i].Y != g2.Nodes[i].Y || g1.Nodes[i].Z != g2.Nodes[i].Z {
			t.Fatalf("non-deterministic layout at %d: %#v vs %#v", i, g1.Nodes[i], g2.Nodes[i])
		}
	}
}
