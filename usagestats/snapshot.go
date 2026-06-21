// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import "time"

// Snapshot is one machine's installation telemetry. It is the wire shape the
// stats.oblikovati.org service ingests; this struct is duplicated from that service's
// internal/report.Snapshot and pinned to it by a JSON round-trip test (the same arrangement
// the bug-report Payload uses), so the two stay in sync without a shared module.
type Snapshot struct {
	MachineUUID   string    `json:"machineUUID"`
	OS            string    `json:"os"`            // runtime.GOOS
	Arch          string    `json:"arch"`          // runtime.GOARCH
	RAMBytes      int64     `json:"ramBytes"`      // total physical memory, 0 if undetected
	CPU           string    `json:"cpu"`           // CPU model string, "" if undetected
	CPUCores      int       `json:"cpuCores"`      // logical cores (runtime.NumCPU)
	StorageBytes  int64     `json:"storageBytes"`  // capacity of the home volume, 0 if undetected
	GPU           string    `json:"gpu"`           // selected Vulkan physical device name, "" if headless
	VulkanVersion string    `json:"vulkanVersion"` // Vulkan/MoltenVK API version, "" if undetected
	AppVersion    string    `json:"appVersion"`    // Oblikovati build version
	AddIns        []AddIn   `json:"addIns"`        // installed add-ins (id + version)
	ReportedAt    time.Time `json:"reportedAt"`    // client clock at upload; the server overrides on ingest
}

// AddIn is one installed add-in's identity in a snapshot.
type AddIn struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
