// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchReferenceIndex, SketchPlacement},
    history::{
        DocumentSketchCreation, SketchDimensionCreation, SketchDimensionCreationReport,
        SketchDimensionEdit, SketchPrimitiveCreationReport,
    },
    sketch::{
        MeasuredDimensionOperands, SketchDimensionLimits, SketchDimensionalConstraintCreation,
        SketchEntityId, SketchPointInput, SketchPrimitiveCreation,
    },
};

struct DeletionScene {
    sketch: DocumentSketchId,
    circles: [SketchPrimitiveCreationReport; 2],
    dimensions: [SketchDimensionCreationReport; 2],
    marker: SketchEntityId,
}

#[path = "entity_delete/authoring.rs"]
mod authoring;
use authoring::{create_circle, create_radius, deletion_owner, dimension_bounds};

fn delete_first_circle(host: &mut Host, document: u64, scene: &DeletionScene) {
    let report = host
        .commands(document)
        .unwrap()
        .delete_sketch_entities(scene.sketch, &[scene.circles[0].entity])
        .unwrap();
    assert_eq!(report.entities, vec![scene.circles[0].entity]);
    assert_eq!(report.constraints[0].id, scene.dimensions[0].constraint);
    assert_eq!(report.constraints.len(), 1);
}

fn owner_snapshot(host: &Host, document: u64) -> &DocumentSnapshot {
    host.workspace().session(document).unwrap().snapshot()
}

fn reference_names(host: &Host, document: u64) -> DocumentSketchReferenceIndex {
    let guid = host.workspace().identity(document).unwrap().internal_name;
    DocumentSketchReferenceIndex::new(&owner_snapshot(host, document).sketches, guid).unwrap()
}

fn assert_deleted_scene(host: &Host, document: u64, scene: &DeletionScene) {
    let snapshot = owner_snapshot(host, document);
    let geometry = snapshot.sketches.get(scene.sketch).unwrap().geometry();
    assert_eq!(
        geometry.entity_ids(),
        &[scene.circles[1].entity, scene.marker]
    );
    assert_eq!(geometry.points().len(), 2);
    assert_eq!(geometry.dimensional_constraints().count(), 1);
    assert!(
        geometry
            .dimension_source(scene.dimensions[0].constraint)
            .is_err()
    );
    assert_retired_source(snapshot, &scene.dimensions[0]);
    assert_surviving_dimension(snapshot, scene);
    snapshot.require_resolved_sketch(scene.sketch).unwrap();
}

fn assert_surviving_dimension(snapshot: &DocumentSnapshot, scene: &DeletionScene) {
    let dimension = &scene.dimensions[1];
    let geometry = snapshot.sketches.get(scene.sketch).unwrap().geometry();
    let source = snapshot.parameters.get(dimension.parameter).unwrap();
    assert_eq!(source.annotations.comment, "authored dimension");
    assert!(source.annotations.exposed_as_property);
    assert_eq!(source.source, "width * 2");
    assert_dimension_metadata(geometry, dimension);
    let property = snapshot
        .properties
        .get(USER_DEFINED, &dimension.parameter_name)
        .unwrap();
    assert_eq!(property.from_parameter, dimension.parameter_name);
}

#[test]
fn both_file_owners_save_reopen_deleted_entities_and_retired_names() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_reopened_deletion(kind);
    }
}

fn verify_reopened_deletion(kind: DocumentType) {
    let (mut host, mut source, document, scene) = deletion_owner(kind);
    let names = reference_names(&host, document);
    let retired = names
        .entity_key(scene.sketch, scene.circles[0].entity)
        .unwrap();
    let retained = names
        .entity_key(scene.sketch, scene.circles[1].points[0])
        .unwrap();
    delete_first_circle(&mut host, document, &scene);
    assert!(host.workspace().needs_save(document).unwrap());
    let expected = owner_snapshot(&host, document).sketches.clone();
    let reopened = save_reopen_owner(&mut host, document, &mut source);
    assert_deleted_scene(&host, reopened, &scene);
    assert!(expected.same_recipe(&owner_snapshot(&host, reopened).sketches));
    assert_reopened_names(&host, reopened, &names, &retired, &retained);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    assert_next_reservations(&mut host, reopened, &scene);
}

fn assert_next_reservations(host: &mut Host, document: u64, scene: &DeletionScene) {
    let next = create_circle(
        host,
        document,
        scene.sketch,
        SketchPointInput::New([5.0, 6.0]),
    );
    assert_eq!(next.points, vec![SketchEntityId(5)]);
    assert_eq!(next.entity, SketchEntityId(6));
    let dimension = create_radius(host, document, scene.sketch, next.entity, false);
    assert_eq!(dimension.parameter, 4);
    assert_eq!(dimension.constraint.0, 5);
    assert_eq!(dimension.parameter_name, "d2");
}

#[test]
fn both_file_owners_copy_deleted_scene_into_a_separate_name_space() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_copied_deletion(kind);
    }
}

fn verify_copied_deletion(kind: DocumentType) {
    let (mut host, mut source, document, scene) = deletion_owner(kind);
    delete_first_circle(&mut host, document, &scene);
    let original = host.workspace().identity(document).unwrap().clone();
    let recipe = owner_snapshot(&host, document).sketches.clone();
    let copy = save_open_copy(&mut host, document, &mut source);
    assert_deleted_scene(&host, copy, &scene);
    assert!(recipe.same_recipe(&owner_snapshot(&host, copy).sketches));
    assert_copy_names(&host, document, copy, &scene);
    assert_eq!(host.workspace().identity(document).unwrap(), &original);
    assert!(host.workspace().needs_save(document).unwrap());
    assert_next_reservations(&mut host, copy, &scene);
    assert!(recipe.same_recipe(&owner_snapshot(&host, document).sketches));
}

fn assert_copy_names(host: &Host, document: u64, copy: u64, scene: &DeletionScene) {
    let original = reference_names(host, document);
    let copied = reference_names(host, copy);
    let entity = scene.circles[1].points[0];
    let original_key = original.entity_key(scene.sketch, entity).unwrap();
    let copied_key = copied.entity_key(scene.sketch, entity).unwrap();
    assert_ne!(original_key, copied_key);
    assert_eq!(copied.resolve(&original_key), None);
    assert_eq!(copied.resolve(&copied_key), original.resolve(&original_key));
}

#[test]
fn borrowed_reply_failure_preserves_saved_owner_and_file_contents() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_rejected_reply(kind);
    }
}

fn verify_rejected_reply(kind: DocumentType) {
    let (mut host, _, document, scene) = deletion_owner(kind);
    let before = owner_snapshot(&host, document).sketches.clone();
    let bytes = host.files().contents.borrow()[&path("entity-delete.obk")].clone();
    let generation = host.workspace().session(document).unwrap().generation();
    reject_deletion_reply(&mut host, document, &scene);
    assert!(before.same_recipe(&owner_snapshot(&host, document).sketches));
    assert_eq!(
        host.workspace().session(document).unwrap().generation(),
        generation
    );
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_eq!(
        host.files().contents.borrow()[&path("entity-delete.obk")],
        bytes
    );
    assert_restored_source(&host, document, &scene);
}

#[test]
fn failed_save_keeps_deleted_candidate_dirty_and_preserves_disk_recipe() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_failed_save(kind);
    }
}

fn verify_failed_save(kind: DocumentType) {
    let (host, mut source, _, scene) = deletion_owner(kind);
    let bytes = host.files().contents.borrow()[&path("entity-delete.obk")].clone();
    let mut host = refusing_saved_owner(&host);
    let document = open_saved_owner(&mut host, &mut source);
    let original = host.workspace().identity(document).unwrap().clone();
    delete_first_circle(&mut host, document, &scene);
    assert!(host.save(document, "alpha", &mut source).is_err());
    assert_deleted_scene(&host, document, &scene);
    assert!(host.workspace().needs_save(document).unwrap());
    assert_eq!(host.workspace().identity(document).unwrap(), &original);
    assert_eq!(
        host.files().contents.borrow()[&path("entity-delete.obk")],
        bytes
    );
}

fn refusing_saved_owner(host: &Host) -> Host {
    let files = WorkspaceFiles {
        contents: Rc::clone(&host.files().contents),
        fail_write: true,
        ..Default::default()
    };
    FileWorkspace::new(RecordingWorkspace::default(), files)
}

fn assert_retired_source(snapshot: &DocumentSnapshot, dimension: &SketchDimensionCreationReport) {
    assert!(snapshot.parameters.get(dimension.parameter).is_none());
    assert!(
        snapshot
            .properties
            .get(USER_DEFINED, &dimension.parameter_name)
            .is_none()
    );
}
fn assert_dimension_metadata(geometry: &Sketch2D, dimension: &SketchDimensionCreationReport) {
    assert_eq!(
        geometry.dimension_text_point(dimension.constraint).unwrap(),
        Some([-2.0, 3.0])
    );
    assert_eq!(
        geometry.dimension_limits(dimension.constraint).unwrap(),
        Some(dimension_bounds())
    );
    assert!(
        !geometry
            .dimension_is_reference(dimension.constraint)
            .unwrap()
    );
}
fn save_reopen_owner(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    host.save(document, "alpha", source).unwrap();
    host.close_discarding_changes(document).unwrap();
    open_saved_owner(host, source)
}
fn open_saved_owner(host: &mut Host, source: &mut SequenceGuids) -> u64 {
    open_file(host, "entity-delete.obk", source, OpenActivation::Activate)
}
fn save_open_copy(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    host.save_copy(
        document,
        &path("entity-delete-copy.obk"),
        None,
        "alpha",
        source,
    )
    .unwrap();
    open_file(
        host,
        "entity-delete-copy.obk",
        source,
        OpenActivation::Background,
    )
}
fn reject_deletion_reply(host: &mut Host, document: u64, scene: &DeletionScene) {
    let result = host
        .commands(document)
        .unwrap()
        .delete_sketch_entities_projected(scene.sketch, &[scene.circles[0].entity], |reply| {
            assert_eq!(reply.unwrap().constraints.len(), 1);
            Err::<(), _>("reply refused")
        });
    assert!(result.is_err());
}
fn assert_restored_source(host: &Host, document: u64, scene: &DeletionScene) {
    let snapshot = owner_snapshot(host, document);
    let dimension = &scene.dimensions[0];
    let parameter = snapshot.parameters.get(dimension.parameter).unwrap();
    assert!(parameter.annotations.exposed_as_property);
    let property = snapshot
        .properties
        .get(USER_DEFINED, &dimension.parameter_name);
    assert!(property.is_some());
    let geometry = snapshot.sketches.get(scene.sketch).unwrap().geometry();
    assert!(
        geometry
            .dimension_is_reference(dimension.constraint)
            .unwrap()
    );
}

fn assert_reopened_names(
    host: &Host,
    document: u64,
    before: &DocumentSketchReferenceIndex,
    retired: &str,
    retained: &str,
) {
    let current = reference_names(host, document);
    assert_eq!(current.resolve(retired), None);
    assert_eq!(current.resolve(retained), before.resolve(retained));
}
#[test]
fn all_entity_and_dimension_reservations_survive_empty_save_reopen_and_copy() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_empty_owner_reservations(kind);
    }
}
fn verify_empty_owner_reservations(kind: DocumentType) {
    let (mut host, mut source, document, scene) = deletion_owner(kind);
    delete_all_scene(&mut host, document, &scene);
    assert_empty_scene(&host, document, &scene);
    let reopened = save_reopen_owner(&mut host, document, &mut source);
    assert_empty_scene(&host, reopened, &scene);
    let copy = save_open_copy(&mut host, reopened, &mut source);
    assert_empty_scene(&host, copy, &scene);
    assert_next_reservations(&mut host, reopened, &scene);
    assert_next_reservations(&mut host, copy, &scene);
}
fn assert_empty_scene(host: &Host, document: u64, scene: &DeletionScene) {
    let snapshot = owner_snapshot(host, document);
    let geometry = snapshot.sketches.get(scene.sketch).unwrap().geometry();
    assert!(geometry.owned_entity_ids().is_empty());
    assert!(geometry.constraints().is_empty());
    for dimension in &scene.dimensions {
        assert_retired_source(snapshot, dimension);
    }
    snapshot.require_resolved_sketch(scene.sketch).unwrap();
}
#[test]
fn file_owned_undo_restores_reference_dimension_and_saved_baseline() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_file_owned_undo(kind);
    }
}
fn verify_file_owned_undo(kind: DocumentType) {
    let (mut host, _, document, scene) = deletion_owner(kind);
    let before = owner_snapshot(&host, document).sketches.clone();
    delete_first_circle(&mut host, document, &scene);
    host.commands(document).unwrap().undo().unwrap();
    assert!(before.same_recipe(&owner_snapshot(&host, document).sketches));
    assert_restored_source(&host, document, &scene);
    assert!(!host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    assert_deleted_scene(&host, document, &scene);
    assert!(host.workspace().needs_save(document).unwrap());
}

fn delete_all_scene(host: &mut Host, document: u64, scene: &DeletionScene) {
    host.commands(document)
        .unwrap()
        .delete_sketch_entities(
            scene.sketch,
            &[
                scene.circles[0].entity,
                scene.circles[1].entity,
                scene.marker,
            ],
        )
        .unwrap();
}
