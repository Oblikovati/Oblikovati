// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::parameter::ParameterHealth;

#[test]
fn minor_failed_formulas_preserve_geometry_and_repair_to_zero_dof() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for expression in ["missing / 2", "90 deg", "1 cm / 0"] {
            assert_failed_minor(owner, expression);
        }
    }
}
fn assert_failed_minor(owner: DocumentType, expression: &str) {
    let (mut files, document, mut source, ellipse) = ellipse_owner(owner);
    let result = minor(&mut files, &mut source, ellipse, expression, false).unwrap();
    assert_eq!(result.value, 1.0);
    assert_minor_geometry(&files, document, ellipse, [4.0, 1.0]);
    assert_minor_source_failure(&files, document, expression);
    repair_minor_source(&mut files, &mut source);
    assert_restored_minor(&files, document, ellipse);
    assert_eq!(dimension_rows(&files)[1].expression, "width / 2");
}
fn assert_minor_source_failure(files: &DiscoveryWorkspace, document: u64, expression: &str) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let parameter = snapshot
        .parameters
        .get(snapshot.parameters.find("d1").unwrap())
        .unwrap();
    assert_eq!(parameter.source, expression);
    assert!(!matches!(parameter.health, ParameterHealth::Healthy));
    assert!(status_call(files, 0).is_err());
    assert!(
        Client::new(FileQueryHost::new(files))
            .sketch()
            .dimensions(&Default::default())
            .is_err()
    );
}
fn repair_minor_source(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "d1".into(),
            expression: "width / 2".into(),
            ..Default::default()
        })
        .unwrap();
}
#[test]
fn minor_aborted_creation_keeps_major_dimension_and_retires_minor_identity() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_minor_abort(owner);
    }
}
fn assert_minor_abort(owner: DocumentType) {
    let (mut files, document, mut source, ellipse) = ellipse_owner(owner);
    let before = owned_geometry(&files, document).clone();
    let rows = dimension_rows(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Minor radius group")
        .unwrap();
    minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    assert_restored_minor(&files, document, ellipse);
    files.commands(document).unwrap().abort().unwrap();
    assert_minor_rollback(&files, document, &before, &rows);
    let next = minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    assert_eq!(next.parameter, "d2");
    assert_eq!(next.index, 1);
    assert_restored_minor(&files, document, ellipse);
}
fn assert_minor_rollback(
    files: &DiscoveryWorkspace,
    document: u64,
    before: &oblikovati_model::document_sketch::DocumentSketches,
    rows: &[DimensionInfo],
) {
    assert!(before.same_recipe(owned_geometry(files, document)));
    assert_eq!(dimension_rows(files), rows);
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.parameters.find("d1").is_none());
}
#[test]
fn minor_commit_veto_keeps_major_dimension_and_saves_retired_minor_name() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_minor_veto(owner);
    }
}
fn assert_minor_veto(owner: DocumentType) {
    let (mut files, document, mut source, ellipse) = ellipse_owner(owner);
    let document = reopen_minor_veto(&mut files, document, &mut source);
    let before = owned_geometry(&files, document).clone();
    let rows = dimension_rows(&files);
    expect_api_decline(
        minor(&mut files, &mut source, ellipse, "width / 2", false),
        "locked",
    );
    assert_minor_rollback(&files, document, &before, &rows);
    assert!(owner_revision(&files, document).2);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    let next = minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    assert_eq!(next.parameter, "d2");
    assert_restored_minor(&files, document, ellipse);
}
fn reopen_minor_veto(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
) -> u64 {
    reopen_with_observer(
        files,
        document,
        source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    )
}
