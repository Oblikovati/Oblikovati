// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    wire::{DeleteSketchDimensionArgs, DeleteSketchDimensionResult},
};
pub(super) fn populated(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = fresh(kind);
    files
        .commands(document)
        .unwrap()
        .run("Owned dimension", install_dimension)
        .unwrap();
    (files, document, source)
}
fn install_dimension(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let parameter = ids.allocate().unwrap();
    snapshot
        .parameters
        .insert_expression(parameter, "radius", MODEL_PARAM, "1 cm")
        .unwrap();
    let sketch = DocumentSketchId(1);
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    let relation = Radius {
        curve: entity(9),
        parameter,
    };
    geometry
        .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .map_err(|cause| cause.to_string())
}
fn delete_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
) -> Result<DeleteSketchDimensionResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .delete(&DeleteSketchDimensionArgs {
            dimension_index: index,
            ..Default::default()
        })
}
fn state_dof(files: &DiscoveryWorkspace, document: u64) -> usize {
    files
        .workspace()
        .session(document)
        .unwrap()
        .analyze_sketch(DocumentSketchId(1))
        .unwrap()
        .degrees_of_freedom
}
fn assert_dimension_present(files: &DiscoveryWorkspace, document: u64, present: bool) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(2).is_some(), present);
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(
        geometry.dimensional_constraints().count(),
        usize::from(present)
    );
    assert!(snapshot.parameters.get(1).is_some());
}
#[test]
fn canonical_delete_releases_one_dof_in_both_owner_kinds() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = populated(kind);
        let before = state_dof(&files, document);
        let focus = files.workspace().active_document();
        let result = delete_call(&mut files, &mut source, 0).unwrap();
        assert_eq!(usize::try_from(result.dof).unwrap(), before + 1);
        assert_eq!(state_dof(&files, document), before + 1);
        assert_dimension_present(&files, document, false);
        assert_eq!(files.workspace().active_document(), focus);
        assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_DELETE_DIMENSION));
    }
}
#[test]
fn invalid_dimension_indices_and_payloads_preserve_owner_state() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(
            delete_call(&mut files, &mut source, index),
            "dimension index",
        );
    }
    assert_malformed(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    assert_dimension_present(&files, document, true);
}
fn assert_malformed(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let host = FileHost::new(files, source, "test");
    for input in [b"{".as_slice(), b"[]", br#"{"dimensionIndex":"bad"}"#] {
        expect_api_decline(
            host.call(METHOD_SKETCH_DELETE_DIMENSION, input),
            METHOD_SKETCH_DELETE_DIMENSION,
        );
    }
}
#[test]
fn deletion_navigation_and_saved_reopen_preserve_the_cascade() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    delete_call(&mut files, &mut source, 0).unwrap();
    assert_dimension_present(&files, document, false);
    files.commands(document).unwrap().undo().unwrap();
    assert_dimension_present(&files, document, true);
    files.commands(document).unwrap().redo().unwrap();
    assert_dimension_present(&files, document, false);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_dimension_present(&files, reopened, false);
}
#[test]
fn commit_veto_restores_owned_source_and_relation() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let document = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let before = owner_revision(&files, document);
    expect_api_decline(delete_call(&mut files, &mut source, 0), "locked");
    assert_dimension_present(&files, document, true);
    let after = owner_revision(&files, document);
    assert_eq!(after.0, before.0 + 2);
    assert_eq!((after.1, after.2), (before.1, before.2));
}
#[test]
fn workspace_wrapper_deletes_and_group_abort_restores_owned_dimension() {
    let (mut files, document, _) = populated(DOCUMENT_PART);
    let mut commands = files.commands(document).unwrap();
    commands.begin("Delete group").unwrap();
    commands
        .delete_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1))
        .unwrap();
    commands.abort().unwrap();
    assert_dimension_present(&files, document, true);
}
#[test]
fn wrong_document_kind_cannot_delete_a_dimension() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(delete_call(&mut files, &mut source, 0), "part or assembly");
}
