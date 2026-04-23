package references

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/howznguyen/knowns/internal/models"
)

var referenceRE = regexp.MustCompile(`@(task-[A-Za-z0-9.-]+(?:\{[a-z-]+\})?|memory-[A-Za-z0-9-]+(?:\{[a-z-]+\})?|doc/[^\s\)]+)`)
var docRangeSuffixRE = regexp.MustCompile(`:(\d+)-(\d+)$`)
var docLineSuffixRE = regexp.MustCompile(`:(\d+)$`)

var allowedRelations = map[string]struct{}{
	"implements":    {},
	"depends":       {},
	"blocked-by":    {},
	"follows":       {},
	"related":       {},
	"parent":        {},
	"spec":          {},
	"imported-from": {},
	"template-for":  {},
	models.SemanticReferenceRelationReferences: {},
}

// AllRelationKinds returns the full allowlist of relation kinds.
func AllRelationKinds() []string {
	kinds := make([]string, 0, len(allowedRelations))
	for k := range allowedRelations {
		kinds = append(kinds, k)
	}
	return kinds
}

func AllowedRelation(relation string) bool {
	_, ok := allowedRelations[relation]
	return ok
}

func Extract(content string) []models.SemanticReference {
	matches := referenceRE.FindAllString(content, -1)
	refs := make([]models.SemanticReference, 0, len(matches))
	for _, match := range matches {
		ref, ok := Parse(match)
		if ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

func Parse(raw string) (models.SemanticReference, bool) {
	raw = strings.TrimSpace(strings.TrimRight(raw, ".,;"))
	if raw == "" || !strings.HasPrefix(raw, "@") {
		return models.SemanticReference{}, false
	}

	body := raw[1:]
	relation := models.SemanticReferenceRelationReferences
	explicitRelation := false
	validRelation := true

	if open := strings.LastIndex(body, "{"); open >= 0 && strings.HasSuffix(body, "}") {
		relation = body[open+1 : len(body)-1]
		body = body[:open]
		explicitRelation = true
		validRelation = AllowedRelation(relation)
	}

	ref := models.SemanticReference{
		Raw:              raw,
		Relation:         relation,
		ExplicitRelation: explicitRelation,
		ValidRelation:    validRelation,
	}

	switch {
	case strings.HasPrefix(body, "task-"):
		ref.Type = "task"
		ref.Target = strings.TrimPrefix(body, "task-")
	case strings.HasPrefix(body, "memory-"):
		ref.Type = "memory"
		ref.Target = strings.TrimPrefix(body, "memory-")
	case strings.HasPrefix(body, "doc/"):
		ref.Type = "doc"
		ref.Target, ref.Fragment = parseDocTarget(strings.TrimPrefix(body, "doc/"))
	default:
		return models.SemanticReference{}, false
	}

	if strings.TrimSpace(ref.Target) == "" {
		return models.SemanticReference{}, false
	}

	return ref, true
}

func parseDocTarget(value string) (string, *models.DocReferenceFragment) {
	value = strings.TrimSpace(value)
	fragment := &models.DocReferenceFragment{}

	if idx := strings.Index(value, "#"); idx >= 0 {
		fragment.Raw = value[idx:]
		fragment.Heading = value[idx+1:]
		value = value[:idx]
	} else if m := docRangeSuffixRE.FindStringSubmatch(value); m != nil {
		fragment.Raw = m[0]
		fragment.RangeStart, _ = strconv.Atoi(m[1])
		fragment.RangeEnd, _ = strconv.Atoi(m[2])
		value = strings.TrimSuffix(value, m[0])
	} else if m := docLineSuffixRE.FindStringSubmatch(value); m != nil {
		fragment.Raw = m[0]
		fragment.Line, _ = strconv.Atoi(m[1])
		value = strings.TrimSuffix(value, m[0])
	}

	value = strings.TrimRight(value, ".,;")
	if fragment.Raw == "" {
		fragment = nil
	}
	return value, fragment
}
