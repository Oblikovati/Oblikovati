// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadKey, FeatureSequence,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::WorkGeometryDefinition,
        work_plane::{WorkPlaneRecipe, WorkPlaneResolver},
    },
    parameter::ParameterCollection,
    sketch::SketchPlane,
};
type DatumProgram = FeatureSequence<WorkGeometryDefinition>;
fn definition(first: WorkAxisReference, second: WorkAxisReference) -> WorkGeometryDefinition {
    WorkGeometryDefinition::ThroughTwoAxes([first, second])
}
fn append(program: &mut DatumProgram, id: u64, definition: WorkGeometryDefinition) {
    program
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn axis(origin: [f64; 3], direction: [f64; 3]) -> WorkGeometryDefinition {
    WorkGeometryDefinition::Axis(WorkAxisLine::new(origin, direction).unwrap())
}
fn program() -> DatumProgram {
    let mut program = DatumProgram::default();
    append(&mut program, 1, axis([2.0, 3.0, 4.0], [1.0, 0.0, 0.0]));
    append(&mut program, 2, axis([80.0, 90.0, 100.0], [0.0, 1.0, 0.0]));
    append(
        &mut program,
        3,
        definition(
            WorkAxisReference::Feature(FeatureId(1)),
            WorkAxisReference::Feature(FeatureId(2)),
        ),
    );
    program
}
fn frame(program: &DatumProgram) -> SketchPlane {
    program
        .evaluate_work_geometry(&ParameterCollection::new(Default::default()))
        .unwrap()
        .plane(FeatureId(3))
        .unwrap()
}
fn replace(program: &mut DatumProgram, id: u64, definition: WorkGeometryDefinition) {
    program
        .replace(FeatureId(id), definition.clone(), definition.reads())
        .unwrap();
}
#[test]
fn both_current_directions_and_first_current_origin_drive_the_frame() {
    let mut program = program();
    assert_eq!(frame(&program).recipe().origin, [2.0, 3.0, 4.0]);
    assert_eq!(frame(&program).normal(), [0.0, 0.0, 1.0]);
    replace(&mut program, 2, axis([900.0; 3], [0.0, 0.0, 1.0]));
    assert_eq!(frame(&program).normal(), [0.0, -1.0, 0.0]);
    replace(&mut program, 1, axis([5.0, 6.0, 7.0], [0.0, 1.0, 0.0]));
    assert_eq!(frame(&program).recipe().origin, [5.0, 6.0, 7.0]);
    assert_eq!(frame(&program).normal(), [1.0, 0.0, 0.0]);
}
#[test]
fn skew_second_origin_is_not_a_coplanarity_or_placement_input() {
    let mut program = program();
    let before = frame(&program);
    replace(
        &mut program,
        2,
        axis([f64::MAX, -f64::MAX, f64::MAX], [0.0, 1.0, 0.0]),
    );
    assert!(before.same_recipe(frame(&program)));
    assert_eq!(before.recipe().x_axis, [1.0, 0.0, 0.0]);
}
#[test]
fn ordered_axes_control_normal_sign_without_changing_authored_flip() {
    let mut program = program();
    program.flip_plane_normal(FeatureId(3)).unwrap();
    assert_eq!(frame(&program).normal(), [0.0, 0.0, -1.0]);
    let swapped = definition(
        WorkAxisReference::Feature(FeatureId(2)),
        WorkAxisReference::Feature(FeatureId(1)),
    );
    replace(&mut program, 3, swapped);
    assert_eq!(frame(&program).normal(), [0.0, 0.0, 1.0]);
    assert_eq!(frame(&program).recipe().origin, [80.0, 90.0, 100.0]);
}
#[test]
fn parallel_and_suppressed_drivers_have_no_stale_frame_and_recover() {
    let mut program = program();
    for second in [[1.0, 0.0, 0.0], [-1.0, 0.0, 0.0]] {
        replace(&mut program, 2, axis([0.0; 3], second));
        assert_no_frame(&program);
    }
    replace(&mut program, 2, axis([0.0; 3], [0.0, 1.0, 0.0]));
    program.set_suppressed(FeatureId(1), true).unwrap();
    assert_no_frame(&program);
    program.set_suppressed(FeatureId(1), false).unwrap();
    assert_eq!(frame(&program).normal(), [0.0, 0.0, 1.0]);
}
#[test]
fn origin_axes_read_nothing_and_repeated_owned_axes_share_one_dependency() {
    let origins = definition(WorkAxisReference::OriginX, WorkAxisReference::OriginY);
    assert!(origins.reads().keys().is_empty());
    let owned = WorkAxisReference::Feature(FeatureId(1));
    let repeated = definition(owned, owned).reads();
    assert_eq!(
        repeated.keys().iter().copied().collect::<Vec<_>>(),
        [FeatureReadKey::Feature(FeatureId(1))]
    );
    let reads = program().get(FeatureId(3)).unwrap().definition().reads();
    assert_eq!(
        reads.keys().iter().copied().collect::<Vec<_>>(),
        [1, 2].map(|id| FeatureReadKey::Feature(FeatureId(id)))
    );
}
#[test]
fn self_forward_and_wrong_kind_references_are_named_declines() {
    let mut program = program();
    assert_invalid_dependencies(&mut program);
    replace(
        &mut program,
        2,
        WorkGeometryDefinition::Fixed(SketchPlane::xy()),
    );
    let result = program
        .evaluate_work_geometry(&ParameterCollection::new(Default::default()))
        .unwrap();
    assert!(result.plane(FeatureId(3)).is_err());
    assert!(format!("{:?}", result.health[2]).contains("axis"));
}
#[test]
fn exact_sources_round_trip_and_require_new_program_and_wrapper_versions() {
    let program = program();
    let recipe = program.work_geometry_recipe().unwrap();
    assert_eq!(
        (recipe.version, recipe.features[2].definition.version),
        (14, 9)
    );
    assert!(
        program
            .same_work_geometry_recipe(&DatumProgram::from_work_geometry_recipe(&recipe).unwrap())
    );
    let mut old = recipe.clone();
    old.version = 13;
    assert!(DatumProgram::from_work_geometry_recipe(&old).is_err());
    let mut source = recipe.features[2].definition.clone();
    source.version = 8;
    assert!(WorkGeometryDefinition::from_recipe(source.clone()).is_err());
}
#[test]
fn ordered_source_equality_and_unknown_variant_fields_are_checked() {
    let first = definition(WorkAxisReference::OriginX, WorkAxisReference::OriginY);
    let reverse = definition(WorkAxisReference::OriginY, WorkAxisReference::OriginX);
    assert!(!first.same_recipe(&reverse));
    let mut value = serde_json::to_value(first.recipe()).unwrap();
    value["definition"]["ThroughTwoAxes"] = serde_json::json!(["OriginX", "OriginY", "OriginZ"]);
    assert!(serde_json::from_value::<WorkPlaneRecipe>(value).is_err());
}
#[test]
fn generated_small_and_oblique_angles_keep_a_repeatable_right_handed_frame() {
    for exponent in [0, -8, -32, -128, -512, -1022] {
        let small = 2.0_f64.powi(exponent);
        let second = WorkAxisLine::new([0.0; 3], [1.0, small, small])
            .unwrap()
            .direction();
        let actual = axis_frame([0.0; 3], second).unwrap();
        let repeat = axis_frame([0.0; 3], second).unwrap();
        assert!(actual.same_recipe(repeat));
        assert_eq!(actual.recipe().x_axis, [1.0, 0.0, 0.0]);
        assert!((actual.normal()[1] + 0.5_f64.sqrt()).abs() <= actual.options().angular_deviation);
        assert!((actual.normal()[2] - 0.5_f64.sqrt()).abs() <= actual.options().angular_deviation);
    }
}
#[test]
fn axis_direction_constructor_preserves_origin_bits_and_refuses_invalid_frames() {
    let origin = [-0.0, f64::MAX, -0.0];
    let frame = axis_frame(origin, [0.0, 1.0, 0.0]).unwrap();
    assert_eq!(
        frame.recipe().origin.map(f64::to_bits),
        origin.map(f64::to_bits)
    );
    for second in [[0.0; 3], [2.0, 0.0, 0.0], [f64::NAN, 1.0, 0.0]] {
        assert!(axis_frame(origin, second).is_err());
    }
    assert!(axis_frame([f64::INFINITY; 3], [0.0, 1.0, 0.0]).is_err());
}

fn assert_no_frame(program: &DatumProgram) {
    let parameters = ParameterCollection::new(Default::default());
    let evaluated = program.evaluate_work_geometry(&parameters).unwrap();
    assert!(evaluated.plane(FeatureId(3)).is_err());
}
fn axis_frame(
    origin: [f64; 3],
    second: [f64; 3],
) -> Result<SketchPlane, oblikovati_model::sketch::SketchError> {
    SketchPlane::from_axis_directions(origin, [1.0, 0.0, 0.0], second, Default::default())
}

fn assert_invalid_dependencies(program: &mut DatumProgram) {
    let baseline = program.clone();
    for id in [3, 99] {
        let source = definition(
            WorkAxisReference::OriginX,
            WorkAxisReference::Feature(FeatureId(id)),
        );
        assert!(
            program
                .replace(FeatureId(3), source.clone(), source.reads())
                .is_err()
        );
        assert!(program.same_work_geometry_recipe(&baseline));
    }
}
