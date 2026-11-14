// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{
        CircumcircleOptions, PlanarCircumcircle, Sketch2D, SketchCircleConstruction,
        SketchCircularPlacementCheck, SketchError, SketchPlacementRecipe, SolvedCircularSeed,
    },
};
use std::{boxed::Box, cell::RefCell, error::Error};
#[derive(Default)]
struct RecordingCircularVerifier {
    refuse: bool,
    observed: RefCell<Vec<SolvedCircularSeed>>,
}
impl SketchCircleConstruction for RecordingCircularVerifier {
    fn arc_direction(&self, _points: [[f64; 2]; 3]) -> Result<bool, Box<dyn Error>> {
        Err("verifier fake: expected verification only".into())
    }
    fn circumcircle(
        &self,
        _points: [[f64; 2]; 3],
        _options: CircumcircleOptions,
    ) -> Result<PlanarCircumcircle, Box<dyn Error>> {
        Err("verifier fake: expected verification only".into())
    }
    fn verify(
        &self,
        points: [[f64; 2]; 3],
        seed: SolvedCircularSeed,
        options: CircumcircleOptions,
    ) -> Result<(), Box<dyn Error>> {
        assert_eq!(points, check().points);
        assert_eq!(options, check().options);
        self.observed.borrow_mut().push(seed);
        if self.refuse {
            return Err(Box::new(SketchError::Definition(
                "named final seed refusal".into(),
            )));
        }
        Ok(())
    }
}
fn check() -> SketchCircularPlacementCheck {
    SketchCircularPlacementCheck {
        member: 0,
        cap_members: None,
        points: [[5., 0.], [3., 3.], [0., 2.]],
        options: CircumcircleOptions {
            maximum_point_residual: 1e-9,
        },
    }
}
fn checked_arc() -> SketchPlacementRecipe {
    let mut recipe = SketchPlacementRecipe::arc([0., 0.], [[5., 0.], [0., 2.]], true).unwrap();
    recipe.circular_checks.push(check());
    recipe
}
fn arc_session() -> DocumentSession<RecordingObserver> {
    super::sketch_fixture::owned_sketch_session(None, "Arc", Sketch2D::default())
}
#[test]
fn placement_verification_reads_solved_geometry_and_history_replays_it() {
    let mut session = arc_session();
    let verifier = RecordingCircularVerifier::default();
    let report = session
        .create_sketch_placement_with_construction(
            DocumentSketchId(1),
            &checked_arc(),
            &[],
            &verifier,
        )
        .unwrap();
    let observed = verifier.observed.borrow();
    assert_eq!(observed.len(), 1);
    let seed = observed[0];
    let (endpoints, ccw) = seed.arc.unwrap();
    assert!(ccw);
    for point in endpoints {
        assert!(
            ((point[0] - seed.center[0]).hypot(point[1] - seed.center[1]) - seed.radius).abs()
                < 1e-8
        );
    }
    assert_ne!(seed.center, [0., 0.]);
    let saved = session.snapshot().clone();
    assert_eq!(
        saved
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .read_index()
            .circular_seed(report.members[0])
            .unwrap(),
        seed
    );
    session.undo().unwrap();
    assert!(
        session
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .arcs()
            .is_empty()
    );
    session.redo().unwrap();
    assert!(saved.sketches.same_recipe(&session.snapshot().sketches));
}
#[test]
fn refused_verification_discards_candidates_and_retires_ids_within_a_group() {
    let mut session = arc_session();
    session.begin("Arc group").unwrap();
    session
        .create_sketch_placement(
            DocumentSketchId(1),
            &SketchPlacementRecipe::circle([10., 10.], 2.).unwrap(),
            &[],
        )
        .unwrap();
    let before = session.snapshot().clone();
    let reserved = session.history().sketch_reservations().entity;
    let verifier = RecordingCircularVerifier {
        refuse: true,
        ..Default::default()
    };
    let error = session
        .create_sketch_placement_with_construction(
            DocumentSketchId(1),
            &checked_arc(),
            &[],
            &verifier,
        )
        .unwrap_err();
    assert!(error.to_string().contains("named final seed refusal"));
    assert!(session.in_transaction());
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    assert!(session.history().sketch_reservations().entity > reserved);
    session.end().unwrap();
    assert_eq!(session.undo_labels(), vec!["Arc group"]);
}
#[test]
fn missing_service_and_malformed_member_checks_leave_no_geometry() {
    let mut session = arc_session();
    let before = session.snapshot().clone();
    let error = session
        .create_sketch_placement(DocumentSketchId(1), &checked_arc(), &[])
        .unwrap_err();
    assert!(error.to_string().contains("injected construction service"));
    let mut recipe = checked_arc();
    recipe.circular_checks[0].member = 99;
    let verifier = RecordingCircularVerifier::default();
    assert!(
        session
            .create_sketch_placement_with_construction(DocumentSketchId(1), &recipe, &[], &verifier)
            .unwrap_err()
            .to_string()
            .contains("99")
    );
    assert!(verifier.observed.borrow().is_empty());
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    assert!(session.undo_labels().is_empty());
}

#[test]
fn slot_verification_reads_cap_centers_and_refusal_never_adopts_the_slot() {
    let mut recipe =
        SketchPlacementRecipe::arc_slot([0., 0.], [[5., 0.], [0., 2.]], 2., true).unwrap();
    let mut centerline_check = check();
    centerline_check.cap_members = Some([3, 1]);
    recipe.circular_checks.push(centerline_check);
    for refuse in [false, true] {
        let mut session = arc_session();
        let verifier = RecordingCircularVerifier {
            refuse,
            ..Default::default()
        };
        let result = session.create_sketch_placement_with_construction(
            DocumentSketchId(1),
            &recipe,
            &[],
            &verifier,
        );
        assert_eq!(result.is_err(), refuse);
        let observed = verifier.observed.borrow();
        assert_eq!(observed.len(), 1);
        assert_eq!(
            observed[0],
            SolvedCircularSeed {
                center: [0., 0.],
                radius: 5.,
                arc: Some(([[5., 0.], [0., 5.]], true))
            }
        );
        let geometry = session
            .snapshot()
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry();
        assert_eq!(geometry.arcs().len(), if refuse { 0 } else { 4 });
        assert!(session.history().sketch_reservations().entity > 0);
    }
}
