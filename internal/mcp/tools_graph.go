package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/quonaro/gnostis/internal/graph"
)

func tracePathTool() mcp.Tool {
	return mcp.NewTool("trace_path",
		mcp.WithDescription("Trace the call chain between two symbols (BFS over the call graph). Returns the shortest path found within max_depth."),
		mcp.WithString("from", mcp.Required(), mcp.Description("Caller symbol name")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Callee symbol name")),
		mcp.WithString("project", mcp.Description("Project name to restrict resolution")),
		mcp.WithNumber("max_depth", mcp.Description("Maximum hops"), mcp.DefaultNumber(10)),
	)
}

type tracePathArgs struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Project  string `json:"project,omitempty"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

func (s *Server) tracePath(ctx context.Context, _ mcp.CallToolRequest, args tracePathArgs) (*mcp.CallToolResult, error) {
	if args.From == "" || args.To == "" {
		return toolError(errReasonInvalidArgument, "from and to are required", "provide both symbol names"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	result, err := s.indexer.TracePath(ctx, args.From, args.To, args.Project, args.MaxDepth)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), "use find_symbol to verify the symbol name"), nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal trace result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func communitiesTool() mcp.Tool {
	return mcp.NewTool("get_communities",
		mcp.WithDescription("Detect communities (Louvain clustering) in the project call graph: groups of tightly coupled symbols with a label, size, and modularity score"),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project name from list_projects")),
		mcp.WithNumber("min_size", mcp.Description("Minimum community size"), mcp.DefaultNumber(1)),
		mcp.WithNumber("top_k", mcp.Description("Max communities returned"), mcp.DefaultNumber(30)),
		mcp.WithNumber("max_members", mcp.Description("Max member nodes listed per community"), mcp.DefaultNumber(20)),
	)
}

type communitiesArgs struct {
	Project    string `json:"project"`
	MinSize    int    `json:"min_size,omitempty"`
	TopK       int    `json:"top_k,omitempty"`
	MaxMembers int    `json:"max_members,omitempty"`
}

func (s *Server) communities(ctx context.Context, _ mcp.CallToolRequest, args communitiesArgs) (*mcp.CallToolResult, error) {
	if args.Project == "" {
		return toolError(errReasonInvalidArgument, "project is required", "use list_projects to see available projects"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	result, err := s.indexer.Communities(ctx, args.Project, args.MinSize, args.TopK, args.MaxMembers)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), "use list_projects to verify the project name"), nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal communities result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func deadCodeTool() mcp.Tool {
	return mcp.NewTool("dead_code",
		mcp.WithDescription("List symbols with no detected callers. Heuristic: entry points (main, init, tests, exported Go symbols) are excluded."),
		mcp.WithString("project", mcp.Required(), mcp.Description("Project name from list_projects")),
		mcp.WithString("kind", mcp.Description("Filter by kind: function, method. Default: both")),
		mcp.WithNumber("top_k", mcp.Description("Max results"), mcp.DefaultNumber(50)),
	)
}

type deadCodeArgs struct {
	Project string `json:"project"`
	Kind    string `json:"kind,omitempty"`
	TopK    int    `json:"top_k,omitempty"`
}

func (s *Server) deadCode(ctx context.Context, _ mcp.CallToolRequest, args deadCodeArgs) (*mcp.CallToolResult, error) {
	if args.Project == "" {
		return toolError(errReasonInvalidArgument, "project is required", "use list_projects to see available projects"), nil
	}

	if s.indexer == nil {
		return toolError(errReasonNotConfigured, "indexer is not configured", "add a project and run rebuild_index first"), nil
	}

	candidates, err := s.indexer.DeadCode(ctx, args.Project, args.Kind, args.TopK)
	if err != nil {
		return toolError(errReasonNotFound, err.Error(), "use list_projects to verify the project name"), nil
	}

	if candidates == nil {
		candidates = []graph.DeadCodeCandidate{}
	}
	data, err := json.Marshal(candidates)
	if err != nil {
		return toolError(errReasonSearchFailed, "marshal dead code result", ""), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
