# SPDX-License-Identifier: GPL-2.0-only
import csv
from collections import defaultdict
import gzip
import json
import math
from pathlib import Path
import statistics
import sys

def quantile(values, percentile):
    return sorted(values)[math.ceil(len(values) * percentile) - 1]

def timing_groups(samples, meta):
    if meta["mode"] == "continuous":
        return [samples[200:]]
    return [[sample for sample in samples if start <= int(sample["entry_ns"]) <= end]
            for start, end in meta["intervals"]]

def duration_ms(row):
    return (int(row["return_ns"]) - int(row["entry_ns"])) / 1e6

def print_duration(name, durations):
    print(f"  {name}: {len(durations)} calls, p50 {statistics.median(durations):.3f}, "
          f"p95 {quantile(durations, .95):.3f}, max {max(durations):.3f} ms")

def summarize_phases(path, rows, groups, meta):
    with path.open() as stream:
        phases = list(csv.DictReader(stream))
    intervals = ([(groups[0][0], groups[0][-1])] if meta["mode"] == "continuous"
                 else meta["intervals"])
    durations = defaultdict(list)
    for row in phases:
        entered = int(row["entry_ns"])
        if any(start <= entered <= end for start, end in intervals):
            durations[row["function"]].append(duration_ms(row))
    for name, samples in sorted(durations.items()):
        print_duration(name, samples)

    start, end = max(((a, b) for group in groups for a, b in zip(group, group[1:])),
                     key=lambda interval: interval[1] - interval[0])
    traced = sum(max(0, min(end, int(row["return_ns"])) - max(start, int(row["entry_ns"])))
                 for row in rows + phases)
    print(f"  Longest interval {(end - start) / 1e6:.3f} ms, "
          f"{traced / 1e6:.3f} ms summed across traced calls")

def summarize(folders):
    for folder in folders:
        meta = json.loads((folder / "run.json").read_text())
        path = folder / "presents.csv"
        trace = path.open() if path.exists() else gzip.open(path.with_suffix(".csv.gz"), "rt")
        with trace as stream:
            rows = list(csv.DictReader(stream))
        if meta["mode"] != "continuous" and "intervals" not in meta:
            continue
        successful = [row for row in rows if int(row["result"]) == 0]
        selected = timing_groups(successful, meta)
        groups = [[int(row["entry_ns"]) for row in group] for group in selected]
        times_ms = [(b-a) / 1e6 for group in groups for a, b in zip(group, group[1:])]
        if not times_ms:
            print(f"{folder.name}: empty present sample")
            continue
        total = sum(map(len, groups))
        print(f"{folder.name}: {total} frames, {1000/statistics.mean(times_ms):.1f} FPS, "
              f"p50 {statistics.median(times_ms):.2f}, p95 {quantile(times_ms, .95):.2f}, "
              f"p99 {quantile(times_ms, .99):.2f}, max {max(times_ms):.2f} ms, "
              f"{sum(t <= 1000/120 for t in times_ms)}/{len(times_ms)} at 120 Hz budget, "
              f"{len(rows)-len(successful)} non-successful presents")
        if len(groups) > 1:
            print("  Burst frame counts:", [len(group) for group in groups])
        print_duration("vkQueuePresentKHR", [duration_ms(row) for group in selected for row in group])
        phases_path = folder / "phases.csv"
        if phases_path.is_file():
            summarize_phases(phases_path, rows, groups, meta)

summarize(map(Path, sys.argv[1:]))
