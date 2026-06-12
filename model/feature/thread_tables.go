// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// The thread table query surface (M09-F01 PBI-101, #325): the wire-facing view
// over the thread catalog (thread_catalog.go) behind threads.tableQuery —
// thread types, each type's nominal sizes, the standard designations per size,
// and the tolerance classes per side. The thread feature, hole tapping (#326),
// and drawings (M14) all resolve designations through [ParseThreadDesignation]
// against these same tables.

// ThreadTypes lists the thread table names — the catalog standards.
func ThreadTypes() []string {
	stds := ThreadStandards()
	out := make([]string, len(stds))
	for i, s := range stds {
		out[i] = string(s)
	}
	return out
}

// threadStandardOf resolves a wire thread-type spelling to its catalog
// standard, erroring with the known names.
func threadStandardOf(threadType string) (ThreadStandard, error) {
	for _, s := range ThreadStandards() {
		if string(s) == threadType {
			return s, nil
		}
	}
	return "", fmt.Errorf("thread: unknown thread type %q (want one of %v)", threadType, ThreadTypes())
}

// ThreadNominalSizes lists a thread type's nominal sizes (catalog order,
// ascending diameter).
func ThreadNominalSizes(threadType string) ([]string, error) {
	std, err := threadStandardOf(threadType)
	if err != nil {
		return nil, err
	}
	sizes := ThreadSizes(std)
	out := make([]string, len(sizes))
	for i, s := range sizes {
		out[i] = s.Name
	}
	return out, nil
}

// ThreadDesignationsOf lists one nominal size's parseable designations (coarse
// pitch first, then fines), built by the same [ThreadDesignation] the thread
// tool uses.
func ThreadDesignationsOf(threadType, nominalSize string) ([]string, error) {
	std, err := threadStandardOf(threadType)
	if err != nil {
		return nil, err
	}
	size, ok := findSize(std, nominalSize)
	if !ok {
		return nil, fmt.Errorf("thread: %s has no size %q", std, nominalSize)
	}
	out := make([]string, len(size.Pitches))
	for i, p := range size.Pitches {
		if out[i], err = ThreadDesignation(std, nominalSize, p); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ThreadClasses lists the tolerance classes for one side of a thread type:
// internal (nut) or external (bolt). The classes are per-system, not
// per-designation, in these tables.
func ThreadClasses(threadType string, internal bool) ([]string, error) {
	std, err := threadStandardOf(threadType)
	if err != nil {
		return nil, err
	}
	switch {
	case StandardSystem(std) == SystemMetric && internal:
		return []string{"6H"}, nil
	case StandardSystem(std) == SystemMetric:
		return []string{"6g"}, nil
	case internal:
		return []string{"2B", "3B"}, nil
	default:
		return []string{"2A", "3A"}, nil
	}
}

// ThreadTypeOfDesignation reports which table a designation resolves against
// (for classifying a parsed spec back to its type name).
func ThreadTypeOfDesignation(designation string) (string, error) {
	spec, err := ParseThreadDesignation(designation)
	if err != nil {
		return "", err
	}
	return spec.ThreadType, nil
}
