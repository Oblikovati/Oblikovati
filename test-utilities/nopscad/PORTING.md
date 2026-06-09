<!-- SPDX-License-Identifier: GPL-2.0-only -->

# NopSCADlib porting — re-model OpenSCAD parts as native Oblikovati designs

## Goal

**Re-model every NopSCADlib part (OpenSCAD `.scad`) as a native Oblikovati
feature-based design.** The deliverable for each part is a parametric Oblikovati
part document (`.obk`) with a real **feature tree** — sketches plus
extrude / revolve / cut / hole / pattern / … features — that reproduces the
geometry of the OpenSCAD original.

The OpenSCAD source is the **reference to reproduce**, not the thing to import.

### This is NOT

- **Not a mesh/STL import.** Importing the rendered STL gives a dumb triangle
  soup with no features, no parameters, no editability — the opposite of the
  goal. (Golden STLs exist only as a *reference* to check a re-modelled part
  against; see `render_goldens.py` / `goldens/`.)
- **Not an automatic `.scad` → `.obk` converter.** There is no OpenSCAD importer
  and we are not building one. Each part is modelled deliberately with the
  feature API, the way a user would in the UI.
- **Not (only) raw-kernel boolean tests.** The `kernel/brep/nopscad_*_test.go`
  CSG tests build bodies straight from `ops.Boolean` to exercise and harden the
  kernel — they are a useful *byproduct* (and bug-finder) but they are NOT the
  end product. The end product is the feature-based design.

## Why re-model with features (not import)

A NopSCADlib part captures *design intent* (a wall thickness, a bore that stops
short of a face, a rounded corner radius). A feature tree preserves that intent:
the part stays parametric and editable, mass properties and drawings work, and
the re-modelling exercises the real authoring path (sketch solver → feature
recompute → B-rep) end to end. That is exactly the surface we want hardened, and
exactly what a CAD user gets. An STL import preserves none of it.

## Method (per part)

1. Read the OpenSCAD module (e.g. `vitamins/pcb.scad` `usb_C`) and extract its
   dimensions and *intent* (profiles, extrude depths, which cuts are blind vs.
   through, fillet/offset radii).
2. Re-model it in an Oblikovati part document with features — typically:
   `create_document` → `create_sketch` → author the profile (rectangle / arcs /
   rounded corners) → `extrude` (new body) → further `create_sketch` on a face →
   `extrude` with the **cut** operation for bores/pockets, plus holes, fillets,
   patterns as the part needs.
3. Verify the result against the OpenSCAD reference: the golden STL
   (bounding box / volume) and a visual check in the UI.
4. Save the `.obk`. Keep the feature tree clean and named.

Work the catalog (`CATALOG.md` / `catalog.json`) low→high complexity. Any kernel
defect surfaced while re-modelling is fixed first (tessellation/boolean bugs
preempt feature work — see the repo `CLAUDE.md`) and pinned with a regression
test.
