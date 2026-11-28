// SPDX-License-Identifier: GPL-2.0-only
//! Certified Martensite Decal verification: sketch image + face validation, planar wrap refusal, and v49/v18 reopen.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchImage},
    feature::{
        sketch_decal::{SketchDecalDefinition, SketchDecalRecipe},
        sketch_prism::{
            SketchDirectedCurve, SketchPrismDefinition, SketchPrismRecipe, SketchPrismSelection,
        },
        work_geometry::DatumLength,
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchPlaneOptions, SketchPlaneRecipe},
};
use oblikovati_persistence::package::DocumentPackage;

const DOC: &str = "11111111-2222-3333-4444-777777777777";

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

#[test]
fn decal_validates_image_and_face_refuses_planar_wrap_and_reopens_v49() {
    let document: DocumentGuid = DOC.parse().unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();

    let mut base = Sketch2D::default();
    let pts = [[0.0, 0.0], [4.0, 0.0], [4.0, 3.0], [0.0, 3.0]];
    for (i, pos) in pts.iter().enumerate() {
        base.add_point(SketchEntityId((i + 1) as u64), *pos)
            .unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((i + 1) as u64);
        let b = SketchEntityId(((i + 1) % 4 + 1) as u64);
        base.add_line(SketchEntityId(10 + i as u64), [a, b])
            .unwrap();
    }
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Sketch1", SketchPlane::xy(), base)
        .unwrap();

    let top = SketchPlane::from_recipe(
        SketchPlaneRecipe {
            origin: [0.0, 0.0, 2.0],
            x_axis: [1.0, 0.0, 0.0],
            y_axis: [0.0, 1.0, 0.0],
        },
        SketchPlaneOptions::default(),
    )
    .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(2), "Sketch2", top, Sketch2D::default())
        .unwrap();
    snapshot
        .sketches
        .add_image(
            DocumentSketchId(2),
            SketchImage {
                id: SketchEntityId(30),
                reference: "fixtures/cad/decal/decal-pattern.png".into(),
                anchor: [0.5, 0.5],
                width: 1.0,
                height: 1.0,
                rotation: 0.0,
                opacity: 1.0,
            },
        )
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
            snap.create_sketch_prism("Extrusion1", prism, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let body_key = SemanticReferenceName::new(document, base_id.0, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let top_face = SemanticReferenceName::new(document, base_id.0, 1, "sketch-prism", "end", &[1])
        .unwrap()
        .encode()
        .unwrap();

    // 1. Valid Decal on planar face with Automatic Face Chain enabled
    let decal_def = SketchDecalDefinition::from_recipe(&SketchDecalRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        image: SketchEntityId(30),
        target_face: top_face.clone(),
        target_body: body_key.clone(),
        chain_faces: true,
        wrap_to_face: false,
    })
    .unwrap();
    let token = history.begin("Decal1").unwrap();
    let decal_id = history
        .edit(&token, |snap, ids| {
            snap.create_sketch_decal("Decal1", decal_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend = CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
    let eval = backend
        .evaluate_private_candidate(history.snapshot())
        .unwrap();
    let solid = eval.body(decal_id).expect("expected decal solid");
    let vol = solid.volume(&history, document).unwrap();
    assert!(
        (vol - 24.0).abs() < 1e-4,
        "expected preserved volume 24, got {vol}"
    );

    // 2. Wrap to Face on planar face must fail cleanly
    let mut bad_hist = DocumentEditHistory::new(
        history.snapshot().clone(),
        RecipeIdAllocator::restore(Some(history.feature_high_water()), &[]).unwrap(),
    )
    .unwrap();
    let bad_def = SketchDecalDefinition::from_recipe(&SketchDecalRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        image: SketchEntityId(30),
        target_face: top_face,
        target_body: body_key,
        chain_faces: true,
        wrap_to_face: true,
    })
    .unwrap();
    let token = bad_hist.begin("DecalWrap").unwrap();
    let bad_id = bad_hist
        .edit(&token, |snap, ids| {
            snap.create_sketch_decal("DecalWrap", bad_def, ids)
        })
        .unwrap();
    bad_hist.commit(&token, false).unwrap();
    let mut bad_backend =
        CertifiedNamedSolidEvaluator::with_policy(&bad_hist, document, resolution());
    let bad_eval = bad_backend
        .evaluate_private_candidate(bad_hist.snapshot())
        .unwrap();
    assert!(bad_eval.body(bad_id).is_none());

    // 3. Part program v49 + Sketch recipe v18 package round-trip reopen
    let current = history.snapshot();
    assert_eq!(current.work_geometry.part_recipe().unwrap().version, 49);
    assert_eq!(current.sketches.recipe().version, 18);
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_sketch_reservations(&current.sketches, history.sketch_reservations())
        .unwrap()
        .with_work_geometry(&current.work_geometry, history.feature_high_water())
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = document_snapshot::empty_snapshot();
    restored.work_geometry = reopened.restore_work_geometry().unwrap().program;
    restored.sketches = reopened.restore_sketches().unwrap().sketches;
    solve(&mut restored);
    assert_eq!(
        restored
            .sketches
            .get(DocumentSketchId(2))
            .unwrap()
            .images()
            .len(),
        1
    );
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
