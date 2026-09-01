// SPDX-License-Identifier: GPL-2.0-only

// Command testslowest ranks a `go test -json` stream by test time, and can fail a
// run whose packages exceed the tier budget.
//
//	go test -json ./... | go run ./cmd/testslowest -top 25
//	go test -short -json ./... | go run ./cmd/testslowest -package-budget 90
//
// -package-budget is the gate a tier can hold: package WALL time does not move with
// how the tests inside were scheduled. -budget gates a single test instead, and is
// only meaningful on a SEQUENTIAL run (`-p 1 -parallel 1`) — under t.Parallel() every
// test on a busy core reports a stretched time while the package still finishes fast.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"oblikovati.org/test-utilities/testtiming"
)

func main() {
	top := flag.Int("top", 25, "how many of the slowest tests to print")
	budget := flag.Float64("budget", 0, "seconds one test may take on a SEQUENTIAL run; 0 disables the gate")
	pkgBudget := flag.Float64("package-budget", 0, "seconds one package may take; 0 disables the gate")
	flag.Parse()

	ok, err := run(os.Stdin, os.Stdout, os.Stderr, *top, *budget, *pkgBudget)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testslowest:", err)
		os.Exit(1)
	}
	if !ok {
		os.Exit(1)
	}
}

// run reads the stream, writes the ranking to out, writes any budget breach to errOut
// and reports whether every budget held.
func run(in io.Reader, out, errOut io.Writer, top int, budget, pkgBudget float64) (bool, error) {
	runs, pkgs, err := testtiming.ParseAll(in)
	if err != nil {
		return false, err
	}
	report(out, runs, top)
	ok := budget <= 0 || withinBudget(errOut, runs, budget)
	if pkgBudget > 0 && !packagesWithinBudget(errOut, pkgs, pkgBudget) {
		ok = false
	}
	return ok, nil
}

// report prints the suite's total test time and its slowest tests.
func report(w io.Writer, runs []testtiming.TestRun, top int) {
	fmt.Fprintf(w, "%d tests, %.0fs of test time\n", len(runs), testtiming.Total(runs))
	for _, r := range testtiming.Slowest(runs, top) {
		fmt.Fprintln(w, r)
	}
}

// withinBudget reports whether every test finished inside the budget, naming those
// that did not.
func withinBudget(w io.Writer, runs []testtiming.TestRun, budget float64) bool {
	over := testtiming.OverBudget(runs, budget)
	if len(over) == 0 {
		return true
	}
	fmt.Fprintf(w, "\n%d test(s) over the %.0fs tier-1 budget — guard them with testing.Short()\n",
		len(over), budget)
	for _, r := range over {
		fmt.Fprintln(w, " ", r)
	}
	return false
}

// packagesWithinBudget reports whether every package finished inside the budget,
// naming those that did not.
func packagesWithinBudget(w io.Writer, pkgs []testtiming.PackageRun, budget float64) bool {
	over := testtiming.PackagesOverBudget(pkgs, budget)
	if len(over) == 0 {
		return true
	}
	fmt.Fprintf(w, "\n%d package(s) over the %.0fs tier budget — move their corpus tests to tier 2\n",
		len(over), budget)
	for _, p := range over {
		fmt.Fprintln(w, " ", p)
	}
	return false
}
