package service

import (
	"encoding/json"

	"cloud_ai_agent/internal/model"
)

// extractMcpNpmPackages parses tool DSLs and returns the npm package names
// for MCP servers that use "npx" as their command. These packages are then
// pre-installed in the Dockerfile at build time so the container doesn't
// need network access at runtime.
func extractMcpNpmPackages(store interface{ GetTool(id string) (*model.Tool, error) }, toolIDs []string) []string {
	seen := make(map[string]bool)
	var packages []string
	for _, tid := range toolIDs {
		tool, err := store.GetTool(tid)
		if err != nil || tool == nil {
			continue
		}
		var dsl struct {
			Handler struct {
				Type    string   `json:"type"`
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"handler"`
		}
		if json.Unmarshal([]byte(tool.DSLDefinition), &dsl) != nil {
			continue
		}
		if dsl.Handler.Type != "mcp" || dsl.Handler.Command != "npx" {
			continue
		}
		for i, arg := range dsl.Handler.Args {
			if arg == "-y" && i+1 < len(dsl.Handler.Args) {
				pkg := dsl.Handler.Args[i+1]
				if !seen[pkg] {
					seen[pkg] = true
					packages = append(packages, pkg)
				}
				break
			}
		}
	}
	return packages
}
