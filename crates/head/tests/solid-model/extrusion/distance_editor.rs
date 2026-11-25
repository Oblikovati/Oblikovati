// SPDX-License-Identifier: GPL-2.0-only
use super::super::{diagnostics, x11_input};
use super::acceptance::{feature_count, feature_distance, open_extrusion_editor};
use super::viewport_interaction::{click_text_native, replace_distance_native};
use super::*;

#[given("Edit Feature is open for an existing extrusion")]
fn existing_extrusion_editor(world: &mut ApplicationWorld) {
    create_first_extrusion(world);
    open_extrusion_editor(world);
}

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-rect-8mm.opd` is open with Edit Feature active"
)]
fn published_extrusion_editor(world: &mut ApplicationWorld) {
    let source =
        super::super::scenario_root().join("fixtures/cad/extrusion/extrusion-rect-8mm.opd");
    let copy = world.scratch_folder().join("extrusion-rect-8mm.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    open_extrusion_editor(world);
}

#[given("Edit Feature with Distance A 8 mm")]
fn eight_millimetre_tolerance_target(world: &mut ApplicationWorld) {
    existing_extrusion_editor(world);
}

#[when("Tolerance is opened from its dropdown")]
fn open_distance_tolerance(world: &mut ApplicationWorld) {
    open_distance_menu(world);
    click_popup(world, "Tolerance");
}

#[then("Model Value is 8.000 mm and Type is Default")]
fn tolerance_seeds_the_current_model_value(world: &mut ApplicationWorld) {
    world.run_in_window("Tolerance", "ModelValue:8.000mm");
    world.run_in_window("Tolerance", "Default");
    world.run_in_window("Tolerance", "Type");
}

#[then(
    "Precision, evaluated-size controls, Upper and Lower fields, Hole and Shaft fields are visible"
)]
fn tolerance_controls_are_visible(world: &mut ApplicationWorld) {
    for label in [
        "Precision",
        "EvaluatedSize",
        "Upper",
        "Lower",
        "Hole",
        "Shaft",
    ] {
        world.run_in_window("Tolerance", label);
    }
    world.save_capture("distance-tolerance");
}

#[when("the Type list is opened")]
fn open_tolerance_types(world: &mut ApplicationWorld) {
    click_window_label(world, "Tolerance", "Default");
}

#[then(
    "it offers Default, Basic, Symmetric, Deviation, Limits-Stacked, Limits-Linear, MAX, MIN, Limits/Fits-Stacked, Limits/Fits-Linear, Limits/Fits-Show Size Limits and Limits/Fits-Show Tolerance"
)]
fn tolerance_types_are_listed(world: &mut ApplicationWorld) {
    for label in [
        "Default",
        "Basic",
        "Symmetric",
        "Deviation",
        "Limits-Stacked",
        "Limits-Linear",
        "MAX",
        "MIN",
        "Limits/Fits-Stacked",
        "Limits/Fits-Linear",
        "Limits/Fits-ShowSizeLimits",
        "Limits/Fits-ShowTolerance",
    ] {
        popup_label(world, label);
    }
    world.save_capture("distance-tolerance-types");
}

#[when("the Type list is dismissed and Cancel is clicked")]
fn cancel_tolerance(world: &mut ApplicationWorld) {
    x11_input::press_escape();
    world.pump(Vec::new());
    click_window_label(world, "Tolerance", "Cancel");
}

#[then("the extrusion panel remains at Distance A 8 mm")]
fn original_distance_survives_tolerance_cancel(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Tolerance").is_none());
    world.run_in_window("Extrude", "DistanceA");
    let parameter = current_feature_parameter(world);
    world.run_in_window("Extrude", &parameter);
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 8.);
    let document = world.active_id();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let parameter = snapshot.parameters.find(&parameter).unwrap();
    assert_eq!(
        snapshot.parameters.get(parameter).unwrap().tolerance,
        Default::default()
    );
}

fn create_first_extrusion(world: &mut ApplicationWorld) {
    selected_rectangle_preview(world);
    replace_distance_native(world, "8");
    click_text_native(world, "OK");
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 8.);
}

#[when("the user opens the Distance A editor menu")]
fn open_distance_menu(world: &mut ApplicationWorld) {
    let field = world.run_in_window("Extrude", "DistanceA").center();
    let (position, size) = diagnostics::window_geometry("Extrude").unwrap();
    click_position(world, [position[0] + size[0] - 21., field[1]]);
}

#[then("it offers Measure, Select Feature Dimension, Parameters, Tolerance and List Parameters")]
fn menu_offers_distance_tools(world: &mut ApplicationWorld) {
    for label in [
        "Measure",
        "SelectFeatureDimension",
        "Parameters",
        "Tolerance",
        "ListParameters",
    ] {
        popup_label(world, label);
    }
    world.save_capture("distance-editor-menu");
}

#[when("the user chooses List Parameters")]
fn open_parameter_list(world: &mut ApplicationWorld) {
    click_popup(world, "ListParameters");
}

#[then("a list of parameters available in the active part appears")]
fn parameters_are_listed(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        !state.parameters.is_empty(),
        "expected authored parameters in the fixture"
    );
    popup_label(world, "AvailableParameters");
    for parameter in &state.parameters {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|row| world.frame.is_popup_row(row) && row.text.starts_with(&parameter.name)),
            "expected {} in the parameter list",
            parameter.name
        );
    }
    world.save_capture("distance-parameter-list");
}

#[then("the extrusion edit closes without changing the feature")]
#[then("the accepted feature remains unchanged")]
fn accepted_extrusion_stays_unchanged(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Extrude").is_none());
    assert_eq!(feature_count(world), 1);
    let state = world.state();
    let browser = state
        .browser
        .expect("expected a part after cancelling the draft");
    let feature = browser
        .extrusion_edits
        .values()
        .next()
        .expect("expected the accepted extrusion");
    assert_extrusion_is_eight_millimetres(world, &feature.distance);
    assert!(!state.extrude_preview_active);
}

fn assert_extrusion_is_eight_millimetres(world: &mut ApplicationWorld, source: &str) {
    use oblikovati_model::quantity::Dimension;
    let document = world.active_id();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let evaluate = |text: &str| {
        snapshot
            .parameters
            .evaluate_scalar_expression(text, Dimension::Length)
            .unwrap()
            .value
    };
    assert!(
        (evaluate(source) - evaluate("8 mm")).abs() < 1e-12,
        "unexpected depth {source:?}"
    );
}

#[when("the user opens the Distance A editor menu and chooses Select Feature Dimension")]
fn start_dimension_selection(world: &mut ApplicationWorld) {
    open_distance_menu(world);
    click_popup(world, "SelectFeatureDimension");
}

#[then("the status area asks the user to select a feature and then one of its dimensions")]
fn status_prompts_for_dimension(world: &mut ApplicationWorld) {
    let status: String = world
        .window_text("##status-bar")
        .chars()
        .filter(|c| !c.is_whitespace())
        .collect();
    assert!(
        status.contains("Selectafeatureandthenoneofitsdimensions."),
        "{status}"
    );
    world.save_capture("choose-feature-dimension");
}

#[when("the user cancels the extrusion edit")]
fn cancel_edit(world: &mut ApplicationWorld) {
    click_window_label(world, "Extrude", "Cancel");
}

#[when("the user opens the Distance A editor menu and chooses Parameters")]
fn open_parameter_panel(world: &mut ApplicationWorld) {
    open_distance_menu(world);
    click_popup(world, "Parameters");
}

#[then("a Parameters dialog opens for the active part")]
fn part_parameter_panel_opens(world: &mut ApplicationWorld) {
    world.run_in_window("Parameters", "ModelParameters");
    let name = current_feature_parameter(world);
    world.run_in_window("Parameters", &name);
}

#[then(
    "the table includes parameter name, consumed-by, unit or type, equation, nominal value, tolerance and model value columns"
)]
fn table_headers(world: &mut ApplicationWorld) {
    for caption in [
        "Name",
        "ConsumedBy",
        "Unit",
        "Equation",
        "NominalValue",
        "Tolerance",
        "ModelValue",
    ] {
        world.run_in_window("Parameters", caption);
    }
    world.run_in_window("Parameters", "Extrusion1");
    world.save_capture("distance-parameter-table");
}

#[then("the extrusion panel remains open")]
fn distance_draft_remains_open(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Parameters").is_none());
    world.run_in_window("Extrude", "DistanceA");
}

fn current_feature_parameter(world: &mut ApplicationWorld) -> String {
    feature_parameter(world, 0)
}

fn feature_parameter(world: &mut ApplicationWorld, index: usize) -> String {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let snapshot = session.snapshot();
    let parameter = snapshot.work_geometry.features()[index].owned_scalars()[0].parameter;
    snapshot.parameters.get(parameter).unwrap().name.clone()
}

#[test]
fn native_dimension_picker_inserts_a_feature_parameter_into_the_draft() {
    let mut world = ApplicationWorld::new();
    create_first_extrusion(&mut world);
    click_text_native(&mut world, "Extrude");
    replace_distance_native(&mut world, "12");
    click_text_native(&mut world, "OK");
    let driver = feature_parameter(&mut world, 1);
    open_extrusion_editor(&mut world);
    start_dimension_selection(&mut world);
    click_text_native(&mut world, "Extrusion2");
    click_window_label(&mut world, "Extrude", "Selectadimension");
    let target = world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text.starts_with(&driver))
        .expect("expected the second feature's dimension")
        .center();
    click_position(&mut world, target);
    world.run_in_window("Extrude", &driver);
    assert_eq!(feature_distance(&mut world), 8.);
    cancel_edit(&mut world);
    assert_eq!(feature_count(&mut world), 2);
    assert_eq!(feature_distance(&mut world), 8.);
    world.close();
}

fn popup_label<'w>(world: &'w ApplicationWorld, label: &str) -> &'w diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == label)
        .unwrap_or_else(|| {
            panic!(
                "expected {label} in the distance popup, saw {}",
                world.text()
            )
        })
}

fn click_popup(world: &mut ApplicationWorld, label: &str) {
    click_position(world, popup_label(world, label).center());
}

fn click_window_label(world: &mut ApplicationWorld, window: &str, label: &str) {
    click_position(world, world.run_in_window(window, label).center());
}

fn click_position(world: &mut ApplicationWorld, point: [f32; 2]) {
    x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
