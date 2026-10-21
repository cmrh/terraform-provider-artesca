package policydocument

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

// document is an IAM policy document. Fields that hold one value or a list
// are encoded as a string for one value, as IAM writes them.
type document struct {
	Version   string      `json:"Version,omitempty"`
	Statement []statement `json:"Statement"`
}

type statement struct {
	Sid          string                    `json:"Sid,omitempty"`
	Effect       string                    `json:"Effect,omitempty"`
	Principal    any                       `json:"Principal,omitempty"`
	NotPrincipal any                       `json:"NotPrincipal,omitempty"`
	Action       any                       `json:"Action,omitempty"`
	NotAction    any                       `json:"NotAction,omitempty"`
	Resource     any                       `json:"Resource,omitempty"`
	NotResource  any                       `json:"NotResource,omitempty"`
	Condition    map[string]map[string]any `json:"Condition,omitempty"`
}

// stringOrList returns nil for no values, the value for one, and the sorted
// list otherwise.
func stringOrList(values []string) any {
	switch len(values) {
	case 0:
		return nil
	case 1:
		return values[0]
	default:
		sorted := slices.Clone(values)
		slices.Sort(sorted)
		return sorted
	}
}

// principal is one principals block: a principal type and its identifiers.
type principal struct {
	Type        string
	Identifiers []string
}

// principalValue merges principal blocks by type. The type "*" with the
// identifier "*" is the anonymous principal, encoded as "*".
func principalValue(blocks []principal) (any, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	byType := map[string][]string{}
	for _, p := range blocks {
		if p.Type == "*" {
			if len(blocks) != 1 || len(p.Identifiers) != 1 || p.Identifiers[0] != "*" {
				return nil, fmt.Errorf(`a principals block with type "*" must be the only one and have identifiers ["*"]`)
			}
			return "*", nil
		}
		for _, id := range p.Identifiers {
			if !slices.Contains(byType[p.Type], id) {
				byType[p.Type] = append(byType[p.Type], id)
			}
		}
	}
	out := make(map[string]any, len(byType))
	for t, ids := range byType {
		out[t] = stringOrList(ids)
	}
	return out, nil
}

// condition is one condition block.
type condition struct {
	Test     string
	Variable string
	Values   []string
}

func conditionValue(blocks []condition) map[string]map[string]any {
	if len(blocks) == 0 {
		return nil
	}
	merged := map[string]map[string][]string{}
	for _, c := range blocks {
		if merged[c.Test] == nil {
			merged[c.Test] = map[string][]string{}
		}
		for _, v := range c.Values {
			if !slices.Contains(merged[c.Test][c.Variable], v) {
				merged[c.Test][c.Variable] = append(merged[c.Test][c.Variable], v)
			}
		}
	}
	out := make(map[string]map[string]any, len(merged))
	for test, vars := range merged {
		out[test] = make(map[string]any, len(vars))
		for v, values := range vars {
			if len(values) == 1 {
				out[test][v] = values[0]
			} else {
				out[test][v] = values // condition values keep their order
			}
		}
	}
	return out
}

// parseDocument decodes a policy document given as JSON. Statement may be a
// single object or a list.
func parseDocument(raw string) (document, error) {
	var doc struct {
		Version   string          `json:"Version"`
		Id        string          `json:"Id"`
		Statement json.RawMessage `json:"Statement"`
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return document{}, fmt.Errorf("parsing policy document: %w", err)
	}
	out := document{Version: doc.Version}
	trimmed := bytes.TrimSpace(doc.Statement)
	switch {
	case len(trimmed) == 0:
	case trimmed[0] == '[':
		if err := decodeStrict(trimmed, &out.Statement); err != nil {
			return document{}, fmt.Errorf("parsing policy document statements: %w", err)
		}
	default:
		var s statement
		if err := decodeStrict(trimmed, &s); err != nil {
			return document{}, fmt.Errorf("parsing policy document statement: %w", err)
		}
		out.Statement = []statement{s}
	}
	return out, nil
}

func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// merge builds the document from source documents, the configured statements
// and override documents, the way aws_iam_policy_document does:
//   - statements from all source documents are combined; a Sid may appear in
//     only one of them;
//   - a configured statement replaces the source statement with its Sid;
//   - an override statement replaces the statement with its Sid, and one
//     without a Sid, or with a new Sid, is appended.
func merge(version string, sources []document, statements []statement, overrides []document) (document, error) {
	var out []statement
	seen := map[string]bool{}
	for i, src := range sources {
		for _, s := range src.Statement {
			if s.Sid != "" {
				if seen[s.Sid] {
					return document{}, fmt.Errorf("source_policy_documents[%d]: duplicate Sid %q", i, s.Sid)
				}
				seen[s.Sid] = true
			}
			out = append(out, s)
		}
	}

	configured := map[string]bool{}
	for _, s := range statements {
		if s.Sid != "" {
			if configured[s.Sid] {
				return document{}, fmt.Errorf("duplicate statement Sid %q", s.Sid)
			}
			configured[s.Sid] = true
		}
		out = replaceOrAppend(out, s)
	}

	for _, ov := range overrides {
		for _, s := range ov.Statement {
			out = replaceOrAppend(out, s)
		}
	}

	return document{Version: version, Statement: out}, nil
}

// replaceOrAppend replaces the statement with s's Sid, or appends s.
func replaceOrAppend(statements []statement, s statement) []statement {
	if s.Sid != "" {
		for i := range statements {
			if statements[i].Sid == s.Sid {
				statements[i] = s
				return statements
			}
		}
	}
	return append(statements, s)
}

func encode(doc document) (string, error) {
	if doc.Statement == nil {
		doc.Statement = []statement{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	return string(bytes.TrimRight(buf.Bytes(), "\n")), nil
}
