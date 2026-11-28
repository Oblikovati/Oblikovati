// SPDX-License-Identifier: GPL-2.0-only
//! Certified Martensite Derived Part verification: 1x derive, 2x scale, YZ mirror, and v47 reopen.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        derived_part::{DeriveMirrorPlane, DeriveStyle, DerivedPartDefinition, DerivedPartRecipe},
        sketch_prism::{
            SketchDirectedCurve, SketchPrismDefinition, SketchPrismRecipe, SketchPrismSelection,
        },
        work_geometry::DatumLength,
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
use std::collections::BTreeMap;

const DOC: &str = "11111111-2222-3333-4444-555555555555";
const SOURCE_PATH: &str = "/virtual/Part7.opd";

fn resolution() -> PrismResolutionPolicy {
    PrismResolutionPolicy::Fixed(PrismResolution {
        version: 1,
        weld: 1e-7,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 0.01,
        display_angular: 0.1,
        maximum_tolerance: 1e-4,
    })
}

fn solve(snapshot: &mut DocumentSnapshot) {
    let solved = snapshot.solve_document_sketches(Default::default(), || false);
    snapshot.sketches = solved.candidate;
}

fn box_part(x0: f64, x1: f64, y0: f64, y1: f64, height: f64) -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut sketch = Sketch2D::default();
    let pts = [[x0, y0], [x1, y0], [x1, y1], [x0, y1]];
    for (i, pos) in pts.iter().enumerate() {
        sketch
            .add_point(SketchEntityId((i + 1) as u64), *pos)
            .unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((i + 1) as u64);
        let b = SketchEntityId(((i + 1) % 4 + 1) as u64);
        sketch
            .add_line(SketchEntityId(10 + i as u64), [a, b])
            .unwrap();
    }
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Sketch1", SketchPlane::xy(), sketch)
        .unwrap();
    solve(&mut snapshot);
    let mut history = DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap();
    let selection = SketchPrismSelection {
        outer: (10..14)
            .map(|id| SketchDirectedCurve {
                curve: SketchEntityId(id),
                reversed: false,
            })
            .collect(),
        holes: Vec::new(),
    };
    let prism = SketchPrismDefinition::from_recipe(&SketchPrismRecipe {
        version: 1,
        sketch: DocumentSketchId(1),
        selection,
        additional_outputs: Vec::new(),
        displacement: [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(height),
        ],
        reversed_normal: false,
        centered_normal: false,
        second_depth: None,
        taper: None,
        extent: None,
        cut_target: None,
        join_target: None,
        intersection_target: None,
    })
    .unwrap();
    let token = history.begin("Extrusion1").unwrap();
    history
        .edit(&token, |snap, ids| {
            snap.create_sketch_prism("Extrusion1", prism, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    history
}

fn recipe(scale: f64, mirror: bool, mirror_plane: DeriveMirrorPlane) -> DerivedPartDefinition {
    DerivedPartDefinition::from_recipe(&DerivedPartRecipe {
        version: 1,
        source_path: SOURCE_PATH.into(),
        source_name: "Part7.opd".into(),
        style: DeriveStyle::SingleSolid,
        include_bodies: true,
        include_sketches: false,
        include_work_geometry: false,
        include_parameters: false,
        scale,
        mirror,
        mirror_plane,
        link_face_color: false,
        link_sketch_format: false,
        show_all_objects: false,
        linked: true,
        model_state: "[Primary]".into(),
        design_view: "[Primary]".into(),
    })
    .unwrap()
}

#[test]
fn derive_imports_scales_mirrors_and_reopens_v47() {
    let document: DocumentGuid = DOC.parse().unwrap();
    // Source Part7: box [1..4] x [0..2] x [0..2], volume = 12.0, centroid X = 2.5
    let source_history = box_part(1.0, 4.0, 0.0, 2.0, 2.0);
    let mut sources = BTreeMap::new();
    sources.insert(SOURCE_PATH.to_string(), source_history.snapshot().clone());
    // Target Part15 already has 1 base solid: [0..2] x [0..2] x [0..1]
    let target_base = box_part(0.0, 2.0, 0.0, 2.0, 1.0);

    let run = |label: &str, def: DerivedPartDefinition| {
        let mut history = DocumentEditHistory::new(
            target_base.snapshot().clone(),
            RecipeIdAllocator::restore(Some(target_base.feature_high_water()), &[]).unwrap(),
        )
        .unwrap();
        let token = history.begin(label).unwrap();
        let feature = history
            .edit(&token, |snap, ids| {
                snap.create_derived_part("Part7.opd", def, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        let mut backend =
            CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution())
                .with_derived_sources(&sources);
        let part = backend
            .evaluate_private_candidate(history.snapshot())
            .unwrap();
        (history, feature, part)
    };

    // 1. 1x derive yields 2 solid bodies in Part15, derived body volume = 12.0, center X = +2.5
    let (h1, f1, p1) = run("derive-1x", recipe(1.0, false, DeriveMirrorPlane::XY));
    assert_eq!(p1.ordered_bodies().count(), 2, "{:?}", p1.health());
    let b1 = p1.body(f1).expect("derived solid");
    let v1 = b1.volume(&h1, document).unwrap();
    assert!((v1 - 12.0).abs() < 1e-4, "volume {v1} vs 12.0");
    let key1 = b1.body_reference_key(&h1, document).unwrap();
    let center1 = b1.body_bounds_center(&h1, document, key1).unwrap();
    assert!((center1[0] - 2.5).abs() < 1e-4, "center1 {center1:?}");

    // 2. 2x scale yields 8x volume (96.0) and doubled center X = +5.0
    let (h2, f2, p2) = run("derive-2x", recipe(2.0, false, DeriveMirrorPlane::XY));
    let b2 = p2.body(f2).expect("scaled derived solid");
    let v2 = b2.volume(&h2, document).unwrap();
    assert!((v2 - 96.0).abs() < 1e-3, "scaled volume {v2} vs 96.0");
    let key2 = b2.body_reference_key(&h2, document).unwrap();
    let center2 = b2.body_bounds_center(&h2, document, key2).unwrap();
    assert!((center2[0] - 5.0).abs() < 1e-4, "center2 {center2:?}");

    // 3. YZ mirror reflects X -> -X (center X = -2.5) with equal volume (12.0)
    let (hm, fm, pm) = run("derive-mirror-yz", recipe(1.0, true, DeriveMirrorPlane::YZ));
    let bm = pm.body(fm).expect("mirrored derived solid");
    let vm = bm.volume(&hm, document).unwrap();
    assert!((vm - 12.0).abs() < 1e-4, "mirrored volume {vm} vs 12.0");
    let keym = bm.body_reference_key(&hm, document).unwrap();
    let centerm = bm.body_bounds_center(&hm, document, keym).unwrap();
    assert!((centerm[0] + 2.5).abs() < 1e-4, "centerm {centerm:?}");

    // 4. Part program v47 package round-trip reopen
    let current = hm.snapshot();
    assert_eq!(current.work_geometry.part_recipe().unwrap().version, 47);
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_sketch_reservations(&current.sketches, hm.sketch_reservations())
        .unwrap()
        .with_work_geometry(&current.work_geometry, hm.feature_high_water())
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = document_snapshot::empty_snapshot();
    restored.work_geometry = reopened.restore_work_geometry().unwrap().program;
    restored.sketches = reopened.restore_sketches().unwrap().sketches;
    solve(&mut restored);
    let reopened_history =
        DocumentEditHistory::new(restored, RecipeIdAllocator::default()).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::with_policy(&reopened_history, document, resolution())
            .with_derived_sources(&sources);
    let part = backend
        .evaluate_private_candidate(reopened_history.snapshot())
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 2, "{:?}", part.health());
}
