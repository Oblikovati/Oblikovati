// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature,
    work_plane::{WorkPlaneOrigin, WorkPlaneReference},
};
fn fixed_request(construction: bool, visible: bool) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_FIXED.0.into_owned(),
        origin: Some(vec![0.0, 0.0, 5.0]),
        x_axis: Some(vec![2.0, 0.0, 0.0]),
        y_axis: Some(vec![0.0, 3.0, 0.0]),
        construction,
        visible: Some(std::boxed::Box::new(visible)),
        ..Default::default()
    }
}
fn fixed(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    construction: bool,
    visible: bool,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&fixed_request(construction, visible))
        .unwrap()
}
fn plane_rows(files: &DiscoveryWorkspace, include: bool) -> Vec<WorkPlaneInfo> {
    Client::new(FileQueryHost::new(files))
        .work_planes()
        .list_with_options(&ListWorkPlanesArgs {
            include_construction: include,
            ..Default::default()
        })
        .unwrap()
        .planes
        .unwrap()
}
#[test]
fn default_plane_query_lists_origin_frames_without_edits() {
    let (files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    let rows = Client::new(FileQueryHost::new(&files))
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    assert_eq!(
        rows.iter().map(|row| row.ref_.as_str()).collect::<Vec<_>>(),
        [WORK_REF_XY_PLANE, WORK_REF_XZ_PLANE, WORK_REF_YZ_PLANE]
    );
    assert!(
        rows.iter()
            .all(|row| row.is_origin && row.grounded && row.healthy && !row.visible)
    );
    assert_eq!(rows[0].normal, Some(vec![0.0, 0.0, 1.0]));
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn fixed_planes_share_ids_and_preserve_full_indices_and_visibility() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let hidden = fixed(&mut files, &mut source, true, false);
    let point = create_point(&mut files, &mut source, [1.0; 3], false);
    let shown = fixed(&mut files, &mut source, false, true);
    assert_eq!((hidden.index, shown.index), (3, 4));
    assert_ne!(point.ref_, hidden.ref_);
    assert_eq!(plane_rows(&files, false)[3].ref_, shown.ref_);
    let rows = plane_rows(&files, true);
    assert!(!rows[3].visible && rows[3].construction);
    assert_eq!(rows[3].origin, Some(vec![0.0, 0.0, 5.0]));
    assert_eq!(rows[3].normal, Some(vec![0.0, 0.0, 1.0]));
    assert_eq!(rows[3].kind, WORK_PLANE_FIXED.0);
}
fn insert_offset(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Offset fixture", |snapshot, ids| -> Result<(), String> {
            snapshot
                .parameters
                .set_expression(1, "2 cm")
                .map_err(|cause| cause.to_string())?;
            let id = ids.allocate_feature().map_err(|cause| cause.to_string())?;
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(id, "Offset", offset_definition()))
                .map(|_| ())
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
fn offset_definition() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(1),
    }
}
#[test]
fn offset_lists_use_current_parameters_preferred_units_and_local_health() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    insert_offset(&mut files, document);
    assert_offset_inputs(&files, 2.0, 20.0);
    change_offset(&mut files, document);
    assert_offset_inputs(&files, 4.0, 40.0);
    files
        .commands(document)
        .unwrap()
        .run("Suppress offset", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(FeatureId(1), true)
        })
        .unwrap();
    let row = &plane_rows(&files, false)[3];
    assert!(!row.healthy && row.origin.is_none() && row.normal.is_none());
    assert!(!row.reason.is_empty());
}
fn assert_offset_inputs(files: &DiscoveryWorkspace, height: f64, displayed: f64) {
    let row = &plane_rows(files, false)[3];
    assert_eq!(row.origin, Some(vec![0.0, 0.0, height]));
    assert_eq!(row.kind, WORK_PLANE_OFFSET.0);
    assert_eq!(row.scalars.as_ref().unwrap()[0].value, displayed);
    assert_eq!(row.scalars.as_ref().unwrap()[0].unit, "mm");
    assert_eq!(row.slots.as_ref().unwrap()[0].label, "Base plane");
}
#[test]
fn fixed_plane_identity_and_hidden_metadata_survive_undo_and_storage() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = fixed(&mut files, &mut source, true, false);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(plane_rows(&files, true).len(), 3);
    files.commands(document).unwrap().redo().unwrap();
    files.save(document, "test", &mut source).unwrap();
    reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    let restored = &plane_rows(&files, true)[3];
    assert_eq!(restored.ref_, created.ref_);
    assert!(!restored.visible && restored.construction);
}
#[test]
fn malformed_frames_and_unported_constructors_do_not_mutate_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for request in invalid_frames() {
        let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .work_planes()
            .create(&request);
        assert!(result.is_err());
    }
    assert_eq!(owner_revision(&files, document), before);
}
fn invalid_frames() -> Vec<CreateWorkPlaneArgs> {
    let mut zero = fixed_request(false, true);
    zero.x_axis = Some(vec![0.0; 3]);
    let mut parallel = fixed_request(false, true);
    parallel.x_axis = parallel.y_axis.clone();
    let mut short = fixed_request(false, true);
    short.origin = Some(vec![0.0; 2]);
    let offset = CreateWorkPlaneArgs {
        kind: WORK_PLANE_OFFSET.0.into_owned(),
        ..Default::default()
    };
    vec![zero, parallel, short, offset]
}

fn change_offset(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Change offset", |snapshot, _| {
            snapshot.parameters.set_expression(1, "4 cm")
        })
        .unwrap();
}
#[test]
fn plane_scope_queries_decline_assembly_and_occurrence_contexts() {
    let (files, _, _) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    assert_datum_decline(
        Client::new(FileQueryHost::new(&files)).work_planes().list(),
        "expected",
    );
    let (files, _, _) = configured_files(DOCUMENT_PART, Default::default());
    let request = ListWorkPlanesArgs {
        occurrence: Some(vec!["part".into()]),
        ..Default::default()
    };
    assert_datum_decline(
        Client::new(FileQueryHost::new(&files))
            .work_planes()
            .list_with_options(&request),
        "local part scope",
    );
}
#[test]
fn aborted_plane_creation_retires_its_key_before_the_next_creation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .begin("Plane group")
        .unwrap();
    let retired = fixed(&mut files, &mut source, false, true);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(plane_rows(&files, true).len(), 3);
    let kept = fixed(&mut files, &mut source, false, true);
    assert_ne!(kept.ref_, retired.ref_);
    assert_eq!(kept.index, retired.index);
}
#[test]
fn plane_creation_commit_veto_keeps_the_document_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let request = fixed_request(true, false);
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .create(&request);
    assert_datum_decline(result, "locked");
    assert_eq!(plane_rows(&files, true).len(), 3);
    assert_eq!(owner_revision(&files, document).1, before.1);
}
