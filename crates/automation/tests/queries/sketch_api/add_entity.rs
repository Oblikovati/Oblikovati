// SPDX-License-Identifier: GPL-2.0-only
use super::entities::{activate_copy, configured_scaled, rows, verify_keys};
use super::*;
use oblikovati_api::wire::SketchEntityInfo;
fn create_empty_sketch(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
}
fn fresh() -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_empty_sketch(&mut files, &mut source);
    (files, document, source)
}
fn literal_input(kind: &str, points: &[[f64; 2]]) -> AddSketchEntityArgs {
    AddSketchEntityArgs {
        kind: kind.into(),
        points: Some(points.iter().map(|pair| Some(pair.to_vec())).collect()),
        ..Default::default()
    }
}
fn circle_input() -> AddSketchEntityArgs {
    let mut input = literal_input("circle", &[[1.0, 2.0]]);
    input.radius = "10 mm".into();
    input
}
fn arc_input() -> AddSketchEntityArgs {
    let mut input = literal_input("arc", &[[0.0; 2], [2.0, 0.0], [0.0, 2.0]]);
    input.ccw = true;
    input
}
fn add_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddSketchEntityArgs,
) -> Result<AddSketchEntityResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .add_entity(input)
}
fn row(files: &DiscoveryWorkspace, id: u64) -> SketchEntityInfo {
    rows(files).into_iter().find(|row| row.id == id).unwrap()
}
fn expected_points(points: &[[f64; 2]]) -> Option<Vec<Option<Vec<f64>>>> {
    Some(points.iter().map(|pair| Some(pair.to_vec())).collect())
}
#[test]
fn canonical_constructors_return_real_owned_ids_points_and_current_keys() {
    let (mut files, _, mut source) = fresh();
    let point = add_call(
        &mut files,
        &mut source,
        &literal_input("point", &[[3.0, 4.0]]),
    )
    .unwrap();
    let circle = add_call(&mut files, &mut source, &circle_input()).unwrap();
    let arc = add_call(&mut files, &mut source, &arc_input()).unwrap();
    assert_creation(&point, "point", 2, &[2]);
    assert_creation(&circle, "circle", 4, &[3]);
    assert_creation(&arc, "arc", 8, &[5, 6, 7]);
    assert_eq!(row(&files, circle.entity_id).radius, 1.0);
    assert_eq!(row(&files, arc.entity_id).radius, 2.0);
    verify_keys(&files, &rows(&files));
}
fn assert_creation(reply: &AddSketchEntityResult, kind: &str, entity: u64, points: &[u64]) {
    assert_eq!((reply.entity_id, reply.kind.as_str()), (entity, kind));
    assert_eq!(reply.point_i_ds.as_deref(), Some(points));
    assert!(reply.entity_i_ds.is_none());
    assert!(reply.inferred_constraints.is_none());
    assert!(reply.inferred_points.is_none());
}
#[test]
fn explicit_variants_curve_roles_and_both_arc_directions_survive_automatic_recompute() {
    let (mut files, document, mut source) = fresh();
    let mut circle = circle_input();
    circle.variant = "centerRadius".into();
    circle.construction = true;
    let circle = add_call(&mut files, &mut source, &circle).unwrap();
    assert!(row(&files, circle.entity_id).construction);
    for ccw in [true, false] {
        let mut arc = arc_input();
        arc.variant = "centerStartEnd".into();
        arc.ccw = ccw;
        arc.construction = true;
        let created = add_call(&mut files, &mut source, &arc).unwrap();
        assert!(row(&files, created.entity_id).construction);
        assert_arc_direction(&files, document, created.entity_id, ccw);
    }
}
#[test]
fn plain_points_ignore_curve_format_flags_as_the_go_constructor_does() {
    let (mut files, _, mut source) = fresh();
    let mut input = literal_input("point", &[[3.0, 4.0]]);
    input.construction = true;
    input.centerline = true;
    let created = add_call(&mut files, &mut source, &input).unwrap();
    assert!(!row(&files, created.entity_id).construction);
}
#[test]
fn expression_points_override_literal_points_and_radius_formulas_use_current_parameters() {
    let (mut files, _, mut source) = fresh();
    let mut input = circle_input();
    input.points = Some(vec![None]);
    input.point_exprs = Some(vec![Some(vec!["width * 2".into(), "width + 5 mm".into()])]);
    input.radius = "width / 2 + 5 mm".into();
    let created = add_call(&mut files, &mut source, &input).unwrap();
    let inspected = row(&files, created.entity_id);
    assert_eq!(inspected.points, expected_points(&[[2.0, 1.5]]));
    assert_eq!(inspected.radius, 1.0);
}
#[test]
fn literal_coordinates_and_length_expressions_use_different_input_units_consistently() {
    for scale in [0.1, 2.0] {
        let (mut files, _, mut source) = configured_scaled(scale);
        create_empty_sketch(&mut files, &mut source);
        let literal = add_call(&mut files, &mut source, &circle_input()).unwrap();
        assert_eq!(
            row(&files, literal.entity_id).points,
            expected_points(&[[1.0, 2.0]])
        );
        assert_eq!(row(&files, literal.entity_id).radius, 1.0);
        let mut expression = circle_input();
        expression.point_exprs = Some(vec![Some(vec!["10 mm".into(), "20 mm".into()])]);
        let expression = add_call(&mut files, &mut source, &expression).unwrap();
        assert_eq!(
            row(&files, expression.entity_id).points,
            expected_points(&[[1.0, 2.0]])
        );
        assert_eq!(row(&files, expression.entity_id).radius, 1.0);
    }
}
#[test]
fn bare_radius_literals_use_document_display_units_without_creating_a_parameter() {
    let (mut files, document, mut source) = fresh();
    let before = files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .clone();
    let mut input = circle_input();
    input.radius = "10".into();
    let created = add_call(&mut files, &mut source, &input).unwrap();
    assert_eq!(row(&files, created.entity_id).radius, 1.0);
    assert!(owned_parameters(&files, document).same_recipe(&before));
}
fn assert_refused(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    input: &AddSketchEntityArgs,
    expected: &str,
) {
    let before = owner_revision(files, document);
    let geometry = owned_geometry(files, document).clone();
    expect_api_decline(add_call(files, source, input), expected);
    assert_eq!(owner_revision(files, document), before);
    assert!(geometry.same_evaluated_state(owned_geometry(files, document)));
}
#[test]
fn bad_shapes_counts_and_null_coordinates_are_atomic_and_do_not_reserve_ids() {
    let (mut files, document, mut source) = fresh();
    for points in bad_point_shapes() {
        let mut input = circle_input();
        input.points = points;
        assert_refused(&mut files, document, &mut source, &input, "expected");
    }
    assert_refused(
        &mut files,
        document,
        &mut source,
        &literal_input("arc", &[[1.0; 2]]),
        "exactly 3",
    );
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn unknown_kinds_and_constructor_overloads_decline_by_name() {
    let (mut files, document, mut source) = fresh();
    for kind in ["bogus", "ellipticalArc", "spline", "slot"] {
        assert_refused(
            &mut files,
            document,
            &mut source,
            &literal_input(kind, &[[1.0; 2]]),
            "not ported",
        );
    }
    refuse_constructor_overloads(&mut files, document, &mut source);
}
#[test]
fn invalid_radius_and_coordinate_expressions_decline_without_partial_geometry() {
    let (mut files, document, mut source) = fresh();
    for radius in ["", "missing", "1 rad", "-1 mm", "0 mm", "1 cm / 0"] {
        let mut input = circle_input();
        input.radius = radius.into();
        assert_refused(&mut files, document, &mut source, &input, "expected");
    }
    for pair in [
        None,
        Some(vec![]),
        Some(vec!["1 cm".into()]),
        Some(vec!["missing".into(), "0".into()]),
        Some(vec!["0".into(), "1 rad".into()]),
    ] {
        let mut input = circle_input();
        input.point_exprs = Some(vec![pair]);
        assert_refused(&mut files, document, &mut source, &input, "expected");
    }
}
#[test]
fn empty_expression_array_uses_literal_coordinates_and_conversion_overflow_declines() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    create_empty_sketch(&mut files, &mut source);
    let mut input = circle_input();
    input.point_exprs = Some(vec![]);
    add_call(&mut files, &mut source, &input).unwrap();
    input.points = expected_points(&[[f64::MAX, 0.0]]);
    assert_refused(
        &mut files,
        document,
        &mut source,
        &input,
        "representable in working units",
    );
}
#[test]
fn public_creation_supports_part_and_assembly_owners_and_refuses_read_only_hosts() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, _, mut source) = configured_files(kind, Default::default());
        create_empty_sketch(&mut files, &mut source);
        let created = add_call(&mut files, &mut source, &circle_input()).unwrap();
        assert_eq!(row(&files, created.entity_id).radius, 1.0);
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .add_entity(&circle_input()),
            "sketch.addEntity",
        );
    }
}
#[test]
fn canonical_geometry_survives_save_reopen_and_copy_with_new_keys() {
    let (mut files, document, mut source) = fresh();
    add_call(&mut files, &mut source, &circle_input()).unwrap();
    add_call(&mut files, &mut source, &arc_input()).unwrap();
    let before = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), before);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/api-created-copy.obk"),
    );
    let copied = rows(&files);
    assert_eq!(copied.len(), before.len());
    assert_copy_namespaces(&before, &copied);
    verify_keys(&files, &copied);
}
#[test]
fn api_creation_groups_undo_and_observer_veto_use_the_normal_document_boundary() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Create circles")
        .unwrap();
    add_call(&mut files, &mut source, &circle_input()).unwrap();
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Create circles"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    assert_vetoed_creation(&mut files, document, &mut source, &before);
}

fn refuse_constructor_overloads(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
) {
    for kind in ["circle", "arc", "point"] {
        let mut input = literal_input(kind, &[[1.0; 2]]);
        input.variant = "unportedVariant".into();
        assert_refused(
            files,
            document,
            source,
            &input,
            "other overloads are not ported",
        );
    }
}

fn assert_copy_namespaces(before: &[SketchEntityInfo], copied: &[SketchEntityInfo]) {
    for (first, second) in before.iter().zip(copied) {
        assert_eq!(
            (first.id, &first.points, first.radius),
            (second.id, &second.points, second.radius)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
}

fn assert_vetoed_creation(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    before: &[SketchEntityInfo],
) {
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(files, document, source, observer);
    expect_api_decline(add_call(files, source, &circle_input()), "locked");
    assert_eq!(rows(files), before);
    // #3771: veto discards geometry, but retired IDs must still be saved.
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(files, document, source, Default::default());
    assert_next_point(files, source, 6);
}

fn owned_parameters(
    files: &DiscoveryWorkspace,
    document: u64,
) -> &oblikovati_model::parameter::ParameterCollection<u64> {
    &files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
}
fn assert_arc_direction(files: &DiscoveryWorkspace, document: u64, id: u64, ccw: bool) {
    assert_eq!(
        owned_geometry(files, document).sketches()[0]
            .geometry()
            .arc(SketchEntityId(id))
            .unwrap()
            .counter_clockwise,
        ccw
    );
}
fn bad_point_shapes() -> [Option<Vec<Option<Vec<f64>>>>; 5] {
    [
        None,
        Some(vec![]),
        Some(vec![None]),
        Some(vec![Some(vec![1.0])]),
        Some(vec![Some(vec![1.0, 2.0, 3.0])]),
    ]
}
fn assert_next_point(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, expected: u64) {
    assert_eq!(
        add_call(files, source, &literal_input("point", &[[0.0; 2]]))
            .unwrap()
            .entity_id,
        expected
    );
}
#[test]
fn unhealthy_parameter_reads_decline_and_repaired_reads_use_the_new_value() {
    let (mut files, document, mut source) = fresh();
    set_width(&mut files, &mut source, "missing");
    let mut input = circle_input();
    input.radius = "width".into();
    assert_refused(
        &mut files,
        document,
        &mut source,
        &input,
        "healthy finite length",
    );
    set_width(&mut files, &mut source, "2 cm");
    let created = add_call(&mut files, &mut source, &input).unwrap();
    assert_eq!(row(&files, created.entity_id).radius, 2.0);
}
#[test]
fn constructor_expressions_are_seeds_and_do_not_invent_a_live_dimension_binding() {
    let (mut files, _, mut source) = fresh();
    let mut input = circle_input();
    input.radius = "width".into();
    let first = add_call(&mut files, &mut source, &input).unwrap();
    let original = row(&files, first.entity_id);
    set_width(&mut files, &mut source, "2 cm");
    assert_eq!(row(&files, first.entity_id), original);
    let second = add_call(&mut files, &mut source, &input).unwrap();
    assert_eq!(row(&files, second.entity_id).radius, 2.0);
}

fn set_width(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, expression: &str) {
    Client::new(FileHost::new(files, source, "test"))
        .parameters()
        .set(&numeric("width", expression))
        .unwrap();
}
#[test]
fn invalid_sketch_indices_and_document_kinds_cannot_adopt_geometry() {
    let (mut files, document, mut source) = fresh();
    for index in [-1, 1] {
        let mut input = circle_input();
        input.sketch_index = index;
        assert_refused(&mut files, document, &mut source, &input, "ordered index");
    }
    let (mut files, document, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    assert_refused(
        &mut files,
        document,
        &mut source,
        &circle_input(),
        "part or assembly document",
    );
}

#[path = "add_entity/line.rs"]
mod line;

#[path = "add_entity/deletion_attributes.rs"]
mod deletion_attributes;
#[path = "add_entity/ellipse.rs"]
mod ellipse;
#[path = "add_entity/point_ownership.rs"]
mod point_ownership;
#[path = "add_entity/polygon.rs"]
mod polygon;
#[path = "add_entity/polyline.rs"]
mod polyline;
#[path = "add_entity/rectangle.rs"]
mod rectangle;
#[path = "add_entity/three_point.rs"]
mod three_point;
