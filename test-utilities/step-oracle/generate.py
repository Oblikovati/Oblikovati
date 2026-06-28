#!/usr/bin/env python3
"""Generate the OCC STEP oracle fixtures for TestOCCOracleVolumes.

OpenCASCADE (via the gmsh SDK) is the oracle: it emits the full real-world STEP geometry
vocabulary our importer must accept — FILE_NAME before FILE_DESCRIPTION, SURFACE_CURVE/
SEAM_CURVE edge carriers, FACE_BOUND for every bound (no FACE_OUTER_BOUND), VERTEX_LOOP poles,
rational B-spline fillets — and reports each solid's EXACT volume via occ.getMass. We commit
the .step files plus oracle.json (the exact volumes) so the Go test runs with no gmsh
dependency; regenerate with this script when adding scenarios.

Usage:
    pip install gmsh   # or download the SDK and set PYTHONPATH=<sdk>/lib
    python3 generate.py

Writes ../../kernel/exchange/step/testdata/occ/{*.step, oracle.json}.
"""
import json
import os

import gmsh

OUT = os.path.normpath(os.path.join(os.path.dirname(__file__), "..", "..", "kernel", "exchange", "step", "testdata", "occ"))


def freeform_trimmed(o):
    """A mild B-spline barrel loft (radii 9,10,9 over z=0,8,16) with an axial Ø6 through-bore.

    Genuinely freeform: addThruSections(makeRuled=False) emits B_SPLINE_SURFACE side faces and the
    closed cross-sections are B_SPLINE_CURVEs, so it exercises the trimmed-NURBS mesher (the bore
    trims the caps and adds an inner face). The mild convex curvature imports faithfully (our volume
    converges to OCC getMass), so it gates fold + over-enclosure for #584 without masking a bug.
    """
    import math

    def ring(z, r, n=28):
        pts = [o.addPoint(r * math.cos(2 * math.pi * i / n), r * math.sin(2 * math.pi * i / n), z) for i in range(n)]
        pts.append(pts[0])
        return o.addWire([o.addBSpline(pts)])

    solid = o.addThruSections([ring(0, 9), ring(8, 10), ring(16, 9)], makeSolid=True, makeRuled=False)[0][1]
    o.cut([(3, solid)], [(3, o.addCylinder(0, 0, -2, 0, 0, 20, 3))])


def bulged_duct(o):
    """A STRONGLY curved B-spline barrel loft (radii 6,13,6 over z=0,9,18) with an axial Ø6 through-bore.

    Unlike the mild freeform_trimmed (radii 9,10,9), the sharp 6→13→6 bulge gives the B_SPLINE_SURFACE
    side faces high curvature, so the importer + tessellator must carry a large chord error on a real
    freeform NURBS surface and the trimmed (bored) caps must stay watertight against it. It is the
    committed default subject of the imported-NURBS duct guards (TestImportedNurbsDuct*), so they run in
    CI on every machine; the heavy EDF bell-mouth duct — whose self-proximal lip and many interacting
    trimmed analytic faces are what reproduced the #585 over-enclosure — stays a developer-local
    OBK_PERF_STEP override, since that emergent failure does not miniaturize into a small fixture.
    """
    import math

    def ring2(z, r, n=28):
        pts = [o.addPoint(r * math.cos(2 * math.pi * i / n), r * math.sin(2 * math.pi * i / n), z) for i in range(n)]
        pts.append(pts[0])
        return o.addWire([o.addBSpline(pts)])

    solid = o.addThruSections([ring2(0, 6), ring2(9, 13), ring2(18, 6)], makeSolid=True, makeRuled=False)[0][1]
    o.cut([(3, solid)], [(3, o.addCylinder(0, 0, -2, 0, 0, 22, 3))])


def main():
    gmsh.initialize()
    gmsh.option.setNumber("General.Terminal", 0)
    oracle = {}

    def emit(name):
        gmsh.model.occ.synchronize()
        vols = gmsh.model.occ.getEntities(3)
        total = sum(gmsh.model.occ.getMass(3, t) for (_, t) in vols)
        gmsh.write(f"{OUT}/{name}.step")
        oracle[name] = {"volume": round(total, 4), "solids": len(vols)}
        gmsh.clear()

    o = gmsh.model.occ
    o.addBox(0, 0, 0, 10, 20, 30); emit("box")
    o.addSphere(0, 0, 0, 7); emit("sphere")
    o.addCylinder(0, 0, 0, 0, 0, 20, 5); emit("cylinder")
    o.addCone(0, 0, 0, 0, 0, 15, 8, 3); emit("cone_frustum")
    o.addCone(0, 0, 0, 0, 0, 12, 6, 0); emit("cone_sharp")
    o.addTorus(0, 0, 0, 10, 3); emit("torus")
    o.addWedge(0, 0, 0, 10, 10, 10); emit("wedge")
    b = o.addBox(0, 0, 0, 20, 20, 5); c = o.addCylinder(10, 10, -1, 0, 0, 7, 4)
    o.cut([(3, b)], [(3, c)]); emit("drilled_box")
    b = o.addBox(0, 0, 0, 20, 20, 20); o.fillet([b], list(range(1, 13)), [3]); emit("filleted_box")
    b = o.addBox(0, 0, 0, 20, 20, 20); o.chamfer([b], [1, 2, 3, 4], [1, 1, 1, 1], [2]); emit("chamfered_box")
    b1 = o.addBox(0, 0, 0, 10, 10, 10); b2 = o.addBox(5, 5, 5, 10, 10, 10)
    o.fuse([(3, b1)], [(3, b2)]); emit("fused_boxes")
    b = o.addBox(0, 0, 0, 10, 10, 10); c = o.addCylinder(5, 5, 10, 0, 0, 8, 3)
    o.fuse([(3, b)], [(3, c)]); emit("box_with_boss")
    o.addBox(0, 0, 0, 5, 5, 5); o.addBox(20, 0, 0, 5, 5, 5); emit("two_solids")
    o.addCylinder(0, 0, 0, 0, 0, 10, 5, angle=1.5); emit("partial_cylinder")
    o.addTorus(0, 0, 0, 10, 3, angle=3.14159); emit("partial_torus")
    o.addSphere(0, 0, 0, 8, angle3=1.0); emit("partial_sphere")
    freeform_trimmed(o); emit("freeform_trimmed")
    bulged_duct(o); emit("bulged_duct")

    json.dump(oracle, open(f"{OUT}/oracle.json", "w"), indent=1, sort_keys=True)
    gmsh.finalize()
    print(f"wrote {len(oracle)} fixtures + oracle.json to {OUT}")


if __name__ == "__main__":
    main()
