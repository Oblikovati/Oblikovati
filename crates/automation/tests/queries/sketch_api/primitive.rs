// SPDX-License-Identifier: GPL-2.0-only
use super::entities::{activate_copy, rows, verify_keys};
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement,
    history::{DocumentSketchCreation, SketchPrimitiveCreationReport},
    sketch::{SketchPlane, SketchPointInput, SketchPrimitiveCreation},
};
pub(super) fn create_owner(files: &mut DiscoveryWorkspace, document: u64) -> DocumentSketchId {
    files
        .commands(document)
        .unwrap()
        .create_sketch(&DocumentSketchCreation {
            name: "Typed geometry".into(),
            placement: SketchPlacement::Fixed(SketchPlane::xy()),
            orientation: None,
            project_origin: true,
        })
        .unwrap()
}
fn circle_input(anchor: SketchEntityId) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Circle {
        center: SketchPointInput::Owned(anchor),
        radius: 2.0,
        construction: true,
    }
}
pub(super) fn create_circle(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> SketchPrimitiveCreationReport {
    files
        .commands(document)
        .unwrap()
        .create_sketch_primitive(sketch, &circle_input(SketchEntityId(1)))
        .unwrap()
}
#[test]
fn typed_authoring_reaches_canonical_inspection_and_reference_queries_for_both_document_kinds() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, _) = configured_files(kind, Default::default());
        let sketch = create_owner(&mut files, document);
        let report = create_circle(&mut files, document, sketch);
        assert_eq!(report.created, [SketchEntityId(2)]);
        assert_eq!(report.points, [SketchEntityId(1)]);
        let inspected = rows(&files);
        assert_eq!(inspected.len(), 2);
        assert_circle_row(&inspected[1]);
        assert_eq!(inspected[1].moveable_status, "freeToMove");
        verify_keys(&files, &inspected);
    }
}
#[test]
fn created_shared_geometry_survives_save_reopen_and_copy_with_new_namespaces() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let sketch = create_owner(&mut files, document);
    create_circle(&mut files, document, sketch);
    let original = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), original);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/typed-copy.obk"),
    );
    let copied = rows(&files);
    assert_eq!(copied.len(), original.len());
    assert_copy_namespaces(&original, &copied);
    verify_keys(&files, &copied);
}
#[test]
fn grouped_creation_commit_and_undo_use_the_callers_label() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    let sketch = create_owner(&mut files, document);
    files
        .commands(document)
        .unwrap()
        .begin("Author profile")
        .unwrap();
    create_circle(&mut files, document, sketch);
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Author profile"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files).len(), 1);
    files.commands(document).unwrap().redo().unwrap();
    assert!(rows(&files)[1].construction);
}
#[test]
fn response_error_and_save_reopen_keep_retired_ids_reserved() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let sketch = create_owner(&mut files, document);
    let result = files
        .commands(document)
        .unwrap()
        .create_sketch_primitive_projected(sketch, &circle_input(SketchEntityId(1)), |report| {
            assert_eq!(report.unwrap().entity, SketchEntityId(2));
            Err::<(), _>("reply rejected")
        });
    assert!(result.is_err());
    assert_eq!(rows(&files).len(), 1);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(
        create_circle(&mut files, document, sketch).entity,
        SketchEntityId(3)
    );
}
#[test]
fn healthy_original_names_survive_appending_with_no_focus_change() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    let sketch = create_owner(&mut files, document);
    let original = rows(&files);
    create_circle(&mut files, document, sketch);
    assert_eq!(rows(&files)[0], original[0]);
    verify_keys(&files, &original);
    assert_eq!(files.workspace().active_document(), Some(document));
}

fn assert_circle_row(row: &oblikovati_api::wire::SketchEntityInfo) {
    assert_eq!((row.id, row.construction, row.radius), (2, true, 2.0));
}
fn assert_copy_namespaces(
    original: &[oblikovati_api::wire::SketchEntityInfo],
    copied: &[oblikovati_api::wire::SketchEntityInfo],
) {
    for (first, second) in original.iter().zip(copied) {
        assert_eq!(
            (first.id, first.construction, &first.points),
            (second.id, second.construction, &second.points)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
}
