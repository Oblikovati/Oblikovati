// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchFrameEditDefinition, SketchFrameOrigin},
    feature::{
        AuthoredFeature,
        work_geometry::{DatumLength, WorkPointReference},
        work_plane::WorkPlaneDefinition,
    },
    sketch::SketchPlane,
};
use oblikovati_persistence::package::DocumentPackage;

#[path = "freshness_tests/fakes.rs"]
mod fakes;
use fakes::{FakeFiles, FakeGuids, FakeObserver};

#[test]
fn file_owned_preview_rejects_old_datums_after_edit_and_history() {
    let mut files = FileWorkspace::new(FakeObserver, FakeFiles);
    let document = files
        .create_unbound(
            DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Part"),
            FakeObserver,
            "test",
            &mut FakeGuids,
        )
        .unwrap()
        .session_id;
    let sketch = add_datum_sketch(&mut files, document);
    let mut owner = PartReplayService::default();
    let initial = files.workspace().session(document).unwrap().snapshot();
    let old = owner.preview_datums(&files, document, initial).unwrap();
    assert_eq!(
        initial
            .resolve_sketch_plane_with_datums(sketch, &old)
            .unwrap()
            .recipe()
            .origin,
        [2., 0., 0.]
    );
    files
        .commands(document)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "3 cm")
        })
        .unwrap();
    let changed = files.workspace().session(document).unwrap().snapshot();
    assert!(
        changed
            .resolve_sketch_plane_with_datums(sketch, &old)
            .is_err()
    );
    let fresh = owner.preview_datums(&files, document, changed).unwrap();
    assert_eq!(
        changed
            .resolve_sketch_plane_with_datums(sketch, &fresh)
            .unwrap()
            .recipe()
            .origin,
        [3., 0., 0.]
    );
    files.commands(document).unwrap().undo().unwrap();
    assert!(
        files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .resolve_sketch_plane_with_datums(sketch, &fresh)
            .is_err()
    );
    files.commands(document).unwrap().redo().unwrap();
    assert!(
        files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .resolve_sketch_plane_with_datums(sketch, &fresh)
            .is_ok()
    );
}
fn add_datum_sketch(
    files: &mut FileWorkspace<FakeObserver, FakeObserver, FakeFiles>,
    document: u64,
) -> DocumentSketchId {
    files
        .commands(document)
        .unwrap()
        .run("Create datum sketch", |snapshot, ids| {
            let parameter = ids.allocate().unwrap();
            assert_eq!(parameter, 1);
            snapshot
                .parameters
                .insert_expression(parameter, "width", USER_PARAM, "2 cm")
                .unwrap();
            let feature = ids.allocate_feature().unwrap();
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    feature,
                    "Point",
                    WorkPlaneDefinition::Point([
                        DatumLength::Parameter(parameter),
                        DatumLength::Constant(0.),
                        DatumLength::Constant(0.),
                    ]),
                ))
                .unwrap();
            let sketch = ids.allocate_sketch().unwrap();
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
                            feature,
                        ))),
                        alignment: None,
                    }),
                )
                .unwrap();
            Ok::<_, String>(sketch)
        })
        .unwrap()
}
