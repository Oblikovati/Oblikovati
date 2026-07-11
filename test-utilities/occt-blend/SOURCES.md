<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OCCT blend test-data fixture provenance

## What's here

`data/` holds a flat copy of the `tests/blend/*` fixture files referenced by
`restore [locate_data_file <name>] s` across the 477 OCCT blend cases (`../OCCT/tests/blend`,
checkout commit `d3056ef80c9668f395da40f5fd7be186cae4501f`, 2026-05-06). The oracle
(`oracle/occt_blend_oracle`) points `CSF_TestDataPath` at this directory; `locate_data_file`
searches it recursively, so no subdirectory structure is required.

## Source used — and why it's not the repo the task plan named

The task plan called for cloning `https://github.com/Open-Cascade-SAS/occt-test-data.git`.
**That repository does not exist** — verified 2026-07-11:

```
$ git clone --depth 1 https://github.com/Open-Cascade-SAS/occt-test-data.git
Cloning into 'occt-test-data'...
remote: Repository not found.
fatal: repository 'https://github.com/Open-Cascade-SAS/occt-test-data.git/' not found
```

OCCT's own documentation confirms why: most `tests/*` fixture files are **confidential**
and live only inside Open Cascade SAS's internal system —
<https://dev.opencascade.org/doc/overview/html/occt_contribution__tests.html> states "many
tests are based on data files that are confidential and thus available only at OPEN
CASCADE," and that new fixtures are submitted by attaching them to a GitHub issue, not by
pushing to a public data repo. There is no public git mirror of the full fixture set.

What OPEN CASCADE *did* publish is a one-off sample: the "OCCT testing dataset" archive
announced on the dev.opencascade.org forum
(<https://dev.opencascade.org/content/open-cascade-technology-testing-dataset-published>,
posted 2021-03-29, "more than 2500 shapes," raised test-suite data coverage from ~30% to
~60%). That archive is what `fetch-data.sh` downloads:

- URL: `https://dev.opencascade.org/sites/default/files/free/shapes_7.5.0.tgz`
- SHA-256: `6b1684db87fdad137753a34d7e83d3230302bc6c4402ea967269edd269710ea9`
- Size: 65,580,939 bytes (downloaded 2026-07-11)
- Layout: flat category directories `brep/ geom/ iges/ msv/ others/ step/`, 3223 files
  total; every `tests/blend` fixture that resolved was found directly under `brep/`.
- No newer version is published: `shapes_7.6.0.tgz` … `shapes_8.0.0.tgz` all 404 on the
  same host (checked 2026-07-11); `7.5.0` is the only release.

This is a legitimate, reproducible, publicly-downloadable OPEN CASCADE artifact — not a
substitute invented to paper over a gap. It is a strict subset of what the real (private)
fixture set contains, which is exactly why 27 names don't resolve (below).

## Resolution count

**140 / 167** distinct fixture names referenced by `tests/blend/*` resolved and were copied
into `data/`.

## Unresolved fixtures (27) — NOT silently dropped

All 27 unresolved names are absent from the public `shapes_7.5.0.tgz` archive (confirmed by
exact and case-insensitive search over the full archive listing — no near-miss, no path
mismatch). They form one clean, explainable cluster: STEP-conformance-suite fixtures
(`CFI_*`, `CTO*_*`, and bare `cts*`/`pro*` case IDs) from the CAX-IF / PDES STEP
conformance-test corpus, which is a *different*, still-confidential dataset than the public
sample archive covers:

```
cfi900H2.rle
CFI_cfi90fjc.rle
CFI_cts20006.rle
CFI_cts20970.rle
CFI_cts21020.rle
CFI_cts21183.rle
CFI_cts21630.rle
CFI_fra60610.rle
CFI_ger60206.rle
CFI_jap50078.rle
CFI_pro10117.rle
CFI_pro10522.rle
CFI_pro10631.rle
CFI_pro12894.rle
CFI_pro13892.rle
CFI_pro5203.rle
CFI_pro5477.rle
CFI_pro5545.rle
CFI_pro9067.rle
CFI_pro9169.rle
CFI_pro9523.rle
CTO900_pro12880c.rle
CTO904_hkg60206.rle
cts16288.rle
cts21363.rle
pro12832.rle
pro13893.rle
```

Blend cases that `restore` one of these 27 fixtures cannot run through the oracle until the
files are sourced some other way (e.g. attached to an upstream OCCT GitHub issue per their
own contribution process, or obtained directly from OPEN CASCADE). They are a small tail
(27 of 477 grid cases at most reference one of these — most blend cases either construct
their input inline or use a fixture that *did* resolve) and are tracked as a known gap, not
silently skipped: any grid case whose `restore` target isn't in `data/` will fail loudly
(DRAWEXE `locate_data_file` errors) rather than pass spuriously.

## Reproducing

```bash
grep -rhoE 'locate_data_file [^]} ]+' ../OCCT/tests/blend/*/ \
  | sed 's/locate_data_file //' | sort -u > /tmp/needed-fixtures.txt   # 167 names
bash test-utilities/occt-blend/fetch-data.sh                            # downloads+copies
```
