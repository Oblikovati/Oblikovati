// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"os"
	"time"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
)

// Linked external-file attachments (M03-F08, #609): a document's named
// references to foreign (non-.obk) files — linked spreadsheets, sketch images,
// design-data side files. Linked attachments track the path and its last-known
// modification time for out-of-date detection; embedded attachments carry the
// file's bytes inside the document. These are the persistence backbone for
// spreadsheet-linked parameter tables (M15) and sketch images (M21).

// ExternalFileProbe is the filesystem seam attachments resolve through —
// foreign files live outside the document [Store], so the workspace carries
// its own probe (injected; tests use a named fake).
type ExternalFileProbe interface {
	// StatFile returns the file's modification time, exists=false when absent.
	StatFile(fullFileName string) (modTime time.Time, exists bool)
	// ReadFile returns the file's bytes (the embed path).
	ReadFile(fullFileName string) ([]byte, error)
}

// FileAttachment is one attachment record on a document.
type FileAttachment struct {
	owner             *Document
	name              string
	kind              types.AttachmentKind
	fullFileName      string
	payload           []byte // embedded bytes; nil for linked/generic
	resourceID        string // minted id addressing an embedded payload
	lastKnownFileTime time.Time
	browserVisible    bool
}

// Name returns the unique name other objects bind to.
func (a *FileAttachment) Name() string { return a.name }

// Kind returns whether the foreign file is linked, embedded or generic.
func (a *FileAttachment) Kind() types.AttachmentKind { return a.kind }

// FullFileName returns the as-recorded path (the origin file for embedded).
func (a *FileAttachment) FullFileName() string { return a.fullFileName }

// ResourceID returns the id addressing an embedded payload, "" otherwise.
func (a *FileAttachment) ResourceID() string { return a.resourceID }

// Payload returns a copy of an embedded attachment's bytes, nil otherwise.
func (a *FileAttachment) Payload() []byte {
	out := make([]byte, len(a.payload))
	copy(out, a.payload)
	return out
}

// LastKnownFileTime returns the target's modification time as recorded at
// attach time; the zero time when unknown or embedded.
func (a *FileAttachment) LastKnownFileTime() time.Time { return a.lastKnownFileTime }

// BrowserVisible reports whether the attachment shows in the model browser.
func (a *FileAttachment) BrowserVisible() bool { return a.browserVisible }

// SetBrowserVisible toggles the attachment's browser visibility.
func (a *FileAttachment) SetBrowserVisible(visible bool) { a.browserVisible = visible }

// probe returns the owning workspace's external-file probe, nil for a
// standalone document (status is then unknown).
func (a *FileAttachment) probe() ExternalFileProbe {
	if a.owner == nil || a.owner.graph == nil {
		return nil
	}
	return a.owner.graph.ws.externalFiles
}

// ResolvedFileName returns where the linked path resolves right now, "" while
// missing. Embedded payloads travel with the document and resolve to nothing.
func (a *FileAttachment) ResolvedFileName() string {
	if a.kind == types.AttachmentEmbedded {
		return ""
	}
	p := a.probe()
	if p == nil {
		return ""
	}
	if _, exists := p.StatFile(a.fullFileName); exists {
		return a.fullFileName
	}
	return ""
}

// Status derives the attachment's freshness: embedded payloads are always up
// to date; a linked file that vanished is missing; a linked file modified
// after the recorded time is out of date.
func (a *FileAttachment) Status() types.ReferenceStatus {
	if a.kind == types.AttachmentEmbedded {
		return types.ReferenceUpToDate
	}
	p := a.probe()
	if p == nil {
		return types.ReferenceUnknown
	}
	mod, exists := p.StatFile(a.fullFileName)
	switch {
	case !exists:
		return types.ReferenceMissing
	case a.kind == types.AttachmentLinked && !a.lastKnownFileTime.IsZero() && mod.After(a.lastKnownFileTime):
		return types.ReferenceOutOfDate
	default:
		return types.ReferenceUpToDate
	}
}

// FileAttachments is a document's attachment collection (lazily seeded).
type FileAttachments struct {
	d      *Document
	byName map[string]*FileAttachment
	order  []string
}

// Attachments returns the document's attachment collection.
func (d *Document) Attachments() *FileAttachments {
	if d.attachments == nil {
		d.attachments = &FileAttachments{d: d, byName: map[string]*FileAttachment{}}
	}
	return d.attachments
}

// Count returns the number of attachment records.
func (c *FileAttachments) Count() int { return len(c.order) }

// Names returns the attachment names in insertion order.
func (c *FileAttachments) Names() []string {
	out := make([]string, len(c.order))
	copy(out, c.order)
	return out
}

// ByName returns the named attachment, ok=false when absent. It returns the
// contract view (interface returns keep the collection assignable to
// [contract.FileAttachments]); in-package callers use [FileAttachments.Record].
func (c *FileAttachments) ByName(name string) (contract.FileAttachment, bool) {
	a, ok := c.byName[name]
	if !ok {
		return nil, false
	}
	return a, true
}

// Record returns the named concrete attachment, nil when absent.
func (c *FileAttachments) Record(name string) *FileAttachment { return c.byName[name] }

// Add attaches the file at fullFileName under the unique name. A linked
// attachment records the file's current modification time; an embedded one
// reads and carries its bytes. The document is marked dirty.
func (c *FileAttachments) Add(name string, kind types.AttachmentKind, fullFileName string) (contract.FileAttachment, error) {
	if name == "" || fullFileName == "" {
		return nil, fmt.Errorf("doc: attachment needs a name and a file (got %q, %q)", name, fullFileName)
	}
	if _, taken := c.byName[name]; taken {
		return nil, fmt.Errorf("doc: attachment %q already exists", name)
	}
	a := &FileAttachment{owner: c.d, name: name, kind: kind, fullFileName: fullFileName, browserVisible: true}
	if err := c.captureSource(a); err != nil {
		return nil, err
	}
	c.byName[name] = a
	c.order = append(c.order, name)
	c.d.MarkDirty()
	return a, nil
}

// captureSource records what Add needs from the foreign file: bytes for an
// embedded attachment, the modification time for a linked one.
func (c *FileAttachments) captureSource(a *FileAttachment) error {
	p := a.probe()
	switch {
	case a.kind == types.AttachmentEmbedded:
		if p == nil {
			return fmt.Errorf("doc: cannot embed %q: document has no workspace", a.fullFileName)
		}
		payload, err := p.ReadFile(a.fullFileName)
		if err != nil {
			return fmt.Errorf("doc: embed %q: %w", a.fullFileName, err)
		}
		a.payload, a.resourceID = payload, mintFileGUID()
	case a.kind == types.AttachmentLinked && p != nil:
		if mod, exists := p.StatFile(a.fullFileName); exists {
			a.lastKnownFileTime = mod
		}
	}
	return nil
}

// Remove deletes the named record (and an embedded payload with it),
// reporting whether it existed.
func (c *FileAttachments) Remove(name string) bool {
	if _, ok := c.byName[name]; !ok {
		return false
	}
	delete(c.byName, name)
	for i, n := range c.order {
		if n == name {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.d.MarkDirty()
	return true
}

// All returns the attachment records in insertion order.
func (c *FileAttachments) All() []*FileAttachment {
	out := make([]*FileAttachment, len(c.order))
	for i, name := range c.order {
		out[i] = c.byName[name]
	}
	return out
}

// osFileProbe is the default [ExternalFileProbe]: the real filesystem.
type osFileProbe struct{}

func (osFileProbe) StatFile(fullFileName string) (time.Time, bool) {
	info, err := os.Stat(fullFileName)
	if err != nil || info.IsDir() {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func (osFileProbe) ReadFile(fullFileName string) ([]byte, error) {
	return os.ReadFile(fullFileName)
}

// FileAttachmentRecord is the persistence shape of one attachment — plain data
// the store round-trips (the doc/persistence bridge, like [Resource]).
type FileAttachmentRecord struct {
	Name              string
	Kind              string // wire spelling; readable in the YAML file
	FullFileName      string
	ResourceID        string
	Payload           []byte // embedded bytes; nil for linked/generic
	LastKnownFileTime string // RFC 3339; "" when unknown
	BrowserVisible    bool
}

// AttachmentRecords renders the collection for persistence.
func (d *Document) AttachmentRecords() []FileAttachmentRecord {
	all := d.Attachments().All()
	out := make([]FileAttachmentRecord, len(all))
	for i, a := range all {
		out[i] = FileAttachmentRecord{
			Name: a.name, Kind: a.kind.String(), FullFileName: a.fullFileName,
			ResourceID: a.resourceID, Payload: a.payload,
			LastKnownFileTime: formatFileTime(a.lastKnownFileTime),
			BrowserVisible:    a.browserVisible,
		}
	}
	return out
}

// SetAttachmentRecords restores persisted attachment records (the load path).
func (d *Document) SetAttachmentRecords(records []FileAttachmentRecord) {
	c := &FileAttachments{d: d, byName: map[string]*FileAttachment{}}
	for _, rec := range records {
		kind, ok := types.ParseAttachmentKind(rec.Kind)
		if !ok {
			kind = types.AttachmentGeneric
		}
		c.byName[rec.Name] = &FileAttachment{
			owner: d, name: rec.Name, kind: kind, fullFileName: rec.FullFileName,
			resourceID: rec.ResourceID, payload: rec.Payload,
			lastKnownFileTime: parseFileTime(rec.LastKnownFileTime),
			browserVisible:    rec.BrowserVisible,
		}
		c.order = append(c.order, rec.Name)
	}
	d.attachments = c
}

// formatFileTime / parseFileTime are the RFC 3339 round-trip for the
// last-known file time; the zero time persists as "".
func formatFileTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseFileTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
