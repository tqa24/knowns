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
	"github.com/howznguyen/knowns/internal/search"
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

const silentSupplementalWarning = "Silent supplemental context. Do not quote unless asked."

const (
	defaultMaxItems  = 5
	defaultMaxBytes  = 2500
	maxPreviewBody   = 320
	baselineMaxItems = 4
)

var tokenRE = regexp.MustCompile(`[a-z0-9]+`)

var lowSignalPromptTokens = map[string]struct{}{
	"again":    {},
	"continue": {},
	"go":       {},
	"hello":    {},
	"hey":      {},
	"hi":       {},
	"next":     {},
	"no":       {},
	"ok":       {},
	"okay":     {},
	"retry":    {},
	"sure":     {},
	"thank":    {},
	"thanks":   {},
	"yes":      {},
}

var globalPreferencePhrases = []string{
	"i want",
	"i prefer",
	"please",
	"always",
	"never",
	"default to",
	"from now on",
	"toi muon",
	"toi thich",
	"uu tien",
	"luon",
	"mac dinh",
	"tu gio",
	"ve sau",
	"dung",
	"khong doi",
}

var assistantScopePhrases = []string{
	"assistant",
	"agent",
	" ai ",
	"memory",
	"save memory",
	"reply",
	"review",
	"commit",
	"luu memory",
	"tra loi",
	"review code",
}

var projectScopePhrases = []string{
	"repo",
	"repository",
	"project",
	"codebase",
	"this repo",
	"this project",
	"repo nay",
	"project nay",
	"trong repo",
	"trong project",
	"knowns.md",
	"agents.md",
	"claude.md",
	"opencode.md",
	"copilot-instructions.md",
	"shim",
	"runtime",
	"package",
	"module",
	"file",
}

var instructionPhrases = []string{
	"must",
	"should",
	"need to",
	"do not",
	"don't",
	"never",
	"always",
	"keep",
	"use",
	"phai",
	"nen",
	"khong duoc",
	"dung",
	"luon",
	"giu",
	"dung ",
	"hay ",
	"bat",
}

var workingContextPhrases = []string{
	"currently",
	"for now",
	"this session",
	"temporary",
	"temporarily",
	"investigating",
	"debugging",
	"blocked on",
	"workaround",
	"hien tai",
	"tam thoi",
	"phien nay",
	"dang debug",
	"dang dieu tra",
	"bi chan",
}

type captureCandidate struct {
	Title      string
	Category   string
	Layer      string
	Content    string
	Tags       []string
	Confidence float64
}

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
	Retrieval string    `json:"retrieval,omitempty"`
	MatchedBy []string  `json:"matchedBy,omitempty"`
	Reasons   []string  `json:"reasons,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}

type hybridCandidate struct {
	entry     *models.MemoryEntry
	score     float64
	matchedBy []string
}

type candidate struct {
	item Item
}

var lookupHybridCandidates = defaultHybridCandidates

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
			Runtime:        "codex",
			DisplayName:    "Codex",
			HookKind:       HookNative,
			NativeHooks:    true,
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
	isSessionBaseline := shouldUseSessionBaseline(input.ActionType, input.UserPrompt)
	if shouldSkipPrompt(input.UserPrompt) && !isSessionBaseline {
		return pack, nil
	}
	maxItems := input.MaxItems
	if maxItems <= 0 {
		maxItems = defaultMaxItems
	}
	if isSessionBaseline && input.MaxItems <= 0 {
		maxItems = baselineMaxItems
	}
	maxBytes := input.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	candidates, err := buildCandidates(store, input, maxItems, isSessionBaseline)
	if err != nil {
		return pack, err
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
	for _, candidate := range candidates {
		if len(selected) >= maxItems {
			break
		}
		selected = append(selected, candidate.item)
	}

	serialized := serializePrefix(input.Runtime)
	if isSessionBaseline || len(selected) > 0 {
		block := serializeKNOWNSSummary(store, maxBytes-len(serialized))
		if block != "" {
			serialized += block
		}
	}

	if len(selected) == 0 && strings.TrimSpace(serialized) == strings.TrimSpace(serializePrefix(input.Runtime)) {
		pack.Bytes = len(serializePrefix(input.Runtime))
		return pack, nil
	}
	if !isSessionBaseline && len(selected) > 0 && !passesInjectionThreshold(selected) {
		pack.Bytes = len(serializePrefix(input.Runtime))
		return pack, nil
	}

	pack.Items = selected
	pack.Serialized = serialized
	pack.Bytes = len(serialized)
	pack.Status = StatusCandidate
	return pack, nil
}

func Capture(store *storage.Store, input Input) (*models.MemoryEntry, bool, error) {
	if store == nil || shouldSkipPrompt(input.UserPrompt) {
		return nil, false, nil
	}
	candidate, ok := inferCaptureCandidate(input)
	if !ok {
		return nil, false, nil
	}
	entries, err := store.Memory.List("")
	if err != nil {
		return nil, false, err
	}
	if hasDuplicateCapture(entries, candidate) {
		return nil, false, nil
	}
	entry := &models.MemoryEntry{
		Title:    candidate.Title,
		Layer:    candidate.Layer,
		Category: candidate.Category,
		Content:  candidate.Content,
		Tags:     append([]string(nil), candidate.Tags...),
	}
	if err := store.Memory.Create(entry); err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

func buildCandidates(store *storage.Store, input Input, maxItems int, baseline bool) ([]candidate, error) {
	if baseline {
		entries, err := store.Memory.List("")
		if err != nil {
			return nil, err
		}
		return buildBaselineItems(entries), nil
	}
	limit := max(maxItems*4, 20)
	if hybrid, ok := lookupHybridCandidates(store, input, limit); ok {
		candidates := buildHybridItems(hybrid, input)
		if len(candidates) > 0 {
			return candidates, nil
		}
	}

	entries, err := store.Memory.List("")
	if err != nil {
		return nil, err
	}
	return buildHeuristicItems(entries, input), nil
}

func buildBaselineItems(entries []*models.MemoryEntry) []candidate {
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if !allowedCategory(entry.Category) {
			continue
		}
		if entry.Layer != models.MemoryLayerProject && entry.Layer != models.MemoryLayerGlobal {
			continue
		}
		if hasMemoryTag(entry, "probe") || strings.Contains(strings.ToLower(entry.Title), "probe") {
			continue
		}
		score, reasons := baselineScore(entry)
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
			Retrieval: "session-baseline",
			Reasons:   reasons,
			Tags:      append([]string(nil), entry.Tags...),
		}})
	}
	return candidates
}

func inferCaptureCandidate(input Input) (captureCandidate, bool) {
	normalizedPrompt := normalizedPrompt(input.UserPrompt)
	if normalizedPrompt == "" {
		return captureCandidate{}, false
	}
	if looksLikeHookPayload(input.UserPrompt, normalizedPrompt) {
		return captureCandidate{}, false
	}

	if candidate, ok := inferGlobalPreferenceCandidate(input.UserPrompt, normalizedPrompt); ok {
		return candidate, true
	}
	if candidate, ok := inferProjectDecisionCandidate(input, normalizedPrompt); ok {
		return candidate, true
	}
	if candidate, ok := inferWorkingContextCandidate(input.UserPrompt, normalizedPrompt); ok {
		return candidate, true
	}
	return captureCandidate{}, false
}

func inferGlobalPreferenceCandidate(rawPrompt, normalized string) (captureCandidate, bool) {
	if !hasAnyPhrase(normalized, globalPreferencePhrases) {
		return captureCandidate{}, false
	}
	if !hasAnyPhrase(" "+normalized+" ", assistantScopePhrases) {
		return captureCandidate{}, false
	}
	if looksRepoSpecific(rawPrompt, normalized) {
		return captureCandidate{}, false
	}
	content := normalizeCapturedContent(rawPrompt)
	title := "User collaboration preference"
	tags := []string{"assistant", "preference"}
	if strings.Contains(normalized, "memory") || strings.Contains(normalized, "luu memory") || strings.Contains(normalized, "save memory") {
		title = "Memory capture preference"
		content = "User prefers the assistant to proactively save durable memory without waiting for explicit reminders."
		tags = append(tags, "memory")
	}
	if strings.Contains(normalized, "tra loi") || strings.Contains(normalized, "reply") || strings.Contains(normalized, "language") {
		title = "Response preference"
		tags = append(tags, "response")
	}
	return captureCandidate{
		Title:      title,
		Category:   "preference",
		Layer:      models.MemoryLayerGlobal,
		Content:    content,
		Tags:       uniqueStrings(tags),
		Confidence: 0.92,
	}, true
}

func inferProjectDecisionCandidate(input Input, normalized string) (captureCandidate, bool) {
	if !hasAnyPhrase(normalized, instructionPhrases) {
		return captureCandidate{}, false
	}
	if !looksRepoSpecific(input.UserPrompt, normalized) {
		return captureCandidate{}, false
	}
	content := normalizeCapturedContent(input.UserPrompt)
	title := "Project workflow decision"
	tags := []string{"project", "workflow"}
	if strings.Contains(normalized, "knowns.md") && strings.Contains(normalized, "agents.md") {
		title = "Instruction source of truth"
		content = "Compatibility shim files such as `AGENTS.md` must defer behavior and memory policy to `KNOWNS.md`, which is the canonical instruction source."
		tags = append(tags, "knowns", "agents")
	}
	return captureCandidate{
		Title:      title,
		Category:   "decision",
		Layer:      models.MemoryLayerProject,
		Content:    content,
		Tags:       uniqueStrings(tags),
		Confidence: 0.88,
	}, true
}

func inferWorkingContextCandidate(rawPrompt, normalized string) (captureCandidate, bool) {
	if !hasAnyPhrase(normalized, workingContextPhrases) {
		return captureCandidate{}, false
	}
	return captureCandidate{
		Title:      "Session working context",
		Category:   "context",
		Layer:      models.MemoryLayerProject,
		Content:    normalizeCapturedContent(rawPrompt),
		Tags:       []string{"session", "working-context"},
		Confidence: 0.84,
	}, true
}

func hasDuplicateCapture(entries []*models.MemoryEntry, candidate captureCandidate) bool {
	content := normalizeComparableText(candidate.Content)
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		existingContent := normalizeComparableText(entry.Content)
		if existingContent == content {
			return true
		}
		if entry.Layer == candidate.Layer && normalizeComparableText(entry.Title) == normalizeComparableText(candidate.Title) {
			if existingContent == "" || content == "" || strings.Contains(existingContent, content) || strings.Contains(content, existingContent) {
				return true
			}
		}
	}
	return false
}

func normalizedPrompt(prompt string) string {
	return normalizeComparableText(normalizeWhitespace(strings.ToLower(strings.TrimSpace(prompt))))
}

func normalizeCapturedContent(prompt string) string {
	prompt = normalizeWhitespace(strings.TrimSpace(prompt))
	if prompt == "" {
		return ""
	}
	last := prompt[len(prompt)-1]
	if last != '.' && last != '!' && last != '?' {
		prompt += "."
	}
	return prompt
}

func normalizeComparableText(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ắ", "a", "ằ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ấ", "a", "ầ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"é", "e", "è", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ế", "e", "ề", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"í", "i", "ì", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"ó", "o", "ò", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ố", "o", "ồ", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ớ", "o", "ờ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"ú", "u", "ù", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ứ", "u", "ừ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"ý", "y", "ỳ", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
		"đ", "d",
	)
	return normalizeWhitespace(replacer.Replace(strings.ToLower(strings.TrimSpace(s))))
}

func hasAnyPhrase(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if phrase == "" {
			continue
		}
		if strings.Contains(text, normalizeComparableText(phrase)) {
			return true
		}
	}
	return false
}

func looksRepoSpecific(rawPrompt, normalized string) bool {
	if hasAnyPhrase(normalized, projectScopePhrases) {
		return true
	}
	rawPrompt = strings.TrimSpace(rawPrompt)
	return strings.Contains(rawPrompt, "`") || strings.Contains(rawPrompt, "/") || strings.Contains(rawPrompt, ".go") || strings.Contains(rawPrompt, ".md")
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeComparableText(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func buildHeuristicItems(entries []*models.MemoryEntry, input Input) []candidate {
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if !allowedCategory(entry.Category) {
			continue
		}
		score, reasons, _ := scoreEntry(entry, input, true)
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
			Retrieval: "heuristic-fallback",
			Reasons:   append(reasons, "heuristic-fallback"),
			Tags:      append([]string(nil), entry.Tags...),
		}})
	}
	return candidates
}

func buildHybridItems(hits []hybridCandidate, input Input) []candidate {
	candidates := make([]candidate, 0, len(hits))
	for _, hit := range hits {
		if hit.entry == nil || !allowedCategory(hit.entry.Category) {
			continue
		}
		if !containsString(hit.matchedBy, "semantic") {
			continue
		}
		score, reasons, promptOverlaps := scoreEntry(hit.entry, input, false)
		if promptOverlaps == 0 {
			continue
		}
		score += hybridSearchBoost(hit.score)
		reasons = append(reasons, "hybrid-retrieval")
		reasons = append(reasons, "semantic-match")
		if containsString(hit.matchedBy, "keyword") {
			reasons = append(reasons, "keyword-match")
		}
		if score <= 0.75 {
			continue
		}
		candidates = append(candidates, candidate{item: Item{
			ID:        hit.entry.ID,
			Title:     hit.entry.Title,
			Category:  hit.entry.Category,
			Layer:     hit.entry.Layer,
			UpdatedAt: hit.entry.UpdatedAt,
			Content:   normalizeWhitespace(hit.entry.Content),
			Score:     score,
			Retrieval: "hybrid",
			MatchedBy: append([]string(nil), hit.matchedBy...),
			Reasons:   reasons,
			Tags:      append([]string(nil), hit.entry.Tags...),
		}})
	}
	return candidates
}

func defaultHybridCandidates(store *storage.Store, input Input, limit int) ([]hybridCandidate, bool) {
	if store == nil || strings.TrimSpace(input.UserPrompt) == "" {
		return nil, false
	}
	embedder, vecStore, err := search.InitSemantic(store)
	if err != nil {
		return nil, false
	}
	if embedder != nil {
		defer embedder.Close()
	}
	if vecStore != nil {
		defer vecStore.Close()
	}
	engine := search.NewEngine(store, embedder, vecStore)
	if !engine.SemanticAvailable() {
		return nil, false
	}
	results, err := engine.Search(search.SearchOptions{
		Query: strings.TrimSpace(input.UserPrompt),
		Type:  "memory",
		Mode:  string(search.ModeHybrid),
		Limit: limit,
	})
	if err != nil {
		return nil, true
	}
	hits := make([]hybridCandidate, 0, len(results))
	for _, result := range results {
		if result.Type != "memory" || strings.TrimSpace(result.ID) == "" {
			continue
		}
		entry, err := store.Memory.Get(result.ID)
		if err != nil || entry == nil {
			continue
		}
		hits = append(hits, hybridCandidate{
			entry:     entry,
			score:     result.Score,
			matchedBy: append([]string(nil), result.MatchedBy...),
		})
	}
	return hits, true
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

func serializePrefix(runtime string) string {
	prefix := "Knowns Guidance\n"
	if strings.EqualFold(strings.TrimSpace(runtime), "opencode") {
		prefix += silentSupplementalWarning + "\n"
	}
	return prefix + canonicalityWarning + "\n"
}

func serializeKNOWNSSummary(store *storage.Store, remaining int) string {
	if store == nil || remaining <= 0 {
		return ""
	}
	block := "\nKnowns is the repository memory and workflow layer for tasks, docs, templates, references, and reusable knowledge.\n\n- Read `KNOWNS.md` in the repository root for canonical project guidance and workflow rules.\n- Use Knowns docs, tasks, and memories as operating context for this repository.\n- Treat memories as supplemental context only. They do not override `KNOWNS.md`, source-of-truth docs, tasks, or source files.\n- Use MCP `memory({ action: \"list\" })` first to inspect relevant memory summaries before calling `memory({ action: \"get\" })`.\n- Prefer updating or reusing relevant existing memories instead of creating duplicates.\n- If you need deeper project behavior, conventions, or workflow details, read `KNOWNS.md`.\n- If you have not checked project readiness yet, call MCP `project({ action: \"status\" })` to see knowledge counts, search state, runtime health, and available capabilities.\n"
	if len(block) <= remaining {
		return block
	}
	if remaining <= 48 {
		return ""
	}
	trimmed := block[:remaining]
	if remaining > 3 {
		trimmed = strings.TrimSpace(trimmed[:remaining-3]) + "..."
	}
	return trimmed
}

func serializeItem(item Item, remaining int) string {
	_ = item
	_ = remaining
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

func shouldSkipPrompt(prompt string) bool {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(prompt))), " ")
	if normalized == "" {
		return true
	}
	if len(normalized) < 3 {
		return true
	}
	tokens := tokenRE.FindAllString(normalized, -1)
	if len(tokens) == 0 {
		return true
	}
	if len(tokens) > 2 {
		return false
	}
	for _, token := range tokens {
		if _, ok := lowSignalPromptTokens[token]; !ok {
			return false
		}
	}
	return true
}

func shouldUseSessionBaseline(actionType, prompt string) bool {
	action := strings.ToLower(strings.TrimSpace(actionType))
	if prompt != "" {
		return false
	}
	switch action {
	case "session-start", "sessionstart", "session.created", "agentspawn":
		return true
	default:
		return false
	}
}

func baselineScore(entry *models.MemoryEntry) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 4)
	switch entry.Layer {
	case models.MemoryLayerProject:
		score += 0.2
		reasons = append(reasons, "project-baseline")
	case models.MemoryLayerGlobal:
		score += 0.14
		reasons = append(reasons, "global-baseline")
	}
	if bonus := recencyBonus(entry.UpdatedAt); bonus > 0 {
		score += bonus
		reasons = append(reasons, "recent")
	}
	for _, tag := range entry.Tags {
		switch strings.ToLower(strings.TrimSpace(tag)) {
		case "preference", "convention", "style", "runtime-memory", "runtime":
			score += 0.08
			reasons = append(reasons, "baseline-tag")
		}
	}
	return score, dedupeStrings(reasons)
}

func hasMemoryTag(entry *models.MemoryEntry, target string) bool {
	for _, tag := range entry.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), target) {
			return true
		}
	}
	return false
}

func looksLikeHookPayload(rawPrompt, normalized string) bool {
	trimmed := strings.TrimSpace(rawPrompt)
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"hook_event_name\"") {
		return true
	}
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"session_id\"") {
		return true
	}
	if strings.Contains(normalized, "hook_event_name") || strings.Contains(normalized, "session_id") || strings.Contains(normalized, "transcript_path") || strings.Contains(normalized, "permission_mode") {
		return true
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func passesInjectionThreshold(items []Item) bool {
	if len(items) == 0 {
		return false
	}
	total := 0.0
	for _, item := range items {
		total += item.Score
	}
	if items[0].Score < 0.85 {
		return false
	}
	if len(items) == 1 {
		return total >= 1.1
	}
	return total >= 1.4
}

func scoreEntry(entry *models.MemoryEntry, input Input, requirePromptMatch bool) (float64, []string, int) {
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
	if entry.Layer == models.MemoryLayerProject {
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
	if promptOverlaps == 0 && requirePromptMatch {
		return 0, nil, 0
	}
	if promptOverlaps > 0 {
		score += float64(promptOverlaps) * 0.35
		reasons = append(reasons, fmt.Sprintf("keyword-overlap:%d", promptOverlaps))
	}

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
	return score, reasons, promptOverlaps
}

func hybridSearchBoost(raw float64) float64 {
	if raw < 0 {
		return 0
	}
	if raw > 1.2 {
		return 1.2
	}
	return raw
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
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
