// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
)

// File-level reference descriptors (M03-F07, #608): the persisted "as-saved"
// record of one file-to-file reference. The live [RefGraph] drives resolution
// this session; these records make the reference inspectable when the target
// is gone — an unresolved file still reports what it tried to reference, why
// it failed, and offers [FileReference.ReplaceReference] as the repair.

// FileReference is one persisted file-to-file reference record. Its methods
// satisfy both descriptor contracts: the file-side flags view and the
// document-side status/bridge view.
type FileReference struct {
	owner                  *Document
	fullFileName           string // the as-saved target
	relativeFileName       string
	libraryName            string
	locationType           types.FileLocationType
	referencedInternalName string
	saveCounter            int
	replacement            string // a repair target; "" until ReplaceReference
}

// FullFileName returns the reference's as-saved full file name.
func (r *FileReference) FullFileName() string { return r.fullFileName }

// RelativeFileName returns the owner-relative spelling, "" when the target
// lies outside the owner's directory tree.
func (r *FileReference) RelativeFileName() string { return r.relativeFileName }

// LibraryName returns the owning library's name — "" until project libraries
// participate in resolution (the M03-F04 project layer records workspaces
// only today).
func (r *FileReference) LibraryName() string { return r.libraryName }

// LocationType classifies where the as-saved name points.
func (r *FileReference) LocationType() types.FileLocationType { return r.locationType }

// ReferencedFileInternalName returns the target's identity GUID as saved, ""
// when the target was never loaded while the owner saved.
func (r *FileReference) ReferencedFileInternalName() string { return r.referencedInternalName }

// FileSaveCounter returns the target's save counter as saved, the comparison
// basis for out-of-date detection.
func (r *FileReference) FileSaveCounter() int { return r.saveCounter }

// effectiveTarget is the name resolution acts on: the repair target once
// re-pointed, the as-saved name otherwise.
func (r *FileReference) effectiveTarget() string {
	if r.replacement != "" {
		return r.replacement
	}
	return r.fullFileName
}

// liveTarget returns the open document the reference resolves to, if any.
func (r *FileReference) liveTarget() (*Document, bool) {
	if r.owner == nil || r.owner.graph == nil {
		return nil, false
	}
	d, ok := r.owner.graph.ws.byName[r.effectiveTarget()]
	return d, ok && d.open
}

// ResolvedFileName returns where the reference resolves right now: the open
// target's name, an existing file on disk, or "" while missing.
func (r *FileReference) ResolvedFileName() string {
	if _, ok := r.liveTarget(); ok {
		return r.effectiveTarget()
	}
	if r.owner != nil && r.owner.graph != nil {
		if store := r.owner.graph.ws.store; store != nil && store.Exists(r.effectiveTarget()) {
			return r.effectiveTarget()
		}
	}
	return ""
}

// ReferenceMissing reports that the target cannot be found anywhere.
func (r *FileReference) ReferenceMissing() bool { return r.ResolvedFileName() == "" }

// ReferenceReplaced reports the reference was re-pointed and the owner has not
// been saved since (saving makes the repair the as-saved truth).
func (r *FileReference) ReferenceReplaced() bool { return r.replacement != "" }

// ReferenceLocationDifferent reports the reference resolves somewhere other
// than the as-saved location.
func (r *FileReference) ReferenceLocationDifferent() bool {
	resolved := r.ResolvedFileName()
	return resolved != "" && resolved != r.fullFileName
}

// ReferenceInternalNameDifferent reports the resolved file carries a different
// identity GUID than the one saved against (a swapped-in file).
func (r *FileReference) ReferenceInternalNameDifferent() bool {
	target, ok := r.liveTarget()
	if !ok || r.referencedInternalName == "" {
		return false
	}
	return target.identity.InternalName != r.referencedInternalName
}

// referenceOutOfDate reports the target has been saved since the owner
// recorded it (only assessable while the target is open).
func (r *FileReference) referenceOutOfDate() bool {
	target, ok := r.liveTarget()
	if !ok || r.saveCounter == 0 {
		return false
	}
	return target.identity.SaveCounter > r.saveCounter
}

// Status derives the single status vocabulary from the resolution state.
func (r *FileReference) Status() types.ReferenceStatus {
	switch {
	case r.ReferenceReplaced():
		return types.ReferenceReplaced
	case r.ReferenceMissing():
		return types.ReferenceMissing
	case r.ReferenceInternalNameDifferent() || r.referenceOutOfDate():
		return types.ReferenceOutOfDate
	default:
		return types.ReferenceUpToDate
	}
}

// DisplayName returns the reference's human-readable name (document-side view).
func (r *FileReference) DisplayName() string { return derivedDisplayName(r.fullFileName) }

// DocumentFound reports whether a document resolves for this reference.
func (r *FileReference) DocumentFound() bool { return !r.ReferenceMissing() }

// DifferentDocument reports the resolved document is not the one saved
// against — a repair or a swapped-in file.
func (r *FileReference) DifferentDocument() bool {
	return r.ReferenceReplaced() || r.ReferenceInternalNameDifferent() || r.ReferenceLocationDifferent()
}

// ReplaceReference re-points this record at fullFileName (the broken-reference
// repair). The replacement must exist; the matching live graph edge is
// re-pointed too, so resolution this session follows the repair, and the next
// save persists it as the as-saved truth.
func (r *FileReference) ReplaceReference(fullFileName string) error {
	if r.owner == nil || r.owner.graph == nil {
		return fmt.Errorf("doc: reference %q has no workspace; cannot replace", r.fullFileName)
	}
	ws := r.owner.graph.ws
	if ws.store == nil || !ws.store.Exists(fullFileName) {
		return fmt.Errorf("doc: replacement %q does not exist", fullFileName)
	}
	old := r.effectiveTarget()
	r.replacement = fullFileName
	ws.graph.repointReference(r.owner, old, fullFileName)
	return nil
}

// FileReferences returns the document's persisted file-to-file reference
// records, in order.
func (d *Document) FileReferences() []*FileReference {
	out := make([]*FileReference, len(d.fileReferences))
	copy(out, d.fileReferences)
	return out
}

// FileReferenceRecord is the persistence shape of one record — plain data the
// store round-trips (the doc/persistence bridge, like [Resource]).
type FileReferenceRecord struct {
	FullFileName           string
	RelativeFileName       string
	LibraryName            string
	LocationType           string // wire spelling; readable in the YAML file
	ReferencedInternalName string
	SaveCounter            int
}

// FileReferenceRecords renders the as-saved snapshot for persistence.
func (d *Document) FileReferenceRecords() []FileReferenceRecord {
	out := make([]FileReferenceRecord, len(d.fileReferences))
	for i, r := range d.fileReferences {
		out[i] = FileReferenceRecord{
			FullFileName: r.fullFileName, RelativeFileName: r.relativeFileName,
			LibraryName: r.libraryName, LocationType: r.locationType.String(),
			ReferencedInternalName: r.referencedInternalName, SaveCounter: r.saveCounter,
		}
	}
	return out
}

// SetFileReferenceRecords restores persisted records (the load path).
func (d *Document) SetFileReferenceRecords(records []FileReferenceRecord) {
	d.fileReferences = make([]*FileReference, len(records))
	for i, rec := range records {
		loc, ok := types.ParseFileLocationType(rec.LocationType)
		if !ok {
			loc = types.LocationUnknown
		}
		d.fileReferences[i] = &FileReference{
			owner: d, fullFileName: rec.FullFileName, relativeFileName: rec.RelativeFileName,
			libraryName: rec.LibraryName, locationType: loc,
			referencedInternalName: rec.ReferencedInternalName, saveCounter: rec.SaveCounter,
		}
	}
}

// snapshotFileReferences rebuilds the records from the live reference graph at
// save time: every graph edge becomes a record carrying where the target is
// and what identity it had; a pending repair becomes the as-saved truth.
// Identity details of an unloaded target survive from the prior record.
func (d *Document) snapshotFileReferences() {
	if d.graph == nil {
		return
	}
	descs := d.graph.descriptors(d.fullDocumentName)
	if len(descs) == 0 && len(d.fileReferences) > 0 {
		// The graph is rebuilt lazily after load (a reopened document's edges
		// only reappear as features re-touch their sources), so an empty graph
		// must not wipe the persisted records — they may be the only memory of
		// what this file references.
		return
	}
	prior := d.fileReferences
	d.fileReferences = make([]*FileReference, len(descs))
	for i, desc := range descs {
		d.fileReferences[i] = d.snapshotOneReference(desc.fullDocumentName, prior)
	}
}

// snapshotOneReference builds the as-saved record for one graph edge.
func (d *Document) snapshotOneReference(target string, prior []*FileReference) *FileReference {
	rec := &FileReference{owner: d, fullFileName: target}
	rec.relativeFileName, rec.locationType = relateToOwner(d.fullDocumentName, target)
	if t, ok := d.graph.ws.byName[target]; ok && t.open {
		rec.referencedInternalName = t.identity.InternalName
		rec.saveCounter = t.identity.SaveCounter
		return rec
	}
	for _, p := range prior {
		if p.effectiveTarget() == target {
			rec.referencedInternalName, rec.saveCounter = p.referencedInternalName, p.saveCounter
		}
	}
	return rec
}

// relateToOwner expresses target relative to the owner file's directory. A
// sibling-tree target is an ownerDirectory reference (portable across moves of
// the whole tree); anything else is recorded absolute with unknown location
// until project search roots join resolution. The relative spelling is
// normalized to forward slashes — a .obk saved on one OS must resolve on
// another (ADR-0020 portability).
func relateToOwner(ownerName, target string) (string, types.FileLocationType) {
	rel, err := filepath.Rel(filepath.Dir(ownerName), target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", types.LocationUnknown
	}
	return filepath.ToSlash(rel), types.LocationOwnerDirectory
}
