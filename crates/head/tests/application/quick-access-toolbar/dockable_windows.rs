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
    let run = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == label)
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .unwrap();
    let point = [run.minimum[0] - 40., run.center()[1]];
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
