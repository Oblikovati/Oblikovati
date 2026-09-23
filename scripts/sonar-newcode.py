#!/usr/bin/env python3
"""Coverage and duplication on NEW code, for a local SonarQube that cannot compute them itself.

WHY this exists rather than asking the server. SonarCloud decides "new code" by diffing against the
PR's target branch. Locally that is `sonar.newCode.referenceBranch`, which needs Developer Edition;
Community Build refuses it. Community's only alternative, a PREVIOUS_VERSION period, attributes new
lines by SCM blame DATE relative to the baseline analysis — so a baseline scanned a minute ago marks
almost nothing new (measured: 86 lines against a true 5704). The comparison has to come from git.

So the split is explicit: the server measures everything it can — rules, issues, whole-project
coverage and duplication — and this computes the three gates that need a branch comparison, from the
server's OWN issue and duplication data and the SAME exclusions sonar-project.properties declares.
Nothing here re-implements a Sonar rule; it intersects Sonar's answers with `git diff`.

    scripts/sonar-newcode.py <base-ref> <sonar-host> <user:password> <project-key> [coverage.out]
"""
from __future__ import annotations

import base64
import collections
import json
import re
import subprocess
import sys
import urllib.error
import urllib.request

COVERAGE_GATE = 80.0  # CLAUDE.md: coverage >80%
DUPLICATION_GATE = 3.0  # CLAUDE.md: duplication <3%


def properties(path: str = "sonar-project.properties") -> dict[str, str]:
    """The exclusions must come from the file CI reads, never from a copy of them here."""
    out: dict[str, str] = {}
    with open(path) as fh:
        for line in fh:
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, value = line.split("=", 1)
                out[key.strip()] = value.strip()
    return out


def globs(spec: str) -> list[re.Pattern[str]]:
    """Sonar's `**/x` glob syntax, as regexes."""
    pats = []
    for raw in filter(None, (g.strip() for g in spec.split(","))):
        rx = re.escape(raw).replace(r"\*\*/", "(?:.*/)?").replace(r"\*\*", ".*").replace(r"\*", "[^/]*")
        pats.append(re.compile(f"^{rx}$"))
    return pats


def excluded(path: str, pats: list[re.Pattern[str]]) -> bool:
    return any(p.match(path) for p in pats)


def git(*args: str) -> str:
    return subprocess.run(["git", *args], capture_output=True, text=True, check=True).stdout


def added_lines(base: str) -> dict[str, set[int]]:
    """path -> the line numbers this branch adds, counting the WORKING TREE, not only HEAD.

    Two deliberate choices. The comparison is against the MERGE BASE, so a commit that landed on
    `base` after the fork does not read as this branch deleting it. And the diff has no second
    endpoint, so uncommitted and untracked files count as new: a pre-commit gate that measured HEAD
    would pass the very change being committed, which is the one thing it exists to judge.
    `-U0` keeps the added lines themselves and no context.
    """
    fork = git("merge-base", base, "HEAD").strip() or base
    added: dict[str, set[int]] = collections.defaultdict(set)
    path, line = None, 0
    for raw in git("diff", "-U0", fork).splitlines():
        if raw.startswith("+++ b/"):
            path = raw[6:]
        elif raw.startswith("@@"):
            m = re.search(r"\+(\d+)", raw)
            line = int(m.group(1)) if m else 0
        elif raw.startswith("+") and not raw.startswith("+++") and path:
            added[path].add(line)
            line += 1
    for untracked in git("ls-files", "--others", "--exclude-standard").splitlines():
        # A file git has never seen is new in its entirety. Sonar analysed it (the scanner reads the
        # working tree), so its issues are the branch's to answer for.
        with open(untracked, "rb") as fh:
            added[untracked] = set(range(1, sum(1 for _ in fh) + 1))
    return added


def coverage_by_line(profile: str, module: str) -> dict[str, dict[int, bool]]:
    """A line ran if ANY profile block covering it ran — the same rule `go tool cover` applies."""
    ran: dict[str, dict[int, bool]] = collections.defaultdict(dict)
    with open(profile) as fh:
        for raw in fh:
            if raw.startswith("mode:") or not raw.strip():
                continue
            loc, _stmts, count = raw.rsplit(" ", 2)
            file_part, span = loc.split(":", 1)
            start, end = span.split(",")
            path = file_part[len(module):] if file_part.startswith(module) else file_part
            hit = int(count) > 0
            for ln in range(int(start.split(".")[0]), int(end.split(".")[0]) + 1):
                ran[path][ln] = ran[path].get(ln, False) or hit
    return ran


def api(host: str, auth: str, path: str) -> dict:
    req = urllib.request.Request(f"{host}/api/{path}")
    req.add_header("Authorization", "Basic " + base64.b64encode(auth.encode()).decode())
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.load(resp)


def duplicated_lines(host: str, auth: str, key: str, path: str) -> set[int]:
    """Sonar's OWN duplication blocks for one file, as line numbers. Empty when it has none."""
    try:
        data = api(host, auth, f"duplications/show?key={key}:{path}")
    except urllib.error.URLError:  # HTTPError derives from it
        return set()
    lines: set[int] = set()
    for dup in data.get("duplications", []):
        for block in dup.get("blocks", []):
            # `_ref` "1" is the file asked about; other refs are its clone partners elsewhere.
            if block.get("_ref") == "1":
                lines.update(range(block["from"], block["from"] + block["size"]))
    return lines


def issues_on(host: str, auth: str, key: str, added: dict[str, set[int]]) -> list[tuple[str, str, int, str]]:
    """Sonar's OWN open issues that sit on a line this branch added.

    Whole-project issue counts cannot answer "did MY change add a smell?" — this project carries 542
    open smells and five of them were the branch's. Asking per changed file would be 300 requests, so
    the whole open set is paged once and intersected here, the same way coverage and duplication are.
    """
    found: list[tuple[str, str, int, str]] = []
    page, page_size = 1, 500
    while True:
        data = api(host, auth, f"issues/search?componentKeys={key}&resolved=false"
                               f"&ps={page_size}&p={page}")
        batch = data.get("issues", [])
        for issue in batch:
            component = issue.get("component", "")
            path = component.split(":", 1)[1] if ":" in component else component
            line = issue.get("line")
            if line is not None and line in added.get(path, ()):
                found.append((issue.get("rule", "?"), path, line, issue.get("message", "")))
        if len(batch) < page_size or page * page_size >= data.get("total", 0):
            return sorted(found)
        page += 1


def report_coverage(go_added: dict[str, set[int]], ran: dict[str, dict[int, bool]],
                    cov_excl: list[re.Pattern[str]]) -> bool:
    """The coverage gate. True when it passes, or when there is nothing measurable to gate."""
    total = covered = measured_files = unmeasured_files = 0
    worst: list[tuple[float, str, int, int]] = []
    for path, lines in go_added.items():
        if excluded(path, cov_excl):
            continue
        prof = ran.get(path, {})
        # A file absent from the profile was never instrumented — with a SCOPED profile that is most
        # of them. Counting it as 0% would be wrong and skipping it silently would be worse, because
        # the percentage then describes a sample nobody chose knowingly.
        if not prof:
            unmeasured_files += 1
            continue
        measured_files += 1
        # Only lines the compiler treated as executable appear in a profile at all; a blank line or a
        # declaration is neither covered nor uncovered.
        executable = [ln for ln in lines if ln in prof]
        if not executable:
            continue
        hit = [ln for ln in executable if prof[ln]]
        total += len(executable)
        covered += len(hit)
        worst.append((len(hit) / len(executable), path, len(hit), len(executable)))
    if not total:
        print("    coverage on NEW code           no new executable lines in the profile")
        return True
    pct = 100 * covered / total
    ok = pct > COVERAGE_GATE
    print(f"    coverage on NEW code           {pct:>7.2f}%   gate >80%  "
          f"{'PASS' if ok else 'FAIL'}   ({covered}/{total} new executable lines in "
          f"{measured_files} file(s))")
    if unmeasured_files:
        # The headline must not be read as covering the whole change set.
        print(f"      NOT MEASURED: {unmeasured_files} changed file(s) are absent from the profile — "
              f"it was scoped. The figure above describes only what it instrumented; "
              f"`make sonar` measures the rest.")
    if not ok:
        print("      least-covered new files:")
        for _, path, hit, tot in sorted(worst)[:8]:
            print(f"        {100*hit/tot:>5.1f}%  {hit:>4}/{tot:<4}  {path}")
    return ok


def report_duplication(go_added: dict[str, set[int]], cpd_excl: list[re.Pattern[str]],
                       host: str, auth: str, key: str) -> bool:
    """The duplication gate, read off Sonar's own clone blocks rather than re-detected here."""
    dup_new = dup_total = 0
    for path, lines in go_added.items():
        if excluded(path, cpd_excl):
            continue
        dup_total += len(lines)
        dup_new += len(lines & duplicated_lines(host, auth, key, path))
    if not dup_total:
        print("    duplication on NEW code        no new non-excluded lines")
        return True
    pct = 100 * dup_new / dup_total
    ok = pct < DUPLICATION_GATE
    print(f"    duplication on NEW code        {pct:>7.2f}%   gate <3%   "
          f"{'PASS' if ok else 'FAIL'}   ({dup_new}/{dup_total} new lines in a clone)")
    return ok


def report_issues(go_added: dict[str, set[int]], host: str, auth: str, key: str) -> bool:
    """The issue gate: a smell on a line you just wrote is cheapest to fix before it is committed."""
    found = issues_on(host, auth, key, go_added)
    if not found:
        print("    issues on NEW code                   0   gate 0     PASS")
        return True
    print(f"    issues on NEW code             {len(found):>7}   gate 0     FAIL")
    for rule, path, line, message in found[:20]:
        print(f"        {rule:<14} {path}:{line}  {message}")
    if len(found) > 20:
        print(f"        … {len(found) - 20} more")
    return False


def main() -> int:
    base, host, auth, key = sys.argv[1:5]
    profile = sys.argv[5] if len(sys.argv) > 5 else None

    props = properties()
    module = "oblikovati.org/"
    cov_excl = globs(props.get("sonar.coverage.exclusions", "") + ",**/*_test.go")
    cpd_excl = globs(props.get("sonar.cpd.exclusions", ""))
    src_excl = globs(props.get("sonar.exclusions", ""))

    added = added_lines(base)
    go_added = {p: lines for p, lines in added.items()
                if p.endswith(".go") and not excluded(p, src_excl)}

    if profile:
        passed = report_coverage(go_added, coverage_by_line(profile, module), cov_excl)
    else:
        print("    coverage on NEW code           not measured (no coverage profile)")
        passed = True
    # `and` would short-circuit: every gate reports, whatever an earlier one decided. A gate that
    # goes quiet because another failed is how a second defect ships behind the first.
    passed = report_duplication(go_added, cpd_excl, host, auth, key) and passed
    # Issues are read for EVERY changed file, .go or not: the Python and shell tooling under scripts/
    # is new code too, and the coverage and duplication gates above deliberately see only Go.
    passed = report_issues({p: v for p, v in added.items() if not excluded(p, src_excl)},
                           host, auth, key) and passed

    # Two denominators, deliberately: coverage excludes sonar.coverage.exclusions (tests included),
    # duplication excludes sonar.cpd.exclusions (tests excluded here). Printing one total for both
    # would misdescribe whichever gate it did not belong to.
    print(f"    added Go lines, vs {base:<11} {sum(len(v) for v in go_added.values())}"
          f" across {len(go_added)} file(s)")
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
