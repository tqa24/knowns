package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/howznguyen/knowns/internal/server/routes"
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Code intelligence commands",
	Long: `Code intelligence commands for AST-based indexing and graph analysis.

Recommended context flow:
  1. Use 'knowns code search' to find the most relevant symbol or entry point.
  2. Use 'knowns code symbols' to verify what was actually indexed in a file or scope.
  3. Use 'knowns code deps' to inspect raw relationships such as calls, imports, ownership, and inheritance.

Examples:
  knowns code ingest
  knowns code watch
  knowns code search backfill --neighbors 5
  knowns code deps --type calls
  knowns code symbols --kind function`,
}

var codeGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Inspect code graph data",
	RunE:  runCodeGraph,
}

var codeDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Inspect code dependency data",
	RunE:  runCodeDeps,
}

var codeSymbolsCmd = &cobra.Command{
	Use:   "symbols",
	Short: "Inspect indexed code symbols",
	RunE:  runCodeSymbols,
}

func runCodeGraph(cmd *cobra.Command, args []string) error {
	store := getStore()
	nodes, edges := routes.BuildCodeGraph(store)
	if nodes == nil {
		nodes = []routes.GraphNode{}
	}
	if edges == nil {
		edges = []routes.GraphEdge{}
	}

	if isJSON(cmd) {
		printJSON(map[string]any{"nodes": nodes, "edges": edges})
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", StyleBold.Render("Code Graph"))
	fmt.Fprintf(&b, "%s\n", RenderField("Nodes", fmt.Sprintf("%d", len(nodes))))
	fmt.Fprintf(&b, "%s\n\n", RenderField("Edges", fmt.Sprintf("%d", len(edges))))

	nodeCounts := map[string]int{}
	for _, node := range nodes {
		kind, _ := node.Data["kind"].(string)
		if kind == "" {
			kind = node.Type
		}
		nodeCounts[kind]++
	}
	if len(nodeCounts) > 0 {
		fmt.Fprintln(&b, RenderSectionHeader("Node Kinds"))
		keys := sortedMapKeys(nodeCounts)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s\n", RenderField(k, fmt.Sprintf("%d", nodeCounts[k])))
		}
		fmt.Fprintln(&b)
	}

	edgeCounts := map[string]int{}
	for _, edge := range edges {
		edgeCounts[edge.Type]++
	}
	if len(edgeCounts) > 0 {
		fmt.Fprintln(&b, RenderSectionHeader("Edge Types"))
		keys := sortedMapKeys(edgeCounts)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s\n", RenderField(k, fmt.Sprintf("%d", edgeCounts[k])))
		}
		fmt.Fprintln(&b)
	}

	if isPlain(cmd) {
		printPaged(cmd, b.String())
	} else {
		renderOrPage(cmd, "Code Graph", b.String())
	}
	return nil
}

func runCodeDeps(cmd *cobra.Command, args []string) error {
	store := getStore()
	db := store.SemanticDB()
	if db == nil {
		if isJSON(cmd) {
			printJSON([]map[string]any{})
		} else {
			fmt.Println(RenderWarning("No code dependency index found. Run 'knowns code ingest' first."))
		}
		return nil
	}
	defer db.Close()

	typeFilter, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 {
		limit = 200
	}

	rows, err := db.Query(`SELECT from_id, to_id, edge_type, from_path, to_path, raw_target, resolution_status, resolution_confidence FROM code_edges WHERE (? = '' OR edge_type = ?) ORDER BY from_id, edge_type, to_id LIMIT ?`, typeFilter, typeFilter, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var fromID, toID, edgeType, fromPath, toPath, rawTarget, status, confidence string
		if err := rows.Scan(&fromID, &toID, &edgeType, &fromPath, &toPath, &rawTarget, &status, &confidence); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"from":       fromID,
			"to":         toID,
			"type":       edgeType,
			"fromPath":   fromPath,
			"toPath":     toPath,
			"rawTarget":  rawTarget,
			"status":     status,
			"confidence": confidence,
		})
	}

	if isJSON(cmd) {
		printJSON(items)
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", StyleBold.Render("Code Dependencies"))
	fmt.Fprintf(&b, "%s\n", RenderField("Edges", fmt.Sprintf("%d", len(items))))
	if typeFilter != "" {
		fmt.Fprintf(&b, "%s\n", RenderField("Type filter", typeFilter))
	}
	fmt.Fprintln(&b)

	for _, item := range items {
		fmt.Fprintf(&b, "  %s %s\n",
			RenderBadge(strings.ToUpper(fmt.Sprint(item["type"])), colorBlue),
			StyleBold.Render(fmt.Sprintf("%s -> %s", item["from"], item["to"])))
		if fromPath := fmt.Sprint(item["fromPath"]); fromPath != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(fmt.Sprintf("from: %s", fromPath)))
		}
		if toPath := fmt.Sprint(item["toPath"]); toPath != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(fmt.Sprintf("to: %s", toPath)))
		}
		if rawTarget := fmt.Sprint(item["rawTarget"]); rawTarget != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(fmt.Sprintf("target: %s", rawTarget)))
		}
		status := fmt.Sprint(item["status"])
		confidence := fmt.Sprint(item["confidence"])
		if status != "" || confidence != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(fmt.Sprintf("resolution: %s %s", status, confidence)))
		}
		fmt.Fprintln(&b)
	}

	if isPlain(cmd) {
		printPaged(cmd, b.String())
	} else {
		renderOrPage(cmd, "Code Dependencies", b.String())
	}
	return nil
}

func runCodeSymbols(cmd *cobra.Command, args []string) error {
	store := getStore()
	db := store.SemanticDB()
	if db == nil {
		if isJSON(cmd) {
			printJSON([]map[string]any{})
		} else {
			fmt.Println(RenderWarning("No code symbol index found. Run 'knowns code ingest' first."))
		}
		return nil
	}
	defer db.Close()

	pathFilter, _ := cmd.Flags().GetString("path")
	kindFilter, _ := cmd.Flags().GetString("kind")
	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.Query(`SELECT id, doc_path, field, COALESCE(name, ''), COALESCE(signature, '') FROM chunks WHERE type = 'code' AND (? = '' OR doc_path = ?) AND (? = '' OR field = ?) ORDER BY doc_path, name, id LIMIT ?`, pathFilter, pathFilter, kindFilter, kindFilter, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id, docPath, kind, name, signature string
		if err := rows.Scan(&id, &docPath, &kind, &name, &signature); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":        id,
			"path":      docPath,
			"kind":      kind,
			"name":      name,
			"signature": signature,
		})
	}

	if isJSON(cmd) {
		printJSON(items)
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", StyleBold.Render("Code Symbols"))
	fmt.Fprintf(&b, "%s\n", RenderField("Symbols", fmt.Sprintf("%d", len(items))))
	if pathFilter != "" {
		fmt.Fprintf(&b, "%s\n", RenderField("Path filter", pathFilter))
	}
	if kindFilter != "" {
		fmt.Fprintf(&b, "%s\n", RenderField("Kind filter", kindFilter))
	}
	fmt.Fprintln(&b)

	for _, item := range items {
		name := fmt.Sprint(item["name"])
		if name == "" {
			name = fmt.Sprint(item["id"])
		}
		fmt.Fprintf(&b, "  %s %s\n",
			RenderBadge(strings.ToUpper(fmt.Sprint(item["kind"])), colorMagenta),
			StyleBold.Render(name))
		fmt.Fprintf(&b, "    %s\n", StyleDim.Render(fmt.Sprintf("path: %s", item["path"])))
		if sig := fmt.Sprint(item["signature"]); sig != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(sig))
		}
		fmt.Fprintln(&b)
	}

	if isPlain(cmd) {
		printPaged(cmd, b.String())
	} else {
		renderOrPage(cmd, "Code Symbols", b.String())
	}
	return nil
}

func sortedMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	codeDepsCmd.Flags().String("type", "", "Filter dependency edges by type")
	codeDepsCmd.Flags().Int("limit", 200, "Limit dependency results")
	codeSymbolsCmd.Flags().String("path", "", "Filter symbols by file path")
	codeSymbolsCmd.Flags().String("kind", "", "Filter symbols by kind")
	codeSymbolsCmd.Flags().Int("limit", 100, "Limit symbol results")

	codeCmd.AddCommand(ingestCmd)
	codeCmd.AddCommand(watchCmd)
	codeCmd.AddCommand(codeSearchCmd)
	codeCmd.AddCommand(codeDepsCmd)
	codeCmd.AddCommand(codeSymbolsCmd)
	rootCmd.AddCommand(codeCmd)

	ingestCmd.Hidden = false
	watchCmd.Hidden = false
	codeSearchCmd.Hidden = false
	codeDepsCmd.Hidden = false
	codeSymbolsCmd.Hidden = false

	codeSearchCmd.Flags().Int("limit", 10, "Limit code matches")
	codeSearchCmd.Flags().Int("neighbors", 5, "Max neighbors per match (1-hop)")
	codeSearchCmd.Flags().String("edge-types", "", "Comma-separated edge types to expand")
	codeSearchCmd.Flags().Bool("keyword", false, "Force keyword-only search")
	codeSearchCmd.Flags().Bool("show-snippet", false, "Show code snippet/preview for each match")
}
