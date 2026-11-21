// SPDX-License-Identifier: GPL-2.0-only
//! Drive shared dockable windows with the native pointer.
use super::*;

#[test]
fn quick_access_select_changes_the_viewport_selection_mode() {
    use oblikovati_gui::GuiSelectionMode;
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    for (label, expected) in [
        ("Faces", GuiSelectionMode::Faces),
        ("Components", GuiSelectionMode::Components),
        ("All Types", GuiSelectionMode::All),
    ] {
        world.click(row_center(&world, "Select"));
        world.click_text(label);
        world.pump(Vec::new());
        assert_eq!(world.state().selection_mode, expected);
    }
    world.close();
}

fn close_title(world: &mut ApplicationWorld, title: &str) {
    let (position, size) = window_geometry(title).expect("expected a visible tool window");
    let center = [position[0] + size[0] - 13., position[1] + 10.];
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

fn enter_parameter_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let point = if window_geometry("Parameters").is_some() {
        let kind = world.run_in_window("Parameters", "Numeric");
        let offset = match label {
            "Name" => 125.,
            "Equation" => 273.,
            _ => panic!("parameter field {label:?}: expected Name or Equation"),
        };
        [kind.minimum[0] + offset, kind.center()[1]]
    } else {
        let run = world.run(label);
        [run.minimum[0] - 40., run.center()[1]]
    };
    world.click(super::super::diagnostics::pointer_pixels(point));
    world.type_text(value);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
}

#[test]
fn quick_access_update_recomputes_the_active_part_without_a_history_step() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    assert!(on_row(&world, "Update"));
    let history = world.state().undo_labels.clone();
    world.click(row_center(&world, "Update"));
    world.pump(Vec::new());
    assert_eq!(world.state().undo_labels, history);
    part_stays(&mut world);
    world.close();
}

#[test]
fn quick_access_parameters_opens_closes_and_can_be_removed() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    assert!(on_row(&world, "Parameters"));
    world.click(row_center(&world, "Parameters"));
    assert!(window_geometry("Parameters").is_some());
    enter_parameter_field(&mut world, "Name", "quick_size");
    enter_parameter_field(&mut world, "Equation", "10 mm");
    world.save_capture("quick-access-parameter-input");
    world.click_text("Add");
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .parameters
            .iter()
            .any(|row| row.name == "quick_size"),
        "visible text: {}",
        world.text()
    );
    close_title(&mut world, "Parameters");
    assert!(window_geometry("Parameters").is_none());
    open_customization(&mut world);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Parameters")
        .unwrap()
        .center();
    world.click(super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    assert!(!on_row(&world, "Parameters"));
    part_stays(&mut world);
    world.close();
}

#[test]
fn dockable_material_browser_closes_and_reopens_from_the_title_bar() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    world.click(row_center(&world, "MaterialBrowser"));
    assert!(window_geometry("Material Browser").is_some());
    close_title(&mut world, "Material Browser");
    assert!(window_geometry("Material Browser").is_none());
    part_stays(&mut world);
    world.click(row_center(&world, "MaterialBrowser"));
    assert!(window_geometry("Material Browser").is_some());
    close_title(&mut world, "Material Browser");
    world.close();
}

#[test]
fn dockable_tool_windows_can_be_docked_together() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    world.click_text("Tools");
    world.click_text("Application Options");
    world.click(row_center(&world, "MaterialBrowser"));
    let (source, source_size) = window_geometry("Material Browser").unwrap();
    let (target, target_size) = window_geometry("Application Options").unwrap();
    world.save_capture("dockable-before-drag");
    let from = [source[0] + source_size[0] * 0.4, source[1] + 10.];
    let to = [target[0] + target_size[0] * 0.5, target[1] + 10.];
    world.drag(
        super::super::diagnostics::pointer_pixels(from),
        super::super::diagnostics::pointer_pixels(to),
    );
    world.pump(Vec::new());
    world.save_capture("dockable-after-drag");
    let first = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    let second = super::super::diagnostics::window_dock_id("Application Options").unwrap();
    let mouse_down = unsafe { imgui::sys::igGetIO().as_ref().unwrap().MouseDown[0] };
    assert!(
        first != 0 && first == second,
        "expected one shared dock node, got {first} and {second}; mouse down {mouse_down}; source {source:?} {source_size:?}, target {target:?} {target_size:?}, after {:?}",
        window_geometry("Material Browser")
    );
    world.app().window.native.view.resize([1000, 700]).unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(first)
    );
    assert_eq!(
        super::super::diagnostics::window_dock_id("Application Options"),
        Some(first)
    );
    let (minimum, maximum) = super::super::diagnostics::window_dock_tab_bounds("Material Browser")
        .expect("expected the browser's dock tab");
    let height = maximum[1] - minimum[1];
    let close = [maximum[0] - height * 0.5, (minimum[1] + maximum[1]) * 0.5];
    world.click(super::super::diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.save_capture("dockable-after-close");
    assert!(window_geometry("Material Browser").is_none());
    assert!(window_geometry("Application Options").is_some());
    world.close();
}

#[test]
fn quick_access_measure_starts_and_closes_the_real_measure_tool() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    open_customization(&mut world);
    world.click_text("Measure");
    world.pump(Vec::new());
    assert!(on_row(&world, "Measure"));
    world.click(row_center(&world, "Measure"));
    world.pump(Vec::new());
    assert!(world.state().measure.is_some());
    assert!(window_geometry("Measure").is_some());
    close_title(&mut world, "Measure");
    assert!(world.state().measure.is_none());
    world.close();
}

fn has_motor_title(world: &mut ApplicationWorld) -> bool {
    use oblikovati_api::types::VariantValue;
    world
        .state()
        .document_properties
        .iter()
        .any(|p| p.name == "Title" && p.value.value == VariantValue::String("Motor".into()))
}
fn create_motor_title(world: &mut ApplicationWorld) {
    open_customization(world);
    world.click_text("iProperties");
    world.pump(Vec::new());
    assert!(on_row(world, "iProperties"));
    world.click(row_center(world, "iProperties"));
    assert!(window_geometry("iProperties").is_some());
    enter_parameter_field(world, "Name", "Title");
    enter_parameter_field(world, "Value", "Motor");
    world.click_text("Set property");
    world.pump(Vec::new());
    assert!(has_motor_title(world));
    close_title(world, "iProperties");
}
#[test]
fn quick_access_iproperties_edits_and_undoes_document_metadata() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    create_motor_title(&mut world);
    world.click(row_center(&world, "Undo"));
    world.pump(Vec::new());
    assert!(!has_motor_title(&mut world));
    world.click(row_center(&world, "Redo"));
    world.pump(Vec::new());
    assert!(has_motor_title(&mut world));
    world.close();
}

#[test]
fn iproperties_closes_when_the_active_document_changes() {
    let mut world = ApplicationWorld::new();
    part_and_assembly(&mut world);
    let part = world.parts[0];
    world.click_tab(part);
    world.pump(Vec::new());
    open_customization(&mut world);
    world.click_text("iProperties");
    world.click(row_center(&world, "iProperties"));
    assert!(window_geometry("iProperties").is_some());
    world.click_tab(world.other_document.unwrap());
    world.pump(Vec::new());
    assert!(window_geometry("iProperties").is_none());
    assert!(world.state().document_properties.is_empty());
    world.close();
}

#[test]
fn iproperties_text_survives_save_and_reopen() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    create_motor_title(&mut world);
    world.press_save_shortcut();
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    assert!(has_motor_title(&mut world));
    world.close();
}

fn add_typed_property(world: &mut ApplicationWorld, name: &str, kind: &str, value: &str) {
    world.click(row_center(world, "iProperties"));
    enter_parameter_field(world, "Name", name);
    world.click_text("Text");
    world.click_text(kind);
    enter_parameter_field(world, "Value", value);
    assert!(world.run_is_enabled("Set property"));
    world.click_text("Set property");
    world.pump(Vec::new());
    close_title(world, "iProperties");
}

fn typed_property(
    world: &mut ApplicationWorld,
    name: &str,
) -> Option<oblikovati_api::types::VariantValue> {
    world
        .state()
        .document_properties
        .into_iter()
        .find(|property| property.name == name)
        .map(|property| property.value.value)
}

#[test]
fn iproperties_typed_values_undo_redo_and_reopen() {
    use oblikovati_api::{json::Bytes, types::VariantValue};
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    open_customization(&mut world);
    world.click_text("iProperties");
    for (name, kind, text, value) in [
        ("Count", "Integer", "-12", VariantValue::Integer(-12)),
        ("Factor", "Number", "1.25", VariantValue::Double(1.25)),
        ("Approved", "Boolean", "true", VariantValue::Boolean(true)),
        (
            "Payload",
            "Bytes (hex)",
            "00af",
            VariantValue::Bytes(Bytes(Some(vec![0, 175]))),
        ),
    ] {
        add_typed_property(&mut world, name, kind, text);
        assert_eq!(typed_property(&mut world, name), Some(value));
    }
    world.click(row_center(&world, "Undo"));
    assert_eq!(typed_property(&mut world, "Payload"), None);
    world.click(row_center(&world, "Redo"));
    assert_eq!(
        typed_property(&mut world, "Payload"),
        Some(VariantValue::Bytes(Bytes(Some(vec![0, 175]))))
    );
    add_typed_property(&mut world, "Count", "Integer", "17");
    assert_eq!(
        typed_property(&mut world, "Count"),
        Some(VariantValue::Integer(17))
    );
    world.click(row_center(&world, "Undo"));
    assert_eq!(
        typed_property(&mut world, "Count"),
        Some(VariantValue::Integer(-12))
    );
    world.click(row_center(&world, "Redo"));
    assert_eq!(
        typed_property(&mut world, "Count"),
        Some(VariantValue::Integer(17))
    );
    world.press_save_shortcut();
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    assert_eq!(
        typed_property(&mut world, "Count"),
        Some(VariantValue::Integer(17))
    );
    assert_eq!(
        typed_property(&mut world, "Factor"),
        Some(VariantValue::Double(1.25))
    );
    assert_eq!(
        typed_property(&mut world, "Approved"),
        Some(VariantValue::Boolean(true))
    );
    assert_eq!(
        typed_property(&mut world, "Payload"),
        Some(VariantValue::Bytes(Bytes(Some(vec![0, 175]))))
    );
    world.close();
}

#[test]
fn iproperties_refuses_invalid_numeric_input_without_an_edit() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    open_customization(&mut world);
    world.click_text("iProperties");
    world.click(row_center(&world, "iProperties"));
    enter_parameter_field(&mut world, "Name", "Invalid");
    world.click_text("Text");
    world.click_text("Number");
    enter_parameter_field(&mut world, "Value", "NaN");
    assert!(!world.run_is_enabled("Set property"));
    world.expect_text("Enter a finite number.");
    assert_eq!(typed_property(&mut world, "Invalid"), None);
    world.close();
}
