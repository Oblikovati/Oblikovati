// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestProjectionSourceKindTags: each reference source reports the kind tag persistence uses to
// rebuild it (#1268).
func TestProjectionSourceKindTags(t *testing.T) {
	d := NewPartComponentDefinition()
	cases := []struct {
		kind string
		got  string
	}{
		{"edge", NewEdgeRefSource(d, "k").SourceKind()},
		{"vertex", NewVertexRefSource(d, "k").SourceKind()},
		{"workPoint", NewWorkPointRefSource(d, feature.OriginCenter).SourceKind()},
		{"workAxis", NewWorkAxisRefSource(d, feature.OriginXAxis).SourceKind()},
		{"workPlane", NewWorkPlaneRefSource(d, feature.OriginXYPlane, sketch.XYPlane()).SourceKind()},
	}
	for _, c := range cases {
		if c.got != c.kind {
			t.Errorf("SourceKind = %q, want %q", c.got, c.kind)
		}
	}
}

// TestProjectionSourceBuilders: the rebind builders construct a source for every known kind and
// reject an unknown one (the frozen fallback).
func TestProjectionSourceBuilders(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, ok := d.pointRefSource("vertex", "k"); !ok {
		t.Error("vertex point source should build")
	}
	if _, ok := d.pointRefSource("workPoint", string(feature.OriginCenter)); !ok {
		t.Error("workPoint source should build")
	}
	if _, ok := d.pointRefSource("mystery", "k"); ok {
		t.Error("unknown point kind must not build")
	}
	plane := sketch.XYPlane()
	if _, ok := d.curveRefSource("edge", "k", plane); !ok {
		t.Error("edge curve source should build")
	}
	if _, ok := d.curveRefSource("workAxis", string(feature.OriginXAxis), plane); !ok {
		t.Error("workAxis source should build")
	}
	if _, ok := d.curveRefSource("workPlane", string(feature.OriginXZPlane), plane); !ok {
		t.Error("workPlane source should build")
	}
	if _, ok := d.curveRefSource("mystery", "k", plane); ok {
		t.Error("unknown curve kind must not build")
	}
}
