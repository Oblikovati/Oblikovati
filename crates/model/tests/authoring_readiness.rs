// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchReadinessCause},
    history::DocumentSnapshot,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
};
fn snapshot(driven: bool) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    if driven {
        snapshot
            .parameters
            .insert_expression(1, "span", oblikovati_api::types::USER_PARAM, "3 cm")
            .unwrap();
        geometry.add_point(SketchEntityId(2), [3.0, 0.0]).unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(1),
                PlanarConstraint::Distance {
                    points: [SketchEntityId(1), SketchEntityId(2)],
                    parameter: 1,
                    measure: DistanceMeasure::Aligned,
                },
            )
            .unwrap();
    }
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Authoring",
            SketchPlane::xy(),
            geometry,
        )
        .unwrap();
    snapshot
}
fn recompute(snapshot: &mut DocumentSnapshot) {
    snapshot.sketches = snapshot
        .solve_pending_document_sketches(Default::default(), || false)
        .candidate;
}
#[test]
fn authoring_requires_current_convergence_but_does_not_require_zero_dof() {
    let mut snapshot = snapshot(false);
    let sketch = DocumentSketchId(1);
    assert_eq!(
        snapshot.require_converged_sketch(sketch).unwrap_err().cause,
        SketchReadinessCause::Unevaluated
    );
    recompute(&mut snapshot);
    assert_eq!(
        snapshot
            .require_converged_sketch(sketch)
            .unwrap()
            .points()
            .len(),
        1
    );
    assert_eq!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .require_fully_constrained(&snapshot.parameters)
            .unwrap_err()
            .cause,
        SketchReadinessCause::FreeVariables(2)
    );
    let missing = snapshot
        .require_converged_sketch(DocumentSketchId(99))
        .unwrap_err();
    assert_eq!(missing.sketch, DocumentSketchId(99));
    assert!(matches!(missing.cause, SketchReadinessCause::Failed(_)));
}
#[test]
fn stale_and_failed_parameter_reads_cannot_publish_an_old_preview() {
    let mut snapshot = snapshot(true);
    let sketch = DocumentSketchId(1);
    recompute(&mut snapshot);
    assert!(snapshot.require_converged_sketch(sketch).is_ok());
    snapshot.parameters.set_expression(1, "4 cm").unwrap();
    assert_eq!(
        snapshot.require_converged_sketch(sketch).unwrap_err().cause,
        SketchReadinessCause::Stale(vec![1])
    );
    recompute(&mut snapshot);
    assert!(snapshot.require_converged_sketch(sketch).is_ok());
    snapshot
        .parameters
        .set_expression(1, "missing_parameter")
        .unwrap();
    recompute(&mut snapshot);
    assert!(matches!(
        snapshot.require_converged_sketch(sketch).unwrap_err().cause,
        SketchReadinessCause::Failed(_)
    ));
}
