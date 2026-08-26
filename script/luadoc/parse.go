// SPDX-License-Identifier: GPL-2.0-only

package luadoc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// collectMethods parses the api/wire and api/client packages and joins them into the
// scriptable method set: every client method carries its wire constant, its mcp:summary, and
// (via the wire request DTO) its argument fields.
func collectMethods(apiDir string) ([]Method, error) {
	constVals, structs, err := parseWire(filepath.Join(apiDir, "wire"))
	if err != nil {
		return nil, err
	}
	clientMethods, err := parseClient(filepath.Join(apiDir, "client"))
	if err != nil {
		return nil, err
	}
	var out []Method
	for _, cm := range clientMethods {
		wireName, ok := constVals[cm.wireConst]
		if !ok || !strings.Contains(wireName, ".") {
			continue // not a dotted wire method (or an unknown constant)
		}
		group, leaf, _ := strings.Cut(wireName, ".")
		out = append(out, Method{
			Wire: wireName, Group: group, Leaf: leaf,
			Summary: cm.summary, Args: structs[cm.argsType],
		})
	}
	return out, nil
}

// clientMethod is one parsed api/client method: the wire constant it calls, its mcp:summary,
// and the name of its request-DTO type (empty when it sends no args).
type clientMethod struct {
	wireConst string
	summary   string
	argsType  string
}

// parseClient walks api/client and returns one clientMethod per method that invokes the
// transport (c.call(wire.Method…, args, &r)).
func parseClient(dir string) ([]clientMethod, error) {
	files, err := goFiles(dir)
	if err != nil {
		return nil, err
	}
	var out []clientMethod
	fset := token.NewFileSet()
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, decl := range af.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				continue
			}
			if cm, ok := clientMethodOf(fn); ok {
				out = append(out, cm)
			}
		}
	}
	return out, nil
}

// clientMethodOf extracts the wire constant, summary and args type from one client method,
// reporting false when the function does not call the transport.
func clientMethodOf(fn *ast.FuncDecl) (clientMethod, bool) {
	call, wireArg := findTransportCall(fn.Body)
	if call == nil || len(call.Args) <= wireArg {
		return clientMethod{}, false
	}
	wireConst := selectorName(call.Args[wireArg])
	if wireConst == "" {
		return clientMethod{}, false
	}
	cm := clientMethod{wireConst: wireConst, summary: mcpSummary(fn.Doc)}
	if argsIdx := wireArg + 1; len(call.Args) > argsIdx {
		cm.argsType = argsTypeName(call.Args[argsIdx], fn.Type.Params, localWireTypes(fn.Body))
	}
	return cm, true
}

// localWireTypes maps each local variable assigned a wire type to that type name, covering the
// common `args := wire.Foo{…}` and `var args wire.Foo` request-building patterns.
func localWireTypes(body *ast.BlockStmt) map[string]string {
	out := map[string]string{}
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			recordAssignedWireTypes(s, out)
		case *ast.DeclStmt:
			recordDeclaredWireTypes(s, out)
		}
		return true
	})
	return out
}

// recordAssignedWireTypes records `name := wire.Foo{…}` bindings.
func recordAssignedWireTypes(s *ast.AssignStmt, out map[string]string) {
	for i, lhs := range s.Lhs {
		id, ok := lhs.(*ast.Ident)
		if !ok || i >= len(s.Rhs) {
			continue
		}
		if t := wireTypeOfExpr(s.Rhs[i]); t != "" {
			out[id.Name] = t
		}
	}
}

// recordDeclaredWireTypes records `var name wire.Foo` declarations.
func recordDeclaredWireTypes(s *ast.DeclStmt, out map[string]string) {
	gd, ok := s.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
		return
	}
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || vs.Type == nil {
			continue
		}
		if t := selectorName(vs.Type); t != "" {
			for _, id := range vs.Names {
				out[id.Name] = t
			}
		}
	}
}

// wireTypeOfExpr returns the wire type name an expression constructs (a composite literal
// wire.Foo{…} or a &wire.Foo{…}), or "".
func wireTypeOfExpr(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.CompositeLit:
		return selectorName(x.Type)
	case *ast.UnaryExpr:
		return wireTypeOfExpr(x.X)
	}
	return ""
}

// findTransportCall returns the first `<recv>.call(...)` invocation in a body, which is how
// every client method reaches the wire transport.
// findTransportCall locates the call that dispatches a wire method in a client method body and the
// argument index holding the wire.Method constant. It handles both the legacy method form
// `receiver.call(wire.Method, args, &r)` (wire const at arg 0) and the generic helper form
// `call[Resp](c, wire.Method, args)` introduced by the G2 refactor (Oblikovati.API #1650), whose
// first argument is the *Client receiver, so the wire const is at arg 1.
func findTransportCall(body *ast.BlockStmt) (*ast.CallExpr, int) {
	var found *ast.CallExpr
	wireArg := 0
	ast.Inspect(body, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := ce.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "call" {
			found, wireArg = ce, 0
			return false
		}
		if isGenericCallHelper(ce.Fun) {
			found, wireArg = ce, 1
			return false
		}
		return true
	})
	return found, wireArg
}

// isGenericCallHelper reports whether fun is the generic package-level call[...] helper — an Ident
// "call" instantiated with one (IndexExpr) or more (IndexListExpr) type arguments.
func isGenericCallHelper(fun ast.Expr) bool {
	switch e := fun.(type) {
	case *ast.IndexExpr:
		id, ok := e.X.(*ast.Ident)
		return ok && id.Name == "call"
	case *ast.IndexListExpr:
		id, ok := e.X.(*ast.Ident)
		return ok && id.Name == "call"
	default:
		return false
	}
}

// mcpSummary returns the text of the `mcp:summary …` doc line, or "".
func mcpSummary(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	for line := range strings.SplitSeq(doc.Text(), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "mcp:summary "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// argsTypeName resolves the request-DTO type name from the transport call's second argument:
// a composite literal wire.Foo{…}, or an identifier bound to wire.Foo as a parameter or a
// local variable. Returns "" for nil or an unrecognised shape.
func argsTypeName(arg ast.Expr, params *ast.FieldList, locals map[string]string) string {
	switch a := arg.(type) {
	case *ast.CompositeLit:
		return selectorName(a.Type)
	case *ast.UnaryExpr:
		return wireTypeOfExpr(a)
	case *ast.Ident:
		if a.Name == "nil" {
			return ""
		}
		if t := paramTypeName(a.Name, params); t != "" {
			return t
		}
		return locals[a.Name]
	}
	return ""
}

// paramTypeName returns the wire type name of the named parameter (e.g. param `args
// wire.CreateDocumentArgs` → "CreateDocumentArgs"), or "".
func paramTypeName(name string, params *ast.FieldList) string {
	if params == nil {
		return ""
	}
	for _, p := range params.List {
		for _, id := range p.Names {
			if id.Name == name {
				return selectorName(p.Type)
			}
		}
	}
	return ""
}

// selectorName returns the Sel name of a `wire.Name` selector (e.g. wire.MethodX → "MethodX"),
// or "" when expr is not a selector on the wire package.
func selectorName(expr ast.Expr) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "wire" {
		return ""
	}
	return sel.Sel.Name
}

// parseWire walks api/wire and returns the string-const values (name→dotted value) and the
// request-DTO field sets (struct name→fields).
func parseWire(dir string) (map[string]string, map[string][]Field, error) {
	files, err := goFiles(dir)
	if err != nil {
		return nil, nil, err
	}
	consts := map[string]string{}
	structs := map[string][]Field{}
	fset := token.NewFileSet()
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, nil, err
		}
		for _, decl := range af.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			collectConstsAndStructs(gen, consts, structs)
		}
	}
	return consts, structs, nil
}

// collectConstsAndStructs records string consts and struct field sets from one declaration.
func collectConstsAndStructs(gen *ast.GenDecl, consts map[string]string, structs map[string][]Field) {
	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *ast.ValueSpec:
			if gen.Tok == token.CONST {
				recordStringConst(s, consts)
			}
		case *ast.TypeSpec:
			if st, ok := s.Type.(*ast.StructType); ok {
				structs[s.Name.Name] = structFields(st)
			}
		}
	}
}

// recordStringConst stores a `Name = "value"` const.
func recordStringConst(s *ast.ValueSpec, consts map[string]string) {
	if len(s.Names) != 1 || len(s.Values) != 1 {
		return
	}
	lit, ok := s.Values[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	if v, err := strconv.Unquote(lit.Value); err == nil {
		consts[s.Names[0].Name] = v
	}
}

// structFields returns a struct's scriptable fields: the JSON key (tag name, else field name)
// and a readable Go type, skipping embedded and json:"-" fields.
func structFields(st *ast.StructType) []Field {
	var out []Field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // embedded
		}
		name := jsonName(f)
		if name == "-" {
			continue
		}
		for _, id := range f.Names {
			key := name
			if key == "" {
				key = id.Name
			}
			out = append(out, Field{Name: key, Type: typeString(f.Type)})
		}
	}
	return out
}

// jsonName returns the field's json tag name ("" when untagged, "-" when omitted).
func jsonName(f *ast.Field) string {
	if f.Tag == nil {
		return ""
	}
	tag, err := strconv.Unquote(f.Tag.Value)
	if err != nil {
		return ""
	}
	js := reflect.StructTag(tag).Get("json")
	if js == "" {
		return ""
	}
	return strings.Split(js, ",")[0]
}

// typeString renders an AST type as compact Go source (e.g. "[]string", "*FaceRef").
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.InterfaceType:
		return "any"
	default:
		return "?"
	}
}

// goFiles lists the non-test .go files in dir.
func goFiles(dir string) ([]string, error) {
	all, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	out := all[:0]
	for _, f := range all {
		if !strings.HasSuffix(f, "_test.go") {
			out = append(out, f)
		}
	}
	return out, nil
}
