package adk

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentmesh/agentmesh/pkg/graph"
)

// ProjectInspectionResult holds the extracted graph and any diagnostic notes or warnings.
type ProjectInspectionResult struct {
	Graph                 *graph.AgentGraph `json:"graph"`
	SourcePath            string            `json:"sourcePath"`
	FilesScanned          int               `json:"filesScanned"`
	IdentifiedTools       []string          `json:"identifiedTools"`
	IdentifiedDelegations []string          `json:"identifiedDelegations"`
	IdentifiedApprovals   []string          `json:"identifiedApprovals"`
	UnsupportedConstructs []string          `json:"unsupportedConstructs"`
}

// StaticProjectInspector analyzes a Go project statically using AST inspection without executing code.
type StaticProjectInspector struct{}

// NewStaticProjectInspector creates an inspector instance.
func NewStaticProjectInspector() *StaticProjectInspector {
	return &StaticProjectInspector{}
}

// InspectProject recursively parses all Go files in projectRoot and extracts an AgentGraph.
func (spi *StaticProjectInspector) InspectProject(projectRoot, agentID, orgID string) (*ProjectInspectionResult, error) {
	info, err := os.Stat(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot access path %s: %w", projectRoot, err)
	}

	result := &ProjectInspectionResult{
		SourcePath:            projectRoot,
		IdentifiedTools:       make([]string, 0),
		IdentifiedDelegations: make([]string, 0),
		IdentifiedApprovals:   make([]string, 0),
		UnsupportedConstructs: make([]string, 0),
	}

	g := graph.NewAgentGraph(
		fmt.Sprintf("graph_%s", agentID),
		orgID,
		"default",
		agentID,
		"1.0.0",
	)

	// Collect files
	var goFiles []string
	if !info.IsDir() {
		if strings.HasSuffix(projectRoot, ".go") {
			goFiles = append(goFiles, projectRoot)
		}
	} else {
		err := filepath.Walk(projectRoot, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !f.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				goFiles = append(goFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking directory: %w", err)
		}
	}

	result.FilesScanned = len(goFiles)
	if len(goFiles) == 0 {
		return nil, errors.New("no Go source files found in project path")
	}

	fset := token.NewFileSet()
	entrypointNode := "node_entrypoint"
	g.Entrypoint = entrypointNode
	g.Nodes = append(g.Nodes, graph.Node{
		ID:          entrypointNode,
		Name:        fmt.Sprintf("%s Entrypoint", agentID),
		Type:        graph.NodeTypeAgent,
		Description: "Workflow starting point detected by ADK AST inspector",
	})

	lastNodeID := entrypointNode
	nodeCounter := 1

	for _, file := range goFiles {
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			result.UnsupportedConstructs = append(result.UnsupportedConstructs, fmt.Sprintf("Failed to parse file %s: %v", filepath.Base(file), err))
			continue
		}

		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			fnName := spi.extractFunctionName(call)
			if fnName == "" {
				return true
			}

			// Recognize ADK / Agent patterns:
			// Tool calls: WithTool("bigquery.read"), CallTool("bigquery.read"), Use("bigquery.read")
			if strings.Contains(strings.ToLower(fnName), "tool") || strings.EqualFold(fnName, "Use") {
				argStr := spi.extractFirstStringLiteral(call)
				if argStr != "" {
					result.IdentifiedTools = append(result.IdentifiedTools, argStr)
					nodeID := fmt.Sprintf("tool_%d", nodeCounter)
					nodeCounter++
					g.Nodes = append(g.Nodes, graph.Node{
						ID:          nodeID,
						Name:        argStr,
						Type:        graph.NodeTypeTool,
						Target:      argStr,
						Description: fmt.Sprintf("Tool invocation: %s", argStr),
					})
					g.Edges = append(g.Edges, graph.Edge{FromID: lastNodeID, ToID: nodeID})
					lastNodeID = nodeID
				}
			}

			// Delegation calls: DelegateTo("finance-agent"), AddDelegate("finance-agent")
			if strings.Contains(strings.ToLower(fnName), "delegate") {
				argStr := spi.extractFirstStringLiteral(call)
				if argStr != "" {
					result.IdentifiedDelegations = append(result.IdentifiedDelegations, argStr)
					nodeID := fmt.Sprintf("delegation_%d", nodeCounter)
					nodeCounter++
					g.Nodes = append(g.Nodes, graph.Node{
						ID:          nodeID,
						Name:        argStr,
						Type:        graph.NodeTypeDelegation,
						Target:      argStr,
						Description: fmt.Sprintf("Delegation peer: %s", argStr),
					})
					g.Edges = append(g.Edges, graph.Edge{FromID: lastNodeID, ToID: nodeID})
					lastNodeID = nodeID
				}
			}

			// Human approval calls: RequireApproval("purchase_request")
			if strings.Contains(strings.ToLower(fnName), "approval") {
				argStr := spi.extractFirstStringLiteral(call)
				if argStr != "" {
					result.IdentifiedApprovals = append(result.IdentifiedApprovals, argStr)
					nodeID := fmt.Sprintf("approval_%d", nodeCounter)
					nodeCounter++
					g.Nodes = append(g.Nodes, graph.Node{
						ID:          nodeID,
						Name:        argStr,
						Type:        graph.NodeTypeHumanApproval,
						Target:      argStr,
						Description: fmt.Sprintf("Human-in-the-loop approval checkpoint: %s", argStr),
					})
					g.Edges = append(g.Edges, graph.Edge{FromID: lastNodeID, ToID: nodeID})
					lastNodeID = nodeID
				}
			}

			return true
		})
	}

	g.Tools = deduplicate(result.IdentifiedTools)
	g.Delegations = deduplicate(result.IdentifiedDelegations)
	g.ApprovalPoints = deduplicate(result.IdentifiedApprovals)

	if len(g.Nodes) > 1 {
		g.Exitpoints = []string{lastNodeID}
	} else {
		g.Exitpoints = []string{entrypointNode}
	}

	result.Graph = g
	return result, nil
}

func (spi *StaticProjectInspector) extractFunctionName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	default:
		return ""
	}
}

func (spi *StaticProjectInspector) extractFirstStringLiteral(call *ast.CallExpr) string {
	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			// strip surrounding quotes
			return strings.Trim(lit.Value, `"'` + "`")
		}
	}
	return ""
}

func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			res = append(res, item)
		}
	}
	return res
}
