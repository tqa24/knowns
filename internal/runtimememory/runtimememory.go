package runtimememory

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/howznguyen/knowns/internal/models"
	"github.com/howznguyen/knowns/internal/storage"
)

const (
	ModeOff    = "off"
	ModeAuto   = "auto"
	ModeManual = "manual"
	ModeDebug  = "debug"

	StatusNone      = "none"
	StatusCandidate = "candidate"
	StatusInjected  = "injected"

	HookNative            = "native"
	HookProxyPreExecution = "proxy-pre-execution"
	HookWrapper           = "wrapper"
	HookAdapter           = "adapter"

	HeaderMode   = "X-Knowns-Runtime-Memory-Mode"
	HeaderInject = "X-Knowns-Runtime-Memory-Inject"
	HeaderStatus = "X-Knowns-Runtime-Memory-Status"
	HeaderItems  = "X-Knowns-Runtime-Memory-Items"
	HeaderPack   = "X-Knowns-Runtime-Memory-Pack"
)

const canonicalityWarning = "Knowns memory is supplemental context only and does not override KNOWNS.md, source-of-truth docs, tasks, or source files."

const (
	defaultMaxItems = 5
	defaultMaxBytes = 2500
	maxPreviewBody  = 320
)

var tokenRE = regexp.MustCompile(`[a-z0-9]+`)

type Settings struct {
	Mode     string
	MaxItems int
	MaxBytes int
}

type Input struct {
	Runtime     string
	ProjectRoot string
	WorkingDir  string
	ActionType  string
	UserPrompt  string
	Mode        string
	MaxItems    int
	MaxBytes    int
}

type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Layer     string    `json:"layer"`
	UpdatedAt time.Time `json:"updatedAt"`
	Content   string    `json:"content"`
	Score     float64   `json:"score"`
	Reasons   []string  `json:"reasons,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}

type Pack struct {
	Runtime string `json:"runtime"`
	Mode    string `json:"mode"`
	Status  string `json:"status"`

	Warning    string `json:"warning"`
	Items      []Item `json:"items"`
	Serialized string `json:"serialized,omitempty"`
	Bytes      int    `json:"bytes"`
}

type Adapter struct {
	Runtime        string   `json:"runtime"`
	DisplayName    string   `json:"displayName"`
	HookKind       string   `json:"hookKind"`
	NativeHooks    bool     `json:"nativeHooks"`
	SupportedModes []string `json:"supportedModes"`
}

func DefaultAdapters() []Adapter {
	return []Adapter{
		{
			Runtime:        "kiro",
			DisplayName:    "Kiro",
			HookKind:       HookNative,
			NativeHooks:    true,
			SupportedModes: []string{ModeOff, ModeAuto, ModeManual, ModeDebug},
		},
		{
			Runtime:        "claude-code",
			DisplayName:    "Claude Code",
			HookKind:       HookWrapper,
			SupportedModes: []string{ModeOff, ModeAuto, ModeManual, ModeDebug},
		},
		{
			Runtime:        "opencode",
			DisplayName:    "OpenCode",
			HookKind:       HookProxyPreExecution,
			SupportedModes: []string{ModeOff, ModeAuto, ModeManual, ModeDebug},
		},
		{
			Runtime:        "antigravity",
			DisplayName:    "Antigravity",
			HookKind:       HookAdapter,
			SupportedModes: []string{ModeOff, ModeAuto, ModeManual, ModeDebug},
		},
	}
}

func LookupAdapter(runtime string) (Adapter, bool) {
	runtime = strings.TrimSpace(strings.ToLower(runtime))
	for _, adapter := range DefaultAdapters() {
		if adapter.Runtime == runtime {
			return adapter, true
		}
	}
	return Adapter{}, false
}

func NormalizeSettings(cfg *models.RuntimeMemorySettings) Settings {
	settings := Settings{
		Mode:     ModeAuto,
		MaxItems: defaultMaxItems,
		MaxBytes: defaultMaxBytes,
	}
	if cfg == nil {
		return settings
	}
	if mode := NormalizeMode(cfg.Mode); mode != "" {
		settings.Mode = mode
	}
	if cfg.MaxItems > 0 {
		settings.MaxItems = cfg.MaxItems
	}
	if cfg.MaxBytes > 0 {
		settings.MaxBytes = cfg.MaxBytes
	}
	return settings
}

func NormalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ModeOff:
		return ModeOff
	case ModeManual:
		return ModeManual
	case ModeDebug:
		return ModeDebug
	case "", ModeAuto:
		return ModeAuto
	default:
		return ModeAuto
	}
}

func Build(store *storage.Store, input Input) (Pack, error) {
	pack := Pack{
		Runtime: input.Runtime,
		Mode:    NormalizeMode(input.Mode),
		Status:  StatusNone,
		Warning: canonicalityWarning,
	}
	if store == nil {
		return pack, nil
	}
	if _, ok := LookupAdapter(input.Runtime); !ok {
		return pack, fmt.Errorf("unsupported runtime adapter: %s", input.Runtime)
	}
	entries, err := store.Memory.List("")
	if err != nil {
		return pack, err
	}

	maxItems := input.MaxItems
	if maxItems <= 0 {
		maxItems = defaultMaxItems
	}
	maxBytes := input.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	type candidate struct {
		item Item
	}
	var candidates []candidate
	for _, entry := range entries {
		if !allowedCategory(entry.Category) {
			continue
		}
		score, reasons := scoreEntry(entry, input)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, candidate{item: Item{
			ID:        entry.ID,
			Title:     entry.Title,
			Category:  entry.Category,
			Layer:     entry.Layer,
			UpdatedAt: entry.UpdatedAt,
			Content:   normalizeWhitespace(entry.Content),
			Score:     score,
			Reasons:   reasons,
			Tags:      append([]string(nil), entry.Tags...),
		}})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].item.Score != candidates[j].item.Score {
			return candidates[i].item.Score > candidates[j].item.Score
		}
		if !candidates[i].item.UpdatedAt.Equal(candidates[j].item.UpdatedAt) {
			return candidates[i].item.UpdatedAt.After(candidates[j].item.UpdatedAt)
		}
		return candidates[i].item.ID < candidates[j].item.ID
	})

	selected := make([]Item, 0, maxItems)
	serialized := serializePrefix()
	for _, candidate := range candidates {
		if len(selected) >= maxItems {
			break
		}
		item := candidate.item
		remaining := maxBytes - len(serialized)
		if remaining <= 0 {
			break
		}
		block := serializeItem(item, remaining)
		if block == "" {
			continue
		}
		selected = append(selected, item)
		serialized += block
	}

	if len(selected) == 0 {
		pack.Bytes = len(serializePrefix())
		return pack, nil
	}

	pack.Items = selected
	pack.Serialized = serialized
	pack.Bytes = len(serialized)
	pack.Status = StatusCandidate
	return pack, nil
}

func InjectSystemPrompt(existingSystem, serialized string) string {
	serialized = strings.TrimSpace(serialized)
	if serialized == "" {
		return existingSystem
	}
	if strings.TrimSpace(existingSystem) == "" {
		return serialized
	}
	return strings.TrimSpace(existingSystem) + "\n\n" + serialized
}

func EncodePackHeader(pack Pack) string {
	preview := struct {
		Runtime string `json:"runtime"`
		Mode    string `json:"mode"`
		Status  string `json:"status"`
		Warning string `json:"warning"`
		Items   []Item `json:"items,omitempty"`
	}{
		Runtime: pack.Runtime,
		Mode:    pack.Mode,
		Status:  pack.Status,
		Warning: pack.Warning,
		Items:   pack.Items,
	}
	data, err := json.Marshal(preview)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func serializePrefix() string {
	return "Knowns Memory Pack\n" + canonicalityWarning + "\n"
}

func serializeItem(item Item, remaining int) string {
	if remaining <= 0 {
		return ""
	}
	content := item.Content
	if len(content) > maxPreviewBody {
		content = strings.TrimSpace(content[:maxPreviewBody]) + "..."
	}
	base := fmt.Sprintf("\n- %s [%s/%s] updated %s\n", item.Title, item.Category, item.Layer, item.UpdatedAt.UTC().Format(time.RFC3339))
	reason := ""
	if len(item.Reasons) > 0 {
		reason = "  Why: " + strings.Join(item.Reasons, "; ") + "\n"
	}
	bodyPrefix := "  Memory: "
	block := base + reason + bodyPrefix + content + "\n"
	if len(block) <= remaining {
		return block
	}
	compactBase := fmt.Sprintf("\n- %s [%s/%s]\n", item.Title, item.Category, item.Layer)
	available := remaining - len(base) - len(reason) - len(bodyPrefix) - len("\n")
	if available > 16 {
		if available < len(content) {
			content = strings.TrimSpace(content[:available-3]) + "..."
		}
		block = base + reason + bodyPrefix + content + "\n"
		if len(block) <= remaining {
			return block
		}
	}
	available = remaining - len(base) - len(bodyPrefix) - len("\n")
	if available > 16 {
		content = item.Content
		if len(content) > available {
			content = strings.TrimSpace(content[:available-3]) + "..."
		}
		block = base + bodyPrefix + content + "\n"
		if len(block) <= remaining {
			return block
		}
	}
	available = remaining - len(compactBase) - len(bodyPrefix) - len("\n")
	if available > 8 {
		content = item.Content
		if len(content) > available {
			content = strings.TrimSpace(content[:available-3]) + "..."
		}
		block = compactBase + bodyPrefix + content + "\n"
		if len(block) <= remaining {
			return block
		}
	}
	if len(compactBase) <= remaining {
		return compactBase
	}
	return ""
}

func allowedCategory(category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "decision", "pattern", "preference", "warning", "failure":
		return true
	default:
		return false
	}
}

func scoreEntry(entry *models.MemoryEntry, input Input) (float64, []string) {
	mode := NormalizeMode(input.Mode)
	_ = mode
	promptTokens := uniqueTokens(input.UserPrompt)
	contextTokens := uniqueTokens(
		input.Runtime,
		filepathBase(input.ProjectRoot),
		filepathBase(input.WorkingDir),
		input.ActionType,
	)
	textTokens := uniqueTokens(entry.Title, entry.Category, strings.Join(entry.Tags, " "), entry.Content)
	textSet := make(map[string]struct{}, len(textTokens))
	for _, token := range textTokens {
		textSet[token] = struct{}{}
	}

	score := 0.0
	reasons := make([]string, 0, 4)
	if entry.Layer == models.MemoryLayerProject || entry.Layer == models.MemoryLayerWorking {
		score += 0.12
		reasons = append(reasons, "project-scoped")
	} else if entry.Layer == models.MemoryLayerGlobal {
		score += 0.04
		reasons = append(reasons, "global-memory")
	}

	promptOverlaps := 0
	for _, token := range promptTokens {
		if _, ok := textSet[token]; ok {
			promptOverlaps++
		}
	}
	if promptOverlaps == 0 {
		return 0, nil
	}
	score += float64(promptOverlaps) * 0.35
	reasons = append(reasons, fmt.Sprintf("keyword-overlap:%d", promptOverlaps))

	contextOverlaps := 0
	for _, token := range contextTokens {
		if _, ok := textSet[token]; ok {
			contextOverlaps++
		}
	}
	if contextOverlaps > 0 {
		score += float64(contextOverlaps) * 0.05
	}
	if tokenMatches(textSet, strings.ToLower(strings.TrimSpace(input.Runtime))) {
		score += 0.08
		reasons = append(reasons, "runtime-match")
	}
	if tokenMatches(textSet, strings.ToLower(strings.TrimSpace(input.ActionType))) {
		score += 0.08
		reasons = append(reasons, "action-match")
	}
	if bonus := recencyBonus(entry.UpdatedAt); bonus > 0 {
		score += bonus
		reasons = append(reasons, "recent")
	}
	return score, reasons
}

func recencyBonus(updatedAt time.Time) float64 {
	if updatedAt.IsZero() {
		return 0
	}
	ageDays := time.Since(updatedAt).Hours() / 24
	switch {
	case ageDays <= 7:
		return 0.12
	case ageDays <= 30:
		return 0.06
	case ageDays <= 90:
		return 0.03
	default:
		return 0
	}
}

func filepathBase(path string) string {
	path = strings.ReplaceAll(path, `\\`, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return ""
	}
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func tokenMatches(set map[string]struct{}, value string) bool {
	for _, token := range uniqueTokens(value) {
		if _, ok := set[token]; ok {
			return true
		}
	}
	return false
}

func uniqueTokens(parts ...string) []string {
	seen := map[string]struct{}{}
	var tokens []string
	for _, part := range parts {
		for _, token := range tokenRE.FindAllString(strings.ToLower(part), -1) {
			if len(token) < 3 {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			tokens = append(tokens, token)
		}
	}
	sort.Strings(tokens)
	return tokens
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
