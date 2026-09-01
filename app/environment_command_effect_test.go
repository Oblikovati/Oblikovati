// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2039 shipped six controls registered on the 3D Sketch tab whose implementation only ever
// handled the 2D sketch: every one answered ok and changed nothing. Neither archguard's
// dialog-path guard nor its activation-seam guard could see it — those check TOOLS, and these
// were plain CommandDefinitions.
//
// What actually catches that shape is running the commands. This walks every command registered
// for a sketch environment, executes it against a fixture with geometry selected in THAT
// environment, and fails when nothing observable changes (#2051). It is the audit's live sweep,
// headless: no frame hashes, no idle noise — just the state a command is supposed to move.

// environmentEffectExempt are the sketch-environment commands that legitimately change nothing
// observable when run against the fixture below. Each entry names why; an unexplained no-op
// fails instead.
var environmentEffectExempt = map[string]string{
	"Sketch.Finish":            "leaves the environment — measured by InSketch, which the fingerprint takes before it can act",
	"Sketch3D.Finish":          "leaves the environment, as above",
	"Sketch.Create2D":          "refuses to nest a second sketch, correctly",
	"Sketch.Create3D":          "refuses to nest a second sketch, correctly",
	"Sketch.LineType":          "a selection list: the head sets the value, the command itself is the anchor",
	"Sketch.Color":             "a selection list, as above",
	"Sketch.LineWeight":        "a selection list, as above",
	"Sketch3D.LineType":        "a selection list, as above",
	"Sketch3D.Color":           "a selection list, as above",
	"Sketch3D.LineWeight":      "a selection list, as above",
	"Sketch.RelaxMode":         "toggles a solver mode the fingerprint does not read",
	"Sketch.ShowConstraints":   "toggles an overlay the fingerprint does not read",
	"Sketch3D.ShowConstraints": "toggles the 3D constraint overlay the fingerprint does not read (#1998)",
	"Sketch.ShowFormat":        "toggles a persisted application option, not sketch state",
	"Sketch3D.ShowFormat":      "toggles a persisted application option, as above",
	"Sketch.DimensionsVisible": "toggles a display flag the fingerprint does not read",
}

// sketchState is what "the command did something" means here: the geometry, the construction
// census, the per-entity formatting, the armed creation modes, the selection, the undo stream,
// and which tool is armed — arming a tool IS the effect of a tool command, and what happens
// after the picks is the dialog-path and activation-seam guards' business.
func sketchState(s *Session) map[string]string {
	modes := s.FormatModes()
	entities, formats, dims, construction := 0, uint64(0), 0, 0
	if sk := s.ActiveSketch(); sk != nil {
		entities, formats, dims = len(sk.Entities()), sk.FormatRevision(), len(sk.DimensionConstraints().All())
		construction = countConstruction(sk.Entities())
	}
	if sk := s.ActiveSketch3D(); sk != nil {
		entities, formats = sk.EntityCount(), sk.FormatRevision()
		dims, construction = sk.DimensionConstraints3D().Count(), countConstruction(sk.Entities())
	}
	tool := ""
	if t := s.ActiveTool(); t != nil {
		tool = t.Name()
	}
	return map[string]string{
		"entities":     fmt.Sprint(entities),
		"construction": fmt.Sprint(construction),
		"formats":      fmt.Sprint(formats),
		"dimensions":   fmt.Sprint(dims),
		"modes":        fmt.Sprintf("%+v", modes),
		"selection":    fmt.Sprint(s.Selection().Count()),
		"undo":         s.UndoLabel(),
		"tool":         tool,
		"environment":  fmt.Sprintf("%t/%t", s.InSketch(), s.InSketch3D()),
	}
}

// changedFields lists which parts of the sketch state a command moved.
func changedFields(before, after map[string]string) []string {
	var out []string
	for k, v := range before {
		if after[k] != v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// countConstruction counts the entities flagged construction (which centerlines imply) — the
// state the Format panel's convert branch moves, and the one the audit found unmoved in 3D.
func countConstruction(ents []sketch.Entity) int {
	n := 0
	for _, e := range ents {
		if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
			n++
		}
	}
	return n
}

// TestSketch2DCommandsHaveAnEffect walks the 2D sketch environment's commands.
func TestSketch2DCommandsHaveAnEffect(t *testing.T) {
	t.Parallel()
	assertEnvironmentCommandsAct(t, SketchEnvironment, sketch2DEffectFixture)
}

// sketch2DEffectFixture is a 2D sketch holding one line, with that line selected.
func sketch2DEffectFixture(t *testing.T) *Session {
	t.Helper()
	s := registeredSession(t)
	enterSketchEnv(t, s)
	l := s.ActiveSketch().Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	s.Selection().Add(SketchEntityHandle{Entity: l})
	return s
}

// sketch3DEffectFixture is the same in a 3D sketch.
func sketch3DEffectFixture(t *testing.T) *Session {
	t.Helper()
	s := registeredSession(t)
	sk, err := s.CreateSketch3D()
	if err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(4, 0, 0))
	s.Selection().Add(SketchEntityHandle{Entity: l})
	return s
}

// TestSketch3DCommandsHaveAnEffect walks the 3D sketch environment's commands — the ones the
// audit found inert.
func TestSketch3DCommandsHaveAnEffect(t *testing.T) {
	t.Parallel()
	assertEnvironmentCommandsAct(t, Sketch3DEnvironment, sketch3DEffectFixture)
}

// assertEnvironmentCommandsAct executes every enabled command registered for env against a fresh
// fixture and fails on one that moves no observable state.
func assertEnvironmentCommandsAct(t *testing.T, env Environment, fixture func(*testing.T) *Session) {
	t.Helper()
	for _, id := range environmentCommandIDs(t, env, fixture) {
		t.Run(id, func(t *testing.T) {
			s := fixture(t)
			c, ok := s.Commands().ByID(id)
			if !ok || !c.IsEnabled(s) {
				return // registered but not live on this fixture: the enable predicate's business
			}
			before := sketchState(s)
			if err := s.Execute(id); err != nil {
				return // a command that REFUSES is not inert; it reported why
			}
			if len(changedFields(before, sketchState(s))) > 0 {
				return
			}
			if why, exempt := environmentEffectExempt[id]; exempt {
				t.Logf("%s: no observable effect, exempt (%s)", id, why)
				return
			}
			t.Errorf("%s is registered for %v, runs without error, and changes nothing observable "+
				"— the shape of the inert 3D Format panel (#2039). Wire it in this environment, "+
				"unregister it here, or declare it in environmentEffectExempt with the reason.", id, env)
		})
	}
}

// environmentCommandIDs lists the commands registered for env, sorted so failures are stable.
func environmentCommandIDs(t *testing.T, env Environment, fixture func(*testing.T) *Session) []string {
	t.Helper()
	s := fixture(t)
	var ids []string
	for _, c := range s.Commands().All() {
		if c.Environment() == env {
			ids = append(ids, c.ID())
		}
	}
	if len(ids) == 0 {
		t.Fatalf("no commands registered for %v — the environment scan is broken, not the code", env)
	}
	sort.Strings(ids)
	return ids
}

// TestSketchCommandParityAcrossEnvironments is the guard for the shape #2039 actually had.
//
// The inert 3D Format controls were not silent: with nothing selected they armed a creation
// mode, so they DID move session state and an effect check passes them. What was dead was the
// half that acts on a selection — the 3D convert branch and the armed mode's application — while
// the 2D twin, registered from the same shared list, worked.
//
// So: for every command registered under BOTH sketch environments, run it in each against a
// fixture with one line selected and require the SAME KIND of state to move. A 2D control that
// converts the selection while its 3D twin only flips a flag fails here.
func TestSketchCommandParityAcrossEnvironments(t *testing.T) {
	t.Parallel()
	for _, pair := range pairedSketchCommands(t) {
		t.Run(pair.name, func(t *testing.T) {
			got2D := runAndDiff(t, pair.id2D, sketch2DEffectFixture)
			got3D := runAndDiff(t, pair.id3D, sketch3DEffectFixture)
			if why, exempt := environmentParityExempt[pair.name]; exempt {
				t.Logf("%s: parity not required (%s); 2D moved %v, 3D moved %v", pair.name, why, got2D, got3D)
				return
			}
			if !sameFields(got2D, got3D) {
				t.Errorf("%s moves %v in the 2D sketch but %v in the 3D one — one environment's "+
					"implementation is missing, which is how the 3D Format panel shipped inert "+
					"(#2039). Wire the missing half, unregister the control there, or declare the "+
					"pair in environmentParityExempt with the reason.", pair.name, got2D, got3D)
			}
		})
	}
}

// environmentParityExempt are the 2D/3D command pairs that legitimately differ, with why.
var environmentParityExempt = map[string]string{
	"Finish": "2D exits to the base environment and swings the camera back; 3D has no host plane",
}

// sketchCommandPair is one control registered under both sketch environments.
type sketchCommandPair struct{ name, id2D, id3D string }

// pairedSketchCommands finds every Sketch.X with a Sketch3D.X twin.
func pairedSketchCommands(t *testing.T) []sketchCommandPair {
	t.Helper()
	s := registeredSession(t)
	have := map[string]bool{}
	for _, c := range s.Commands().All() {
		have[c.ID()] = true
	}
	var out []sketchCommandPair
	for _, c := range s.Commands().All() {
		name, ok := strings.CutPrefix(c.ID(), "Sketch.")
		if !ok || !have["Sketch3D."+name] {
			continue
		}
		out = append(out, sketchCommandPair{name: name, id2D: c.ID(), id3D: "Sketch3D." + name})
	}
	if len(out) == 0 {
		t.Fatal("no 2D/3D command pairs found — the pairing scan is broken, not the code")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// runAndDiff executes one command against a fresh fixture and reports which state it moved.
func runAndDiff(t *testing.T, id string, fixture func(*testing.T) *Session) []string {
	t.Helper()
	s := fixture(t)
	c, ok := s.Commands().ByID(id)
	if !ok || !c.IsEnabled(s) {
		return nil
	}
	before := sketchState(s)
	if err := s.Execute(id); err != nil {
		return nil
	}
	return changedFields(before, sketchState(s))
}

// sameFields compares two changed-field lists as sets.
func sameFields(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
