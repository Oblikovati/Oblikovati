// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{AddDimensionArgs, AddDimensionResult, DimensionInfo};
fn ellipse_owner(owner: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids, u64) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    create_empty_sketch(&mut files, &mut source);
    let mut request = input();
    request.construction = false;
    let created = add_call(&mut files, &mut source, &request).unwrap();
    constrain_frame(&mut files, &mut source, created.entity_id);
    dimension(
        &mut files,
        &mut source,
        created.entity_id,
        DIM_CONSTRAINT_ELLIPSE_RADIUS,
        "width * 4",
        false,
    )
    .unwrap();
    (files, document, source, created.entity_id)
}
fn constrain_frame(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, ellipse: u64) {
    for kind in [GEO_CONSTRAINT_GROUND, GEO_CONSTRAINT_HORIZONTAL] {
        Client::new(FileHost::new(files, source, "test"))
            .constrain()
            .add(&oblikovati_api::wire::AddConstraintArgs {
                kind: kind.0.to_string(),
                entities: Some(vec![ellipse]),
                ..Default::default()
            })
            .unwrap();
    }
}
fn dimension(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    ellipse: u64,
    kind: DimensionConstraintKind,
    expression: &str,
    reference: bool,
) -> Result<AddDimensionResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .add(&AddDimensionArgs {
            kind: kind.0.to_string(),
            entities: Some(vec![ellipse]),
            expression: expression.into(),
            driven: reference,
            text_point: Some(vec![2.5, -7.25]),
            ..Default::default()
        })
}
fn minor(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    ellipse: u64,
    expression: &str,
    reference: bool,
) -> Result<AddDimensionResult, oblikovati_api::Error> {
    dimension(
        files,
        source,
        ellipse,
        DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS,
        expression,
        reference,
    )
}
fn dimension_rows(files: &DiscoveryWorkspace) -> Vec<DimensionInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap()
}
#[test]
fn ellipse_minor_dimension_completes_zero_dof_and_keeps_both_sources_live() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_live_minor(owner);
    }
}
fn assert_live_minor(owner: DocumentType) {
    let (mut files, document, mut source, ellipse) = ellipse_owner(owner);
    let result = minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    assert_eq!(result.kind, "ellipseMinorRadius");
    assert_eq!(result.parameter, "d1");
    assert_eq!(result.dof, 0);
    assert!((result.value - 0.5).abs() < 1e-8);
    assert_minor_geometry(&files, document, ellipse, [4.0, 0.5]);
    change_ellipse_width(&mut files, &mut source);
    assert_minor_geometry(&files, document, ellipse, [6.0, 0.75]);
    assert_eq!(status_call(&files, 0).unwrap().dof, 0);
    let rows = dimension_rows(&files);
    assert_eq!(rows[0].expression, "width * 4");
    assert_eq!(rows[1].expression, "width / 2");
    assert_eq!(rows[1].text_point, Some(vec![2.5, -7.25]));
}
fn change_ellipse_width(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "width".into(),
            expression: "1.5 cm".into(),
            ..Default::default()
        })
        .unwrap();
}
fn assert_minor_geometry(
    files: &DiscoveryWorkspace,
    document: u64,
    ellipse: u64,
    expected: [f64; 2],
) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let value = geometry.ellipse(SketchEntityId(ellipse)).unwrap();
    for (actual, expected) in value.radii().into_iter().zip(expected) {
        assert!((actual - expected).abs() < 1e-8);
    }
    assert_eq!(geometry.point(value.center()).unwrap().position, [1.0, 2.0]);
    assert!(value.major_axis()[1].abs() < 1e-8);
    assert!(!geometry.is_construction(value.id()).unwrap());
}
#[test]
fn ellipse_minor_dimension_history_and_storage_keep_kind_formula_and_identity() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_minor_history(owner);
    }
}
fn assert_minor_history(owner: DocumentType) {
    let (mut files, document, mut source, ellipse) = ellipse_owner(owner);
    minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    let expected = dimension_rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(dimension_rows(&files).len(), 1);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimension_rows(&files), expected);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/minor-dimension-copy.obk"),
    );
    assert_eq!(dimension_rows(&files), expected);
    let copy = files.workspace().active_document().unwrap();
    assert_restored_minor(&files, copy, ellipse);
    delete_and_recreate_minor(&mut files, &mut source, ellipse);
}
fn assert_restored_minor(files: &DiscoveryWorkspace, document: u64, ellipse: u64) {
    assert_minor_geometry(files, document, ellipse, [4.0, 0.5]);
    assert_eq!(status_call(files, 0).unwrap().dof, 0);
}
fn delete_and_recreate_minor(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    ellipse: u64,
) {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .delete(&oblikovati_api::wire::DeleteSketchDimensionArgs {
            dimension_index: 1,
            ..Default::default()
        })
        .unwrap();
    let result = minor(files, source, ellipse, "width / 2", false).unwrap();
    assert_eq!(result.parameter, "d2");
    assert_eq!(result.index, 1);
    assert_eq!(result.dof, 0);
}
#[test]
fn ellipse_minor_reference_measures_current_geometry_without_constraining_it() {
    let (mut files, document, mut source, ellipse) = ellipse_owner(DOCUMENT_PART);
    let result = minor(&mut files, &mut source, ellipse, "width / 2", true).unwrap();
    assert_eq!(result.value, 1.0);
    assert_eq!(result.dof, 1);
    assert_minor_geometry(&files, document, ellipse, [4.0, 1.0]);
    assert!(dimension_rows(&files)[1].driven);
}
#[test]
fn ellipse_minor_crossings_and_nonpositive_targets_do_not_publish_partial_recipes() {
    let (mut files, document, mut source, ellipse) = ellipse_owner(DOCUMENT_PART);
    let before = owned_geometry(&files, document).clone();
    for expression in ["0 cm", "-1 cm", "5 cm"] {
        assert!(minor(&mut files, &mut source, ellipse, expression, false).is_err());
        assert!(before.same_evaluated_state(owned_geometry(&files, document)));
        assert_eq!(dimension_rows(&files).len(), 1);
    }
    let next = minor(&mut files, &mut source, ellipse, "width / 2", false).unwrap();
    assert_eq!(next.parameter, "d4");
}
