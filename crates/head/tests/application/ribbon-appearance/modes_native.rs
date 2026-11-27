// SPDX-License-Identifier: GPL-2.0-only
//! Window-level checks for ribbon appearance and retained document state.
use super::*;

#[test]
fn appearance_choices_reflow_model_commands_and_restore_labels() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    for appearance in ["Text Off", "Small", "Compact", "Large"] {
        let title = diagnostics::pointer_pixels(world.run("Modify").center());
        world.right_click(title);
        world.click_text("Ribbon Appearance");
        assert!(
            world.run_is_enabled(appearance),
            "{appearance} must be available"
        );
        world.click_text(appearance);
        world.pump(Vec::new());
        world.expect_no_text("Extrude");
        world.expect_no_text("UCS");
        world.expect_text("Modify");
        world.save_capture(&format!("appearance-{appearance}"));
    }
    let title = diagnostics::pointer_pixels(world.run("Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
    assert!(world.checked("Large"));
    world.click_text("Normal");
    world.pump(Vec::new());
    world.expect_text("Extrude");
    world.close();
}

#[test]
fn command_icons_change_size_and_row_count_with_the_appearance() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    choose_appearance(&mut world, "Modify", "Text Off");
    let large = super::icon_quads::ribbon_icon_quads(ribbon_bottom());
    let large_height = ribbon_bottom();
    assert!(large.iter().any(|(min, max)| max[0] - min[0] >= 39.));
    choose_appearance(&mut world, "Modify", "Small");
    let small = super::icon_quads::ribbon_icon_quads(ribbon_bottom());
    assert!(ribbon_bottom() < large_height);
    assert!(small.len() >= 4 && small.iter().all(|(min, max)| max[0] - min[0] < 30.));
    choose_appearance(&mut world, "Modify", "Compact");
    let compact = super::icon_quads::ribbon_icon_quads(ribbon_bottom());
    assert!(compact.len() >= 4);
    let row = compact[0].0[1];
    assert!(compact.iter().all(|(min, _)| (min[1] - row).abs() < 1.));
    choose_appearance(&mut world, "Modify", "Large");
    let large = super::icon_quads::ribbon_icon_quads(ribbon_bottom());
    assert!(large.len() >= 4 && large.iter().all(|(min, max)| max[0] - min[0] >= 39.));
    assert!(
        large
            .iter()
            .any(|(min, _)| (min[1] - large[0].0[1]).abs() > 20.)
    );
    world.close();
}

#[test]
fn appearance_is_restored_after_reopening_the_application() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    choose_appearance(&mut world, "Modify", "Compact");
    world.reopen_application();
    world.add_part();
    open_appearance_from(&mut world, "Modify");
    assert!(world.checked("Compact"));
    world.close();
}

#[test]
fn appearance_changes_keep_the_editable_sketch_unchanged() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    world.record_initial_sketch();
    for appearance in ["Small", "Compact", "Large"] {
        choose_appearance(&mut world, "Create", appearance);
        assert_eq!(
            world.state().authoring,
            oblikovati_gui::GuiAuthoringContext::Sketch
        );
        let current = world.edited_geometry();
        assert!(current.same_recipe(&world.observed_sketch.as_ref().unwrap().geometry));
        assert_no_ribbon_labels(
            &world,
            &[
                "Rectangle",
                "Horizontal",
                "Construction",
                "Center Point",
                "Move",
                "Delete",
                "Image",
                "Points",
            ],
        );
        world.save_capture(&format!("sketch-appearance-{appearance}"));
    }
    choose_appearance(&mut world, "Create", "Normal");
    world.expect_text("Rectangle");
    world.close();
}

#[test]
fn small_sketch_rectangle_icon_starts_and_cancels_the_real_tool() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    choose_appearance(&mut world, "Create", "Small");
    let title = world.run("Create").clone();
    let icon = [title.minimum[0] + 10., title.minimum[1] - 44.];
    world.click(diagnostics::pointer_pixels(icon));
    world.pump(Vec::new());
    assert!(
        !world.state().sketch_at_ready,
        "rectangle icon must start a sketch command"
    );
    world.escape();
    world.pump(Vec::new());
    assert!(
        world.state().sketch_at_ready,
        "Escape must cancel the sketch command"
    );
    world.close();
}

#[test]
fn icon_only_tools_persists_across_tab_changes() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    select_tab(&mut world, "Tools");
    choose_appearance(&mut world, "Options", "Text Off");
    assert_no_ribbon_labels(
        &world,
        &[
            "Measure",
            "Application Options",
            "Document Settings",
            "Find",
        ],
    );
    world.expect_text("Options");
    assert!(super::icon_quads::ribbon_icon_quads(ribbon_bottom()).len() >= 4);
    select_tab(&mut world, "3D Model");
    assert_no_ribbon_labels(&world, &["Extrude", "Revolve", "UCS"]);
    select_tab(&mut world, "Tools");
    open_appearance_from(&mut world, "Options");
    assert!(world.checked("Text Off"));
    world.close();
}

#[test]
fn compact_drawing_ribbon_keeps_its_document_and_commands() {
    let mut world = ApplicationWorld::new();
    let drawing = world.add_drawing();
    choose_appearance(&mut world, "Create", "Compact");
    assert_eq!(world.active_id(), drawing);
    assert_no_ribbon_labels(
        &world,
        &[
            "Base View",
            "Projected View",
            "Break",
            "Crop",
            "Start Sketch",
            "New Sheet",
        ],
    );
    assert!(super::icon_quads::ribbon_icon_quads(ribbon_bottom()).len() >= 4);
    world.save_capture("drawing-appearance-compact");
    world.close();
}

#[test]
fn small_assembly_ribbon_keeps_its_document_and_commands() {
    let mut world = ApplicationWorld::new();
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    choose_appearance(&mut world, "Component", "Small");
    assert_eq!(world.active_id(), assembly);
    assert_no_ribbon_labels(&world, &["Place", "Create"]);
    assert!(super::icon_quads::ribbon_icon_quads(ribbon_bottom()).len() >= 2);
    world.save_capture("assembly-appearance-small");
    world.close();
}

#[test]
fn small_sheet_metal_ribbon_retains_icon_commands() {
    let mut world = ApplicationWorld::new();
    let part = world.add_template(oblikovati_gui::GuiTemplateKind::SheetMetal);
    choose_appearance(&mut world, "Sheet Metal", "Small");
    assert_eq!(world.active_id(), part);
    assert_no_ribbon_labels(&world, &["Flange", "Flat Pattern"]);
    assert!(super::icon_quads::ribbon_icon_quads(ribbon_bottom()).len() >= 2);
    world.close();
}

#[test]
fn compact_presentation_ribbon_retains_insert_command() {
    let mut world = ApplicationWorld::new();
    let presentation = world.add_template(oblikovati_gui::GuiTemplateKind::Presentation);
    choose_appearance(&mut world, "Workshop", "Compact");
    assert_eq!(world.active_id(), presentation);
    assert_no_ribbon_labels(&world, &["Insert Model"]);
    assert!(!super::icon_quads::ribbon_icon_quads(ribbon_bottom()).is_empty());
    world.close();
}

#[test]
fn icon_only_view_keeps_window_controls_and_theme_selectors_available() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    select_tab(&mut world, "View");
    choose_appearance(&mut world, "Windows", "Compact");
    assert_no_ribbon_labels(&world, &["User Interface", "Switch", "New", "Edit Theme"]);
    world.expect_text("Windows");
    world.expect_text("Theme");
    world.expect_text("Colour scheme");
    assert!(super::icon_quads::ribbon_icon_quads(ribbon_bottom()).len() >= 4);
    world.save_capture("view-appearance-compact");
    world.close();
}

#[test]
fn modify_additional_commands_reflow_without_closing() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Modify");
    world.expect_text("Move Bodies");
    super::super::ribbon_panel_expansion::toggle_pin(&mut world);
    let normal = diagnostics::window_geometry("##ribbon-additional-PartModify").unwrap();
    choose_appearance(&mut world, "Modify", "Text Off");
    let compact = diagnostics::window_geometry("##ribbon-additional-PartModify").unwrap();
    assert!(
        compact.1[0] < normal.1[0],
        "normal {normal:?}, text-off {compact:?}"
    );
    for label in ["Move Bodies", "Bend Part", "Copy Object"] {
        world.expect_no_text(label);
    }
    choose_appearance(&mut world, "Modify", "Normal");
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.expect_text("Move Bodies");
    world.close();
}

#[test]
fn simplification_expansion_keeps_icons_and_no_command_labels() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Simplification");
    world.expect_text("Remove Details");
    choose_appearance(&mut world, "Modify", "Text Off");
    let (origin, size) =
        diagnostics::window_geometry("##ribbon-additional-PartSimplification").unwrap();
    let end = [origin[0] + size[0], origin[1] + size[1]];
    assert!(super::icon_quads::icon_quads_inside((origin, end)).len() >= 3);
    for label in ["Remove Details", "Fill Voids", "Define Envelopes"] {
        world.expect_no_text(label);
    }
    world.close();
}

#[test]
fn constrain_expansion_keeps_icons_and_no_command_labels() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    choose_appearance(&mut world, "Create", "Small");
    world.click_text("Constrain");
    assert!(diagnostics::window_geometry("##ribbon-additional-Constrain").is_some());
    choose_appearance(&mut world, "Create", "Text Off");
    let (origin, size) = diagnostics::window_geometry("##ribbon-additional-Constrain").unwrap();
    let end = [origin[0] + size[0], origin[1] + size[1]];
    assert!(super::icon_quads::icon_quads_inside((origin, end)).len() >= 2);
    for label in ["Edit Coordinate System", "Constraint Inference Scope"] {
        world.expect_no_text(label);
    }
    world.close();
}

pub(super) fn ribbon_bottom() -> f32 {
    let (origin, size) = diagnostics::window_geometry("##application-ribbon")
        .expect("expected the visible ribbon band");
    origin[1] + size[1]
}

pub(super) fn assert_no_ribbon_labels(world: &ApplicationWorld, labels: &[&str]) {
    let title_top = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            [
                "Create",
                "Modify",
                "Options",
                "Windows",
                "Visibility",
                "Component",
                "Workshop",
                "Sheet Metal",
                "SheetMetal",
            ]
            .iter()
            .any(|label| diagnostics::panel_caption_matches(&run.text, label))
                && run.minimum[1] >= 58.
        })
        .map(|run| run.minimum[1])
        .min_by(f32::total_cmp)
        .unwrap_or_else(|| panic!("expected a ribbon panel title in {}", world.text()));
    for &label in labels {
        assert!(
            !world.frame.runs.iter().any(|run| run.minimum[1] >= 58.
                && run.minimum[1] < title_top - 2.
                && !run.text.is_empty()
                && label.starts_with(&run.text)),
            "visible command label {label:?}"
        );
    }
}

pub(super) fn open_appearance_from(world: &mut ApplicationWorld, panel: &str) {
    let title = diagnostics::pointer_pixels(world.run(panel).center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
}
pub(super) fn choose_appearance(world: &mut ApplicationWorld, panel: &str, appearance: &str) {
    open_appearance_from(world, panel);
    assert!(world.run_is_enabled(appearance));
    world.click_text(appearance);
    world.pump(Vec::new());
}
