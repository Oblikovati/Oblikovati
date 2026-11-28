// SPDX-License-Identifier: GPL-2.0-only
//! Certified Martensite Rib verification: valid normal/parallel webs, invalid boundary/out-of-bounds refusals, and v48 reopen.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        sketch_prism::{
            SketchDirectedCurve, SketchPrismDefinition, SketchPrismRecipe, SketchPrismSelection,
        },
        sketch_rib::{
            RibDirection, RibThickenSide, RibThicknessPlane, SketchRibDefinition, SketchRibRecipe,
        },
        work_geometry::{DatumAngle, DatumLength},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchPlaneOptions, SketchPlaneRecipe},
};
use oblikovati_persistence::package::DocumentPackage;

const DOC: &str = "11111111-2222-3333-4444-666666666666";

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

fn plane(origin: [f64; 3], x_axis: [f64; 3], y_axis: [f64; 3]) -> SketchPlane {
    SketchPlane::from_recipe(
        SketchPlaneRecipe {
            origin,
            x_axis,
            y_axis,
        },
        SketchPlaneOptions::default(),
    )
    .unwrap()
}

/// Build a hollow U-channel solid (outer 4x3x2 cm, interior pocket 3.2x2.2x1.5 cm open at z=2).
fn u_channel_part() -> (DocumentEditHistory, Vec<u8>) {
    let document: DocumentGuid = DOC.parse().unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    // Sketch1: outer 4x3 rectangle on XY
    let mut s1 = Sketch2D::default();
    let pts = [[0.0, 0.0], [4.0, 0.0], [4.0, 3.0], [0.0, 3.0]];
    for (i, pos) in pts.iter().enumerate() {
        s1.add_point(SketchEntityId((i + 1) as u64), *pos).unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((i + 1) as u64);
        let b = SketchEntityId(((i + 1) % 4 + 1) as u64);
        s1.add_line(SketchEntityId(10 + i as u64), [a, b]).unwrap();
    }
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Sketch1", SketchPlane::xy(), s1)
        .unwrap();

    // Sketch2: pocket rectangle on z=2.0 (0.4..3.6 x 0.4..2.6) cut down by 1.5 cm
    let mut s2 = Sketch2D::default();
    let inner = [[0.4, 0.4], [3.6, 0.4], [3.6, 2.6], [0.4, 2.6]];
    for (i, pos) in inner.iter().enumerate() {
        s2.add_point(SketchEntityId((21 + i) as u64), *pos).unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((21 + i) as u64);
        let b = SketchEntityId((21 + (i + 1) % 4) as u64);
        s2.add_line(SketchEntityId(30 + i as u64), [a, b]).unwrap();
    }
    snapshot
        .sketches
        .add(
            DocumentSketchId(2),
            "Sketch2",
            plane([0.0, 0.0, 2.0], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
            s2,
        )
        .unwrap();

    // Sketch3: valid open line spanning pocket at z=2.0 from (0.6, 1.5) to (3.4, 1.5)
    let mut s3 = Sketch2D::default();
    s3.add_point(SketchEntityId(41), [0.6, 1.5]).unwrap();
    s3.add_point(SketchEntityId(42), [3.4, 1.5]).unwrap();
    s3.add_line(SketchEntityId(43), [SketchEntityId(41), SketchEntityId(42)])
        .unwrap();
    snapshot
        .sketches
        .add(
            DocumentSketchId(3),
            "Sketch3",
            plane([0.0, 0.0, 2.0], [1.0, 0.0, 0.0], [0.0, 1.0, 0.0]),
            s3,
        )
        .unwrap();

    // Sketch4: invalid open line extending beyond solid X=4.0 on side face Y=0
    let mut s4 = Sketch2D::default();
    s4.add_point(SketchEntityId(51), [2.0, 1.0]).unwrap();
    s4.add_point(SketchEntityId(52), [5.0, 1.0]).unwrap();
    s4.add_line(SketchEntityId(53), [SketchEntityId(51), SketchEntityId(52)])
        .unwrap();
    snapshot
        .sketches
        .add(
            DocumentSketchId(4),
            "Sketch4",
            plane([0.0, 0.0, 0.0], [1.0, 0.0, 0.0], [0.0, 0.0, 1.0]),
            s4,
        )
        .unwrap();

    solve(&mut snapshot);
    let mut history = DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap();
    let sel = |start: u64| SketchPrismSelection {
        outer: (start..start + 4)
            .map(|id| SketchDirectedCurve {
                curve: SketchEntityId(id),
                reversed: false,
            })
            .collect(),
        holes: Vec::new(),
    };
    let base_prism = SketchPrismDefinition::from_recipe(&SketchPrismRecipe {
        version: 1,
        sketch: DocumentSketchId(1),
        selection: sel(10),
        additional_outputs: Vec::new(),
        displacement: [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(2.0),
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
    let base_id = history
        .edit(&token, |snap, ids| {
            snap.create_sketch_prism("Extrusion1", base_prism, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let base_key = SemanticReferenceName::new(document, base_id.0, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let cut_prism = SketchPrismDefinition::from_recipe(&SketchPrismRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        selection: sel(30),
        additional_outputs: Vec::new(),
        displacement: [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(1.5),
        ],
        reversed_normal: true,
        centered_normal: false,
        second_depth: None,
        taper: None,
        extent: None,
        cut_target: Some(base_key),
        join_target: None,
        intersection_target: None,
    })
    .unwrap();
    let token = history.begin("Extrusion2").unwrap();
    let pocket_id = history
        .edit(&token, |snap, ids| {
            snap.create_sketch_prism("Extrusion2", cut_prism, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
    let evaluated = backend
        .evaluate_private_candidate(history.snapshot())
        .unwrap();
    let pocket_key = evaluated
        .body(pocket_id)
        .expect("pocket solid")
        .body_reference_key(&history, document)
        .unwrap()
        .to_vec();
    (history, pocket_key)
}

#[test]
fn rib_creates_internal_web_refuses_out_of_bounds_and_reopens_v48() {
    let document: DocumentGuid = DOC.parse().unwrap();
    let (base_history, body_key) = u_channel_part();
    let mut base_backend =
        CertifiedNamedSolidEvaluator::with_policy(&base_history, document, resolution());
    let base_eval = base_backend
        .evaluate_private_candidate(base_history.snapshot())
        .unwrap();
    let base_volume = base_eval
        .ordered_bodies()
        .next()
        .unwrap()
        .volume(&base_history, document)
        .unwrap();

    // 1. Valid NormalToSketchPlane rib inside the pocket (0.2 cm thick, extend to walls)
    let mut ok_hist = DocumentEditHistory::new(
        base_history.snapshot().clone(),
        RecipeIdAllocator::restore(Some(base_history.feature_high_water()), &[]).unwrap(),
    )
    .unwrap();
    let rib_def = SketchRibDefinition::from_recipe(&SketchRibRecipe {
        version: 1,
        sketch: DocumentSketchId(3),
        profile_curves: vec![SketchEntityId(43)],
        thickness: DatumLength::Constant(0.2),
        depth: None,
        to_next: true,
        direction: RibDirection::NormalToSketchPlane,
        flip_direction: false,
        thicken_side: RibThickenSide::Symmetric,
        extend_profile: true,
        draft: DatumAngle::constant(0.0).unwrap(),
        thickness_plane: RibThicknessPlane::AtTop,
        target_body: body_key.clone(),
    })
    .unwrap();
    let token = ok_hist.begin("Rib1").unwrap();
    let rib_id = ok_hist
        .edit(&token, |snap, ids| {
            snap.create_sketch_rib("Rib1", rib_def, ids)
        })
        .unwrap();
    ok_hist.commit(&token, false).unwrap();

    let mut backend = CertifiedNamedSolidEvaluator::with_policy(&ok_hist, document, resolution());
    let eval = backend
        .evaluate_private_candidate(ok_hist.snapshot())
        .unwrap();
    let rib_solid = eval.body(rib_id).expect("expected valid rib solid");
    let rib_volume = rib_solid.volume(&ok_hist, document).unwrap();
    assert!(
        rib_volume > base_volume + 0.5,
        "expected rib volume ({rib_volume}) > base ({base_volume}) + 0.5"
    );

    // 2. Out-of-bounds side-face profile must fail cleanly
    let mut bad_hist = DocumentEditHistory::new(
        base_history.snapshot().clone(),
        RecipeIdAllocator::restore(Some(base_history.feature_high_water()), &[]).unwrap(),
    )
    .unwrap();
    let bad_def = SketchRibDefinition::from_recipe(&SketchRibRecipe {
        version: 1,
        sketch: DocumentSketchId(4),
        profile_curves: vec![SketchEntityId(53)],
        thickness: DatumLength::Constant(0.1),
        depth: None,
        to_next: true,
        direction: RibDirection::NormalToSketchPlane,
        flip_direction: false,
        thicken_side: RibThickenSide::Symmetric,
        extend_profile: true,
        draft: DatumAngle::constant(0.0).unwrap(),
        thickness_plane: RibThicknessPlane::AtTop,
        target_body: body_key,
    })
    .unwrap();
    let token = bad_hist.begin("RibBad").unwrap();
    let bad_id = bad_hist
        .edit(&token, |snap, ids| {
            snap.create_sketch_rib("RibBad", bad_def, ids)
        })
        .unwrap();
    bad_hist.commit(&token, false).unwrap();
    let mut bad_backend =
        CertifiedNamedSolidEvaluator::with_policy(&bad_hist, document, resolution());
    let bad_eval = bad_backend
        .evaluate_private_candidate(bad_hist.snapshot())
        .unwrap();
    assert!(
        bad_eval.body(bad_id).is_none(),
        "expected out-of-bounds rib profile to fail cleanly"
    );

    // 3. Part program v48 package round-trip reopen
    let current = ok_hist.snapshot();
    assert_eq!(current.work_geometry.part_recipe().unwrap().version, 48);
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_sketch_reservations(&current.sketches, ok_hist.sketch_reservations())
        .unwrap()
        .with_work_geometry(&current.work_geometry, ok_hist.feature_high_water())
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = document_snapshot::empty_snapshot();
    restored.work_geometry = reopened.restore_work_geometry().unwrap().program;
    restored.sketches = reopened.restore_sketches().unwrap().sketches;
    solve(&mut restored);
    let reopened_hist = DocumentEditHistory::new(restored, RecipeIdAllocator::default()).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::with_policy(&reopened_hist, document, resolution());
    let reopened_eval = backend
        .evaluate_private_candidate(reopened_hist.snapshot())
        .unwrap();
    assert_eq!(
        reopened_eval.ordered_bodies().count(),
        1,
        "{:?}",
        reopened_eval.health()
    );
}
