// SPDX-License-Identifier: GPL-2.0-only
//! Sketch Constrain expansion while the part is outside sketch edit.
use super::super::workspace_layout::browser_feature::{
    expect_solid_visible, open_visible_extruded_part,
};
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};
use cucumber::{given, then, when};

#[path = "sketch_constrain_finish.rs"]
mod finish_checks;
use finish_checks::{edit_sketch, finish_sketch, finish_unavailable_outside_edit};
#[path = "sketch_constrain_pinned.rs"]
mod pinned_checks;

const WINDOW: &str = "##ribbon-additional-Constrain";

#[given("a part is active in the MCAD Software outside sketch edit")]
fn active_part(world: &mut ApplicationWorld) {
    world.add_part();
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}
#[given("the full Sketch ribbon is selected")]
fn sketch_tab(world: &mut ApplicationWorld) {
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        click_sketch_tab(world);
    }
    world.expect_text("Constrain");
}
#[given("the Constrain additional-command area is closed")]
fn constrain_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[when("the user opens the Constrain additional-command area")]
pub(super) fn open_constrain(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Constrain");
}
#[then("the additional area appears below the Sketch ribbon")]
fn area_below_ribbon(_: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(WINDOW).expect("expected Constrain expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(position[1] >= ribbon[1] + size[1]);
}
#[then("its visible commands are unavailable outside sketch edit")]
fn commands_unavailable(world: &mut ApplicationWorld) {
    for label in ["Edit Coordinate System", "Constraint Inference Scope"] {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected unavailable"
        );
    }
}
#[then("the Sketch additional area closes")]
fn area_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[then("the 3D Model ribbon is visible")]
fn model_visible(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[when("the user returns to the Sketch ribbon tab")]
fn return_sketch(world: &mut ApplicationWorld) {
    click_sketch_tab(world);
}
fn click_sketch_tab(world: &mut ApplicationWorld) {
    let center = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Sketch")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("expected the Sketch tab")
        .center();
    world.click(diagnostics::pointer_pixels(center));
}
#[then("the Sketch ribbon is visible with its Constrain additional area closed")]
fn sketch_area_stays_closed(world: &mut ApplicationWorld) {
    world.expect_text("Constrain");
    area_closes(world);
}
#[cfg(target_os = "linux")]
#[test]
fn x11_switching_tabs_closes_unpinned_constrain_area() {
    let mut world = ApplicationWorld::new();
    active_part(&mut world);
    sketch_tab(&mut world);
    let title = diagnostics::pointer_pixels(world.run("Constrain").center());
    super::super::x11_input::click(title);
    world.pump(Vec::new());
    area_below_ribbon(&mut world);
    let tab = diagnostics::pointer_pixels(world.run("3D Model").center());
    super::super::x11_input::click(tab);
    world.pump(Vec::new());
    area_closes(&mut world);
    model_visible(&mut world);
    world.close();
}
#[given("a part contains an existing editable sketch")]
fn part_with_sketch(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
}
#[given("the full Sketch ribbon is visible outside sketch edit")]
fn sketch_visible_outside_edit(world: &mut ApplicationWorld) {
    sketch_tab(world);
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    finish_unavailable_outside_edit(world);
}
#[then("sketch commands become available and the Exit group shows Finish")]
fn sketch_commands_available(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert!(world.run_is_enabled("Finish Sketch"));
    assert!(world.run_is_enabled("Line"));
}
#[then("Edit Coordinate System and Constraint Inference Scope are visible")]
fn constrain_commands_visible(world: &mut ApplicationWorld) {
    world.expect_text("Edit Coordinate System");
    world.expect_text("Constraint Inference Scope");
}
#[then("sketch edit ends and the part view returns")]
fn returns_to_part(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    assert!(world.app().owner.active_sketch_edit().is_none());
    expect_solid_visible(world);
}
#[then("the Constrain additional-command area closes")]
fn constrain_area_closes(world: &mut ApplicationWorld) {
    area_closes(world);
}
#[then("the original part geometry remains visible")]
fn original_part_visible(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}

#[given("a part containing an existing sketch is active in the MCAD Software")]
fn existing_sketch_part(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    world.previous_document = Some(world.active_id());
}
#[given("the full Sketch ribbon is selected outside sketch edit")]
fn select_sketch_outside_edit(world: &mut ApplicationWorld) {
    sketch_visible_outside_edit(world);
}
#[given("the Create additional-command area is closed")]
fn create_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-Create").is_none());
}
#[when("the user opens the Create additional-command area")]
fn open_create(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Create");
}
#[then("Create Block and Precise Input are visible but unavailable")]
fn create_commands_unavailable(world: &mut ApplicationWorld) {
    let window = "##ribbon-additional-Create";
    for label in ["CreateBlock", "PreciseInput"] {
        let run = world.run_in_window(window, label);
        assert!(run.alpha < u8::MAX, "{label}: expected unavailable");
    }
}
#[then("no sketch edit is active")]
fn sketch_edit_inactive(world: &mut ApplicationWorld) {
    assert!(world.app().owner.active_sketch_edit().is_none());
}

#[given("a part has an active editable sketch in the MCAD Software")]
#[given("a part contains an active editable sketch in the MCAD Software")]
pub(super) fn active_editable_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    world.record_initial_sketch();
}
#[given("the Sketch Create panel's additional command area is open")]
fn create_area_open(world: &mut ApplicationWorld) {
    open_create(world);
}
#[when("the user selects Create Block from that area")]
#[when("I select Create Block from that area")]
fn choose_create_block(world: &mut ApplicationWorld) {
    let control = world.run_in_window("##ribbon-additional-Create", "CreateBlock");
    assert_eq!(control.alpha, u8::MAX);
    world.click(diagnostics::pointer_pixels(control.center()));
    world.pump(Vec::new());
}
#[then("the Create Block dialog opens over the sketch")]
fn create_dialog_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Create Block").is_some());
}
#[then("the dialog offers geometry and insert-point selectors")]
fn create_dialog_selectors(world: &mut ApplicationWorld) {
    let text = world.window_text("Create Block");
    assert!(text.contains("Geometry"));
    assert!(text.contains("InsertPoint"));
}
#[then("the block name and description fields are visible")]
fn create_dialog_fields(world: &mut ApplicationWorld) {
    let text = world.window_text("Create Block");
    assert!(text.contains("BlockName"));
    assert!(text.contains("Description"));
    assert!(text.contains("Block1"));
}
fn cancel_create_dialog(world: &mut ApplicationWorld) {
    let cancel = world.run_in_window("Create Block", "Cancel");
    world.click(diagnostics::pointer_pixels(cancel.center()));
    world.pump(Vec::new());
}
#[then("the dialog closes without creating a block")]
fn create_dialog_closed(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Create Block").is_none());
    assert!(world.app().owner.active_sketch_edit().is_some());
}
#[then("the sketch remains active and unchanged")]
fn sketch_remains_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}

#[test]
fn create_block_opens_and_cancels_through_the_native_ribbon() {
    let mut world = ApplicationWorld::new();
    active_editable_sketch(&mut world);
    let before = world.state().undo_labels;
    create_area_open(&mut world);
    choose_create_block(&mut world);
    create_dialog_open(&mut world);
    create_dialog_selectors(&mut world);
    create_dialog_fields(&mut world);
    cancel_create_dialog(&mut world);
    create_dialog_closed(&mut world);
    sketch_remains_active(&mut world);
    assert_eq!(world.state().undo_labels, before);
    world.close();
}

#[test]
fn create_block_groups_four_mouse_selected_edges_and_supports_undo_redo() {
    use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let original = world.edited_geometry();
    assert_eq!(original.lines().len(), 4);
    let segments: Vec<_> = (0..4)
        .map(|index| [world.line_point(index, 0.3), world.line_point(index, 0.7)])
        .collect();
    create_area_open(&mut world);
    choose_create_block(&mut world);
    let (position, _) = diagnostics::window_geometry("Create Block").unwrap();
    world.drag(
        [(position[0] + 80.) as i32, (position[1] + 10.) as i32],
        [360, 210],
    );
    for index in 0..4 {
        let point = visible_line_point(&mut world, index);
        world.click(point);
        let selection = world.state().sketch_selection.entities;
        assert_eq!(selection.len(), index + 1);
        assert!(
            selection
                .iter()
                .any(|(_, entity)| *entity == original.lines()[index].id)
        );
    }
    let ok = world.run_in_window("Create Block", "OK");
    assert_eq!(ok.alpha, u8::MAX);
    world.click(diagnostics::pointer_pixels(ok.center()));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Create Block").is_none());
    assert_eq!(world.edited_geometry().lines().len(), 0);
    assert_eq!(world.state().sketch_block_names, ["Block1"]);
    world.expect_text("Blocks");
    world.expect_text("Block1:1");
    for [from, to] in &segments {
        world.expect_segment_drawn(*from, *to);
    }
    let midpoint = std::array::from_fn(|axis| (segments[0][0][axis] + segments[0][1][axis]) / 2);
    world.click(midpoint);
    let picked = world.state().sketch_selection.entities;
    assert_eq!(picked.len(), 1);
    assert_eq!(
        picked[0].1,
        world.state().browser.unwrap().block_instances[0].id
    );
    world.click([1150, 650]);
    assert!(world.state().sketch_selection.entities.is_empty());
    let unselected = super::super::capture_framebuffer(&mut world.app().window.native);
    world.click_text("Block1:1");
    let selection = world.state().sketch_selection.entities;
    assert_eq!(selection.len(), 1);
    let browser = world.state().browser.unwrap();
    assert_eq!(selection[0].1, browser.block_instances[0].id);
    let selected = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(segment_color_changed(&unselected, &selected, segments[0]));
    world.key(ViewportPhysicalKey::Delete, false);
    assert!(world.state().browser.unwrap().block_instances.is_empty());
    assert_eq!(world.state().sketch_block_names, ["Block1"]);
    world.expect_no_text("Block1:1");
    world.expect_segment_absent(segments[0][0], segments[0][1]);
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_text("Block1:1");
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    world.expect_no_text("Block1:1");
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_text("Block1:1");
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    assert!(world.edited_geometry().same_recipe(&original));
    assert!(world.state().sketch_block_names.is_empty());
    world.expect_no_text("Block1:1");
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    assert_eq!(world.edited_geometry().lines().len(), 0);
    assert_eq!(world.state().sketch_block_names, ["Block1"]);
    world.expect_text("Block1:1");
    for [from, to] in segments {
        world.expect_segment_drawn(from, to);
    }
    world.close();
}
pub(super) fn segment_color_changed(
    before: &oblikovati_viewport::ViewportCapture,
    after: &oblikovati_viewport::ViewportCapture,
    segment: [[i32; 2]; 2],
) -> bool {
    let point: [i32; 2] = std::array::from_fn(|axis| (segment[0][axis] + segment[1][axis]) / 2);
    (-4..=4).any(|dy| {
        (-4..=4).any(|dx| {
            let x = (point[0] + dx) as usize;
            let y = (point[1] + dy) as usize;
            let index = (y * before.extent[0] as usize + x) * 4;
            before.rgba[index..index + 4] != after.rgba[index..index + 4]
        })
    })
}

#[test]
fn viewport_picks_rotated_and_scaled_block_instance() {
    use crate::HeadWindow;
    use oblikovati_api::{client::Client, wire::AddSketchBlockArgs};
    use oblikovati_model::{document_sketch::SketchBlockInstance, sketch::SketchEntityId};
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let geometry = world.edited_geometry();
    let line = geometry.lines()[0];
    let endpoints = line
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let midpoint = std::array::from_fn(|axis| (endpoints[0][axis] + endpoints[1][axis]) / 2.);
    let desired = world
        .app()
        .window
        .sketch_point([1100, 550], edit.plane)
        .unwrap()
        .unwrap();
    let transform = SketchBlockInstance::new(
        SketchEntityId(1),
        "Rib",
        [0., 0.],
        std::f64::consts::FRAC_PI_2,
        -2.,
    )
    .unwrap();
    let transformed = transform.to_sketch(midpoint).unwrap();
    let translation: [f64; 2] = std::array::from_fn(|axis| desired[axis] - transformed[axis]);
    let scale = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .parameters
        .units()
        .working_scale();
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Rib", edit.sketch, &[line.id])
        })
        .unwrap();
    let added = Client::new(world.app().owner.document_host())
        .sketch()
        .add_block_instance(&AddSketchBlockArgs {
            definition: "Rib".into(),
            position: Some(translation.map(|value| value * scale).to_vec()),
            rotation_angle: "90 deg".into(),
            scale: -2.,
            ..Default::default()
        })
        .unwrap();
    let second = Client::new(world.app().owner.document_host())
        .sketch()
        .add_block_instance(&AddSketchBlockArgs {
            definition: "Rib".into(),
            position: Some(vec![0., 0.]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_text("Rib:1");
    world.expect_text("Rib:2");
    world.click([1100, 550]);
    assert_eq!(
        world.state().sketch_selection.entities[0].1.0,
        added.entity_id
    );
    world.key(oblikovati_viewport::ViewportPhysicalKey::Delete, false);
    world.expect_no_text("Rib:1");
    world.expect_text("Rib:2");
    let rows = world.state().browser.unwrap().block_instances;
    assert_eq!((rows[0].id.0, rows[0].ordinal), (second.entity_id, 2));
    world.close();
}

fn visible_line_point(world: &mut ApplicationWorld, line: usize) -> [i32; 2] {
    let (position, size) = diagnostics::window_geometry("Create Block").unwrap();
    for fraction in [0.5, 0.3, 0.7, 0.15, 0.85, 0.05, 0.95] {
        let point = world.line_point(line, fraction);
        let outside = (0..2).any(|axis| {
            (point[axis] as f32) < position[axis] - 5.
                || (point[axis] as f32) > position[axis] + size[axis] + 5.
        });
        if outside {
            return point;
        }
    }
    panic!("line {line}: expected a pick outside the dialog");
}

#[test]
fn create_expansion_opens_the_existing_precise_input_for_a_line() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.select_line_tool();
    assert!(world.state().precise_target.is_some());
    open_create(&mut world);
    let control = world.run_in_window("##ribbon-additional-Create", "PreciseInput");
    assert_eq!(control.alpha, u8::MAX);
    let center = diagnostics::pointer_pixels(control.center());
    world.click(center);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    let footer = world.run_in_window("##ribbon-additional-Create", "Create");
    world.click(diagnostics::pointer_pixels(footer.center()));
    let control = world.run_in_window("##application-ribbon", "PreciseInput");
    world.click(diagnostics::pointer_pixels(control.center()));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Precise Input").is_none());
    world.close();
}
