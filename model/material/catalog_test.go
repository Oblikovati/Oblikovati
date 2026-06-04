// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"io/fs"
	"testing"

	"github.com/Oblikovati/oblikovati/persistence/yamlcodec"
)

// TestCatalogPopulatesLibrary guards against a regression where an embedded file fails to
// load silently (e.g. a renamed glob): the shipped catalog must seed a substantial set.
func TestCatalogPopulatesLibrary(t *testing.T) {
	lib := NewLibrary()
	if got := len(lib.Materials()); got < 100 {
		t.Errorf("built-in materials = %d, want >= 100 (catalog failed to load?)", got)
	}
	if got := len(lib.Appearances()); got < 80 {
		t.Errorf("built-in appearances = %d, want >= 80 (catalog failed to load?)", got)
	}
}

// TestCatalogIDsUnique fails loudly on a duplicate id across files. AddAppearance/
// AddMaterial replace silently, so a copy-paste collision would otherwise hide one entry.
func TestCatalogIDsUnique(t *testing.T) {
	apprIDs, matIDs := map[string]string{}, map[string]string{}
	forEachCatalogFile(t, func(name string, rd RecipeData) {
		for _, a := range rd.Appearances {
			if prev, dup := apprIDs[a.ID]; dup {
				t.Errorf("appearance id %q duplicated in %q and %q", a.ID, prev, name)
			}
			apprIDs[a.ID] = name
		}
		for _, m := range rd.Materials {
			if prev, dup := matIDs[m.ID]; dup {
				t.Errorf("material id %q duplicated in %q and %q", m.ID, prev, name)
			}
			matIDs[m.ID] = name
		}
	})
}

// TestEveryMaterialHasReliableProperties enforces the catalog's admission rule: an entry
// ships only when every mechanical, thermal and electrical field is present (non-zero).
func TestEveryMaterialHasReliableProperties(t *testing.T) {
	for _, m := range NewLibrary().Materials() {
		mech, th, el := m.Mechanical(), m.Thermal(), m.Electrical()
		checks := []struct {
			label string
			v     float64
		}{
			{"density", m.Density()},
			{"youngsModulus", mech.YoungsModulus},
			{"poissonsRatio", mech.PoissonsRatio},
			{"yieldStrength", mech.YieldStrength},
			{"ultimateTensileStrength", mech.UltimateTensileStrength},
			{"conductivity", th.Conductivity},
			{"expansionCoeff", th.ExpansionCoeff},
			{"specificHeat", th.SpecificHeat},
			{"resistivity", el.Resistivity},
			{"relativePermittivity", el.RelativePermittivity},
		}
		for _, c := range checks {
			if c.v <= 0 {
				t.Errorf("material %q: %s = %v, want > 0 (incomplete data must not ship)", m.ID(), c.label, c.v)
			}
		}
	}
}

// TestAppearanceValuesInRange keeps PBR inputs physical: metallic/roughness/opacity in
// [0,1], and opacity strictly positive so no built-in renders fully invisible.
func TestAppearanceValuesInRange(t *testing.T) {
	for _, a := range NewLibrary().Appearances() {
		if a.Metallic() < 0 || a.Metallic() > 1 {
			t.Errorf("appearance %q: metallic = %v, want [0,1]", a.ID(), a.Metallic())
		}
		if a.Roughness() < 0 || a.Roughness() > 1 {
			t.Errorf("appearance %q: roughness = %v, want [0,1]", a.ID(), a.Roughness())
		}
		if a.Opacity() <= 0 || a.Opacity() > 1 {
			t.Errorf("appearance %q: opacity = %v, want (0,1]", a.ID(), a.Opacity())
		}
	}
}

// TestRequiredIDsPresent pins the handful of ids that other packages and tests reference,
// so a catalog edit can't silently break material assignment elsewhere.
func TestRequiredIDsPresent(t *testing.T) {
	lib := NewLibrary()
	for _, id := range []string{DefaultAppearanceID, "steel", "aluminum", "oak", "abs-black"} {
		if _, ok := lib.Appearance(id); !ok {
			t.Errorf("required built-in appearance %q missing", id)
		}
	}
	for _, id := range []string{"steel", "aluminum-6061"} {
		if _, ok := lib.Material(id); !ok {
			t.Errorf("required built-in material %q missing", id)
		}
	}
}

// TestAnisotropicMaterialsAreComplete enforces the simulation-readiness invariant: a
// material declared orthotropic/transversely-isotropic must carry a full positive elastic
// group, and conversely a populated elastic group must declare a non-isotropic class (so a
// forgotten `isotropy:` tag can't ship). Metals and bulk plastics stay isotropic with none.
func TestAnisotropicMaterialsAreComplete(t *testing.T) {
	for _, m := range NewLibrary().Materials() {
		o := m.Anisotropic()
		anis := m.IsotropyClass().Anisotropic()
		if anis {
			for _, c := range []struct {
				label string
				v     float64
			}{
				{"e1", o.E1}, {"e2", o.E2}, {"e3", o.E3},
				{"g12", o.G12}, {"g23", o.G23}, {"g13", o.G13},
			} {
				if c.v <= 0 {
					t.Errorf("material %q (%s): orthotropic %s = %v, want > 0", m.ID(), m.IsotropyClass(), c.label, c.v)
				}
			}
			continue
		}
		if o.E1 != 0 || o.E2 != 0 || o.E3 != 0 {
			t.Errorf("material %q is isotropic but carries orthotropic data (%+v) — missing isotropy tag?", m.ID(), o)
		}
	}
}

// TestOrthotropicSurvivesRecipeRoundTrip locks the YAML persistence: an anisotropic
// material keeps its elastic group through marshal/unmarshal, an isotropic one writes no
// orthotropic block (pointer stays nil, so saved files don't gain empty {e1:0,...} noise).
func TestOrthotropicSurvivesRecipeRoundTrip(t *testing.T) {
	lib := NewLibrary()
	oak, _ := lib.Material("oak-red")
	r := materialToRecipe(oak)
	if r.Orthotropic == nil {
		t.Fatal("oak-red recipe lost its orthotropic block")
	}
	back := recipeToMaterial(r, SourceBuiltin)
	if back.Anisotropic() != oak.Anisotropic() {
		t.Errorf("orthotropic round-trip changed data: %+v -> %+v", oak.Anisotropic(), back.Anisotropic())
	}

	steel, _ := lib.Material("steel")
	if got := materialToRecipe(steel); got.Orthotropic != nil {
		t.Errorf("isotropic steel emitted an orthotropic block: %+v", *got.Orthotropic)
	}
}

// forEachCatalogFile parses every embedded catalog file and invokes fn — a fake-free way
// to assert over the shipped data directly (the embed.FS is the production source).
func forEachCatalogFile(t *testing.T, fn func(name string, rd RecipeData)) {
	t.Helper()
	names, err := fs.Glob(catalogFiles, "catalog/*.yaml")
	if err != nil {
		t.Fatalf("glob catalog: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no catalog files embedded")
	}
	for _, name := range names {
		data, err := catalogFiles.ReadFile(name)
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		var rd RecipeData
		if err := yamlcodec.Unmarshal(data, &rd); err != nil {
			t.Fatalf("parse %q: %v", name, err)
		}
		fn(name, rd)
	}
}
