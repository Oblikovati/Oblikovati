// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"oblikovati.org/build"
)

// File identity (M03-F07 #608, #159): every document file carries a stable
// GUID minted at creation plus revision stamps maintained on save, so a
// referencing file can detect that its target changed. Retrofitting identity
// onto files in the wild is painful, so it is persisted from the first .obk.

// FileIdentity is the persisted identity block of one document file.
type FileIdentity struct {
	// InternalName is the stable GUID minted at creation; it survives renames
	// and every ordinary save. A save-copy mints a fresh one (two files must
	// never share an identity).
	InternalName string
	// RevisionID stamps the file content as of the last save; every save
	// mints a new one.
	RevisionID string
	// DatabaseRevisionID stamps only the model (recipe) content: it is
	// re-minted on save only when the recipe changed, so property-only saves
	// keep it (#159 — derived parts and assemblies key on it).
	DatabaseRevisionID string
	// SaveCounter counts how many times the file has been saved.
	SaveCounter int
	// VersionCreated / VersionSaved record the software versions that created
	// and last saved the file.
	VersionCreated string
	VersionSaved   string
	// ModelDigest is the recipe digest as of the last save — the comparison
	// basis that decides whether DatabaseRevisionID re-mints.
	ModelDigest string
}

// mintFileGUID returns a fresh RFC 4122 version-4 UUID string.
func mintFileGUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand never fails on supported platforms; failing loud beats
		// silently minting duplicate identities.
		panic(fmt.Sprintf("doc: minting file guid: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// newFileIdentity mints the identity of a freshly created (never saved) file.
func newFileIdentity() FileIdentity {
	return FileIdentity{InternalName: mintFileGUID(), VersionCreated: build.Version}
}

// nextForSave returns the identity as it will be after a save that persists
// modelDigest: counter bumped, content revision re-minted, and the database
// revision re-minted only when the recipe actually changed.
func (id FileIdentity) nextForSave(modelDigest string) FileIdentity {
	next := id
	next.SaveCounter++
	next.RevisionID = mintFileGUID()
	next.VersionSaved = build.Version
	if next.DatabaseRevisionID == "" || modelDigest != id.ModelDigest {
		next.DatabaseRevisionID = mintFileGUID()
		next.ModelDigest = modelDigest
	}
	return next
}

// recipeDigest fingerprints the document's recipe so a save can tell whether
// the model actually changed (DatabaseRevisionID re-mints only then). A
// recipe-less or unmarshalable content digests empty, which conservatively
// re-mints.
func recipeDigest(d *Document) string {
	rc, ok := d.Content().(RecipeContent)
	if !ok {
		return ""
	}
	b, err := rc.MarshalRecipe()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// FileIdentity returns the document file's identity block.
func (d *Document) FileIdentity() FileIdentity { return d.identity }

// SetFileIdentity restores a persisted identity (the load path). A zero
// identity (a pre-identity .obk) keeps the minted one, so old files gain an
// identity on first load and persist it on their next save.
func (d *Document) SetFileIdentity(id FileIdentity) {
	if id.InternalName == "" {
		return
	}
	d.identity = id
}
