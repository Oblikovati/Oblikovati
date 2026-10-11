// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Caller;
use oblikovati_model::parameter::ParameterHealth;

pub(super) fn scaled_operands(scale: f64) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) =
        super::super::super::super::entities::configured_scaled(scale);
    files
        .commands(document)
        .unwrap()
        .run("Scaled constructor operands", install_operands)
        .unwrap();
    (files, document, source)
}
#[test]
fn creation_converts_lengths_annotations_and_angles_at_three_working_scales() {
    for scale in [1e-6, 1.0, 1e6] {
        assert_scaled_length(scale);
        assert_scaled_angle(scale);
    }
}
fn assert_scaled_length(scale: f64) {
    let (mut files, document, mut source) = scaled_operands(scale);
    let input = dimension_request("radius", &[9]);
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(result.value, scale);
    assert_working_annotation(&files, document, scale);
    assert_eq!(
        dimensions::dimensions(&files)[0].text_point,
        input.text_point
    );
}
fn assert_working_annotation(files: &DiscoveryWorkspace, document: u64, scale: f64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let id = geometry.dimensional_constraints().next().unwrap().0.id;
    assert_eq!(
        geometry.dimension_text_point(id).unwrap(),
        Some([2.5 / scale, -7.25 / scale])
    );
}

fn assert_scaled_angle(scale: f64) {
    let (mut files, _, mut source) = scaled_operands(scale);
    let mut input = dimension_request("threePointAngle", &[1, 2, 3]);
    input.expression = "90 deg".into();
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(result.value, std::f64::consts::FRAC_PI_2);
    assert_eq!(
        dimensions::dimensions(&files)[0].expression,
        input.expression
    );
}
#[test]
fn created_formulas_annotations_and_permanent_names_survive_reopen_and_copy() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_saved_creation(owner);
    }
}
fn assert_saved_creation(owner: DocumentType) {
    let (mut files, document, mut source) = fresh(owner);
    let mut input = dimension_request("radius", &[9]);
    input.expression = "width * 2".into();
    create_dimension(&mut files, &mut source, &input).unwrap();
    let rows = dimensions::dimensions(&files);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions::dimensions(&files), rows);
    super::super::super::super::entities::activate_copy(
        &mut files,
        reopened,
        &mut source,
        Path::new("/fixtures/created-dimension-copy.obk"),
    );
    assert_eq!(dimensions::dimensions(&files), rows);
    let second = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(second.parameter, "d1");
    assert_eq!(second.index, 1);
}
#[test]
fn veto_publishes_no_owned_pair_but_keeps_reservations_dirty() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
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
    let input = dimension_request("radius", &[9]);
    expect_api_decline(create_dimension(&mut files, &mut source, &input), "locked");
    assert_absent_pair(&files, document);
    let after = owner_revision(&files, document);
    assert_eq!(after.1, before.1);
    // A veto must keep consumed identities, so saving is needed even without an adopted recipe.
    assert!(after.2);
}
pub(super) fn assert_absent_pair(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.parameters.get(2).is_none());
    assert!(dimensions::dimensions(files).is_empty());
}
#[test]
fn aborted_creation_consumes_permanent_names_without_publishing_the_pair() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    files
        .commands(document)
        .unwrap()
        .begin("Create group")
        .unwrap();
    create_dimension(&mut files, &mut source, &dimension_request("radius", &[9])).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_absent_pair(&files, document);
    let next =
        create_dimension(&mut files, &mut source, &dimension_request("radius", &[9])).unwrap();
    assert_eq!(next.parameter, "d1");
    assert_eq!(next.index, 0);
}
#[test]
fn unavailable_driving_source_keeps_explicit_health_and_repairs_after_binding() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let mut input = dimension_request("radius", &[9]);
    input.driven = false;
    input.expression = "missing * 2".into();
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(result.value, 1.0);
    assert_unavailable_driver(&files, document);
    bind_missing(&mut files, document);
    let row = &dimensions::dimensions(&files)[0];
    assert_eq!(row.expression, input.expression);
    assert!((row.value - 3.0).abs() < 1e-8);
}
pub(super) fn assert_unavailable_driver(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(!matches!(
        snapshot.parameters.get(2).unwrap().health,
        ParameterHealth::Healthy
    ));
    assert!(
        Client::new(FileQueryHost::new(files))
            .sketch()
            .dimensions(&Default::default())
            .is_err()
    );
}

pub(super) fn bind_missing(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Bind missing source", |snapshot, ids| {
            let parameter = ids.allocate().map_err(|cause| cause.to_string())?;
            snapshot
                .parameters
                .insert_expression(parameter, "missing", USER_PARAM, "1.5 cm")
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
#[test]
fn bad_host_indices_and_json_decline_before_creation() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        let mut input = dimension_request("radius", &[9]);
        input.sketch_index = index;
        expect_api_decline(
            create_dimension(&mut files, &mut source, &input),
            "ordered index",
        );
    }
    assert_bad_payloads(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
}
fn assert_bad_payloads(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    for payload in [
        b"[]".as_slice(),
        br#"{"entities":"bad"}"#,
        br#"{"driven":1}"#,
    ] {
        expect_api_decline(
            FileHost::new(files, source, "test").call(METHOD_SKETCH_ADD_DIMENSION, payload),
            METHOD_SKETCH_ADD_DIMENSION,
        );
    }
}
#[test]
fn drawing_owner_cannot_create_dimensions() {
    let (mut drawing, _, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(
        create_dimension(
            &mut drawing,
            &mut source,
            &dimension_request("radius", &[9]),
        ),
        "part or assembly",
    );
}

#[test]
fn rejected_wire_measurement_consumes_names_without_publishing_a_partial_recipe() {
    let (mut files, document, mut source) = scaled_operands(1e100);
    install_large_radius(&mut files, document);
    let input = dimension_request("radius", &[9]);
    expect_api_decline(
        create_dimension(&mut files, &mut source, &input),
        "finite centimetre",
    );
    assert_absent_pair(&files, document);
    let result =
        create_dimension(&mut files, &mut source, &dimension_request("radius", &[10])).unwrap();
    assert_eq!(result.parameter, "d1");
    assert_eq!(result.index, 0);
    assert_eq!(result.value, 1e100);
    assert_retired_ids(&files, document);
}
#[test]
fn incompatible_and_arithmetic_sources_remain_authored_with_failure_health() {
    for expression in ["90 deg", "1 cm / 0"] {
        let (mut files, document, mut source) = fresh(DOCUMENT_PART);
        let mut input = dimension_request("radius", &[9]);
        input.expression = expression.into();
        input.driven = false;
        let result = create_dimension(&mut files, &mut source, &input).unwrap();
        assert_eq!(result.value, 1.0);
        assert_unavailable_driver(&files, document);
    }
}

fn install_large_radius(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Finite geometry beyond wire units", |snapshot, _| {
            let mut geometry = Sketch2D::default();
            geometry.add_point(entity(1), [0.0; 2]).unwrap();
            geometry.add_circle(entity(9), entity(1), 1e220).unwrap();
            geometry.add_circle(entity(10), entity(1), 1.0).unwrap();
            snapshot
                .sketches
                .replace_geometry(DocumentSketchId(1), geometry)
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
pub(super) fn assert_retired_ids(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let relation = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .dimensional_constraints()
        .next()
        .unwrap()
        .0;
    assert_eq!(relation.id, SketchConstraintId(2));
    assert!(snapshot.parameters.get(3).is_some());
}
