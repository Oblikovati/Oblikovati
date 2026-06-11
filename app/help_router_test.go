// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestHelpTopicResolvesThroughOpener(t *testing.T) {
	s := NewSession()
	opener := &FakeURLOpener{}
	s.SetURLOpener(opener)
	if err := s.RegisterHelpContext("com.x.sim", "https://docs.example.org/sim/"); err != nil {
		t.Fatalf("RegisterHelpContext: %v", err)
	}

	if err := s.DisplayHelpTopic("com.x.sim", "mesh-setup"); err != nil {
		t.Fatalf("DisplayHelpTopic: %v", err)
	}
	if len(opener.opened) != 1 || opener.opened[0] != "https://docs.example.org/sim/mesh-setup" {
		t.Fatalf("opened = %v, want the joined topic URL", opener.opened)
	}

	// "" routes to the host documentation.
	if err := s.DisplayHelpTopic("", ""); err != nil {
		t.Fatalf("DisplayHelpTopic(host): %v", err)
	}
	if base, _ := s.HelpPath(""); opener.opened[1] != base {
		t.Errorf("host help opened %q, want %q", opener.opened[1], base)
	}

	if err := s.DisplayHelpTopic("ghost", "x"); err == nil {
		t.Error("an unregistered source should fail")
	}
	if err := s.RegisterHelpContext("bad", "not-a-base"); err == nil {
		t.Error("a relative non-URL base should be rejected")
	}
}

func TestHelpInterceptorConsumesFirst(t *testing.T) {
	s := NewSession()
	opener := &FakeURLOpener{}
	s.SetURLOpener(opener)
	var seen []string
	s.SetHelpInterceptor(func(source, topic string) bool {
		seen = append(seen, source+"/"+topic)
		return source == "mine"
	})
	_ = s.RegisterHelpContext("mine", "https://example.org/")

	if err := s.DisplayHelpTopic("mine", "a"); err != nil {
		t.Fatalf("DisplayHelpTopic: %v", err)
	}
	if len(opener.opened) != 0 {
		t.Error("the interceptor consumed the request; nothing should open")
	}
	if err := s.DisplayHelpTopic("", "b"); err != nil {
		t.Fatalf("DisplayHelpTopic(passthrough): %v", err)
	}
	if len(opener.opened) != 1 || len(seen) != 2 {
		t.Errorf("opened=%v seen=%v, want one open and two interceptor calls", opener.opened, seen)
	}
}

func TestLocaleNormalizes(t *testing.T) {
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	if got := Locale(); got != "pt-BR" {
		t.Errorf("Locale = %q, want pt-BR", got)
	}
	t.Setenv("LC_ALL", "C")
	t.Setenv("LANG", "")
	if got := Locale(); got != "en-US" {
		t.Errorf("Locale fallback = %q, want en-US", got)
	}
}
