// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"errors"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// moduleSourceImporter type-checks this workspace's own packages FROM SOURCE, so an expression's
// type is known even when it comes from oblikovati.org/math or oblikovati.org/api. The gc
// export-data importer (go/importer.Default) cannot see either module — they are in neither GOROOT
// nor GOPATH — and it fails SILENTLY: the affected expressions simply carry no type, so a guard
// built on it passes vacuously. Used by TestNoFusableProductSums (#3528).
type moduleSourceImporter struct {
	fset     *token.FileSet
	dirs     map[string]string // import-path prefix -> directory
	cache    map[string]*types.Package
	inFlight map[string]bool
	fallback types.Importer
}

func newModuleSourceImporter(fset *token.FileSet, repoRoot, apiRoot string) *moduleSourceImporter {
	return &moduleSourceImporter{
		fset:     fset,
		dirs:     map[string]string{"oblikovati.org/api": apiRoot, "oblikovati.org": repoRoot},
		cache:    map[string]*types.Package{},
		inFlight: map[string]bool{},
		fallback: importer.Default(),
	}
}

func (im *moduleSourceImporter) dirFor(path string) (string, bool) {
	if path == "oblikovati.org/api" {
		return im.dirs["oblikovati.org/api"], true
	}
	for _, prefix := range []string{"oblikovati.org/api/", "oblikovati.org/"} {
		if strings.HasPrefix(path, prefix) {
			return filepath.Join(im.dirs[strings.TrimSuffix(prefix, "/")], filepath.FromSlash(strings.TrimPrefix(path, prefix))), true
		}
	}
	return "", false
}

func (im *moduleSourceImporter) Import(path string) (*types.Package, error) {
	if p, ok := im.cache[path]; ok {
		return p, nil
	}
	dir, ours := im.dirFor(path)
	if !ours {
		return im.fallback.Import(path)
	}
	if im.inFlight[path] {
		return nil, errors.New("import cycle at " + path)
	}
	im.inFlight[path] = true
	defer delete(im.inFlight, path)
	pkg, _, err := im.check(path, dir, nil)
	if err != nil {
		return nil, err
	}
	im.cache[path] = pkg
	return pkg, nil
}

// check type-checks the package in dir under the given import path, filling info when non-nil.
func (im *moduleSourceImporter) check(path, dir string, info *types.Info) (*types.Package, []*ast.File, error) {
	files, err := parseNonTest(im.fset, dir)
	if err != nil {
		return nil, nil, err
	}
	conf := types.Config{Importer: im, Sizes: types.SizesFor("gc", "amd64")}
	pkg, err := conf.Check(path, im.fset, files, info)
	return pkg, files, err
}

// parseNonTest parses every non-test .go file in dir, keeping the dominant package only.
func parseNonTest(fset *token.FileSet, dir string) ([]*ast.File, error) {
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		if strings.HasSuffix(fi.Name(), "_test.go") {
			return false
		}
		ok, err := build.Default.MatchFile(dir, fi.Name())
		return err == nil && ok
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var best []*ast.File
	for _, pkg := range pkgs {
		var names []string
		for n := range pkg.Files {
			names = append(names, n)
		}
		sort.Strings(names)
		files := make([]*ast.File, 0, len(names))
		for _, n := range names {
			files = append(files, pkg.Files[n])
		}
		if len(files) > len(best) {
			best = files
		}
	}
	return best, nil
}
