// SPDX-License-Identifier: GPL-2.0-only
//! Certified Martensite Emboss verification: raise, engrave, wrap-to-planar-face, from-plane refusal, and v46 reopen.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        sketch_emboss::{EmbossType, SketchEmbossDefinition, SketchEmbossRecipe},
        sketch_prism::{
            SketchDirectedCurve, SketchPrismDefinition, SketchPrismRecipe, SketchPrismSelection,
        },
        work_geometry::{DatumAngle, DatumLength},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator, SemanticReferenceName},
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchPlaneOptions, SketchPlaneRecipe},
};
use oblikovati_persistence::package::DocumentPackage;

const DOC: &str = "11111111-2222-3333-4444-555555555555";

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

fn top_plane(z: f64) -> SketchPlane {
    SketchPlane::from_recipe(
        SketchPlaneRecipe {
            origin: [0.0, 0.0, z],
            x_axis: [1.0, 0.0, 0.0],
            y_axis: [0.0, 1.0, 0.0],
        },
        SketchPlaneOptions::default(),
    )
    .unwrap()
}

fn block_with_top_circle() -> (DocumentEditHistory, Vec<u8>, Vec<u8>, SketchPrismSelection) {
    let document: DocumentGuid = DOC.parse().unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut base_sketch = Sketch2D::default();
    let pts = [[0.0, 0.0], [4.0, 0.0], [4.0, 3.0], [0.0, 3.0]];
    for (i, pos) in pts.iter().enumerate() {
        base_sketch
            .add_point(SketchEntityId((i + 1) as u64), *pos)
            .unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((i + 1) as u64);
        let b = SketchEntityId(((i + 1) % 4 + 1) as u64);
        base_sketch
            .add_line(SketchEntityId(10 + i as u64), [a, b])
            .unwrap();
    }
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Sketch1",
            SketchPlane::xy(),
            base_sketch,
        )
        .unwrap();

    let mut top_sketch = Sketch2D::default();
    top_sketch
        .add_point(SketchEntityId(20), [2.0, 1.5])
        .unwrap();
    top_sketch
        .add_circle(SketchEntityId(21), SketchEntityId(20), 0.5)
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(2), "Sketch2", top_plane(2.0), top_sketch)
        .unwrap();
    solve(&mut snapshot);

    let mut history = DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap();
    let base_selection = SketchPrismSelection {
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
        selection: base_selection,
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
    let top_face_key =
        SemanticReferenceName::new(document, base_id.0, 1, "sketch-prism", "end", &[1])
            .unwrap()
            .encode()
            .unwrap();
    let circle_selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: SketchEntityId(21),
            reversed: false,
        }],
        holes: Vec::new(),
    };
    (history, body_key, top_face_key, circle_selection)
}

#[test]
fn emboss_raises_engraves_wraps_planar_refuses_flush_plane_and_reopens_v46() {
    let document: DocumentGuid = DOC.parse().unwrap();
    let (base_history, body_key, top_face_key, circle) = block_with_top_circle();
    let base_volume = 4.0 * 3.0 * 2.0;
    let cyl_volume = std::f64::consts::PI * 0.5 * 0.5 * 0.1;

    let run_case = |name: &str, def: SketchEmbossDefinition| {
        let mut history = DocumentEditHistory::new(
            base_history.snapshot().clone(),
            RecipeIdAllocator::restore(Some(base_history.feature_high_water()), &[]).unwrap(),
        )
        .unwrap();
        let token = history.begin(name).unwrap();
        let feature = history
            .edit(&token, |snap, ids| {
                snap.create_sketch_emboss("Emboss1", def, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        let mut backend =
            CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
        let part = backend
            .evaluate_private_candidate(history.snapshot())
            .unwrap();
        (history, feature, part)
    };

    // 1. Raised emboss (+0.1 cm)
    let raised_def = SketchEmbossDefinition::from_recipe(&SketchEmbossRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        selection: circle.clone(),
        depth: DatumLength::Constant(0.1),
        emboss_type: EmbossType::EmbossFromFace,
        direction_reversed: false,
        taper: DatumAngle::constant(0.0).unwrap(),
        wrap_to_face: false,
        wrap_face: None,
        target_body: body_key.clone(),
    })
    .unwrap();
    let (raised_hist, emboss_id, part) = run_case("raised", raised_def);
    let solid = part.body(emboss_id).expect("expected raised emboss solid");
    let volume = solid.volume(&raised_hist, document).unwrap();
    assert!(
        (volume - (base_volume + cyl_volume)).abs() < 1e-4,
        "raised emboss volume {volume} vs expected {}",
        base_volume + cyl_volume
    );

    // 2. Engraved recess (-0.1 cm)
    let cut_def = SketchEmbossDefinition::from_recipe(&SketchEmbossRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        selection: circle.clone(),
        depth: DatumLength::Constant(0.1),
        emboss_type: EmbossType::EngraveFromFace,
        direction_reversed: false,
        taper: DatumAngle::constant(0.0).unwrap(),
        wrap_to_face: false,
        wrap_face: None,
        target_body: body_key.clone(),
    })
    .unwrap();
    let (cut_hist, cut_id, cut_part) = run_case("engrave", cut_def);
    let cut_solid = cut_part
        .body(cut_id)
        .expect("expected engraved recess solid");
    let cut_volume = cut_solid.volume(&cut_hist, document).unwrap();
    assert!(
        (cut_volume - (base_volume - cyl_volume)).abs() < 1e-4,
        "engraved emboss volume {cut_volume} vs expected {}",
        base_volume - cyl_volume
    );

    // 3. Wrap to planar face
    let wrap_def = SketchEmbossDefinition::from_recipe(&SketchEmbossRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        selection: circle.clone(),
        depth: DatumLength::Constant(0.1),
        emboss_type: EmbossType::EmbossFromFace,
        direction_reversed: false,
        taper: DatumAngle::constant(0.0).unwrap(),
        wrap_to_face: true,
        wrap_face: Some(top_face_key),
        target_body: body_key.clone(),
    })
    .unwrap();
    let (wrap_hist, wrap_id, wrap_part) = run_case("wrap", wrap_def);
    let wrap_solid = wrap_part
        .body(wrap_id)
        .expect("expected wrapped planar solid");
    let wrap_volume = wrap_solid.volume(&wrap_hist, document).unwrap();
    assert!(
        (wrap_volume - (base_volume + cyl_volume)).abs() < 1e-4,
        "wrapped emboss volume {wrap_volume} vs expected {}",
        base_volume + cyl_volume
    );

    // 4. Third operation (fromPlane) on flush planar face must fail cleanly
    let plane_def = SketchEmbossDefinition::from_recipe(&SketchEmbossRecipe {
        version: 1,
        sketch: DocumentSketchId(2),
        selection: circle,
        depth: DatumLength::Constant(0.1),
        emboss_type: EmbossType::EmbossEngraveFromPlane,
        direction_reversed: false,
        taper: DatumAngle::constant(0.0).unwrap(),
        wrap_to_face: false,
        wrap_face: None,
        target_body: body_key,
    })
    .unwrap();
    let (_plane_hist, plane_id, plane_part) = run_case("from-plane", plane_def);
    assert!(
        plane_part.body(plane_id).is_none(),
        "expected fromPlane emboss on flush planar face to fail"
    );

    // 5. Part program v46 package round-trip reopen
    let current = wrap_hist.snapshot();
    assert_eq!(current.work_geometry.part_recipe().unwrap().version, 46);
    let package = DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\n")
        .unwrap()
        .with_sketch_reservations(&current.sketches, wrap_hist.sketch_reservations())
        .unwrap()
        .with_work_geometry(&current.work_geometry, wrap_hist.feature_high_water())
        .unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = document_snapshot::empty_snapshot();
    restored.work_geometry = reopened.restore_work_geometry().unwrap().program;
    restored.sketches = reopened.restore_sketches().unwrap().sketches;
    solve(&mut restored);
    let reopened_history =
        DocumentEditHistory::new(restored, RecipeIdAllocator::default()).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::with_policy(&reopened_history, document, resolution());
    let part = backend
        .evaluate_private_candidate(reopened_history.snapshot())
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 1, "{:?}", part.health());
}
