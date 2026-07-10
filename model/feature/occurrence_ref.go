// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"strings"

	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// Occurrence-qualified work-feature references name a datum that lives inside a sub-component
// occurrence, reached from an assembly context — the wire counterpart of Inventor's
// WorkPlane/Axis/PointProxy (NativeObject + ContainingOccurrence, #1857). The encoding is
// "occ/<b64path>/<native>" where b64path is the URL-safe base64 of the NUL-joined occurrence path
// (instance names, top-down) and native is the component-local datum ref ("plane/3", "axis/1",
// "point/0"). URL-safe base64 never contains '/', so the native ref (which does) is unambiguous.

const occurrenceRefPrefix = "occ/"

// EncodeOccurrenceRef builds the occurrence-qualified reference for a component-local datum named
// through the given occurrence path.
func EncodeOccurrenceRef(path []string, native WorkRef) WorkRef {
	b64 := base64.RawURLEncoding.EncodeToString([]byte(strings.Join(path, "\x00")))
	return WorkRef(occurrenceRefPrefix + b64 + "/" + string(native))
}

// ParseOccurrenceRef decodes an occurrence-qualified reference into its occurrence path (instance
// names) and the component-local native datum ref. ok is false for any other reference.
func ParseOccurrenceRef(ref WorkRef) (path []string, native WorkRef, ok bool) {
	s := string(ref)
	if !strings.HasPrefix(s, occurrenceRefPrefix) {
		return nil, "", false
	}
	rest := s[len(occurrenceRefPrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return nil, "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(rest[:slash])
	if err != nil {
		return nil, "", false
	}
	nativeRef := rest[slash+1:]
	if nativeRef == "" {
		return nil, "", false
	}
	names := strings.Split(string(raw), "\x00")
	return names, WorkRef(nativeRef), true
}

// isOccurrenceRef reports whether ref is an occurrence-qualified reference.
func isOccurrenceRef(ref WorkRef) bool { return strings.HasPrefix(string(ref), occurrenceRefPrefix) }

// ExternalDatumResolver resolves occurrence-qualified references (which name a datum in another
// component, reached through an occurrence) to their geometry in the resolving context's space. An
// assembly injects one into its WorkGeometry so occurrence-qualified refs are accepted as datum
// inputs to assembly sketch/feature tools (#1857). Each method returns ok=false when the reference
// does not resolve (a lost occurrence, a deleted datum), so the consuming feature goes unhealthy.
type ExternalDatumResolver interface {
	OccurrencePlane(ref WorkRef) (sketch.Plane, bool)
	OccurrenceAxisLine(ref WorkRef) (origin math.Point3, dir math.UnitVector3, ok bool)
	OccurrencePoint(ref WorkRef) (math.Point3, bool)
}

// SetExternalDatumResolver installs the resolver for occurrence-qualified references. Nil (the
// default) means occurrence refs do not resolve — a plain part has no occurrences.
func (g *WorkGeometry) SetExternalDatumResolver(r ExternalDatumResolver) { g.external = r }

// newOccurrenceAxis wraps a resolved occurrence axis line as a transient grounded WorkAxis (not part
// of any collection) so it can be returned from axis() like a local datum axis.
func newOccurrenceAxis(ref WorkRef, origin math.Point3, dir math.UnitVector3) *WorkAxis {
	return &WorkAxis{
		key:      ref,
		name:     "Occurrence Axis",
		def:      fixedAxisDef{origin: origin, dir: dir},
		origin:   origin,
		dir:      dir,
		grounded: true,
		health:   health.Healthy,
	}
}
