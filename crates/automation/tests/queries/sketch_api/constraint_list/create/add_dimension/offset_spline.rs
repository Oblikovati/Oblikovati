// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::super::spline_requests::{fit_request, offset_request};
use super::*;
use oblikovati_model::sketch::SketchEntityId;

fn create_offset(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> u64 {
    let client = Client::new(FileHost::new(files, source, "test"));
    client.sketch().create(&Default::default()).unwrap();
    let fit = client
        .sketch()
        .add_entity(&fit_request("chord", false))
        .unwrap();
    client
        .sketch()
        .add_entity(&offset_request(fit.entity_id, "-5 mm"))
        .unwrap()
        .entity_id
}

fn offset_dimension(offset: u64, expression: &str, reference: bool) -> AddDimensionArgs {
    AddDimensionArgs {
        kind: "offsetSplineDim".into(),
        entities: Some(vec![offset]),
        expression: expression.into(),
        driven: reference,
        text_point: Some(vec![2.5, -7.25]),
        ..Default::default()
    }
}

#[test]
fn typed_offset_dimension_drives_signed_side_for_both_document_owners() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_offset_dimension(owner);
    }
}

fn verify_offset_dimension(owner: DocumentType) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    let offset = create_offset(&mut files, &mut source);
    let request = offset_dimension(offset, "width * 2", false);
    let reply = create_dimension(&mut files, &mut source, &request).unwrap();
    assert_eq!(reply.kind, "offsetSplineDim");
    assert_eq!(dimensions::dimensions(&files)[0].expression, "width * 2");
    assert_eq!(
        dimensions::dimensions(&files)[0].text_point,
        request.text_point
    );
    assert_offset_distance(&files, document, offset, -2.);
    change_source(&mut files, document);
    assert_offset_distance(&files, document, offset, -3.);
}

fn assert_offset_distance(files: &DiscoveryWorkspace, document: u64, id: u64, expected: f64) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let distance = geometry
        .offset_spline(SketchEntityId(id))
        .unwrap()
        .distance();
    assert!(
        (distance - expected).abs() < 1e-8,
        "{distance}: expected {expected}"
    );
}

#[test]
fn offset_dimension_history_reopen_and_copy_keep_side_source_and_name() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_offset_dimension_storage(owner);
    }
}

fn verify_offset_dimension_storage(owner: DocumentType) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    let offset = create_offset(&mut files, &mut source);
    create_dimension(
        &mut files,
        &mut source,
        &offset_dimension(offset, "1 cm", false),
    )
    .unwrap();
    let rows = dimensions::dimensions(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert!(dimensions::dimensions(&files).is_empty());
    assert_offset_distance(&files, document, offset, -0.5);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions::dimensions(&files), rows);
    assert_offset_distance(&files, document, offset, -1.);
    verify_offset_storage_copy(&mut files, document, &mut source, &rows);
}

fn verify_offset_storage_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    rows: &[oblikovati_api::wire::DimensionInfo],
) {
    let reopened = reopen_with_observer(files, document, source, Default::default());
    assert_eq!(dimensions::dimensions(files), rows);
    super::super::super::super::entities::activate_copy(
        files,
        reopened,
        source,
        Path::new("/fixtures/offset-dimension-copy.obk"),
    );
    assert_eq!(dimensions::dimensions(files), rows);
}

#[test]
fn offset_dimension_refuses_a_plain_spline_and_keeps_the_first_source_id() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let offset = create_offset(&mut files, &mut source);
    let parent = owned_geometry(&files, document).sketches()[0]
        .geometry()
        .offset_spline(SketchEntityId(offset))
        .unwrap()
        .parent();
    let before = owner_revision(&files, document);
    let cause = create_dimension(
        &mut files,
        &mut source,
        &offset_dimension(parent.0, "5 mm", false),
    )
    .unwrap_err();
    assert!(cause.to_string().contains("OffsetSpline"));
    assert_eq!(owner_revision(&files, document), before);
    let reply = create_dimension(
        &mut files,
        &mut source,
        &offset_dimension(offset, "5 mm", false),
    )
    .unwrap();
    assert_eq!(reply.parameter, "d0");
}

#[test]
fn offset_dimension_reference_edit_and_delete_use_the_canonical_commands() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let offset = create_offset(&mut files, &mut source);
    let reference = offset_dimension(offset, "1 cm", true);
    create_dimension(&mut files, &mut source, &reference).unwrap();
    assert!(dimensions::dimensions(&files)[0].driven);
    assert_offset_distance(&files, document, offset, -0.5);
    edit_offset_dimension(&mut files, &mut source, "2 cm", false);
    assert!(!dimensions::dimensions(&files)[0].driven);
    assert_offset_distance(&files, document, offset, -2.);
    delete_offset_dimension(&mut files, &mut source);
    assert!(dimensions::dimensions(&files).is_empty());
    assert_offset_distance(&files, document, offset, -2.);
}

fn edit_offset_dimension(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    value: &str,
    reference: bool,
) {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .edit(&oblikovati_api::wire::DriveDimensionArgs {
            expression: value.into(),
            set_driven: true,
            driven: reference,
            ..Default::default()
        })
        .unwrap();
}

fn delete_offset_dimension(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .delete(&oblikovati_api::wire::DeleteSketchDimensionArgs::default())
        .unwrap();
}

#[test]
fn offset_dimension_veto_and_abort_publish_no_relation() {
    verify_offset_dimension_veto();
    verify_offset_dimension_abort();
}

fn verify_offset_dimension_veto() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let offset = create_offset(&mut files, &mut source);
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
    let request = offset_dimension(offset, "1 cm", false);
    expect_api_decline(
        create_dimension(&mut files, &mut source, &request),
        "locked",
    );
    assert!(dimensions::dimensions(&files).is_empty());
    assert_offset_distance(&files, document, offset, -0.5);
    assert_eq!(owner_revision(&files, document).1, before.1);
}

fn verify_offset_dimension_abort() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let offset = create_offset(&mut files, &mut source);
    files
        .commands(document)
        .unwrap()
        .begin("Offset dimension group")
        .unwrap();
    create_dimension(
        &mut files,
        &mut source,
        &offset_dimension(offset, "1 cm", false),
    )
    .unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert!(dimensions::dimensions(&files).is_empty());
    assert_offset_distance(&files, document, offset, -0.5);
    let next = create_dimension(
        &mut files,
        &mut source,
        &offset_dimension(offset, "1 cm", false),
    )
    .unwrap();
    assert_eq!(next.parameter, "d1");
}
