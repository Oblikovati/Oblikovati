// SPDX-License-Identifier: GPL-2.0-only
use super::{sketch_fixture::owned_sketch_session, *};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId},
};

fn one_line() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    geometry.add_point(SketchEntityId(2), [1., 0.]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    geometry
}

#[test]
fn scope_is_per_sketch_validated_and_does_not_change_document_history() {
    let mut session = owned_sketch_session(None, "Scope", one_line());
    let sketch = DocumentSketchId(1);
    let generation = session.generation();
    let initial = session.sketch_inference_scope(sketch).unwrap();
    assert_eq!(*initial.mode(), INFERENCE_SCOPE_CURRENT_COMMAND);
    session
        .set_sketch_inference_scope(
            sketch,
            INFERENCE_SCOPE_SELECTED_GEOMETRY,
            &[SketchEntityId(3), SketchEntityId(1)],
        )
        .unwrap();
    let selected = session.sketch_inference_scope(sketch).unwrap();
    assert_eq!(selected.selected(), &[SketchEntityId(1), SketchEntityId(3)]);
    assert!(
        session
            .set_sketch_inference_scope(
                sketch,
                INFERENCE_SCOPE_SELECTED_GEOMETRY,
                &[SketchEntityId(999)]
            )
            .is_err()
    );
    assert_eq!(session.sketch_inference_scope(sketch).unwrap(), selected);
    assert!(
        session
            .sketch_inference_scope(DocumentSketchId(99))
            .is_err()
    );
    assert_eq!(session.generation(), generation);
    assert!(session.undo_labels().is_empty());
}

#[test]
fn deleted_selected_geometry_fails_closed_until_the_scope_is_replaced() {
    let mut session = owned_sketch_session(None, "Scope", one_line());
    let sketch = DocumentSketchId(1);
    session
        .set_sketch_inference_scope(
            sketch,
            INFERENCE_SCOPE_SELECTED_GEOMETRY,
            &[SketchEntityId(3)],
        )
        .unwrap();
    session
        .delete_sketch_entities(sketch, &[SketchEntityId(3)])
        .unwrap();
    assert!(session.sketch_inference_scope(sketch).is_err());
    session
        .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
        .unwrap();
    assert_eq!(
        *session.sketch_inference_scope(sketch).unwrap().mode(),
        INFERENCE_SCOPE_ALL_GEOMETRY
    );
}

#[test]
fn line_chain_scope_matches_preview_and_commit() {
    use oblikovati_model::history::SketchLineChainCreation;
    let mut session = owned_sketch_session(None, "Scope", one_line());
    let sketch = DocumentSketchId(1);
    let input = SketchLineChainCreation {
        points: vec![[0.0, 0.0], [0.5, 0.5]],
        closed: false,
        construction: false,
        centerline: false,
        joins: Default::default(),
        locks: Default::default(),
    };
    let current = session.preview_scoped_line_chain(sketch, &input).unwrap();
    assert!(current.report.lines[0].inference.points.is_empty());
    session
        .set_sketch_inference_scope(
            sketch,
            INFERENCE_SCOPE_SELECTED_GEOMETRY,
            &[SketchEntityId(3)],
        )
        .unwrap();
    let selected = session.preview_scoped_line_chain(sketch, &input).unwrap();
    assert_eq!(
        selected.report.lines[0].inference.points[0].target,
        SketchEntityId(1)
    );
    let committed = session.create_scoped_line_chain(sketch, &input).unwrap();
    assert_eq!(selected.report, committed);
    assert_eq!(
        &selected.geometry,
        session.snapshot().sketches.get(sketch).unwrap().geometry()
    );
}
