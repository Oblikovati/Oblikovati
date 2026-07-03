// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"errors"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/drawing"
)

// progressSink is a named fake (CLAUDE.md: named fakes over inline stubs) that records every
// progress tick a decoder reports, so a test can assert the callback fired and advanced.
type progressSink struct {
	stages    []string
	dones     []int
	lastTotal int
}

func (p *progressSink) fn(stage string, done, total int) bool {
	p.stages = append(p.stages, stage)
	p.dones = append(p.dones, done)
	p.lastTotal = total
	return false
}

func sampleDXF(t *testing.T) []byte {
	t.Helper()
	in := &drawing.Drawing{Units: drawing.INSCentimetres, Entities: []drawing.Entity{
		&drawing.Line{Start: [3]float64{0, 0, 0}, End: [3]float64{10, 5, 0}},
		&drawing.Circle{Center: [3]float64{3, 4, 0}, Radius: 2.5, Normal: [3]float64{0, 0, 1}},
		&drawing.Point{Position: [3]float64{7, 8, 0}},
	}}
	data, err := Encode(in, R2000)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return data
}

// TestDecodeWithProgressReports checks the DXF decoder threads the shared progress seam (#1647): it
// fires at least one "entities" tick with a monotonically non-decreasing done and a sane total.
func TestDecodeWithProgressReports(t *testing.T) {
	var sink progressSink
	if _, _, err := DecodeWithProgress(sampleDXF(t), exchange.TranslationOptions{Progress: sink.fn}); err != nil {
		t.Fatalf("DecodeWithProgress: %v", err)
	}
	if len(sink.dones) == 0 {
		t.Fatal("the DXF decoder reported no progress; the seam is not wired")
	}
	for i := 1; i < len(sink.dones); i++ {
		if sink.dones[i] < sink.dones[i-1] {
			t.Errorf("progress done went backwards: %v", sink.dones)
		}
	}
	if sink.lastTotal < 0 {
		t.Errorf("progress total = %d, want >= 0", sink.lastTotal)
	}
}

// TestDecodeWithProgressCancels checks a ProgressFunc that cancels on the first tick aborts the DXF
// import promptly with an ErrCancelled-wrapping error.
func TestDecodeWithProgressCancels(t *testing.T) {
	cancel := func(string, int, int) bool { return true }
	_, _, err := DecodeWithProgress(sampleDXF(t), exchange.TranslationOptions{Progress: cancel})
	if !errors.Is(err, exchange.ErrCancelled) {
		t.Fatalf("cancelled DXF import error = %v, want it to wrap exchange.ErrCancelled", err)
	}
}
