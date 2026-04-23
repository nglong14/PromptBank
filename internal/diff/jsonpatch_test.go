package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/nglong14/PromptBank/internal/asset"
	"github.com/nglong14/PromptBank/internal/models"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeVersion(vn int, a asset.Assets, fid string, tids []string) *models.PromptVersion {
	raw, _ := json.Marshal(a)
	if tids == nil {
		tids = []string{}
	}
	return &models.PromptVersion{
		VersionNumber: vn,
		Assets:        raw,
		FrameworkID:   fid,
		TechniqueIDs:  tids,
	}
}

func makeDoc(a asset.Assets, fid string, tids []string) VersionDoc {
	if tids == nil {
		tids = []string{}
	}
	return VersionDoc{Assets: a, FrameworkID: fid, TechniqueIDs: tids}
}

// changeMap indexes changes by Path for quick lookup in assertions.
func changeMap(changes []Change) map[string]Change {
	m := make(map[string]Change, len(changes))
	for _, c := range changes {
		m[c.Path] = c
	}
	return m
}

// ---------------------------------------------------------------------------
// Test-only patch applier (round-trip verification)
// ---------------------------------------------------------------------------

// applyPatch applies a slice of Changes to from and returns the result.
// It is intentionally minimal: it supports only the ops and path shapes
// produced by this package.
func applyPatch(from VersionDoc, changes []Change) (VersionDoc, error) {
	// Normalise before marshaling so the patch (computed on a normalised doc)
	// is applied to the same canonical representation.
	from = normaliseDoc(from)
	raw, _ := json.Marshal(from)
	var root any
	_ = json.Unmarshal(raw, &root)

	for _, c := range changes {
		parts := parsePointer(c.Path)
		modified, err := patchNode(root, parts, c.Op, c.New)
		if err != nil {
			return VersionDoc{}, fmt.Errorf("apply %s %s: %w", c.Op, c.Path, err)
		}
		root = modified
	}

	out, _ := json.Marshal(root)
	var result VersionDoc
	_ = json.Unmarshal(out, &result)
	return result, nil
}

// parsePointer splits an RFC 6901 JSON Pointer into unescaped path segments.
func parsePointer(pointer string) []string {
	if pointer == "" {
		return nil
	}
	raw := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	out := make([]string, len(raw))
	for i, p := range raw {
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		out[i] = p
	}
	return out
}

// patchNode recursively navigates node using parts and applies op at the leaf.
// It returns the (possibly new) value at this level so callers can update parent containers.
func patchNode(node any, parts []string, op Op, newVal any) (any, error) {
	if len(parts) == 0 {
		return newVal, nil
	}
	key := parts[0]
	rest := parts[1:]

	switch n := node.(type) {
	case map[string]any:
		if len(rest) == 0 {
			switch op {
			case OpAdd, OpReplace:
				n[key] = newVal
			case OpRemove:
				delete(n, key)
			}
			return n, nil
		}
		child := n[key]
		modified, err := patchNode(child, rest, op, newVal)
		if err != nil {
			return nil, err
		}
		n[key] = modified
		return n, nil

	case []any:
		if key == "-" {
			if len(rest) == 0 && op == OpAdd {
				return append(n, newVal), nil
			}
			return nil, fmt.Errorf("'-' is only valid for array append")
		}
		idx, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("expected array index, got %q", key)
		}
		if len(rest) == 0 {
			switch op {
			case OpAdd:
				result := make([]any, 0, len(n)+1)
				result = append(result, n[:idx]...)
				result = append(result, newVal)
				result = append(result, n[idx:]...)
				return result, nil
			case OpReplace:
				if idx >= len(n) {
					return nil, fmt.Errorf("index %d out of range (len %d)", idx, len(n))
				}
				n[idx] = newVal
				return n, nil
			case OpRemove:
				if idx >= len(n) {
					return nil, fmt.Errorf("index %d out of range (len %d)", idx, len(n))
				}
				result := make([]any, 0, len(n)-1)
				result = append(result, n[:idx]...)
				result = append(result, n[idx+1:]...)
				return result, nil
			}
		}
		if idx >= len(n) {
			return nil, fmt.Errorf("index %d out of range (len %d)", idx, len(n))
		}
		modified, err := patchNode(n[idx], rest, op, newVal)
		if err != nil {
			return nil, err
		}
		n[idx] = modified
		return n, nil

	default:
		return nil, fmt.Errorf("cannot navigate %T with key %q", node, key)
	}
}

// assertRoundTrip applies Diff(from,to) to from and verifies the result is
// semantically equal to to by checking that Diff(patched, to) is empty.
func assertRoundTrip(t *testing.T, from, to VersionDoc) {
	t.Helper()
	changes := Diff(from, to)
	patched, err := applyPatch(from, changes)
	if err != nil {
		t.Fatalf("applyPatch error: %v", err)
	}
	residual := Diff(patched, to)
	if len(residual) != 0 {
		t.Errorf("round-trip failed: residual changes after patch: %v", residual)
	}
}

// ---------------------------------------------------------------------------
// Golden cases — one meaningful edit each
// ---------------------------------------------------------------------------

func TestDiff_PersonaReplace(t *testing.T) {
	from := makeDoc(asset.Assets{Persona: "You are a teacher"}, "", nil)
	to := makeDoc(asset.Assets{Persona: "You are a senior engineer"}, "", nil)

	changes := Diff(from, to)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	c := changes[0]
	if c.Op != OpReplace || c.Path != "/assets/persona" {
		t.Errorf("unexpected change: %+v", c)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_ExampleAdd(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", nil)
	to := makeDoc(asset.Assets{Examples: []asset.Example{{Input: "q", Output: "a"}}}, "", nil)

	changes := Diff(from, to)
	cm := changeMap(changes)

	if _, ok := cm["/assets/examples/-"]; !ok {
		t.Errorf("expected add at /assets/examples/-, got: %v", changes)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_ExampleRemove(t *testing.T) {
	from := makeDoc(asset.Assets{Examples: []asset.Example{{Input: "q", Output: "a"}}}, "", nil)
	to := makeDoc(asset.Assets{}, "", nil)

	changes := Diff(from, to)
	cm := changeMap(changes)

	if _, ok := cm["/assets/examples/0"]; !ok {
		t.Errorf("expected remove at /assets/examples/0, got: %v", changes)
	}
	if cm["/assets/examples/0"].Op != OpRemove {
		t.Errorf("expected OpRemove, got: %v", cm["/assets/examples/0"].Op)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_ExampleModify(t *testing.T) {
	from := makeDoc(asset.Assets{Examples: []asset.Example{{Input: "old q", Output: "old a"}}}, "", nil)
	to := makeDoc(asset.Assets{Examples: []asset.Example{{Input: "new q", Output: "old a"}}}, "", nil)

	changes := Diff(from, to)
	cm := changeMap(changes)

	c, ok := cm["/assets/examples/0/input"]
	if !ok || c.Op != OpReplace {
		t.Errorf("expected replace at /assets/examples/0/input, got: %v", changes)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_TechniqueAdd(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", []string{"cot"})
	to := makeDoc(asset.Assets{}, "", []string{"cot", "few-shot"})

	changes := Diff(from, to)
	cm := changeMap(changes)

	c, ok := cm["/technique_ids/-"]
	if !ok || c.Op != OpAdd {
		t.Errorf("expected add at /technique_ids/-, got: %v", changes)
	}
	if c.New != "few-shot" {
		t.Errorf("expected new value 'few-shot', got: %v", c.New)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_TechniqueRemove(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", []string{"cot", "few-shot"})
	to := makeDoc(asset.Assets{}, "", []string{"cot"})

	changes := Diff(from, to)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	c := changes[0]
	if c.Op != OpRemove || c.Old != "few-shot" {
		t.Errorf("expected remove of 'few-shot', got: %+v", c)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_FrameworkSwap(t *testing.T) {
	from := makeDoc(asset.Assets{}, "raci", nil)
	to := makeDoc(asset.Assets{}, "risen", nil)

	changes := Diff(from, to)
	cm := changeMap(changes)

	c, ok := cm["/framework_id"]
	if !ok || c.Op != OpReplace || c.Old != "raci" || c.New != "risen" {
		t.Errorf("expected replace at /framework_id, got: %v", changes)
	}
	assertRoundTrip(t, from, to)
}

func TestDiff_MultiChange(t *testing.T) {
	from := makeDoc(
		asset.Assets{
			Persona: "You are a teacher",
			Examples: []asset.Example{
				{Input: "old q", Output: "old a"},
			},
		},
		"raci",
		[]string{"cot"},
	)
	to := makeDoc(
		asset.Assets{
			Persona: "You are an engineer",
			Examples: []asset.Example{
				{Input: "old q", Output: "old a"},
				{Input: "new q", Output: "new a"},
			},
		},
		"risen",
		[]string{"cot", "few-shot"},
	)

	changes := Diff(from, to)

	if len(changes) < 3 {
		t.Fatalf("expected at least 3 changes, got %d: %v", len(changes), changes)
	}
	assertRoundTrip(t, from, to)
}

// ---------------------------------------------------------------------------
// Noise cases — should produce zero changes
// ---------------------------------------------------------------------------

func TestDiff_WhitespaceOnly(t *testing.T) {
	from := makeDoc(asset.Assets{Persona: "hello world"}, "", nil)
	to := makeDoc(asset.Assets{Persona: "  hello world  "}, "", nil)

	changes := Diff(from, to)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for whitespace-only edit, got: %v", changes)
	}
}

func TestDiff_TechniqueReorder(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", []string{"cot", "few-shot", "zs"})
	to := makeDoc(asset.Assets{}, "", []string{"zs", "cot", "few-shot"})

	changes := Diff(from, to)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for technique reorder, got: %v", changes)
	}
}

func TestDiff_MissingVsEmptyString(t *testing.T) {
	from := makeDoc(asset.Assets{Tone: ""}, "", nil)
	to := makeDoc(asset.Assets{}, "", nil) // tone absent from JSON → after normalize still ""

	changes := Diff(from, to)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for missing-vs-empty, got: %v", changes)
	}
}

func TestDiff_Identical(t *testing.T) {
	doc := makeDoc(
		asset.Assets{Persona: "test", Examples: []asset.Example{{Input: "i", Output: "o"}}},
		"raci",
		[]string{"cot"},
	)
	changes := Diff(doc, doc)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes for identical docs, got: %v", changes)
	}
}

// ---------------------------------------------------------------------------
// Stats assertions
// ---------------------------------------------------------------------------

func TestCompute_Stats(t *testing.T) {
	from := makeVersion(1,
		asset.Assets{Persona: "old", Examples: []asset.Example{{Input: "q", Output: "a"}}},
		"raci",
		[]string{"cot"},
	)
	to := makeVersion(2,
		asset.Assets{Persona: "new", Examples: []asset.Example{{Input: "q", Output: "a"}, {Input: "q2", Output: "a2"}}},
		"raci",
		[]string{"cot", "few-shot"},
	)

	result, err := Compute(from, to)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}

	if result.VersionFrom != 1 || result.VersionTo != 2 {
		t.Errorf("wrong version numbers: from=%d to=%d", result.VersionFrom, result.VersionTo)
	}
	if result.Stats.Modifications < 1 {
		t.Errorf("expected at least 1 modification (persona), got %d", result.Stats.Modifications)
	}
	if result.Stats.Additions < 2 {
		t.Errorf("expected at least 2 additions (example + technique), got %d", result.Stats.Additions)
	}
	if result.Stats.ByField["assets.persona"] != 1 {
		t.Errorf("expected ByField[assets.persona]=1, got %d", result.Stats.ByField["assets.persona"])
	}
	if result.Stats.ByField["assets.examples"] < 1 {
		t.Errorf("expected ByField[assets.examples]>=1, got %d", result.Stats.ByField["assets.examples"])
	}
	if result.Stats.ByField["technique_ids"] != 1 {
		t.Errorf("expected ByField[technique_ids]=1, got %d", result.Stats.ByField["technique_ids"])
	}
}

func TestCompute_EmptyChanges(t *testing.T) {
	v := makeVersion(1, asset.Assets{Persona: "same"}, "raci", []string{"cot"})
	result, err := Compute(v, v)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}
	if len(result.Changes) != 0 {
		t.Errorf("expected no changes, got: %v", result.Changes)
	}
	if result.Stats.Additions != 0 || result.Stats.Removals != 0 || result.Stats.Modifications != 0 {
		t.Errorf("expected zero stats, got: %+v", result.Stats)
	}
}

// ---------------------------------------------------------------------------
// Round-trip via applyPatch
// ---------------------------------------------------------------------------

func TestRoundTrip_ComplexEdit(t *testing.T) {
	from := makeDoc(
		asset.Assets{
			Persona:  "teacher",
			Context:  "school",
			Examples: []asset.Example{{Input: "a", Output: "b"}, {Input: "c", Output: "d"}},
		},
		"raci",
		[]string{"cot", "few-shot"},
	)
	to := makeDoc(
		asset.Assets{
			Persona:  "engineer",
			Context:  "office",
			Goal:     "solve problems",
			Examples: []asset.Example{{Input: "a", Output: "b"}},
		},
		"risen",
		[]string{"few-shot", "zs"},
	)
	assertRoundTrip(t, from, to)
}

func TestRoundTrip_AddAllFields(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", nil)
	to := makeDoc(
		asset.Assets{
			Persona:     "p",
			Context:     "c",
			Tone:        "t",
			Constraints: "cons",
			Goal:        "g",
			Examples:    []asset.Example{{Input: "i", Output: "o"}},
		},
		"raci",
		[]string{"cot"},
	)
	assertRoundTrip(t, from, to)
}

func TestRoundTrip_RemoveAllFields(t *testing.T) {
	from := makeDoc(
		asset.Assets{
			Persona: "p",
			Context: "c",
			Examples: []asset.Example{{Input: "i", Output: "o"}},
		},
		"raci",
		[]string{"cot"},
	)
	to := makeDoc(asset.Assets{}, "", nil)
	assertRoundTrip(t, from, to)
}

// ---------------------------------------------------------------------------
// RFC 6901 escaping
// ---------------------------------------------------------------------------

func TestEscapeKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"normal", "normal"},
		{"with/slash", "with~1slash"},
		{"with~tilde", "with~0tilde"},
		{"~0already", "~00already"},
		{"a/b~c", "a~1b~0c"},
	}
	for _, tc := range cases {
		if got := escapeKey(tc.in); got != tc.want {
			t.Errorf("escapeKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestByFieldKey(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/assets/persona", "assets.persona"},
		{"/assets/context", "assets.context"},
		{"/assets/examples/0/input", "assets.examples"},
		{"/assets/examples/-", "assets.examples"},
		{"/framework_id", "framework_id"},
		{"/technique_ids/0", "technique_ids"},
		{"/technique_ids/-", "technique_ids"},
	}
	for _, tc := range cases {
		if got := byFieldKey(tc.path); got != tc.want {
			t.Errorf("byFieldKey(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Set semantics: multiple removes emit in descending index order
// ---------------------------------------------------------------------------

func TestDiff_SetMultipleRemoves_Deterministic(t *testing.T) {
	from := makeDoc(asset.Assets{}, "", []string{"a", "b", "c", "d"})
	to := makeDoc(asset.Assets{}, "", []string{"b"})

	changes := Diff(from, to)

	// Only removes should be present.
	for _, c := range changes {
		if c.Op != OpRemove {
			t.Errorf("expected only removes for set shrink, got: %+v", c)
		}
	}

	// Removed indices must appear in descending order for round-trip correctness.
	indices := make([]int, 0, len(changes))
	for _, c := range changes {
		parts := parsePointer(c.Path)
		idx, _ := strconv.Atoi(parts[len(parts)-1])
		indices = append(indices, idx)
	}
	if !sort.IntsAreSorted(reverseInts(indices)) {
		t.Errorf("remove indices not in descending order: %v", indices)
	}

	assertRoundTrip(t, from, to)
}

func reverseInts(s []int) []int {
	r := make([]int, len(s))
	for i, v := range s {
		r[len(s)-1-i] = v
	}
	return r
}
