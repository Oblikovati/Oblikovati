// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"

	"oblikovati.org/api/types"
)

// The add-in data registry on documents (M03-F10, #611): an interest declares
// "client X has data in / depends on this document", carries a client-managed
// data version for migration, and is readable without loading the add-in —
// the discovery layer over the attribute-set payload layer. Interests persist
// with the document so the host can warn when a document opens and the
// responsible add-in is absent.

// InterestRecordView is one registered interest, read through the contract's
// scalar accessors.
type InterestRecordView struct {
	rec types.DocumentInterestRecord
}

// ClientID returns the owning client's id.
func (v InterestRecordView) ClientID() string { return v.rec.ClientID }

// Name returns the interest's name; (ClientID, Name) is its identity.
func (v InterestRecordView) Name() string { return v.rec.Name }

// InterestType returns the interest's strength.
func (v InterestRecordView) InterestType() types.DocumentInterestType { return v.rec.InterestType }

// DataVersion returns the client-managed migration version, 0 non-migrating.
func (v InterestRecordView) DataVersion() int { return v.rec.DataVersion }

// ClientData returns the uninterpreted client payload.
func (v InterestRecordView) ClientData() string { return v.rec.ClientData }

// DocumentInterests is a document's interest registry (lazily seeded).
type DocumentInterests struct {
	d       *Document
	records []types.DocumentInterestRecord
}

// Interests returns the document's interest registry.
func (d *Document) Interests() *DocumentInterests {
	if d.interests == nil {
		d.interests = &DocumentInterests{d: d}
	}
	return d.interests
}

// Count returns the number of interest records.
func (c *DocumentInterests) Count() int { return len(c.records) }

// Records returns the interest records in insertion order.
func (c *DocumentInterests) Records() []types.DocumentInterestRecord {
	out := make([]types.DocumentInterestRecord, len(c.records))
	copy(out, c.records)
	return out
}

// At returns the record at index as the contract's scalar view.
func (c *DocumentInterests) At(index int) (InterestRecordView, error) {
	if index < 0 || index >= len(c.records) {
		return InterestRecordView{}, fmt.Errorf("doc: interest index %d out of range [0,%d)", index, len(c.records))
	}
	return InterestRecordView{rec: c.records[index]}, nil
}

// HasInterest reports whether any record's ClientID or Name matches
// clientIDOrName — the cheap "does anyone hold data here?" probe.
func (c *DocumentInterests) HasInterest(clientIDOrName string) bool {
	for _, rec := range c.records {
		if rec.ClientID == clientIDOrName || rec.Name == clientIDOrName {
			return true
		}
	}
	return false
}

// Add registers the record, updating an existing (ClientID, Name) entry in
// place. An unset InterestType registers as Interested. The document is
// marked dirty.
func (c *DocumentInterests) Add(record types.DocumentInterestRecord) error {
	if record.ClientID == "" || record.Name == "" {
		return fmt.Errorf("doc: interest needs a client id and a name (got %q, %q)", record.ClientID, record.Name)
	}
	if record.InterestType == 0 {
		record.InterestType = types.Interested
	}
	defer c.d.MarkDirty()
	for i, rec := range c.records {
		if rec.ClientID == record.ClientID && rec.Name == record.Name {
			c.records[i] = record
			return nil
		}
	}
	c.records = append(c.records, record)
	return nil
}

// Remove deletes the (clientID, name) record, reporting whether it existed.
func (c *DocumentInterests) Remove(clientID, name string) bool {
	for i, rec := range c.records {
		if rec.ClientID == clientID && rec.Name == name {
			c.records = append(c.records[:i], c.records[i+1:]...)
			c.d.MarkDirty()
			return true
		}
	}
	return false
}

// InterestRecords renders the registry for persistence.
func (d *Document) InterestRecords() []types.DocumentInterestRecord {
	return d.Interests().Records()
}

// SetInterestRecords restores persisted interest records (the load path).
func (d *Document) SetInterestRecords(records []types.DocumentInterestRecord) {
	d.interests = &DocumentInterests{d: d, records: records}
}
