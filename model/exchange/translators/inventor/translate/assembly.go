// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	m "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/persistence"
)

// AssemblyFromInventor translates an Inventor assembly (.iam) at iamPath into a native
// Oblikovati assembly package at outPath. Each referenced component — a sibling
// <component>.ipt part or <component>.iam sub-assembly — is translated to a sibling
// <component>.opd and placed at its decoded transform (model space, cm); sub-assemblies
// recurse. Returns non-fatal warnings (e.g. a component file that could not be found).
//
// It reads from the filesystem (not just bytes) because an assembly references its
// components by file — the .iam alone does not carry their geometry.
func AssemblyFromInventor(iamPath, outPath string) ([]string, error) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	asmDoc, warns, err := buildAssembly(ws, iamPath, outPath, map[string]*doc.Document{}, map[string]bool{})
	if err != nil {
		return warns, err
	}
	if err := ws.Save(asmDoc); err != nil {
		return warns, err
	}
	return warns, nil
}

// buildAssembly builds (without saving) an assembly document at outPath from the .iam at
// iamPath, recursively translating its component parts and sub-assemblies into ws. built
// caches already-translated components by name so a shared component is translated once and
// its .opd path doesn't collide; inProgress guards against reference cycles.
func buildAssembly(ws *doc.Workspace, iamPath, outPath string, built map[string]*doc.Document, inProgress map[string]bool) (*doc.Document, []string, error) {
	d, err := openAssembly(iamPath)
	if err != nil {
		return nil, nil, err
	}
	srcDir, outDir := filepath.Dir(iamPath), filepath.Dir(outPath)
	// Keep only occurrences whose component exists on disk (as a part or sub-assembly); this
	// also rejects the spurious "hash:N" names constraint geometry selections emit.
	placed := d.PlacedOccurrences(func(component string) bool {
		return componentPath(srcDir, component) != ""
	})
	if len(placed) == 0 {
		return nil, nil, fmt.Errorf("assembly %s references no components found next to it", filepath.Base(iamPath))
	}
	components, warns := buildComponents(ws, placed, srcDir, outDir, built, inProgress)
	asmDoc, err := compdef.AddAssembly(ws, outPath, true)
	if err != nil {
		return nil, warns, err
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	warns = append(warns, placeOccurrences(asm, asmDoc, components, placed)...)
	warns = append(warns, constraintWarnings(d)...)
	return asmDoc, warns, nil
}

// componentPath returns the sibling file for a component name — its .iam sub-assembly
// preferred, else its .ipt part — or "" if neither is present.
func componentPath(srcDir, name string) string {
	for _, ext := range []string{".iam", ".ipt"} {
		if p := filepath.Join(srcDir, name+ext); fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// constraintWarnings reports the assembly's placement constraints. Occurrences are placed at
// their solved positions (so the geometry is already correct), but the parametric constraint
// relationships are not yet rebuilt — resolving each geometry selection to a primitive is
// future work. A warning per constraint kind keeps that explicit rather than silent.
func constraintWarnings(d *ipt.Document) []string {
	kinds := ipt.DecodeConstraintKinds(d)
	if len(kinds) == 0 {
		return nil
	}
	counts := map[string]int{}
	for _, k := range kinds {
		counts[k.String()]++
	}
	var warns []string
	for name, n := range counts {
		warns = append(warns, fmt.Sprintf("%d %s constraint(s) placed by solved position but not rebuilt as a relationship", n, name))
	}
	return warns
}

// openAssembly reads and opens the .iam, erroring if it is missing or is not an assembly.
func openAssembly(iamPath string) (*ipt.Document, error) {
	raw, err := os.ReadFile(iamPath)
	if err != nil {
		return nil, err
	}
	d, err := ipt.Open(raw)
	if err != nil {
		return nil, err
	}
	if !d.IsAssembly() {
		return nil, fmt.Errorf("%s is a part, not an assembly (use FromInventor)", filepath.Base(iamPath))
	}
	return d, nil
}

// buildComponents translates each distinct referenced component into a document saved as
// <component>.opd in outDir — a part (.ipt) via buildPart, a sub-assembly (.iam) by
// recursing into buildAssembly — returning them keyed by component name. A missing,
// cyclic, or unreadable component becomes a warning (its occurrences are skipped).
func buildComponents(ws *doc.Workspace, placed []ipt.PlacedOccurrence, srcDir, outDir string, built map[string]*doc.Document, inProgress map[string]bool) (map[string]*doc.Document, []string) {
	components := map[string]*doc.Document{}
	var warns []string
	for _, name := range distinctComponents(placed) {
		if d, ok := built[name]; ok { // already translated (shared component)
			components[name] = d
			continue
		}
		if inProgress[name] {
			warns = append(warns, fmt.Sprintf("component %q: reference cycle, skipped", name))
			continue
		}
		cdoc, cw, err := buildComponent(ws, componentPath(srcDir, name), filepath.Join(outDir, name+".opd"), built, inProgress)
		warns = append(warns, cw...)
		if err != nil {
			warns = append(warns, fmt.Sprintf("component %q: %v", name, err))
			continue
		}
		if err := ws.Save(cdoc); err != nil {
			warns = append(warns, fmt.Sprintf("component %q: save: %v", name, err))
			continue
		}
		built[name] = cdoc
		components[name] = cdoc
	}
	return components, warns
}

// buildComponent translates one component file (built, not saved): a sub-assembly (.iam)
// recurses through buildAssembly (guarded against cycles), a part (.ipt) goes through
// buildPart.
func buildComponent(ws *doc.Workspace, srcPath, outPath string, built map[string]*doc.Document, inProgress map[string]bool) (*doc.Document, []string, error) {
	if srcPath == "" {
		return nil, nil, fmt.Errorf("component file not found")
	}
	if strings.EqualFold(filepath.Ext(srcPath), ".iam") {
		name := componentName(srcPath)
		inProgress[name] = true
		defer delete(inProgress, name)
		return buildAssembly(ws, srcPath, outPath, built, inProgress)
	}
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, nil, err
	}
	pd, err := ipt.Open(raw)
	if err != nil {
		return nil, nil, err
	}
	return buildPart(ws, outPath, pd)
}

// componentName is a component's base name (file name without extension).
func componentName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// placeOccurrences places one occurrence per decoded record at its transform, skipping any
// whose component failed to build.
func placeOccurrences(asm *compdef.AssemblyComponentDefinition, asmDoc *doc.Document, components map[string]*doc.Document, placed []ipt.PlacedOccurrence) []string {
	var warns []string
	for _, p := range placed {
		cdoc, ok := components[p.Component]
		if !ok {
			continue // component build failed; already warned
		}
		name := fmt.Sprintf("%s:%d", p.Component, p.Instance)
		if _, err := asm.PlaceComponentFromFile(asmDoc, cdoc, name, transformOf(p.Transform)); err != nil {
			warns = append(warns, fmt.Sprintf("place %q: %v", name, err))
		}
	}
	return warns
}

// distinctComponents returns the component names referenced by the placed occurrences, in
// first-seen order (one part file to translate per name).
func distinctComponents(placed []ipt.PlacedOccurrence) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range placed {
		if !seen[p.Component] {
			seen[p.Component] = true
			out = append(out, p.Component)
		}
	}
	return out
}

// transformOf converts a decoded row-major transform (cm) to a kernel transform. Both are
// row-major with translation in cells 3/7/11, so the cells map one-to-one.
func transformOf(t ipt.Matrix4) m.Matrix4 {
	var cells [16]m.Scalar
	for i, v := range t {
		cells[i] = m.Scalar(v)
	}
	return m.Matrix4FromCells(cells)
}
