// SPDX-License-Identifier: GPL-2.0-only
use super::planes::plane_rows;
use super::*;
use oblikovati_model::quantity::Dimension;
#[path = "offset/guards.rs"]
mod guards;
#[path = "offset/history.rs"]
mod history;

fn offset_request(source: &str, base: &str) -> CreateWorkPlaneArgs {
    CreateWorkPlaneArgs {
        kind: WORK_PLANE_OFFSET.0.into_owned(),
        offset: source.into(),
        refs: Some(vec![base.into()]),
        ..Default::default()
    }
}
pub(super) fn create_offset(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    expression: &str,
) -> CreateWorkPlaneResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .create(&offset_request(expression, WORK_REF_XY_PLANE))
        .unwrap()
}
fn revision(index: i64, expression: &str, base: &str) -> RedefineWorkPlaneArgs {
    RedefineWorkPlaneArgs {
        index,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: expression.into(),
        }]),
        repick: Some(vec![SlotRepick {
            slot: 0,
            ref_: base.into(),
        }]),
        ..Default::default()
    }
}
fn redefine(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    args: &RedefineWorkPlaneArgs,
) -> Result<RedefineWorkPlaneResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .redefine(args)
}
pub(super) fn scalar_id(files: &DiscoveryWorkspace, document: u64) -> u64 {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[0]
        .owned_scalars()[0]
        .parameter
}
pub(super) fn edit_driver(files: &mut DiscoveryWorkspace, document: u64, expression: &str) {
    files
        .commands(document)
        .unwrap()
        .run("Driver", |snapshot, _| {
            snapshot.parameters.set_expression(1, expression)
        })
        .unwrap();
}
fn assert_source(files: &DiscoveryWorkspace, document: u64, id: u64, expected: &str) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(id).unwrap().source, expected);
    assert_eq!(snapshot.parameters.get(id).unwrap().kind, MODEL_PARAM);
}

#[test]
fn created_offset_keeps_its_live_formula_and_owned_model_parameter() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    assert!(created.healthy);
    let id = scalar_id(&files, document);
    assert_source(&files, document, id, "width / 2");
    edit_driver(&mut files, document, "8 cm");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 4.0])
    );
    let parameters = &files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters;
    assert_eq!(parameters.drivers(id).unwrap(), [1]);
}

#[test]
fn redefine_preserves_scalar_identity_and_applies_reference_with_formula() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    let id = scalar_id(&files, document);
    let args = revision(3, "width * 2", WORK_REF_YZ_PLANE);
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert_eq!(reply.plane.origin, Some(vec![2.0, 0.0, 0.0]));
    assert_eq!(scalar_id(&files, document), id);
    assert_source(&files, document, id, "width * 2");
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 0.5])
    );
    files.commands(document).unwrap().redo().unwrap();
    assert_source(&files, document, id, "width * 2");
}

#[test]
fn bare_input_keeps_original_physical_units_after_preference_change_and_reopen() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "10");
    let id = scalar_id(&files, document);
    change_units(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let document =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_source(&files, document, id, "10");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 1.0])
    );
    let args = revision(3, "2", WORK_REF_XY_PLANE);
    let reply = redefine(&mut files, &mut source, &args).unwrap();
    assert_eq!(reply.plane.origin, Some(vec![0.0, 0.0, 5.08]));
}
fn change_units(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Units", |snapshot, _| {
            let mut units = snapshot.parameters.units().clone();
            units.set_preferred(Dimension::Length, "in").unwrap();
            snapshot.parameters.set_unit_preferences(units)
        })
        .unwrap();
}

#[test]
fn external_driver_failure_has_local_health_without_stale_plane_coordinates() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    edit_driver(&mut files, document, "missing");
    let failed = &plane_rows(&files, true)[3];
    assert!(!failed.healthy && failed.origin.is_none() && failed.normal.is_none());
    assert!(!failed.reason.is_empty());
    assert_source(&files, document, scalar_id(&files, document), "width / 2");
    edit_driver(&mut files, document, "6 cm");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 3.0])
    );
}

#[test]
fn legacy_unowned_driver_is_not_changed_by_first_scalar_edit() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    super::planes::insert_offset(&mut files, document);
    redefine(
        &mut files,
        &mut source,
        &revision(3, "width / 2", WORK_REF_XY_PLANE),
    )
    .unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(1).unwrap().source, "2 cm");
    assert_eq!(snapshot.parameters.get(1).unwrap().kind, USER_PARAM);
    assert_ne!(scalar_id(&files, document), 1);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 1.0])
    );
}

#[test]
fn rejected_repick_leaves_scalar_expression_and_owner_revision_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    let before = owner_revision(&files, document);
    for base in [WORK_REF_CENTER, "datum/v1/999", &created.ref_] {
        assert!(redefine(&mut files, &mut source, &revision(3, "8 cm", base)).is_err());
        assert_source(&files, document, scalar_id(&files, document), "width / 2");
        assert_eq!(owner_revision(&files, document), before);
    }
}

#[test]
fn invalid_inputs_do_not_allocate_features_or_scalars() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for expression in ["missing", "1 deg", "1e309 cm", "width +", "width * width"] {
        let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .work_planes()
            .create(&offset_request(expression, WORK_REF_XY_PLANE));
        assert!(result.is_err(), "{expression}");
    }
    assert_eq!(owner_revision(&files, document), before);
    let created = create_offset(&mut files, &mut source, "1 cm");
    assert_eq!(created.ref_, "datum/v1/1");
    assert_eq!(scalar_id(&files, document), 2);
}

#[test]
fn legacy_constant_is_promoted_to_a_live_owned_expression() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    super::planes::insert_offset(&mut files, document);
    make_constant(&mut files, document);
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 3.0])
    );
    let args = revision(3, "width / 2", WORK_REF_XY_PLANE);
    redefine(&mut files, &mut source, &args).unwrap();
    assert_source(&files, document, scalar_id(&files, document), "width / 2");
    edit_driver(&mut files, document, "8 cm");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 4.0])
    );
}
fn make_constant(files: &mut DiscoveryWorkspace, document: u64) {
    use oblikovati_model::feature::work_plane::{WorkPlaneOrigin, WorkPlaneReference};
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Constant(3.0),
    };
    files
        .commands(document)
        .unwrap()
        .run("Legacy constant", |snapshot, _| {
            snapshot
                .work_geometry
                .replace(FeatureId(1), definition, definition.reads())
        })
        .unwrap();
}

#[test]
fn offset_on_xz_origin_has_the_analytic_normal_and_distance() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let args = offset_request("30 mm", WORK_REF_XZ_PLANE);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_planes()
        .create(&args)
        .unwrap();
    let row = &plane_rows(&files, true)[3];
    assert!(row.healthy);
    assert_eq!(row.origin, Some(vec![0.0, -3.0, 0.0]));
    assert_eq!(row.normal, Some(vec![0.0, -1.0, 0.0]));
}
