// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestHelpOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "help.registerContext",
		`{"source":"com.x.sim","base":"https://docs.example.org/sim/"}`, nil)
	var p wire.HelpPathResult
	call(t, r, s, "help.path", `{"source":"com.x.sim"}`, &p)
	if p.Base != "https://docs.example.org/sim/" {
		t.Fatalf("path = %+v, want the registered base", p)
	}
	// Display fails cleanly without an opener (the headless session has none).
	if _, err := r.Handle(s, "help.display", []byte(`{"source":"com.x.sim","topic":"x"}`)); err == nil {
		t.Error("display without an opener should fail, not vanish")
	}

	var li wire.LanguageInfoResult
	call(t, r, s, "language.info", "{}", &li)
	if li.Locale == "" {
		t.Fatal("locale should never be empty")
	}
}
