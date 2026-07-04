// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
)

// Characterization golden for the boolean manifold-extraction sew (#1726, Slice 0).
//
// This is the REGRESSION GATE for the Weiler radial-edge refactor (ADR-0047): the sew is
// being re-shaped (radialSew → sewPlan → mintEntities), and every NON-degenerate boolean and
// every COPLANAR-FLUSH union must come out BYTE-IDENTICAL — same V/E/F/chi AND the same
// reference keys (ADR-0043 topological naming), because downstream feature references
// (fillets/chamfers keyed on brep:edge#N) break if a key drifts. The signature hashes the
// sorted reference keys of every vertex/edge/face, so it is invariant to build/map order but
// sensitive to any naming or topology change. Captured on develop BEFORE the refactor and held
// invariant through Slices 1–2. (Bowtie line-tangencies are deliberately EXCLUDED — they change
// from nudged to exact in Slice 2 and are certified separately.)

// sewSignature is an order-independent fingerprint of a body's topology and reference-key naming.
func sewSignature(b *topo.Body) string {
	keys := make([]string, 0, len(b.Vertices())+len(b.Edges())+len(b.Faces()))
	for _, v := range b.Vertices() {
		keys = append(keys, "V"+hex.EncodeToString(v.ReferenceKey()))
	}
	for _, e := range b.Edges() {
		keys = append(keys, "E"+hex.EncodeToString(e.ReferenceKey()))
	}
	for _, f := range b.Faces() {
		keys = append(keys, "F"+hex.EncodeToString(f.ReferenceKey()))
	}
	sort.Strings(keys)
	h := sha256.Sum256([]byte(strings.Join(keys, "|")))
	return fmt.Sprintf("V%dE%dF%dchi%d-%s",
		len(b.Vertices()), len(b.Edges()), len(b.Faces()), b.EulerCharacteristic(),
		hex.EncodeToString(h[:8]))
}

// goldenSewCase is one boolean whose exact output must survive the radial-edge refactor.
type goldenSewCase struct {
	name string
	want string
	make func() (*topo.Body, error)
}

func goldenSewCases() []goldenSewCase {
	return []goldenSewCase{
		{"union-overlap-boxes", "V20E30F12chi2-866f1cba152afbc6", func() (*topo.Body, error) {
			return brep.Boolean(brep.Union, box(0, 0, 0, 2, 2, 2), box(1, 1, 1, 2, 2, 2))
		}},
		{"intersect-overlap-boxes", "V8E12F6chi2-f2be246226ca5f9f", func() (*topo.Body, error) {
			return brep.Boolean(brep.Intersection, box(0, 0, 0, 2, 2, 2), box(1, 1, 1, 2, 2, 2))
		}},
		{"diff-notch", "V16E24F10chi0-b69e373caca6bcfa", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, box(0, 0, 0, 3, 3, 1), box(1, 1, -1, 1, 1, 3))
		}},
		{"flush-stack-union", "V12E20F10chi2-1a9804dde167db10", func() (*topo.Body, error) { // shared full face z=1 (coplanar fuse)
			return brep.Boolean(brep.Union, box(0, 0, 0, 2, 2, 1), box(0, 0, 1, 2, 2, 1))
		}},
		{"flush-partial-union", "V18E28F12chi2-70d5f7e3ab6d0048", func() (*topo.Body, error) { // partial coplanar overlap at z=1
			return brep.Boolean(brep.Union, box(0, 0, 0, 2, 2, 1), box(0.5, 0.5, 1, 2, 2, 1))
		}},
		{"prism-drill-box", "V72E108F38chi0-4f66712ca83d1823", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, box(-1, -1, 0, 2, 2, 2), cylinderZAt(0, 0, -0.1, 2.1, 0.5, "drill"))
		}},
		{"union-crossing-prisms", "V192E288F100chi2-ec27452c5531aa21", func() (*topo.Body, error) {
			return brep.Boolean(brep.Union, cylinderZAt(0, 0, 0, 2, 0.6, "a"), cylinderZAt(0, 0, 0.9, 1.1, 1.2, "b"))
		}},
		{"diff-corner-box", "V14E21F9chi2-eb2d66601150fbb8", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, box(0, 0, 0, 2, 2, 2), box(1.5, 1.5, 1.5, 2, 2, 2))
		}},
	}
}

func TestBooleanSewGolden(t *testing.T) {
	for _, c := range goldenSewCases() {
		t.Run(c.name, func(t *testing.T) {
			b, err := c.make()
			if err != nil {
				t.Fatalf("%s: build: %v", c.name, err)
			}
			if b == nil {
				t.Fatalf("%s: nil body", c.name)
			}
			got := sewSignature(b)
			if c.want == "" {
				t.Fatalf("%s: CAPTURE want=%q", c.name, got)
			}
			if got != c.want {
				t.Fatalf("%s: sew signature drifted\n  got  %s\n  want %s", c.name, got, c.want)
			}
		})
	}
}
