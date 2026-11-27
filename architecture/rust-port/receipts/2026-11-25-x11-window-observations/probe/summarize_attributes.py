# SPDX-License-Identifier: GPL-2.0-only
import csv
import gzip
import json
from pathlib import Path
import sys


def summarize(folder):
    meta = json.loads((folder / "run.json").read_text())
    with (folder / "presents.csv").open() as trace:
        presents = [int(row["entry_ns"]) for row in csv.DictReader(trace) if row["result"] == "0"]
    if meta["mode"] == "continuous":
        presents = presents[200:]
        if not presents:
            raise ValueError(f"{folder}: expected more than 200 successful presents")
        windows = [(presents[0], presents[-1])]
    else:
        windows = meta["intervals"]
    frames = sum(any(start <= stamp <= end for start, end in windows) for stamp in presents)
    if not frames:
        raise ValueError(f"{folder}: expected a successful present during the workload")

    attributes = folder / "attributes.csv"
    trace = (attributes.open() if attributes.is_file()
             else gzip.open(attributes.with_suffix(".csv.gz"), "rt"))
    with trace:
        elapsed = [
            int(row["elapsed_ns"])
            for row in csv.DictReader(trace)
            if any(start <= int(row["entry_ns"]) <= end for start, end in windows)
        ]
    duration_ms = sum(elapsed) / 1e6
    print(f"{folder.name}: {len(elapsed)} calls across {frames} presents, "
          f"{len(elapsed) / frames:.1f} calls/present, {duration_ms / frames:.2f} ms/present")


for path in sys.argv[1:]:
    summarize(Path(path))
