// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"io/fs"
	"testing"

	"oblikovati.org/persistence/yamlcodec"
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
				{"e1", o.E1},
				{"e2", o.E2},
				{"e3", o.E3},
				{"g12", o.G12},
				{"g23", o.G23},
				{"g13", o.G13},
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

// TestMagneticMaterialsAreComplete enforces the magnetostatics-readiness invariant: a
// material that declares a magnetic class must carry the constitutive data its solver
// needs (a soft-magnetic core needs μr; a permanent magnet needs Br, Hc and recoil μr).
// Non-magnetic materials (the zero value) carry no magnetic group, so a forgotten `class:`
// tag with stray numbers, or a magnet missing its remanence, can't ship.
func TestMagneticMaterialsAreComplete(t *testing.T) {
	for _, m := range NewLibrary().Materials() {
		mag := m.Magnetic()
		if !mag.IsMagnetic() {
			if mag.RelativePermeability != 0 || mag.Remanence != 0 || mag.Coercivity != 0 {
				t.Errorf("material %q is non-magnetic but carries magnetic data (%+v) — missing class tag?", m.ID(), mag)
			}
			continue
		}
		if mag.RelativePermeability <= 0 {
			t.Errorf("material %q (%s): relativePermeability = %v, want > 0", m.ID(), mag.Class, mag.RelativePermeability)
		}
		if mag.Class == HardMagnetic && (mag.Remanence <= 0 || mag.Coercivity <= 0) {
			t.Errorf("permanent magnet %q: remanence=%v coercivity=%v, both want > 0", m.ID(), mag.Remanence, mag.Coercivity)
		}
		if mag.Class == SoftMagnetic && mag.SaturationFluxDensity <= 0 {
			t.Errorf("soft-magnetic %q: saturationFluxDensity = %v, want > 0", m.ID(), mag.SaturationFluxDensity)
		}
	}
}

// TestMagnetCatalogIDsPresent pins the motor-design / FEMM magnet + core grades other code
// and tests assign by id, so a catalog rename can't silently break the magnetics hand-off.
func TestMagnetCatalogIDsPresent(t *testing.T) {
	lib := NewLibrary()
	for _, id := range []string{"electrical-steel-m270", "magnet-ndfeb-n42", "magnet-smco-2-17", "magnet-ferrite-y30"} {
		m, ok := lib.Material(id)
		if !ok {
			t.Errorf("required magnetic material %q missing", id)
			continue
		}
		if !m.Magnetic().IsMagnetic() {
			t.Errorf("material %q must carry a magnetic class, got %+v", id, m.Magnetic())
		}
	}
}

// TestMagneticSurvivesRecipeRoundTrip locks the YAML persistence: a magnet keeps its full
// magnetic group through marshal/unmarshal, and a non-magnetic material writes no magnetic
// block (pointer stays nil, so saved files don't gain {class:"",...} noise).
func TestMagneticSurvivesRecipeRoundTrip(t *testing.T) {
	lib := NewLibrary()
	n42, _ := lib.Material("magnet-ndfeb-n42")
	r := materialToRecipe(n42)
	if r.Magnetic == nil {
		t.Fatal("magnet-ndfeb-n42 recipe lost its magnetic block")
	}
	back := recipeToMaterial(r, SourceBuiltin)
	if back.Magnetic() != n42.Magnetic() {
		t.Errorf("magnetic round-trip changed data: %+v -> %+v", n42.Magnetic(), back.Magnetic())
	}
	alu, ok := lib.Material("aluminum-6061")
	if !ok {
		t.Fatal("aluminum-6061 missing from catalog")
	}
	if got := materialToRecipe(alu); got.Magnetic != nil {
		t.Errorf("non-magnetic aluminum emitted a magnetic block: %+v", *got.Magnetic)
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
