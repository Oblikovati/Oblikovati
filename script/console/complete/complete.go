// SPDX-License-Identifier: GPL-2.0-only

package complete

import (
	"sort"
	"strings"

	"oblikovati.org/script/console/lualex"
)

// apiRoot is the single global the host API hangs off (ADR-0028); a dotted chain rooted here
// is completed against the wire-method trie, anything else against Lua keywords/builtins.
const apiRoot = "oblikovati"

// Kind tags a candidate so the popup can show an icon and the right colour.
type Kind int

const (
	KindKeyword Kind = iota
	KindBuiltin      // a Lua stdlib global/table or the `oblikovati` root
	KindModule       // an API namespace node (has children)
	KindMethod       // a callable wire method (leaf)
)

// Candidate is one suggestion: Text is the segment to insert (replacing the typed prefix),
// Kind drives its icon, and Detail is supplementary text (the full dotted method name for API
// entries) the popup shows to the right.
type Candidate struct {
	Text   string
	Kind   Kind
	Detail string
}

// Context is where a suggestion applies: ReplaceStart is the rune column at which the typed
// prefix began, so accepting a candidate replaces [ReplaceStart, caret) with its Text.
type Context struct {
	ReplaceStart int
}

// Engine answers completion queries against a fixed API method set plus the Lua keyword/builtin
// vocab. Construct it once per console and rebuild only if the host method set changes.
type Engine struct {
	root     *node
	keywords []string
	builtins []string
}

// New builds an engine from the dotted wire-method names the host publishes (methods()), e.g.
// {"documents.activate", "sketch.rectangle"}.
func New(methods []string) *Engine {
	return &Engine{root: buildTree(methods), keywords: lualex.Keywords(), builtins: lualex.Builtins()}
}

// Suggest returns the ranked candidates for the caret at rune column col in line, with the
// Context describing the span they replace. An empty result means nothing sensible to offer.
func (e *Engine) Suggest(line string, col int) ([]Candidate, Context) {
	chain, start := parseChain([]rune(line), col)
	ctx := Context{ReplaceStart: start}
	if len(chain) <= 1 {
		return e.bareCandidates(lastOf(chain)), ctx
	}
	if chain[0] != apiRoot {
		return nil, ctx
	}
	return e.apiCandidates(chain[1:len(chain)-1], lastOf(chain)), ctx
}

// bareCandidates completes a single unqualified word against Lua keywords, builtins and the API
// root, all filtered by prefix.
func (e *Engine) bareCandidates(prefix string) []Candidate {
	var out []Candidate
	out = appendMatches(out, e.keywords, prefix, KindKeyword)
	out = appendMatches(out, e.builtins, prefix, KindBuiltin)
	sortCandidates(out)
	return out
}

// apiCandidates completes the trailing prefix against the children of the API node reached by
// the already-typed path (the chain's middle segments). An unknown path yields nothing.
func (e *Engine) apiCandidates(path []string, prefix string) []Candidate {
	parent := e.root.walk(path)
	if parent == nil {
		return nil
	}
	dotted := strings.Join(path, ".")
	var out []Candidate
	for name, child := range parent.children {
		if !hasPrefixFold(name, prefix) {
			continue
		}
		out = append(out, apiCandidate(name, child, dotted))
	}
	sortCandidates(out)
	return out
}

// apiCandidate builds a candidate for one trie child, tagging it module vs. method and filling
// Detail with its dotted path under the API root.
func apiCandidate(name string, n *node, parentPath string) Candidate {
	kind := KindModule
	if n.method && len(n.children) == 0 {
		kind = KindMethod
	}
	return Candidate{Text: name, Kind: kind, Detail: dottedDetail(parentPath, name)}
}

// dottedDetail renders the full reference shown beside a candidate, e.g. "oblikovati.sketch.rectangle".
func dottedDetail(parentPath, name string) string {
	if parentPath == "" {
		return apiRoot + "." + name
	}
	return apiRoot + "." + parentPath + "." + name
}

// appendMatches adds every word in words that matches prefix (case-insensitively) as kind k.
func appendMatches(out []Candidate, words []string, prefix string, k Kind) []Candidate {
	for _, w := range words {
		if hasPrefixFold(w, prefix) {
			out = append(out, Candidate{Text: w, Kind: k})
		}
	}
	return out
}

// sortCandidates orders candidates by Text so the popup is stable and predictable.
func sortCandidates(c []Candidate) {
	sort.Slice(c, func(i, j int) bool { return c[i].Text < c[j].Text })
}

// hasPrefixFold reports whether s starts with prefix, ignoring ASCII case so "doc" matches
// "Documents" as well as "documents".
func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// lastOf returns the final element of chain, or "" for an empty chain.
func lastOf(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	return chain[len(chain)-1]
}
