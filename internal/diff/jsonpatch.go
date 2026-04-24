package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/models"
)

// Op: RFC 6902 subset
type Op string

const (
	OpAdd     Op = "add"
	OpRemove  Op = "remove"
	OpReplace Op = "replace"
)

// Change describes a single edit between two versions
// Path is an RFC 6901 JSON Pointer
type Change struct {
	Op   Op     `json:"op"`
	Path string `json:"path"`
	Old  any    `json:"old,omitempty"`
	New  any    `json:"new,omitempty"`
}

// Stats summarises the changes produced by a diff
type Stats struct {
	Additions     int            `json:"additions"`
	Removals      int            `json:"removals"`
	Modifications int            `json:"modifications"`
	ByField       map[string]int `json:"byField"`
}

// Result is the full diff output, suitable for JSON serialisation and caching in diff_from_parent
type Result struct {
	VersionFrom int      `json:"version_from"`
	VersionTo   int      `json:"version_to"`
	Changes     []Change `json:"changes"`
	Stats       Stats    `json:"stats"`
}

// VersionDoc is the virtual document
type VersionDoc struct {
	Assets       asset.Assets `json:"assets"`
	FrameworkID  string       `json:"framework_id"`
	TechniqueIDs []string     `json:"technique_ids"`
}

// FromVersion builds a normalised VersionDoc from a database row
func FromVersion(v *models.PromptVersion) (VersionDoc, error) {
	parsed, err := asset.ParseRaw(v.Assets)
	if err != nil {
		return VersionDoc{}, fmt.Errorf("parse assets: %w", err)
	}
	normalised := asset.Normalize(parsed).Assets

	tids := v.TechniqueIDs
	if tids == nil {
		tids = []string{}
	}

	return VersionDoc{
		Assets:       normalised,
		FrameworkID:  v.FrameworkID,
		TechniqueIDs: tids,
	}, nil
}

// Diff computes the ordered list of Changes between from and to.
func Diff(from, to VersionDoc) []Change {
	stats := Stats{ByField: make(map[string]int)}
	return diffInternal(from, to, &stats)
}

// Normalises both rows, runs the walker, and returns a Result with stats populated.
func Compute(from, to *models.PromptVersion) (Result, error) {
	fromDoc, err := FromVersion(from)
	if err != nil {
		return Result{}, fmt.Errorf("build from-doc: %w", err)
	}
	toDoc, err := FromVersion(to)
	if err != nil {
		return Result{}, fmt.Errorf("build to-doc: %w", err)
	}

	stats := Stats{ByField: make(map[string]int)}
	changes := diffInternal(fromDoc, toDoc, &stats)

	return Result{
		VersionFrom: from.VersionNumber,
		VersionTo:   to.VersionNumber,
		Changes:     changes,
		Stats:       stats,
	}, nil
}

// Normalises both docs then marshals them to generic JSON trees and runs the walker.
func diffInternal(from, to VersionDoc, stats *Stats) []Change {
	from = normaliseDoc(from)
	to = normaliseDoc(to)

	var aAny, bAny any
	aBytes, _ := json.Marshal(from)
	bBytes, _ := json.Marshal(to)
	_ = json.Unmarshal(aBytes, &aAny)
	_ = json.Unmarshal(bBytes, &bAny)

	var changes []Change
	walk("", aAny, bAny, &changes, stats)
	if changes == nil {
		changes = []Change{}
	}
	return changes
}

// normaliseDoc applies asset normalization and coerces nil slices to empty slices
// so that nil-vs-empty distinctions never produce spurious changes.
func normaliseDoc(d VersionDoc) VersionDoc {
	d.Assets = asset.Normalize(d.Assets).Assets
	if d.Assets.Examples == nil {
		d.Assets.Examples = []asset.Example{}
	}
	if d.TechniqueIDs == nil {
		d.TechniqueIDs = []string{}
	}
	return d
}

// arrayPolicy controls how an array at a given JSON Pointer path is compared.
type arrayPolicy int

const (
	policyOrdered arrayPolicy = iota // compare by index; recurse into elements
	policySet                        // compare by value membership; only add/remove
)

var arrayPolicies = map[string]arrayPolicy{
	"/assets/examples": policyOrdered,
	"/technique_ids":   policySet,
}

func policyFor(path string) arrayPolicy {
	if p, ok := arrayPolicies[path]; ok {
		return p
	}
	return policyOrdered
}

func walk(path string, a, b any, changes *[]Change, stats *Stats) {
	a, b = coerceNilContainers(a, b)

	switch av := a.(type) {
	case map[string]any:
		bv := toMap(b)
		for _, key := range unionKeys(av, bv) {
			childPath := path + "/" + escapeKey(key)
			aVal, aOk := av[key]
			bVal, bOk := bv[key]
			switch {
			case aOk && !bOk:
				emit(changes, stats, childPath, OpRemove, aVal, nil)
			case !aOk && bOk:
				emit(changes, stats, childPath, OpAdd, nil, bVal)
			default:
				walk(childPath, aVal, bVal, changes, stats)
			}
		}

	case []any:
		bv := toSlice(b)
		switch policyFor(path) {
		case policySet:
			diffSet(path, av, bv, changes, stats)
		default:
			diffOrdered(path, av, bv, changes, stats)
		}

	default:
		if !scalarEqual(a, b) {
			emit(changes, stats, path, OpReplace, a, b)
		}
	}
}

// diffOrdered diffs two arrays by index position.
// Overlapping indices recurse; tails of the longer side become add/remove.
func diffOrdered(path string, a, b []any, changes *[]Change, stats *Stats) {
	min := len(a)
	if len(b) < min {
		min = len(b)
	}
	for i := 0; i < min; i++ {
		walk(fmt.Sprintf("%s/%d", path, i), a[i], b[i], changes, stats)
	}
	for i := min; i < len(a); i++ {
		emit(changes, stats, fmt.Sprintf("%s/%d", path, i), OpRemove, a[i], nil)
	}
	for i := min; i < len(b); i++ {
		emit(changes, stats, path+"/-", OpAdd, nil, b[i])
	}
}

// diffSet diffs two arrays treated as unordered sets of string scalars.
// Reordering produces no changes. Removes are emitted in descending index order
// so that applying the patch sequentially (removes high-to-low, then adds) is
// always correct for the round-trip property.
func diffSet(path string, a, b []any, changes *[]Change, stats *Stats) {
	aSet := make(map[string]int) // value → original index in a
	for i, v := range a {
		if s, ok := v.(string); ok {
			aSet[s] = i
		}
	}
	bSet := make(map[string]struct{})
	for _, v := range b {
		if s, ok := v.(string); ok {
			bSet[s] = struct{}{}
		}
	}

	var added, removed []string
	for v := range bSet {
		if _, inA := aSet[v]; !inA {
			added = append(added, v)
		}
	}
	for v := range aSet {
		if _, inB := bSet[v]; !inB {
			removed = append(removed, v)
		}
	}

	// Deterministic order: removes descending by original index, adds ascending by value.
	sort.Slice(removed, func(i, j int) bool {
		return aSet[removed[i]] > aSet[removed[j]]
	})
	sort.Strings(added)

	for _, v := range removed {
		emit(changes, stats, fmt.Sprintf("%s/%d", path, aSet[v]), OpRemove, v, nil)
	}
	for _, v := range added {
		emit(changes, stats, path+"/-", OpAdd, nil, v)
	}
}

// emit records one change and updates stats in a single pass.
func emit(changes *[]Change, stats *Stats, path string, op Op, old, new any) {
	*changes = append(*changes, Change{Op: op, Path: path, Old: old, New: new})
	switch op {
	case OpAdd:
		stats.Additions++
	case OpRemove:
		stats.Removals++
	case OpReplace:
		stats.Modifications++
	}
	stats.ByField[byFieldKey(path)]++
}

// byFieldKey maps a JSON Pointer path to its top-level logical section label.
// /assets/examples/0/input → "assets.examples"
func byFieldKey(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.SplitN(trimmed, "/", 3)
	top := parts[0]
	if top == "assets" && len(parts) >= 2 {
		sub := parts[1]
		if sub == "examples" {
			return "assets.examples"
		}
		return "assets." + sub
	}
	return top
}

// escapeKey applies RFC 6901 escaping to an object key.
// ~ must be escaped before / to prevent double-escaping.
func escapeKey(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// scalarEqual returns true when two scalar JSON values should be considered equal.
// nil and "" are treated as equivalent (absent field == empty string).
func scalarEqual(a, b any) bool {
	return normaliseScalar(a) == normaliseScalar(b)
}

func normaliseScalar(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// coerceNilContainers promotes a nil value to an empty container when the
// opposite side is a map or slice.  This prevents nil-vs-container mismatches
// from falling through to the scalar comparator and emitting a spurious replace.
func coerceNilContainers(a, b any) (any, any) {
	if a == nil {
		switch b.(type) {
		case map[string]any:
			a = map[string]any{}
		case []any:
			a = []any{}
		}
	}
	if b == nil {
		switch a.(type) {
		case map[string]any:
			b = map[string]any{}
		case []any:
			b = []any{}
		}
	}
	return a, b
}

// toMap safely casts v to map[string]any, returning an empty map on type mismatch.
func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// toSlice safely casts v to []any, returning nil on type mismatch.
func toSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// unionKeys returns the sorted union of keys from two maps for deterministic traversal.
func unionKeys(a, b map[string]any) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
