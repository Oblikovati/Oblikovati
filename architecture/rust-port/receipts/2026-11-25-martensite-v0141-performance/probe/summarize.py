# SPDX-License-Identifier: GPL-2.0-only
import csv
import gzip
import json
import math
from pathlib import Path
import statistics
import sys

def quantile(values, percentile):
    return sorted(values)[math.ceil(len(values) * percentile) - 1]

def summarize(folders):
    for folder in folders:
        meta = json.loads((folder / "run.json").read_text())
        path = folder / "presents.csv"
        trace = path.open() if path.exists() else gzip.open(path.with_suffix(".csv.gz"), "rt")
        with trace as stream:
            rows = list(csv.DictReader(stream))
        successful = [int(row["entry_ns"]) for row in rows if int(row["result"]) == 0]
        if meta["mode"] == "continuous":
            groups = [successful[200:]]
        elif "intervals" in meta:
            groups = [[stamp for stamp in successful if start <= stamp <= end]
                      for start, end in meta["intervals"]]
        else:
            continue
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

summarize(map(Path, sys.argv[1:]))
