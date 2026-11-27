// SPDX-License-Identifier: GPL-2.0-only
//! Parameter expressions on entity inputs retain a driving dimension.
use super::*;
use oblikovati_api::wire::ParameterSetArgs;

fn add_associative(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddSketchEntityArgs,
) -> Result<AddSketchEntityResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .add_associative_entity(input)
}

type EntityCreator = fn(
    &mut DiscoveryWorkspace,
    &mut QueryGuids,
    &AddSketchEntityArgs,
) -> Result<AddSketchEntityResult, oblikovati_api::Error>;

#[test]
fn circle_radius_expression_survives_parameter_edits_and_history() {
    for create in [add_call as EntityCreator, add_associative] {
        verify_driving_circle_round_trip(create);
    }
}

fn verify_driving_circle_round_trip(create: EntityCreator) {
    let (mut files, document, mut source) = fresh();
    let mut input = circle_input();
    input.radius = "width / 2".into();
    let circle = create(&mut files, &mut source, &input).unwrap();
    assert_driving_circle_dimension(&files, document);
    assert_radius(&files, circle.entity_id, 0.5);
    set_width(&mut files, &mut source, "2 cm");
    assert_radius(&files, circle.entity_id, 1.0);
    files.commands(document).unwrap().undo().unwrap();
    assert_radius(&files, circle.entity_id, 0.5);
    files.commands(document).unwrap().redo().unwrap();
    assert_radius(&files, circle.entity_id, 1.0);
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_radius(&files, circle.entity_id, 1.0);
    set_width(&mut files, &mut source, "3 cm");
    assert_radius(&files, circle.entity_id, 1.5);
}

fn assert_driving_circle_dimension(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let dimension = snapshot
        .parameters
        .find("d0")
        .expect("expected a radius dimension");
    assert_eq!(
        snapshot.parameters.get(dimension).unwrap().source,
        "width / 2"
    );
}

fn assert_radius(files: &DiscoveryWorkspace, entity: u64, expected: f64) {
    let radius = row(files, entity).radius;
    assert!(
        (radius - expected).abs() < 1e-7,
        "radius {radius}: expected {expected}"
    );
}

#[test]
fn invalid_and_unavailable_bindings_leave_geometry_and_history_unchanged() {
    let (mut files, document, mut source) = fresh();
    let before = owner_revision(&files, document);
    let geometry = rows(&files);
    for expression in ["missing", "width + 2 deg", "width / 0"] {
        let mut input = circle_input();
        input.radius = expression.into();
        assert!(add_associative(&mut files, &mut source, &input).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_eq!(rows(&files), geometry);
    }
}

#[test]
fn signed_offset_spline_expression_crosses_zero_and_survives_history_and_reload() {
    let (mut files, document, mut source) = fresh();
    let fit = super::super::spline_requests::fit_request("chord", false);
    let parent = add_call(&mut files, &mut source, &fit).unwrap();
    let request = super::super::spline_requests::offset_request(parent.entity_id, "width");
    let result = add_call(&mut files, &mut source, &request).unwrap();
    assert_offset(&files, document, result.entity_id, 1.0);
    for (expression, expected) in [("0 cm", 0.0), ("-2 cm", -2.0)] {
        set_width(&mut files, &mut source, expression);
        assert_offset(&files, document, result.entity_id, expected);
    }
    files.commands(document).unwrap().undo().unwrap();
    assert_offset(&files, document, result.entity_id, 0.0);
    files.commands(document).unwrap().redo().unwrap();
    assert_offset(&files, document, result.entity_id, -2.0);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_offset(&files, reopened, result.entity_id, -2.0);
    set_width(&mut files, &mut source, "3 cm");
    assert_offset(&files, reopened, result.entity_id, 3.0);
}
#[test]
fn signed_offset_parameter_edit_veto_keeps_geometry_and_history() {
    let (mut files, document, mut source) = fresh();
    let fit = super::super::spline_requests::fit_request("chord", false);
    let parent = add_call(&mut files, &mut source, &fit).unwrap();
    let request = super::super::spline_requests::offset_request(parent.entity_id, "width");
    let result = add_call(&mut files, &mut source, &request).unwrap();
    let reopened = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let before = owner_revision(&files, reopened);
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "-2 cm".into(),
            ..Default::default()
        })
        .unwrap_err();
    assert!(error.to_string().contains("locked"));
    assert_offset(&files, reopened, result.entity_id, 1.0);
    let after = owner_revision(&files, reopened);
    assert_eq!((after.1, after.2), (before.1, before.2));
    let params = owned_parameters(&files, reopened);
    let width = params.find("width").unwrap();
    assert_eq!(params.get(width).unwrap().source, "1 cm");
}
#[test]
fn signed_offset_refuses_invalid_sources_without_geometry_or_history_changes() {
    let (mut files, document, mut source) = fresh();
    let fit = super::super::spline_requests::fit_request("chord", false);
    let parent = add_call(&mut files, &mut source, &fit).unwrap();
    let before = owner_revision(&files, document);
    let geometry = rows(&files);
    for expression in ["missing", "width + 2 deg", "width / 0"] {
        let request = super::super::spline_requests::offset_request(parent.entity_id, expression);
        assert!(add_call(&mut files, &mut source, &request).is_err());
        assert_eq!(owner_revision(&files, document), before);
        assert_eq!(rows(&files), geometry);
    }
}
fn assert_offset(files: &DiscoveryWorkspace, document: u64, entity: u64, expected: f64) {
    let sketch = &owned_geometry(files, document).sketches()[0];
    let offset = sketch
        .geometry()
        .offset_spline(SketchEntityId(entity))
        .unwrap();
    assert!(
        (offset.distance() - expected).abs() < 1e-7,
        "distance {} expected {expected}",
        offset.distance()
    );
}

#[test]
fn ellipse_major_and_minor_expressions_are_distinct_and_persistent() {
    let (mut files, document, mut source) = fresh();
    let mut input = literal_input("ellipse", &[[1., 2.]]);
    input.axis = Some(vec![1., 0.]);
    input.major_radius = "width * 3".into();
    input.minor_radius = "width".into();
    let ellipse = add_associative(&mut files, &mut source, &input).unwrap();
    let read_radii = |files: &DiscoveryWorkspace| {
        let geometry = files
            .workspace()
            .session(files.workspace().active_document().unwrap())
            .unwrap()
            .snapshot()
            .sketches
            .sketches()[0]
            .geometry();
        geometry
            .ellipse(SketchEntityId(ellipse.entity_id))
            .unwrap()
            .radii()
    };
    assert_eq!(read_radii(&files), [3., 1.]);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "2 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let radii = read_radii(&files);
    assert!(
        (radii[0] - 6.).abs() < 1e-6 && (radii[1] - 2.).abs() < 1e-6,
        "radii {radii:?}"
    );
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    let radii = read_radii(&files);
    assert!(
        (radii[0] - 6.).abs() < 1e-6 && (radii[1] - 2.).abs() < 1e-6,
        "radii {radii:?}"
    );
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    let radii = read_radii(&files);
    assert!(
        (radii[0] - 3.).abs() < 1e-6 && (radii[1] - 1.).abs() < 1e-6,
        "radii {radii:?}"
    );
}

#[test]
fn three_point_associative_slot_verification_failure_is_atomic() {
    use super::three_point::CardinalCircleConstruction;
    let (mut files, document, mut source) = fresh();
    let mut input = super::slots::slot_request("arcThreePoint");
    input.width = "width".into();
    let mut backend = CardinalCircleConstruction::default();
    backend.fail_verification = true;
    let before = owner_revision(&files, document);
    let geometry = rows(&files);
    let result = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_sketch_construction(&backend, Default::default()),
    )
    .sketch()
    .add_associative_entity(&input);
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("final circular seed rejected")
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(rows(&files), geometry);
}

#[test]
fn slot_width_expression_drives_both_caps_in_one_undoable_creation() {
    let (mut files, document, mut source) = fresh();
    let mut input = literal_input("slot", &[[0., 0.], [2., 0.]]);
    input.width = "width".into();
    let slot = add_associative(&mut files, &mut source, &input).unwrap();
    let members = slot.entity_i_ds.unwrap();
    assert_eq!(members.len(), 4);
    for id in [members[1], members[3]] {
        assert!((row(&files, id).radius - 0.5).abs() < 1e-7);
    }
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "2 cm".into(),
            ..Default::default()
        })
        .unwrap();
    for id in [members[1], members[3]] {
        assert!((row(&files, id).radius - 1.0).abs() < 1e-7);
    }
    files.commands(document).unwrap().undo().unwrap();
    files.commands(document).unwrap().undo().unwrap();
    assert!(!rows(&files).iter().any(|row| members.contains(&row.id)));
}

#[test]
fn every_slot_variant_keeps_width_and_curvature_after_recompute_and_reload() {
    use super::three_point::CardinalCircleConstruction;
    for variant in [
        "",
        "overall",
        "slotCenter",
        "arc",
        "arcCenterPoint",
        "arcThreePoint",
    ] {
        let (mut files, document, mut source) = fresh();
        let backend = CardinalCircleConstruction::default();
        let mut input = super::slots::slot_request(variant);
        input.width = "width".into();
        let slot = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_sketch_construction(&backend, Default::default()),
        )
        .sketch()
        .add_associative_entity(&input)
        .unwrap();
        let members = slot.entity_i_ds.unwrap();
        assert_slot_width(&files, &members, variant.starts_with("arc"), 1.);
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "2 cm".into(),
                ..Default::default()
            })
            .unwrap();
        assert_slot_width(&files, &members, variant.starts_with("arc"), 2.);
        assert_slot_tangency(&files, document, &members, variant.starts_with("arc"));
        let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_slot_width(&files, &members, variant.starts_with("arc"), 2.);
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "3 cm".into(),
                ..Default::default()
            })
            .unwrap();
        assert_slot_width(&files, &members, variant.starts_with("arc"), 3.);
        assert_slot_tangency(&files, reopened, &members, variant.starts_with("arc"));
    }
}

fn assert_slot_tangency(files: &DiscoveryWorkspace, document: u64, members: &[u64], curved: bool) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot.sketches.sketches()[0].geometry();
    for (left, right) in [(0, 1), (1, 2), (2, 3), (3, 0)] {
        let a = boundary_tangent(geometry, members[left], curved || left % 2 == 1);
        let b = boundary_tangent(geometry, members[right], curved || right % 2 == 1);
        let common =
            a.0.iter()
                .find(|point| b.0.contains(point))
                .expect("expected shared slot endpoint");
        let point = geometry.point(*common).unwrap().position;
        let av = tangent_at(geometry, a.1, point);
        let bv = tangent_at(geometry, b.1, point);
        let cross = av[0] * bv[1] - av[1] * bv[0];
        let norm = av[0].hypot(av[1]) * bv[0].hypot(bv[1]);
        assert!(
            norm > 0. && cross.abs() / norm < 1e-6,
            "slot tangency {left}/{right}: {cross} / {norm}"
        );
    }
}

fn boundary_tangent(
    geometry: &Sketch2D,
    id: u64,
    arc: bool,
) -> ([SketchEntityId; 2], (bool, SketchEntityId)) {
    let entity = SketchEntityId(id);
    if arc {
        let curve = geometry.arc(entity).unwrap();
        return ([curve.start, curve.end], (true, curve.center));
    }
    (geometry.line(entity).unwrap().endpoints, (false, entity))
}

fn tangent_at(geometry: &Sketch2D, operand: (bool, SketchEntityId), point: [f64; 2]) -> [f64; 2] {
    if operand.0 {
        let center = geometry.point(operand.1).unwrap().position;
        return [center[1] - point[1], point[0] - center[0]];
    }
    let ends = geometry
        .line(operand.1)
        .unwrap()
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    [ends[1][0] - ends[0][0], ends[1][1] - ends[0][1]]
}

fn assert_slot_width(files: &DiscoveryWorkspace, members: &[u64], curved: bool, width: f64) {
    for index in [1, 3] {
        let radius = row(files, members[index]).radius;
        assert!(
            (radius * 2. - width).abs() < 1e-6,
            "cap {index} radius {radius}, width {width}"
        );
    }
    if curved {
        let outer = row(files, members[0]).radius;
        let inner = row(files, members[2]).radius;
        assert!(
            (outer - inner - width).abs() < 1e-6,
            "outer {outer}, inner {inner}, width {width}"
        );
    }
}
