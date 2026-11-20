// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for context-sensitive Sketch ribbon commands.
use super::workspace_layout::browser_feature::{SKETCH, right_click_row};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

#[path = "document_kinds.rs"]
mod document_kinds;

#[given("sketch edit is inactive")]
fn sketch_edit_inactive(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    assert!(world.app().owner.active_sketch_edit().is_none());
    assert!(!world.state().browser.unwrap().sketches.is_empty());
}
#[when("the user selects the Sketch ribbon tab")]
#[when("the user selects the Sketch ribbon tab again")]
#[given("the Sketch ribbon tab is selected")]
fn select_sketch_tab(world: &mut ApplicationWorld) {
    let tab = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Sketch")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("expected Sketch ribbon tab")
        .center();
    world.click(diagnostics::pointer_pixels(tab));
}
#[then("Line, Circle, Arc, Rectangle, Fillet, Text and Point are enabled in Create")]
#[then("Line, Circle, Arc, Rectangle, Fillet, Text and Point remain enabled in Create")]
fn create_commands_available(world: &mut ApplicationWorld) {
    for label in [
        "Line",
        "Circle",
        "Arc",
        "Rectangle",
        "Fillet",
        "Text",
        "Point",
    ] {
        assert!(
            world.run_is_enabled(label),
            "{label}: expected enabled in Create"
        );
    }
}
#[then("Project Geometry is disabled")]
#[then("Project Geometry remains disabled")]
fn project_geometry_unavailable(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Project Geometry"));
}
#[then("the commands in Modify, Pattern, Constrain, Insert and Format are disabled")]
#[then("the commands in Modify, Pattern, Constrain, Insert and Format remain disabled")]
fn other_groups_unavailable(world: &mut ApplicationWorld) {
    for (group, commands) in [
        ("Modify", &["Move", "Delete"][..]),
        ("Pattern", &["Rectangular", "Circular"]),
        (
            "Constrain",
            &[
                "Horizontal",
                "Vertical",
                "Coincident",
                "Fix",
                "Parallel",
                "Perpendicular",
                "Collinear",
                "Equal",
                "Concentric",
                "Tangent",
                "Dimension",
            ],
        ),
        ("Insert", &["Image", "Points"]),
        (
            "Format",
            &[
                "Construction",
                "Centerline",
                "Center Point",
                "Precise Input",
            ],
        ),
    ] {
        inspect_unavailable_group(world, group, commands);
    }
}
fn inspect_unavailable_group(world: &mut ApplicationWorld, group: &str, commands: &[&str]) {
    world.expect_text(group);
    let collapsed = !world.shows(commands[0]);
    if collapsed {
        let label = world.run(group).center();
        world.click(diagnostics::pointer_pixels([label[0], label[1] - 35.]));
    }
    for label in commands {
        let wanted: String = label.chars().filter(|c| !c.is_whitespace()).collect();
        let run = world
            .frame
            .runs
            .iter()
            .filter(|run| {
                run.text
                    .chars()
                    .filter(|c| !c.is_whitespace())
                    .eq(wanted.chars())
            })
            .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
            .unwrap_or_else(|| panic!("{group} / {label}: expected visible command"));
        assert_ne!(
            run.alpha,
            u8::MAX,
            "{group} / {label}: expected unavailable"
        );
    }
    if group == "Format" {
        let before = world.state().center_point_format;
        let run = world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "CenterPoint")
            .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
            .expect("expected Center Point format control")
            .center();
        world.click(diagnostics::pointer_pixels(run));
        assert_eq!(world.state().center_point_format, before);
    }
    if collapsed {
        world.escape();
    }
}
#[then("the active part and model browser remain visible")]
fn part_and_browser_visible(world: &mut ApplicationWorld) {
    assert!(world.state().document.is_some());
    assert!(world.state().browser.is_some());
    world.expect_text(SKETCH);
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}
#[when("the user opens the existing sketch from the model browser")]
fn open_sketch(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
}
#[then("the Sketch ribbon tab is selected")]
fn sketch_tab_selected(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.expect_text("Line");
    world.expect_text("Constrain");
    world.expect_text("Finish Sketch");
}
#[then("the 2D sketch workspace shows the existing profile")]
fn existing_profile_visible(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 4);
    let point = world.line_point(0, 0.5);
    let image = super::capture_framebuffer(&mut world.app().window.native);
    let pixel = |x: i32, y: i32| {
        let index = (y as usize * image.extent[0] as usize + x as usize) * 4;
        &image.rgba[index..index + 3]
    };
    let background = pixel(point[0] + 8, point[1] + 8);
    assert!(
        (-3..=3).any(|y| (-3..=3).any(|x| pixel(point[0] + x, point[1] + y) != background)),
        "expected the existing profile in the viewport"
    );
}
#[then("Create, Modify, Pattern, Constrain, Insert and Format groups are displayed")]
fn editing_groups_visible(world: &mut ApplicationWorld) {
    for label in [
        "Create",
        "Modify",
        "Pattern",
        "Constrain",
        "Insert",
        "Format",
    ] {
        world.expect_text(label);
    }
}
#[then("Finish is available to exit sketch editing")]
fn finish_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Finish Sketch"));
}
#[when("the user selects Line")]
fn select_line(world: &mut ApplicationWorld) {
    world.click_text("Line");
}
#[then("the Line command prompt requests a start point")]
fn line_prompt(world: &mut ApplicationWorld) {
    world.expect_status("Select start of line");
}
#[then("the Line tooltip indicates line and arc creation")]
fn line_tooltip(world: &mut ApplicationWorld) {
    let line = world.run("Line").clone();
    world.hover(diagnostics::pointer_pixels(line.center()));
    for _ in 0..32 {
        let mut lists = std::collections::BTreeMap::<usize, String>::new();
        for run in &world.frame.runs {
            lists
                .entry(run.list)
                .or_default()
                .push_str(&run.text.to_lowercase());
        }
        if lists
            .iter()
            .any(|(list, text)| *list != line.list && text.contains("line") && text.contains("arc"))
        {
            return;
        }
        world.pump(Vec::new());
    }
    panic!("expected a Line tooltip describing line and arc creation");
}
#[when("the user exits sketch editing with Finish")]
fn finish_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
}
#[then("the previous part workspace and ribbon context return")]
fn part_context_restored(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.expect_text("Start 2D Sketch");
}
#[then("the existing sketch remains in the model browser")]
fn sketch_remains(world: &mut ApplicationWorld) {
    world.expect_text(SKETCH);
    assert!(!world.state().browser.unwrap().sketches.is_empty());
}

#[test]
fn insert_group_shows_unavailable_commands_during_sketch_edit() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    open_sketch(&mut world);
    world.expect_text("Insert");
    assert!(!world.run_is_enabled("Project Geometry"));
    if !world.shows("Image") {
        let label = world.run("Insert").center();
        world.click(diagnostics::pointer_pixels([label[0], label[1] - 35.]));
    }
    for label in ["Image", "Points"] {
        assert!(!world.run_is_enabled(label));
    }
    world.close();
}

#[test]
fn finishing_an_existing_sketch_restores_the_prior_tools_tab() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    world.click_text("Tools");
    world.expect_text("Application Options");
    open_sketch(&mut world);
    world.expect_text("Finish Sketch");
    world.finish_sketch();
    world.expect_text("Application Options");
    world.expect_text(SKETCH);
    world.close();
}

#[test]
fn non_create_groups_are_unavailable_outside_sketch_edit() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    select_sketch_tab(&mut world);
    project_geometry_unavailable(&mut world);
    other_groups_unavailable(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn line_from_part_enters_the_only_sketch_without_committing_geometry() {
    let mut world = ApplicationWorld::new();
    super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let before = world.state();
    let sketch = before.browser.unwrap().sketches[0].id;
    select_sketch_tab(&mut world);
    assert!(world.run_is_enabled("Line"));
    super::x11_input::focus_window();
    super::x11_input::click(diagnostics::pointer_pixels(world.run("Line").center()));
    world.pump(Vec::new());
    assert_eq!(world.state().active_sketch, Some(sketch));
    line_prompt(&mut world);
    world.save_capture("line-from-part");
    super::x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_status("Ready");
    assert_eq!(world.edited_geometry().lines().len(), 4);
    assert_eq!(world.state().undo_labels, before.undo_labels);
    world.close();
}
