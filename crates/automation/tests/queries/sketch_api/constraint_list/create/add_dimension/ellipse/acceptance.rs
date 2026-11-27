// SPDX-License-Identifier: GPL-2.0-only
use super::super::acceptance as common;
use super::*;

#[test]
fn ellipse_failed_formulas_keep_health_and_geometry_until_repaired() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for expression in ["missing * 4", "90 deg", "1 cm / 0"] {
            assert_failed_ellipse_formula(owner, expression);
        }
    }
}
fn assert_failed_ellipse_formula(owner: DocumentType, expression: &str) {
    let (mut files, document, mut source) = ellipse_owner(owner);
    let before = owned_geometry(&files, document).clone();
    let mut input = ellipse_request(false);
    input.expression = expression.into();
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(result.value, 3.0);
    common::assert_unavailable_driver(&files, document);
    assert_eq!(ellipse_radii(&files, document), [3.0, 1.0]);
    assert!(!before.same_recipe(owned_geometry(&files, document)));
    assert_failed_expression(&files, document, expression);
    repair_ellipse_formula(&mut files, document, &mut source, expression);
    assert_ellipse_frame(&files, document);
    assert!((dimensions::dimensions(&files)[0].value - 6.0).abs() < 1e-8);
}
fn assert_failed_expression(files: &DiscoveryWorkspace, document: u64, expression: &str) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(2).unwrap().source, expression);
    assert!(status_call(files, 0).is_err());
}
fn repair_ellipse_formula(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    expression: &str,
) {
    if expression == "missing * 4" {
        common::bind_missing(files, document);
        return;
    }
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .edit(&oblikovati_api::wire::DriveDimensionArgs {
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
}
fn ellipse_radii(files: &DiscoveryWorkspace, document: u64) -> [f64; 2] {
    owned_geometry(files, document)
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .ellipse(entity(12))
        .unwrap()
        .radii()
}

#[test]
fn ellipse_veto_and_aborted_groups_keep_geometry_and_retire_names() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_ellipse_abort(owner);
        assert_ellipse_veto(owner);
    }
}
fn assert_ellipse_abort(owner: DocumentType) {
    let (mut files, document, mut source) = ellipse_owner(owner);
    let before = owned_geometry(&files, document).clone();
    files
        .commands(document)
        .unwrap()
        .begin("Ellipse group")
        .unwrap();
    create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    assert!((ellipse_radii(&files, document)[0] - 4.0).abs() < 1e-8);
    files.commands(document).unwrap().abort().unwrap();
    common::assert_absent_pair(&files, document);
    assert!(before.same_recipe(owned_geometry(&files, document)));
    let next = create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    assert_eq!(next.parameter, "d1");
    common::assert_retired_ids(&files, document);
}
fn assert_ellipse_veto(owner: DocumentType) {
    let (mut files, document, mut source) = ellipse_owner(owner);
    let document = reopen_ellipse_veto(&mut files, document, &mut source);
    let before = owned_geometry(&files, document).clone();
    expect_api_decline(
        create_dimension(&mut files, &mut source, &ellipse_request(false)),
        "locked",
    );
    common::assert_absent_pair(&files, document);
    assert!(before.same_recipe(owned_geometry(&files, document)));
    assert!(owner_revision(&files, document).2);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    let next = create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    assert_eq!(next.parameter, "d1");
    common::assert_retired_ids(&files, document);
}
fn reopen_ellipse_veto(
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

#[test]
fn edited_ellipse_formula_frame_and_annotation_survive_reopen_and_copy() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for live in [false, true] {
            assert_saved_ellipse_edit(owner, live);
        }
    }
}
fn assert_saved_ellipse_edit(owner: DocumentType, live: bool) {
    let (mut files, document, mut source) = ellipse_owner(owner);
    create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    edit_ellipse_source(&mut files, &mut source, live);
    let rows = dimensions::dimensions(&files);
    assert!((rows[0].value - 5.0).abs() < 1e-8);
    assert_eq!(
        rows[0].expression,
        if live { "width * 5" } else { "5 length" }
    );
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions::dimensions(&files), rows);
    assert_saved_ellipse_copy(&mut files, document, &mut source, live);
    assert_eq!(dimensions::dimensions(&files), rows);
}
fn edit_ellipse_source(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, live: bool) {
    let client = Client::new(FileHost::new(files, source, "test"));
    if live {
        client
            .parameters()
            .set(&oblikovati_api::wire::ParameterSetArgs {
                name: "d0".into(),
                expression: "width * 5".into(),
                ..Default::default()
            })
            .unwrap();
        return;
    }
    // Go driveDimension resolves the input once; only parameters.set retains a live formula.
    let input = oblikovati_api::wire::DriveDimensionArgs {
        expression: "width * 5".into(),
        ..Default::default()
    };
    client.dimension().edit(&input).unwrap();
}

#[test]
fn ellipse_failed_wire_measurement_retires_ids_without_adopting_geometry() {
    let (mut files, document, mut source) = common::scaled_operands(1e100);
    files
        .commands(document)
        .unwrap()
        .run("Oversized ellipse", install_overflow_ellipse)
        .unwrap();
    let before = owned_geometry(&files, document).clone();
    expect_api_decline(
        create_dimension(&mut files, &mut source, &ellipse_request(true)),
        "finite centimetre",
    );
    common::assert_absent_pair(&files, document);
    assert!(before.same_recipe(owned_geometry(&files, document)));
    let next =
        create_dimension(&mut files, &mut source, &dimension_request("radius", &[9])).unwrap();
    assert_eq!(next.parameter, "d1");
    assert_eq!(next.index, 0);
    common::assert_retired_ids(&files, document);
}
fn install_overflow_ellipse(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let id = ids
        .allocate_sketch_entity()
        .map_err(|cause| cause.to_string())?;
    assert_eq!(id, entity(12));
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let ellipse = SketchEllipse2D::new(id, entity(1), [3.0, 4.0], [1e220, 1.0]).unwrap();
    geometry
        .add_ellipse(ellipse)
        .map_err(|cause| cause.to_string())?;
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .map_err(|cause| cause.to_string())
}
fn assert_saved_ellipse_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    live: bool,
) {
    assert_ellipse_frame(files, document);
    super::super::super::super::super::entities::activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/edited-ellipse-copy.obk"),
    );
    let document = files.workspace().active_document().unwrap();
    assert_ellipse_frame(files, document);
    change_source(files, document);
    let expected = if live { 7.5 } else { 5.0 };
    assert!((dimensions::dimensions(files)[0].value - expected).abs() < 1e-8);
    files.commands(document).unwrap().undo().unwrap();
}
