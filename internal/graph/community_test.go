package graph

import (
	"testing"
)

func commNode(id string) Node {
	return Node{ID: id, Path: "/proj/" + id + ".go", Symbol: id, Kind: "function", Language: "go"}
}

func commEdge(from, to string) ResolvedEdge {
	return ResolvedEdge{From: from, To: to}
}

func TestDetectCommunities_TwoClusters(t *testing.T) {
	// Two triangles joined by a single bridge edge.
	nodes := []Node{
		commNode("a1"), commNode("a2"), commNode("a3"),
		commNode("b1"), commNode("b2"), commNode("b3"),
	}
	edges := []ResolvedEdge{
		commEdge("a1", "a2"), commEdge("a2", "a3"), commEdge("a1", "a3"),
		commEdge("b1", "b2"), commEdge("b2", "b3"), commEdge("b1", "b3"),
		commEdge("a1", "b1"),
	}

	res := DetectCommunities(nodes, edges, 0)
	if res.CommunityCount != 2 {
		t.Fatalf("expected 2 communities, got %d: %+v", res.CommunityCount, res.Communities)
	}
	if res.Modularity <= 0 {
		t.Errorf("expected positive modularity, got %v", res.Modularity)
	}
	for _, c := range res.Communities {
		if c.Size != 3 {
			t.Errorf("community %d: expected size 3, got %d", c.ID, c.Size)
		}
	}
	total := 0
	for _, c := range res.Communities {
		total += c.Size
	}
	if total != len(nodes) {
		t.Errorf("community sizes sum to %d, want %d", total, len(nodes))
	}
}

func TestDetectCommunities_Empty(t *testing.T) {
	res := DetectCommunities(nil, nil, 0)
	if res.TotalNodes != 0 || res.CommunityCount != 0 {
		t.Errorf("unexpected result: %+v", res)
	}
	if res.Communities == nil {
		t.Error("expected non-nil empty communities slice")
	}
}

func TestDetectCommunities_IsolatedNodes(t *testing.T) {
	nodes := []Node{commNode("x"), commNode("y"), commNode("z")}
	res := DetectCommunities(nodes, nil, 0)
	if res.CommunityCount != 3 {
		t.Fatalf("expected 3 singleton communities, got %d", res.CommunityCount)
	}
	for _, c := range res.Communities {
		if c.Size != 1 {
			t.Errorf("expected singleton, got size %d", c.Size)
		}
	}
}

func TestDetectCommunities_MaxMembersTruncates(t *testing.T) {
	nodes := []Node{
		commNode("a1"), commNode("a2"), commNode("a3"), commNode("a4"),
	}
	edges := []ResolvedEdge{
		commEdge("a1", "a2"), commEdge("a2", "a3"), commEdge("a3", "a4"),
		commEdge("a1", "a3"), commEdge("a2", "a4"),
	}

	res := DetectCommunities(nodes, edges, 2)
	if res.CommunityCount != 1 {
		t.Fatalf("expected 1 community, got %d", res.CommunityCount)
	}
	c := res.Communities[0]
	if c.Size != 4 {
		t.Errorf("Size = %d, want 4", c.Size)
	}
	if len(c.Members) != 2 {
		t.Errorf("len(Members) = %d, want 2", len(c.Members))
	}
	if !c.Truncated {
		t.Error("expected Truncated=true")
	}
}

func TestDetectCommunities_Label(t *testing.T) {
	nodes := []Node{
		{ID: "/proj/internal/graph/a.go:f1", Path: "/proj/internal/graph/a.go", Symbol: "f1", Kind: "function"},
		{ID: "/proj/internal/graph/b.go:f2", Path: "/proj/internal/graph/b.go", Symbol: "f2", Kind: "function"},
		{ID: "/proj/internal/graph/c.go:f3", Path: "/proj/internal/graph/c.go", Symbol: "f3", Kind: "function"},
	}
	edges := []ResolvedEdge{
		commEdge("/proj/internal/graph/a.go:f1", "/proj/internal/graph/b.go:f2"),
		commEdge("/proj/internal/graph/b.go:f2", "/proj/internal/graph/c.go:f3"),
	}

	res := DetectCommunities(nodes, edges, 0)
	if res.CommunityCount != 1 {
		t.Fatalf("expected 1 community, got %d", res.CommunityCount)
	}
	// common dir is /proj/internal/graph -> last two segments "internal/graph"
	if got := res.Communities[0].Label; got != "internal/graph" {
		t.Errorf("Label = %q, want %q", got, "internal/graph")
	}
}

func TestDetectCommunities_IgnoresForeignEdges(t *testing.T) {
	nodes := []Node{commNode("a"), commNode("b")}
	edges := []ResolvedEdge{
		commEdge("a", "b"),
		commEdge("a", "missing"),
		commEdge("missing", "b"),
	}
	res := DetectCommunities(nodes, edges, 0)
	if res.LinkedEdges != 1 {
		t.Errorf("LinkedEdges = %d, want 1", res.LinkedEdges)
	}
	if res.CommunityCount != 1 {
		t.Fatalf("expected 1 community, got %d", res.CommunityCount)
	}
}
