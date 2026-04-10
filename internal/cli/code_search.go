package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/search"
	"github.com/spf13/cobra"
)

var codeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search indexed code with optional neighbors",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCodeSearch,
}

func runCodeSearch(cmd *cobra.Command, args []string) error {
	store := getStore()
	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")
	if limit <= 0 {
		limit = 10
	}
	neighbors, _ := cmd.Flags().GetInt("neighbors")
	edgeTypes := splitCSV(getFlagString(cmd, "edge-types"))
	keywordOnly, _ := cmd.Flags().GetBool("keyword")
	showSnippet, _ := cmd.Flags().GetBool("show-snippet")
	mode := "hybrid"
	if keywordOnly {
		mode = "keyword"
	}

	embedder, vecStore, _ := initSemanticSearchReal()
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}

	graph, err := search.SearchCodeWithNeighbors(store, embedder, vecStore, models.RetrievalOptions{
		Query: query,
		Mode:  mode,
		Limit: limit,
	}, edgeTypes, neighbors)
	if err != nil {
		return err
	}

	if isJSON(cmd) {
		printJSON(graph)
		return nil
	}

	if len(graph.Matches) == 0 {
		fmt.Println(RenderWarning(fmt.Sprintf("No code results for %q", query)))
		return nil
	}

	nodeByID := make(map[string]search.Chunk, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", StyleBold.Render("Code Search"))
	fmt.Fprintf(&b, "%s\n", RenderField("Query", query))
	fmt.Fprintf(&b, "%s\n", RenderField("Matches", fmt.Sprintf("%d", len(graph.Matches))))
	fmt.Fprintf(&b, "%s\n", RenderField("Neighbor nodes", fmt.Sprintf("%d", len(graph.Nodes))))
	fmt.Fprintf(&b, "%s\n\n", RenderField("Neighbor edges", fmt.Sprintf("%d", len(graph.Edges))))

	fmt.Fprintln(&b, RenderSectionHeader("Matches"))
	for _, item := range graph.Matches {
		title := item.Name
		if title == "" {
			title = item.ID
		}
		fmt.Fprintf(&b, "  %s %s\n", RenderBadge(strings.ToUpper(item.Type), colorBlue), StyleBold.Render(title))
		if item.Path != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(item.Path))
		}
		if item.Signature != "" {
			fmt.Fprintf(&b, "    %s\n", StyleDim.Render(item.Signature))
		}
		if showSnippet && item.Snippet != "" {
			fmt.Fprintf(&b, "    %s\n", strings.TrimSpace(item.Snippet))
		}

		children := buildMatchTree(item.ID, graph.Edges, nodeByID)
		if len(children) > 0 {
			for _, line := range renderTreeChildren(children, "    ") {
				fmt.Fprintln(&b, line)
			}
		}
		fmt.Fprintln(&b)
	}

	if graph.Truncated {
		fmt.Fprintln(&b, RenderHint("Some neighbors were truncated. Increase --neighbors to expand more edges."))
	}

	if isPlain(cmd) {
		printPaged(cmd, b.String())
	} else {
		renderOrPage(cmd, fmt.Sprintf("Code Search: %s", query), b.String())
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type treeNode struct {
	label    string
	children []treeNode
}

func buildMatchTree(matchID string, edges []search.CodeNeighborEdge, nodeByID map[string]search.Chunk) []treeNode {
	grouped := make(map[string][]string)
	for _, edge := range edges {
		if edge.From != matchID && edge.To != matchID {
			continue
		}
		relation := edge.Type
		otherID := edge.To
		if edge.To == matchID {
			relation = relation + " (incoming)"
			otherID = edge.From
		}
		label := otherID
		if chunk, ok := nodeByID[otherID]; ok {
			parts := []string{firstNonEmptyString(chunk.Name, chunk.ID)}
			if chunk.DocPath != "" {
				parts = append(parts, chunk.DocPath)
			}
			label = strings.Join(parts, " - ")
		}
		grouped[relation] = append(grouped[relation], label)
	}
	if len(grouped) == 0 {
		return nil
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	nodes := make([]treeNode, 0, len(keys))
	for _, key := range keys {
		labels := grouped[key]
		sort.Strings(labels)
		children := make([]treeNode, 0, len(labels))
		for _, label := range labels {
			children = append(children, treeNode{label: label})
		}
		nodes = append(nodes, treeNode{label: key, children: children})
	}
	return nodes
}

func renderTreeChildren(nodes []treeNode, prefix string) []string {
	lines := make([]string, 0)
	for i, node := range nodes {
		branch := "├─ "
		nextPrefix := prefix + "│  "
		if i == len(nodes)-1 {
			branch = "└─ "
			nextPrefix = prefix + "   "
		}
		lines = append(lines, prefix+branch+node.label)
		if len(node.children) > 0 {
			lines = append(lines, renderTreeChildren(node.children, nextPrefix)...)
		}
	}
	return lines
}
