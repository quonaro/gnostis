package graph

import (
	"math"
	"math/rand"
	"path"
	"sort"
	"strings"

	"gonum.org/v1/gonum/graph/community"
	"gonum.org/v1/gonum/graph/simple"
)

// Community is a cluster of call-graph nodes detected by Louvain community
// detection over the resolved call graph.
type Community struct {
	ID            int    `json:"id"`
	Label         string `json:"label"`
	Size          int    `json:"size"`
	InternalEdges int    `json:"internal_edges"`
	Members       []Node `json:"members"`
	Truncated     bool   `json:"truncated,omitempty"`
}

// CommunityResult is the outcome of community detection on a call graph.
type CommunityResult struct {
	TotalNodes     int         `json:"total_nodes"`
	LinkedEdges    int         `json:"linked_edges"`
	Modularity     float64     `json:"modularity"`
	CommunityCount int         `json:"community_count"`
	Communities    []Community `json:"communities"`
}

// DetectCommunities runs Louvain community detection over the given nodes and
// resolved edges. Edges whose endpoints are not both present in nodes are
// ignored. The graph is treated as undirected and weighted by edge count.
// maxMembers caps the member list stored per community (0 = no cap); Size
// always reports the full member count. Communities are sorted by size
// descending. The result is deterministic for a given input.
func DetectCommunities(nodes []Node, edges []ResolvedEdge, maxMembers int) *CommunityResult {
	result := &CommunityResult{Communities: []Community{}}
	if len(nodes) == 0 {
		return result
	}
	result.TotalNodes = len(nodes)

	sorted := make([]Node, len(nodes))
	copy(sorted, nodes)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	index := make(map[string]int64, len(sorted))
	g := simple.NewWeightedUndirectedGraph(0, math.MaxInt64)
	for i, n := range sorted {
		index[n.ID] = int64(i)
		g.AddNode(simple.Node(int64(i)))
	}

	type pair struct{ a, b int64 }
	weights := make(map[pair]float64)
	for _, e := range edges {
		a, okA := index[e.From]
		b, okB := index[e.To]
		if !okA || !okB {
			continue
		}
		result.LinkedEdges++
		if a == b {
			continue
		}
		if a > b {
			a, b = b, a
		}
		weights[pair{a, b}]++
	}
	for p, w := range weights {
		g.SetWeightedEdge(simple.WeightedEdge{
			F: simple.Node(p.a),
			T: simple.Node(p.b),
			W: w,
		})
	}

	reduced := community.Modularize(g, 1.0, rand.New(rand.NewSource(1)))
	raw := reduced.Communities()
	result.Modularity = community.Q(g, raw, 1.0)

	commOf := make(map[string]int, len(sorted))
	members := make([][]Node, len(raw))
	for i, group := range raw {
		for _, gn := range group {
			n := sorted[gn.ID()]
			members[i] = append(members[i], n)
			commOf[n.ID] = i
		}
	}

	internal := make([]int, len(members))
	for _, e := range edges {
		cf, okF := commOf[e.From]
		ct, okT := commOf[e.To]
		if okF && okT && cf == ct {
			internal[cf]++
		}
	}

	result.Communities = make([]Community, 0, len(members))
	for i, group := range members {
		sort.Slice(group, func(a, b int) bool {
			if group[a].Path != group[b].Path {
				return group[a].Path < group[b].Path
			}
			return group[a].Symbol < group[b].Symbol
		})

		c := Community{
			Size:          len(group),
			InternalEdges: internal[i],
			Label:         communityLabel(group),
		}
		c.Members = group
		if maxMembers > 0 && len(group) > maxMembers {
			c.Members = group[:maxMembers]
			c.Truncated = true
		}
		result.Communities = append(result.Communities, c)
	}

	sort.Slice(result.Communities, func(i, j int) bool {
		a, b := result.Communities[i], result.Communities[j]
		if a.Size != b.Size {
			return a.Size > b.Size
		}
		if a.Label != b.Label {
			return a.Label < b.Label
		}
		return a.Members[0].Path < b.Members[0].Path
	})
	for i := range result.Communities {
		result.Communities[i].ID = i
	}
	result.CommunityCount = len(result.Communities)

	return result
}

// communityLabel derives a short human-readable label from the deepest
// directory shared by all member paths.
func communityLabel(members []Node) string {
	if len(members) == 0 {
		return ""
	}
	dir := path.Dir(members[0].Path)
	for _, m := range members[1:] {
		dir = commonPathPrefix(dir, path.Dir(m.Path))
		if dir == "" {
			break
		}
	}
	dir = strings.Trim(dir, "/")
	if dir == "" || dir == "." {
		return ""
	}
	parts := strings.Split(dir, "/")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "/")
}

// commonPathPrefix returns the longest common leading directory of a and b.
func commonPathPrefix(a, b string) string {
	as := strings.Split(strings.Trim(a, "/"), "/")
	bs := strings.Split(strings.Trim(b, "/"), "/")
	i := 0
	for i < len(as) && i < len(bs) && as[i] == bs[i] {
		i++
	}
	if i == 0 {
		return ""
	}
	return "/" + strings.Join(as[:i], "/")
}
