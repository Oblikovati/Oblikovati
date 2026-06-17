// SPDX-License-Identifier: GPL-2.0-only

package report

// Payload is the bug report sent to the reporting service. Its JSON shape is the contract
// with reporting.oblikovati.org; by project decision it is duplicated there rather than
// shared through a module, so the field json tags on both sides must stay in lockstep (a
// round-trip test on each side guards against drift). The two PNGs are encoded as base64
// by encoding/json automatically because they are []byte.
type Payload struct {
	Comment        string         `json:"comment"`
	OS             string         `json:"os"`
	Arch           string         `json:"arch"`
	AppVersion     string         `json:"appVersion"`
	AppCommit      string         `json:"appCommit"`
	AppBuildDate   string         `json:"appBuildDate"`
	UserSettings   string         `json:"userSettings"`
	OpenDocuments  []DocumentInfo `json:"openDocuments"`
	TransactionLog []string       `json:"transactionLog"`
	WindowPNG      []byte         `json:"windowPng,omitempty"`
	ViewportPNG    []byte         `json:"viewportPng,omitempty"`
}

// DocumentInfo summarises one open document for the report: its file path, display name,
// kind, dirty flag, and a human-readable dump of its display settings (empty when the
// document uses defaults). It deliberately carries no model geometry — just the context a
// triager needs to reproduce.
type DocumentInfo struct {
	Path            string `json:"path"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Dirty           bool   `json:"dirty"`
	DisplaySettings string `json:"displaySettings,omitempty"`
}
