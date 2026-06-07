// SPDX-License-Identifier: GPL-2.0-only

package examples_test

import (
	"context"
	"testing"
	"time"

	"oblikovati/api/client"
	"oblikovati/script"
	"oblikovati/script/examples"
	"oblikovati/script/gopherlua"
	"oblikovati/script/runner"
)

// exampleMethods is the wire surface the bundled examples touch. It backs both the typed
// sugar (oblikovati.<group>.<method>) and the unit-test fake, decoupling the unit tests
// from the live router.
var exampleMethods = []string{
	"documents.create",
	"parameters.add", "parameters.set", "parameters.get", "parameters.list",
	"sketch.create", "sketch.addEntity", "sketch.rectangle", "sketch.entities",
	"features.add", "model.physicalProperties",
}

// recordingCaller is a named fake client.Caller: it records every method called and
// returns canned JSON so a read-back in the script can proceed, with no real model. It
// lets a unit test assert the *shape* of an example (which wire calls it issues) in
// isolation.
type recordingCaller struct{ methods []string }

func (c *recordingCaller) Call(method string, _ []byte) ([]byte, error) {
	c.methods = append(c.methods, method)
	return cannedReply(method), nil
}

var _ client.Caller = (*recordingCaller)(nil)

// cannedReply returns a minimal valid response for the methods the examples read back.
func cannedReply(method string) []byte {
	switch method {
	case "sketch.create":
		return []byte(`{"sketchIndex":0}`)
	case "parameters.list":
		return []byte(`{"parameters":[{"name":"Width","value":"76.2 mm"}]}`)
	case "parameters.get":
		return []byte(`{"value":"88.9 mm"}`)
	case "sketch.entities":
		return []byte(`{"entities":[{"kind":"line"}]}`)
	case "model.physicalProperties":
		return []byte(`{"volume":12,"area":24,"centroid":[1,1,1]}`)
	default:
		return []byte(`{}`)
	}
}

// callsFor runs an example against the recording fake and returns the methods it issued.
func callsFor(t *testing.T, name string) []string {
	t.Helper()
	c := &recordingCaller{}
	r := runner.New(gopherlua.New(), c, func() []string { return exampleMethods })
	src, err := examples.Source(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	res, err := r.Run(context.Background(), src, script.Limits{Wall: 5 * time.Second}, nil)
	if err != nil {
		t.Fatalf("run %s: %v", name, err)
	}
	if res.Err != nil {
		t.Fatalf("%s script error: %v", name, res.Err)
	}
	return c.methods
}

func count(methods []string, want string) int {
	n := 0
	for _, m := range methods {
		if m == want {
			n++
		}
	}
	return n
}

// requireMethods asserts each wanted method was called at least minCount times.
func requireMethods(t *testing.T, got []string, wanted map[string]int) {
	t.Helper()
	for m, min := range wanted {
		if c := count(got, m); c < min {
			t.Errorf("expected %q at least %d time(s), got %d (calls: %v)", m, min, c, got)
		}
	}
}

func TestUnitCreateParameters(t *testing.T) {
	requireMethods(t, callsFor(t, "create_parameters.lua"), map[string]int{
		"documents.create": 1, "parameters.add": 2, "parameters.list": 1,
	})
}

func TestUnitSetParameter(t *testing.T) {
	requireMethods(t, callsFor(t, "set_parameter.lua"), map[string]int{
		"documents.create": 1, "parameters.add": 1, "parameters.set": 1, "parameters.get": 1,
	})
}

func TestUnitSketchLines(t *testing.T) {
	requireMethods(t, callsFor(t, "sketch_lines.lua"), map[string]int{
		"documents.create": 1, "sketch.create": 1, "sketch.addEntity": 3,
		"sketch.rectangle": 1, "sketch.entities": 1,
	})
}

func TestUnitExtrudeBlock(t *testing.T) {
	requireMethods(t, callsFor(t, "extrude_block.lua"), map[string]int{
		"documents.create": 1, "sketch.create": 1, "sketch.rectangle": 1,
		"features.add": 1, "model.physicalProperties": 1,
	})
}

func TestUnitRevolveTube(t *testing.T) {
	requireMethods(t, callsFor(t, "revolve_tube.lua"), map[string]int{
		"documents.create": 1, "sketch.create": 1, "sketch.addEntity": 4,
		"features.add": 1, "model.physicalProperties": 1,
	})
}

func TestUnitMassProperties(t *testing.T) {
	requireMethods(t, callsFor(t, "mass_properties.lua"), map[string]int{
		"documents.create": 1, "sketch.create": 1, "sketch.rectangle": 1,
		"features.add": 1, "model.physicalProperties": 1,
	})
}
