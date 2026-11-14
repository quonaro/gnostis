package app

import (
	"context"
	"testing"
)

func TestCommunities_ProjectCommunities(t *testing.T) {
	a := testAppWithGraph()

	res, err := a.Communities(context.Background(), "test", 1, 30, 20)
	if err != nil {
		t.Fatalf("Communities: %v", err)
	}
	if res.TotalNodes != 7 {
		t.Errorf("TotalNodes = %d, want 7", res.TotalNodes)
	}
	total := 0
	for _, c := range res.Communities {
		total += c.Size
	}
	if total != 7 {
		t.Errorf("community sizes sum to %d, want 7", total)
	}
	if res.CommunityCount != len(res.Communities) {
		t.Errorf("CommunityCount = %d, len = %d", res.CommunityCount, len(res.Communities))
	}
}

func TestCommunities_MinSizeFilters(t *testing.T) {
	a := testAppWithGraph()

	res, err := a.Communities(context.Background(), "test", 2, 30, 20)
	if err != nil {
		t.Fatalf("Communities: %v", err)
	}
	for _, c := range res.Communities {
		if c.Size < 2 {
			t.Errorf("community size %d below min_size=2", c.Size)
		}
	}
}

func TestCommunities_TopKCaps(t *testing.T) {
	a := testAppWithGraph()

	res, err := a.Communities(context.Background(), "test", 1, 1, 20)
	if err != nil {
		t.Fatalf("Communities: %v", err)
	}
	if len(res.Communities) != 1 {
		t.Fatalf("expected 1 community with top_k=1, got %d", len(res.Communities))
	}
}

func TestCommunities_UnknownProject(t *testing.T) {
	a := testAppWithGraph()

	_, err := a.Communities(context.Background(), "nope", 1, 30, 20)
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
}
