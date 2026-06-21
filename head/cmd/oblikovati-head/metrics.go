//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"oblikovati.org/addincat"
	"oblikovati.org/app"
	"oblikovati.org/build"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/usagestats"
)

// usageReportTimeout bounds the telemetry upload so a slow or missing network never strands
// the goroutine (the submit is best-effort — offline and errors are silently dropped).
const usageReportTimeout = 8 * time.Second

// reportUsage submits one anonymous installation snapshot during startup, off the UI thread,
// when the user has not opted out (#1182). It is fire-and-forget: telemetry must never block
// launch or surface an error. The GPU/Vulkan strings come from the live renderer; everything
// else from usagestats. OBLIKOVATI_STATS_ENDPOINT overrides the service URL (for dev/tests).
func reportUsage(session *app.Session, win *native.Window) {
	if !session.TelemetryEnabled() {
		return
	}
	gpu, vulkanVersion := win.GPUInfo()
	snap := assembleSnapshot(gpu, vulkanVersion)
	go func() { _ = submitSnapshot(snap) }() // best-effort; offline/errors intentionally dropped
}

// submitSnapshot POSTs one snapshot to the telemetry service, honoring the
// OBLIKOVATI_STATS_ENDPOINT override (default stats.oblikovati.org). Synchronous and
// error-returning so it is unit-testable against a local server; reportUsage runs it on a
// goroutine and ignores the result.
func submitSnapshot(snap usagestats.Snapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), usageReportTimeout)
	defer cancel()
	sub := usagestats.NewSubmitter(os.Getenv("OBLIKOVATI_STATS_ENDPOINT"), &http.Client{Timeout: usageReportTimeout})
	return sub.Submit(ctx, snap)
}

// assembleSnapshot gathers the machine's anonymous installation snapshot. The GPU and Vulkan
// version are passed in (only the renderer knows them); the rest comes from usagestats. It is
// pure given those inputs, so the assembly is unit-tested without a window.
func assembleSnapshot(gpu, vulkanVersion string) usagestats.Snapshot {
	id, _ := usagestats.MachineUUID() // a generation failure just yields an empty id
	hw := usagestats.GatherHardware()
	return usagestats.Snapshot{
		MachineUUID:   id,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		RAMBytes:      hw.RAMBytes,
		CPU:           hw.CPU,
		CPUCores:      hw.CPUCores,
		StorageBytes:  hw.StorageBytes,
		GPU:           gpu,
		VulkanVersion: vulkanVersion,
		AppVersion:    build.Version,
		AddIns:        installedAddInsForTelemetry(),
	}
}

// installedAddInsForTelemetry lists the user's installed add-ins (id + version) for the
// snapshot, reusing the catalogue installer's scan of the user add-ins directory. Any
// resolution error yields no add-ins rather than failing the snapshot.
func installedAddInsForTelemetry() []usagestats.AddIn {
	dir, err := addincat.UserAddInsDir()
	if err != nil {
		return nil
	}
	installed, err := addincat.NewInstaller(dir, nil).Installed()
	if err != nil {
		return nil
	}
	out := make([]usagestats.AddIn, 0, len(installed))
	for _, a := range installed {
		out = append(out, usagestats.AddIn{ID: a.Name, Version: a.Version})
	}
	return out
}
