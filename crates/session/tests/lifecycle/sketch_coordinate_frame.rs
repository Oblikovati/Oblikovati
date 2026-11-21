// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin},
    sketch::{Sketch2D, SketchError, SketchPlane},
};

fn move_origin(
    snapshot: &mut DocumentSnapshot,
    _: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<SketchPlane, SketchError> {
    let sketch = DocumentSketchId(1);
    snapshot.sketches.patch_frame_edit(
        sketch,
        SketchFrameEditDefinition {
            origin: Some(SketchFrameOrigin::Position([2., 3., 0.])),
            alignment: None,
        },
    )?;
    snapshot
        .resolve_sketch_plane(sketch)
        .map_err(|error| SketchError::Definition(error.to_string()))
}

#[test]
fn preview_is_transient_and_committed_frame_survives_undo_and_redo() {
    let mut session =
        super::sketch_fixture::owned_sketch_session(None, "Frame", Sketch2D::default());
    let sketch = DocumentSketchId(1);
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    let preview = session
        .preview_projected(move_origin, |_, plane| Ok(plane))
        .unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
    let committed = session
        .run_projected("Edit sketch coordinate system", move_origin, |_, plane| {
            Ok(plane)
        })
        .unwrap();
    assert!(preview.same_recipe(committed));
    assert_current_solve(&session, sketch);
    assert_eq!(
        session
            .snapshot()
            .resolve_sketch_plane(sketch)
            .unwrap()
            .recipe()
            .origin,
        [2., 3., 0.]
    );
    session.undo().unwrap();
    assert!(before.same_recipe(&session.snapshot().sketches));
    assert!(matches!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .health(&session.snapshot().parameters),
        oblikovati_model::document_sketch::PlacedSketchHealth::Unevaluated
    ));
    session.redo().unwrap();
    assert_current_solve(&session, sketch);
    assert_eq!(
        session
            .snapshot()
            .resolve_sketch_plane(sketch)
            .unwrap()
            .recipe()
            .origin,
        [2., 3., 0.]
    );
}

fn assert_current_solve(session: &DocumentSession<RecordingObserver>, sketch: DocumentSketchId) {
    use oblikovati_model::document_sketch::{PlacedSketchHealth, SketchEvaluation};
    let snapshot = session.snapshot();
    assert!(
        matches!(snapshot.sketches.get(sketch).unwrap().health(&snapshot.parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(diagnostics))
        if diagnostics.termination == oblikovati_solve::nonlinear::SolveTermination::Converged)
    );
}
