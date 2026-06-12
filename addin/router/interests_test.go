// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestInterestRegistryOverWire drives documents.addInterest / listInterests /
// hasInterest / removeInterest end to end (M03-F10, #611).
func TestInterestRegistryOverWire(t *testing.T) {
	r, s := seededSession(t)
	id := uint64(s.ActiveDocument().ID())

	call(t, r, s, "documents.addInterest", fmt.Sprintf(
		`{"document":%d,"interest":{"clientId":"com.x.toolpaths","name":"toolpath-recipes","interestType":68866,"dataVersion":2}}`, id), nil)

	var lst wire.ListDocumentInterestsResult
	call(t, r, s, "documents.listInterests", fmt.Sprintf(`{"document":%d}`, id), &lst)
	if len(lst.Interests) != 1 || lst.Interests[0].InterestType != types.Interested || lst.Interests[0].DataVersion != 2 {
		t.Fatalf("interests = %+v, want the migrating toolpath record", lst.Interests)
	}

	var has wire.HasDocumentInterestResult
	call(t, r, s, "documents.hasInterest", fmt.Sprintf(`{"document":%d,"client":"toolpath-recipes"}`, id), &has)
	if !has.HasInterest {
		t.Error("hasInterest by name must report true")
	}

	if _, err := r.Handle(s, "documents.addInterest",
		[]byte(fmt.Sprintf(`{"document":%d,"interest":{"clientId":"","name":"x"}}`, id))); err == nil {
		t.Error("an interest without a client id must fail over the wire")
	}

	call(t, r, s, "documents.removeInterest",
		fmt.Sprintf(`{"document":%d,"clientId":"com.x.toolpaths","name":"toolpath-recipes"}`, id), nil)
	if _, err := r.Handle(s, "documents.removeInterest",
		[]byte(fmt.Sprintf(`{"document":%d,"clientId":"com.x.toolpaths","name":"toolpath-recipes"}`, id))); err == nil {
		t.Error("removing a missing interest must fail")
	}
}
