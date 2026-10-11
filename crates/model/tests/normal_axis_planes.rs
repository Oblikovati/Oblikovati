// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::{
            DatumLength, WorkGeometryDefinition, WorkPlaneLinePoint, WorkPointReference,
        },
        work_plane::WorkPlaneResolver,
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};

fn source(line: WorkAxisReference, point: WorkPointReference) -> WorkGeometryDefinition {
    WorkGeometryDefinition::NormalToAxis(WorkPlaneLinePoint { line, point })
}
fn append(
    program: &mut FeatureSequence<WorkGeometryDefinition>,
    id: u64,
    definition: WorkGeometryDefinition,
) {
    program
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn program() -> FeatureSequence<WorkGeometryDefinition> {
    let mut program = FeatureSequence::default();
    let axis = WorkAxisLine::new([100.0, 200.0, 300.0], [0.0, 0.0, 1.0]).unwrap();
    append(&mut program, 1, WorkGeometryDefinition::Axis(axis));
    let point = [
        DatumLength::Constant(5.0),
        DatumLength::Constant(6.0),
        DatumLength::Parameter(1),
    ];
    append(&mut program, 2, WorkGeometryDefinition::Point(point));
    append(
        &mut program,
        3,
        source(
            WorkAxisReference::Feature(FeatureId(1)),
            WorkPointReference::Feature(FeatureId(2)),
        ),
    );
    program
}
fn parameters(value: &str) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "height", USER_PARAM, value)
        .unwrap();
    parameters
}
fn frame(program: &FeatureSequence<WorkGeometryDefinition>, value: &str) -> SketchPlane {
    program
        .evaluate_work_geometry(&parameters(value))
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
#[test]
fn current_point_sets_origin_and_current_axis_sets_normal_independently() {
    let mut program = program();
    assert_eq!(frame(&program, "7 cm").recipe().origin, [5.0, 6.0, 7.0]);
    assert_eq!(frame(&program, "7 cm").normal(), [0.0, 0.0, 1.0]);
    assert_eq!(frame(&program, "11 cm").recipe().origin, [5.0, 6.0, 11.0]);
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    let replacement = WorkGeometryDefinition::Axis(axis);
    program
        .replace(FeatureId(1), replacement, replacement.reads())
        .unwrap();
    assert_eq!(frame(&program, "11 cm").normal(), [1.0, 0.0, 0.0]);
}
#[test]
fn point_does_not_need_to_lie_on_the_axis_and_origin_references_need_no_reads() {
    let mut program = program();
    let definition = source(WorkAxisReference::OriginX, WorkPointReference::Origin);
    assert!(definition.reads().keys().is_empty());
    program
        .replace(FeatureId(3), definition, definition.reads())
        .unwrap();
    assert_eq!(frame(&program, "0 cm").normal(), [1.0, 0.0, 0.0]);
    assert_eq!(frame(&program, "0 cm").recipe().origin, [0.0; 3]);
}
#[test]
fn driver_suppression_has_no_stale_frame_and_recovery_keeps_the_authored_flip() {
    let mut program = program();
    program.flip_plane_normal(FeatureId(3)).unwrap();
    program.set_suppressed(FeatureId(1), true).unwrap();
    let failed = program.evaluate_work_geometry(&parameters("7 cm")).unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(failed.point(FeatureId(2)).is_ok());
    program.set_suppressed(FeatureId(1), false).unwrap();
    assert_eq!(frame(&program, "7 cm").normal(), [0.0, 0.0, -1.0]);
}
#[test]
fn wrong_driver_kind_is_a_named_local_decline() {
    let mut program = program();
    let wrong = source(
        WorkAxisReference::Feature(FeatureId(2)),
        WorkPointReference::Origin,
    );
    program.replace(FeatureId(3), wrong, wrong.reads()).unwrap();
    let failed = program.evaluate_work_geometry(&parameters("7 cm")).unwrap();
    assert!(failed.plane(FeatureId(3)).is_err());
    assert!(format!("{:?}", failed.health[2]).contains("axis"));
}
#[test]
fn attributed_dependencies_refuse_self_and_forward_source_changes() {
    let mut program = program();
    let before = program.clone();
    let reads = program.get(FeatureId(3)).unwrap().definition().reads();
    let expected = [1, 2].map(|id| FeatureReadKey::Feature(FeatureId(id)));
    assert_eq!(reads.keys().iter().copied().collect::<Vec<_>>(), expected);
    for id in [3, 99] {
        let invalid = source(
            WorkAxisReference::Feature(FeatureId(id)),
            WorkPointReference::Origin,
        );
        assert!(
            program
                .replace(FeatureId(3), invalid, invalid.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&before));
    }
}
#[test]
fn versioned_source_round_trips_exactly_and_older_programs_refuse_it() {
    let program = program();
    let recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 13);
    assert_eq!(recipe.features[2].definition.version, 8);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(program.same_work_geometry_recipe(&restored));
    let mut legacy = recipe.clone();
    legacy.version = 12;
    assert!(FeatureSequence::from_work_geometry_recipe(&legacy).is_err());
    assert!(
        !source(WorkAxisReference::OriginX, WorkPointReference::Origin).same_recipe(source(
            WorkAxisReference::OriginY,
            WorkPointReference::Origin
        ))
    );
}
#[test]
fn source_version_and_unknown_fields_are_refused() {
    let definition = source(WorkAxisReference::OriginZ, WorkPointReference::Origin);
    let mut recipe = definition.recipe();
    recipe.version = 7;
    assert!(WorkGeometryDefinition::from_recipe(recipe).is_err());
    let mut value = serde_json::to_value(definition.recipe()).unwrap();
    value["definition"]["NormalToAxis"]["unknown"] = true.into();
    assert!(
        serde_json::from_value::<oblikovati_model::feature::work_plane::WorkPlaneRecipe>(value)
            .is_err()
    );
}
fn normal_frame(normal: [f64; 3]) -> SketchPlane {
    SketchPlane::from_origin_normal([0.0; 3], normal, Default::default()).unwrap()
}
#[test]
fn signed_principal_normals_preserve_the_go_frame_roll_rule() {
    for (normal, x_axis) in [
        ([1.0, 0.0, 0.0], [0.0, 0.0, 1.0]),
        ([-1.0, 0.0, 0.0], [0.0, 0.0, -1.0]),
        ([0.0, 1.0, 0.0], [0.0, 0.0, -1.0]),
        ([0.0, -1.0, 0.0], [0.0, 0.0, 1.0]),
        ([0.0, 0.0, 1.0], [0.0, 1.0, 0.0]),
        ([0.0, 0.0, -1.0], [0.0, -1.0, 0.0]),
    ] {
        let frame = normal_frame(normal);
        assert_eq!(frame.normal(), normal);
        assert_eq!(frame.recipe().x_axis, x_axis);
    }
}
#[test]
fn exact_seed_boundary_and_adjacent_values_keep_the_source_rule() {
    let boundary = 0.9_f64;
    for sign in [-1.0, 1.0] {
        for (component, expected_x_z) in [
            (f64::from_bits(boundary.to_bits() - 1), -1.0),
            (boundary, -1.0),
            (f64::from_bits(boundary.to_bits() + 1), sign),
        ] {
            let normal = [sign * component, (1.0 - component * component).sqrt(), 0.0];
            let frame = normal_frame(normal);
            assert_eq!(frame.recipe().x_axis[2], expected_x_z);
            assert_direction(frame.normal(), normal, frame.options().angular_deviation);
        }
    }
}
fn assert_direction(actual: [f64; 3], expected: [f64; 3], bound: f64) {
    assert!(
        actual
            .into_iter()
            .zip(expected)
            .all(|(actual, expected)| (actual - expected).abs() <= bound)
    );
}
#[test]
fn generated_oblique_normals_are_orthogonal_right_handed_and_repeatable() {
    for index in -32..=32 {
        let direction = [f64::from(index), 3.0, 4.0];
        let normal = WorkAxisLine::new([0.0; 3], direction).unwrap().direction();
        let first = normal_frame(normal);
        let second = normal_frame(normal);
        assert!(first.same_recipe(second));
        assert_direction(first.normal(), normal, first.options().angular_deviation);
    }
}
#[test]
fn normal_frame_keeps_signed_origin_bits_and_refuses_invalid_inputs() {
    let origin = [-0.0, f64::MAX, -0.0];
    let frame =
        SketchPlane::from_origin_normal(origin, [0.0, 0.0, 1.0], Default::default()).unwrap();
    assert_eq!(
        frame.recipe().origin.map(f64::to_bits),
        origin.map(f64::to_bits)
    );
    for normal in [[0.0; 3], [2.0, 0.0, 0.0], [f64::NAN, 0.0, 1.0]] {
        assert!(SketchPlane::from_origin_normal([0.0; 3], normal, Default::default()).is_err());
    }
    assert!(
        SketchPlane::from_origin_normal(
            [f64::INFINITY, 0.0, 0.0],
            [0.0, 0.0, 1.0],
            Default::default()
        )
        .is_err()
    );
}
