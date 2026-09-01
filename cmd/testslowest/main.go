// SPDX-License-Identifier: GPL-2.0-only

// Command testslowest ranks a `go test -json` stream by test time, and can fail a
// run whose packages exceed the tier budget.
//
//	go test -json ./... | go run ./cmd/testslowest -top 25
//	go test -short -json ./... | go run ./cmd/testslowest -package-budget 90
//	go test -json ./... | go run ./cmd/testslowest -unguarded-budget 60
//
// -unguarded-budget is the gate CI runs. It reads the TIER 2 stream that already
// exists and fails on a slow test that does not guard itself with testing.Short(),
// so catching an unguarded corpus test costs no second run of the suite.
//
// -package-budget is the gate a tier can hold: package WALL time does not move with
// how the tests inside were scheduled. -budget gates a single test instead, and is
// only meaningful on a SEQUENTIAL run (`-p 1 -parallel 1`) — under t.Parallel() every
// test on a busy core reports a stretched time while the package still finishes fast.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"oblikovati.org/test-utilities/testguard"
	"oblikovati.org/test-utilities/testtiming"
)

// errBudgetExceeded is returned when a budget is breached, so the process exits
// non-zero without main having to interpret a boolean.
var errBudgetExceeded = errors.New("time budget exceeded")

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errBudgetExceeded) {
			fmt.Fprintln(os.Stderr, "testslowest:", err)
		}
		os.Exit(1)
	}
}

// run parses args, reads the stream, writes the ranking to out and any budget breach
// to errOut. It returns errBudgetExceeded when a budget did not hold.
func run(args []string, in io.Reader, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("testslowest", flag.ContinueOnError)
	fs.SetOutput(errOut)
	top := fs.Int("top", 25, "how many of the slowest tests to print")
	budget := fs.Float64("budget", 0, "seconds one test may take on a SEQUENTIAL run; 0 disables the gate")
	pkgBudget := fs.Float64("package-budget", 0, "seconds one package may take; 0 disables the gate")
	unguarded := fs.Float64("unguarded-budget", 0,
		"seconds an UNGUARDED test may take in a tier-2 run; 0 disables the gate")
	root := fs.String("module-root", ".", "module root to scan for testing.Short() guards")
	if err := fs.Parse(args); err != nil {
		return err
	}
	runs, pkgs, err := testtiming.ParseAll(in)
	if err != nil {
		return err
	}
	report(out, runs, *top)
	if err := checkBudgets(errOut, runs, pkgs, *budget, *pkgBudget); err != nil {
		return err
	}
	return checkUnguarded(errOut, runs, *root, *unguarded)
}

// checkUnguarded fails when a slow test does not exclude itself from the fast tier.
func checkUnguarded(w io.Writer, runs []testtiming.TestRun, root string, budget float64) error {
	if budget <= 0 {
		return nil
	}
	guards, err := testguard.Scan(root)
	if err != nil {
		return err
	}
	modulePath, err := testguard.ModulePath(root)
	if err != nil {
		return err
	}
	if worst, ok := testtiming.SlowestUnguarded(runs, modulePath, guards); ok {
		fmt.Fprintf(w, "\nslowest UNGUARDED test: %s (budget %.0fs, headroom %.0fs)\n",
			worst, budget, budget-worst.Elapsed)
	}
	over := testtiming.UnguardedOverBudget(runs, modulePath, guards, budget)
	if len(over) == 0 {
		return nil
	}
	fmt.Fprintf(w, "\n%d test(s) over the %.0fs tier-2 limit with no testing.Short() guard —\n"+
		"add one, or they land in the fast tier every developer runs:\n", len(over), budget)
	for _, r := range over {
		fmt.Fprintln(w, " ", r)
	}
	return errBudgetExceeded
}

// checkBudgets applies both gates, reporting every breach before it returns.
func checkBudgets(w io.Writer, runs []testtiming.TestRun, pkgs []testtiming.PackageRun,
	budget, pkgBudget float64,
) error {
	ok := budget <= 0 || withinBudget(w, runs, budget)
	if pkgBudget > 0 && !packagesWithinBudget(w, pkgs, pkgBudget) {
		ok = false
	}
	if !ok {
		return errBudgetExceeded
	}
	return nil
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
