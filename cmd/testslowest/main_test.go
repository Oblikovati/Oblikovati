// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// stream is a two-package `go test -json` transcript: one fast package and one that
// blows every budget.
const stream = `{"Action":"pass","Package":"m/fast","Test":"TestQuick","Elapsed":0.01}
{"Action":"pass","Package":"m/fast","Elapsed":0.5}
{"Action":"pass","Package":"m/slow","Test":"TestCorpus","Elapsed":120}
{"Action":"pass","Package":"m/slow","Elapsed":121}
`

func runStream(t *testing.T, args ...string) (bool, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	err := run(append([]string{"-top", "5"}, args...), strings.NewReader(stream), &out, &errOut)
	if err != nil && !errors.Is(err, errBudgetExceeded) {
		t.Fatalf("run: %v", err)
	}
	return err == nil, out.String(), errOut.String()
}

func TestRunRanksTheSlowestTestFirst(t *testing.T) {
	_, out, _ := runStream(t)
	if !strings.Contains(out, "2 tests, 120s of test time") {
		t.Errorf("report header missing from %q", out)
	}
	if i, j := strings.Index(out, "TestCorpus"), strings.Index(out, "TestQuick"); i < 0 || j < 0 || i > j {
		t.Errorf("TestCorpus must be ranked above TestQuick; got %q", out)
	}
}

func TestRunPassesWhenNoBudgetIsSet(t *testing.T) {
	if ok, _, errOut := runStream(t); !ok || errOut != "" {
		t.Errorf("run() = %v, stderr %q, want a clean pass", ok, errOut)
	}
}

func TestRunFailsAndNamesTheTestOverTheTestBudget(t *testing.T) {
	ok, _, errOut := runStream(t, "-budget", "5")
	if ok {
		t.Error("run() passed, want a failure for the 120s test")
	}
	if !strings.Contains(errOut, "TestCorpus") || !strings.Contains(errOut, "testing.Short()") {
		t.Errorf("stderr must name the test and the remedy; got %q", errOut)
	}
}

func TestRunFailsAndNamesThePackageOverThePackageBudget(t *testing.T) {
	ok, _, errOut := runStream(t, "-package-budget", "90")
	if ok {
		t.Error("run() passed, want a failure for the 121s package")
	}
	if !strings.Contains(errOut, "m/slow") || strings.Contains(errOut, "m/fast") {
		t.Errorf("stderr must name only the slow package; got %q", errOut)
	}
}

func TestRunReportsAMalformedStream(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(nil, errReader{}, &out, &errOut); err == nil {
		t.Fatal("run() succeeded, want the read error")
	}
}

func TestRunRejectsAnUnknownFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"-nonsense"}, strings.NewReader(stream), &out, &errOut); err == nil {
		t.Fatal("run() succeeded, want a flag-parse error")
	}
}

// errReader is a reader that always fails, standing in for a broken pipe.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errFailedRead }

var errFailedRead = errStr("pipe closed")

type errStr string

func (e errStr) Error() string { return string(e) }
