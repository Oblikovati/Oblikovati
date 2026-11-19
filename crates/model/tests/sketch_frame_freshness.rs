// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin},
    feature::{
        AuthoredFeature, FeatureId,
        work_geometry::{DatumLength, WorkPointReference},
        work_plane::WorkPlaneDefinition,
    },
    history::DocumentSnapshot,
    sketch::SketchPlane,
};

fn datum_frame() -> (DocumentSnapshot, DocumentSketchId) {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "2 cm")
        .unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Point",
            WorkPlaneDefinition::Point([
                DatumLength::Parameter(1),
                DatumLength::Constant(0.),
                DatumLength::Constant(0.),
            ]),
        ))
        .unwrap();
    let sketch = DocumentSketchId(1);
    snapshot
        .sketches
        .add(sketch, "Sketch", SketchPlane::xy(), Default::default())
        .unwrap();
    snapshot
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Datum(WorkPointReference::Feature(
                    FeatureId(1),
                ))),
                alignment: None,
            }),
        )
        .unwrap();
    (snapshot, sketch)
}

#[test]
fn retained_datum_evaluation_is_rejected_after_its_parameter_changes() {
    let (mut snapshot, sketch) = datum_frame();
    let old = snapshot
        .effective_work_geometry()
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    assert_eq!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &old)
            .unwrap()
            .recipe()
            .origin,
        [2., 0., 0.]
    );
    assert!(snapshot.bind_sketch_frames(&old).is_ok());
    snapshot.parameters.set_expression(1, "3 cm").unwrap();
    assert!(snapshot.bind_sketch_frames(&old).is_err());
    assert!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &old)
            .is_err()
    );
    let fresh = snapshot
        .effective_work_geometry()
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    assert_eq!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &fresh)
            .unwrap()
            .recipe()
            .origin,
        [3., 0., 0.]
    );
}

#[test]
fn absolute_frame_does_not_depend_on_a_retained_datum_evaluation() {
    let (mut snapshot, sketch) = datum_frame();
    let old = snapshot
        .effective_work_geometry()
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    snapshot
        .sketches
        .set_frame_edit(
            sketch,
            Some(SketchFrameEditDefinition {
                origin: Some(SketchFrameOrigin::Position([5., 0., 0.])),
                alignment: None,
            }),
        )
        .unwrap();
    snapshot.parameters.set_expression(1, "3 cm").unwrap();
    assert_eq!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &old)
            .unwrap()
            .recipe()
            .origin,
        [5., 0., 0.]
    );
}
