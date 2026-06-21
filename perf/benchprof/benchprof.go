// SPDX-License-Identifier: GPL-2.0-only

// Package benchprof is a thin, env-gated profiling harness shared by the large-assembly
// benchmark tooling (M34-B3): the oblikovati-cli generator and the head perf driver. A
// run captures a CPU profile, a heap profile, and a memory summary (allocations, GC
// count, total GC pause, and peak RSS) so a scenario's cost can be attributed without
// each tool re-deriving pprof wiring.
//
// Profiling is OFF unless OBK_PPROF_DIR names a directory; the memory summary is always
// computed (it is cheap), so callers can log RAM/GC even without writing profiles.
// Usage:
//
//	run, _ := benchprof.Start("orbit")
//	... work ...
//	summary, _ := run.Stop()
//	fmt.Println(summary)
package benchprof

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
)

// envDir is the environment variable that enables profile writing and names the output
// directory.
const envDir = "OBK_PPROF_DIR"

// Run is one profiling scope. The zero value is not usable; obtain one from Start. When
// profiling is disabled (OBK_PPROF_DIR unset) a Run still tracks memory so Stop returns
// a valid summary, it just writes no files.
type Run struct {
	label   string
	dir     string
	cpuFile *os.File
	start   runtime.MemStats
}

// MemSummary captures the memory cost of a run: heap live at Stop, bytes/objects
// allocated during the run, GC cycles and total pause incurred, and the process peak
// resident set (0 where the platform does not expose it).
type MemSummary struct {
	Label           string
	HeapAllocBytes  uint64
	TotalAllocBytes uint64
	Mallocs         uint64
	NumGC           uint32
	PauseTotalNs    uint64
	PeakRSSBytes    uint64
}

// Enabled reports whether profile files will be written (OBK_PPROF_DIR is set).
func Enabled() bool { return os.Getenv(envDir) != "" }

// Start begins a profiling scope labelled label. If OBK_PPROF_DIR is set it creates the
// directory and starts CPU profiling into <dir>/<label>.cpu.pprof; otherwise it only
// snapshots memory. label must be non-empty and is used in output filenames.
func Start(label string) (*Run, error) {
	if label == "" {
		return nil, fmt.Errorf("benchprof: label must be non-empty")
	}
	r := &Run{label: label, dir: os.Getenv(envDir)}
	runtime.ReadMemStats(&r.start)
	if r.dir == "" {
		return r, nil
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return nil, fmt.Errorf("benchprof: mkdir %q: %w", r.dir, err)
	}
	f, err := os.Create(filepath.Join(r.dir, label+".cpu.pprof"))
	if err != nil {
		return nil, fmt.Errorf("benchprof: create cpu profile: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("benchprof: start cpu profile: %w", err)
	}
	r.cpuFile = f
	return r, nil
}

// Stop ends the scope, returning the memory summary. When profiling is enabled it stops
// the CPU profile and writes the heap profile and a text memory summary alongside it.
func (r *Run) Stop() (MemSummary, error) {
	summary := r.summary()
	if r.dir == "" {
		return summary, nil
	}
	pprof.StopCPUProfile()
	if err := r.cpuFile.Close(); err != nil {
		return summary, fmt.Errorf("benchprof: close cpu profile: %w", err)
	}
	if err := r.writeHeapProfile(); err != nil {
		return summary, err
	}
	path := filepath.Join(r.dir, r.label+".mem.txt")
	if err := os.WriteFile(path, []byte(summary.String()+"\n"), 0o644); err != nil {
		return summary, fmt.Errorf("benchprof: write mem summary: %w", err)
	}
	return summary, nil
}

// summary reads current memory stats and diffs them against the run's start snapshot.
func (r *Run) summary() MemSummary {
	var end runtime.MemStats
	runtime.ReadMemStats(&end)
	return MemSummary{
		Label:           r.label,
		HeapAllocBytes:  end.HeapAlloc,
		TotalAllocBytes: end.TotalAlloc - r.start.TotalAlloc,
		Mallocs:         end.Mallocs - r.start.Mallocs,
		NumGC:           end.NumGC - r.start.NumGC,
		PauseTotalNs:    end.PauseTotalNs - r.start.PauseTotalNs,
		PeakRSSBytes:    readPeakRSS(),
	}
}

// writeHeapProfile forces a GC so the heap profile reflects live memory, then writes it.
//
// Security note: pprof is a development/benchmark profiling feature, not a runtime
// debug endpoint. It is reachable only when the operator explicitly sets OBK_PPROF_DIR
// (see Start), is never enabled by default in any shipped path, and exposes no network
// surface (it writes local files) — so the "debug feature in production" hotspot here is
// intended and gated.
func (r *Run) writeHeapProfile() error {
	f, err := os.Create(filepath.Join(r.dir, r.label+".heap.pprof"))
	if err != nil {
		return fmt.Errorf("benchprof: create heap profile: %w", err)
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("benchprof: write heap profile: %w", err)
	}
	return nil
}

// String renders the summary as one human-readable line (MB and ms units).
func (m MemSummary) String() string {
	return fmt.Sprintf("[%s] heap=%s alloc=%s mallocs=%d gc=%d pause=%dms peakRSS=%s",
		m.Label, mib(m.HeapAllocBytes), mib(m.TotalAllocBytes), m.Mallocs, m.NumGC,
		m.PauseTotalNs/1e6, mib(m.PeakRSSBytes))
}

// mib formats a byte count as mebibytes.
func mib(b uint64) string { return fmt.Sprintf("%.1fMB", float64(b)/(1<<20)) }
