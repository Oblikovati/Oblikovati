// SPDX-License-Identifier: GPL-2.0-only
//! Certified Martensite Coil kernel verification across helical, spiral, tapered, reversed, closed-end, and package reopen.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        sketch_coil::{
            CoilAxisSource, CoilEndTreatment, CoilHandedness, CoilMethod, SketchCoilDefinition,
            SketchCoilRecipe,
        },
        sketch_prism::{SketchDirectedCurve, SketchPrismSelection},
        work_axis::WorkAxisReference,
        work_geometry::{DatumAngle, DatumLength},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, RecipeIdAllocator},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;

const DOC: &str = "11111111-2222-3333-4444-555555555555";
const SKETCH_ID: DocumentSketchId = DocumentSketchId(1);
const CIRCLE_ID: SketchEntityId = SketchEntityId(3);

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

fn snapshot_with_offset_circle() -> (DocumentSnapshot, SketchPrismSelection) {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(2), [-2.0, -1.25]).unwrap();
    sketch
        .add_circle(CIRCLE_ID, SketchEntityId(2), 0.25)
        .unwrap();
    snapshot
        .sketches
        .add(SKETCH_ID, "Sketch1", SketchPlane::xy(), sketch)
        .unwrap();
    solve(&mut snapshot);
    let selection = SketchPrismSelection {
        outer: vec![SketchDirectedCurve {
            curve: CIRCLE_ID,
            reversed: false,
        }],
        holes: Vec::new(),
    };
    (snapshot, selection)
}

#[test]
fn coil_builds_certified_helix_spiral_taper_reversed_closed_ends_and_reopens() {
    let (snapshot, selection) = snapshot_with_offset_circle();
    let document: DocumentGuid = DOC.parse().unwrap();
    let cases = [
        (
            "helix-left",
            CoilMethod::PitchAndRevolution,
            CoilHandedness::Left,
            0.0,
            false,
            None,
        ),
        (
            "helix-right",
            CoilMethod::PitchAndRevolution,
            CoilHandedness::Right,
            0.0,
            false,
            None,
        ),
        (
            "spiral",
            CoilMethod::Spiral,
            CoilHandedness::Left,
            0.0,
            false,
            None,
        ),
        (
            "taper-5deg",
            CoilMethod::PitchAndRevolution,
            CoilHandedness::Left,
            5.0_f64.to_radians(),
            false,
            None,
        ),
        (
            "taper-reversed",
            CoilMethod::PitchAndRevolution,
            CoilHandedness::Left,
            5.0_f64.to_radians(),
            true,
            None,
        ),
        (
            "closed-ends",
            CoilMethod::PitchAndRevolution,
            CoilHandedness::Right,
            0.0,
            false,
            Some(CoilEndTreatment {
                flat_angle: DatumAngle::constant(90.0_f64.to_radians()).unwrap(),
                transition_angle: DatumAngle::constant(90.0_f64.to_radians()).unwrap(),
            }),
        ),
    ];
    for (name, method, handedness, taper_rad, axis_reversed, end) in cases {
        let def = SketchCoilDefinition::from_recipe(&SketchCoilRecipe {
            version: 1,
            sketch: SKETCH_ID,
            selection: selection.clone(),
            axis: CoilAxisSource::WorkAxis(WorkAxisReference::OriginY),
            axis_reversed,
            method,
            pitch: Some(DatumLength::Constant(1.5)),
            revolutions: Some(2.5),
            height: None,
            taper: DatumAngle::constant(taper_rad).unwrap(),
            handedness,
            start_end: end,
            end_end: end,
            boolean_target: None,
        })
        .unwrap();
        let mut history =
            DocumentEditHistory::new(snapshot.clone(), RecipeIdAllocator::default()).unwrap();
        let token = history.begin(name).unwrap();
        let feature = history
            .edit(&token, |snap, ids| {
                snap.create_sketch_coil("Coil1", def, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        let mut backend =
            CertifiedNamedSolidEvaluator::with_policy(&history, document, resolution());
        let part = backend
            .evaluate_private_candidate(history.snapshot())
            .unwrap();
        assert_eq!(
            part.ordered_bodies().count(),
            1,
            "{name}: {:?}",
            part.health()
        );
        let body = part.body(feature).unwrap();
        assert_eq!(
            body.reference_keys(&history, document).unwrap().len(),
            4,
            "{name}"
        );
        assert!(body.volume(&history, document).unwrap() > 5.0, "{name}");
    }

    // Verify Part program v45 package round-trip reopen.
    let def = SketchCoilDefinition::from_recipe(&SketchCoilRecipe {
        version: 1,
        sketch: SKETCH_ID,
        selection,
        axis: CoilAxisSource::WorkAxis(WorkAxisReference::OriginY),
        axis_reversed: false,
        method: CoilMethod::PitchAndRevolution,
        pitch: Some(DatumLength::Constant(1.5)),
        revolutions: Some(2.5),
        height: None,
        taper: DatumAngle::constant(0.0).unwrap(),
        handedness: CoilHandedness::Left,
        start_end: None,
        end_end: None,
        boolean_target: None,
    })
    .unwrap();
    let mut history = DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap();
    let token = history.begin("Coil1").unwrap();
    let _feature = history
        .edit(&token, |snap, ids| {
            snap.create_sketch_coil("Coil1", def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let program = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 45);

    let current = history.snapshot();
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
    let reopened_history =
        DocumentEditHistory::new(restored, RecipeIdAllocator::default()).unwrap();
    let mut backend =
        CertifiedNamedSolidEvaluator::with_policy(&reopened_history, document, resolution());
    let part = backend
        .evaluate_private_candidate(reopened_history.snapshot())
        .unwrap();
    assert_eq!(part.ordered_bodies().count(), 1, "{:?}", part.health());
}
