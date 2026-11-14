package app

import (
	"context"
	"fmt"

	"github.com/quonaro/gnostis/internal/graph"
)

// Communities detects Louvain communities in the project's call graph.
// minSize filters out smaller communities, topK caps the result count, and
// maxMembers caps the member nodes listed per community.
func (a *App) Communities(_ context.Context, projectName string, minSize, topK, maxMembers int) (*graph.CommunityResult, error) {
	projectPath, err := a.ProjectPath(projectName)
	if err != nil {
		return nil, fmt.Errorf("project %q: %w", projectName, err)
	}

	if minSize <= 0 {
		minSize = 1
	}
	if topK <= 0 {
		topK = 30
	}
	if maxMembers <= 0 {
		maxMembers = 20
	}

	nodes := a.callGraph.NodesByProject(projectPath)
	nodeSet := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeSet[n.ID] = true
	}

	var edges []graph.ResolvedEdge
	for _, e := range a.callGraph.ResolveEdges() {
		if nodeSet[e.From] && nodeSet[e.To] {
			edges = append(edges, e)
		}
	}

	result := graph.DetectCommunities(nodes, edges, maxMembers)

	filtered := make([]graph.Community, 0, len(result.Communities))
	for _, c := range result.Communities {
		if c.Size >= minSize {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}
	result.Communities = filtered
	result.CommunityCount = len(filtered)

	return result, nil
}
