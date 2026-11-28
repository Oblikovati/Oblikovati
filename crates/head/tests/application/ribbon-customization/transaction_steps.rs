// SPDX-License-Identifier: GPL-2.0-only
//! Exercise staged ribbon commands across the native dialog and its targets.
use super::*;

#[given("the source category is All Commands")]
#[given("the command category is \"All Commands\"")]
#[given("All Commands is selected")]
pub(super) fn all_commands(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "AllCommands");
}

pub(super) fn target_is_empty(world: &ApplicationWorld) {
    let rows = target_rows(world);
    assert!(
        rows.is_empty(),
        "expected an empty custom panel list, found {rows:?}"
    );
}

pub(super) fn target_rows(world: &ApplicationWorld) -> Vec<&str> {
    target_row_runs(world)
        .into_iter()
        .map(|row| row.text.as_str())
        .collect()
}

pub(super) fn target_row_runs(world: &ApplicationWorld) -> Vec<&diagnostics::RenderedRun> {
    let panel = world.run_in_window("Customize", "Custompanel");
    let large = world.run_in_window("Customize", "Large");
    let x = panel.minimum[0]..large.minimum[0];
    let top = large.maximum[1];
    let inside_target = [large.center()[0], top + 2. * (top - large.minimum[1])];
    let Some(list) = diagnostics::child_draw_list_at("Customize", inside_target) else {
        return Vec::new();
    };
    let mut rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == list && x.contains(&run.minimum[0]))
        .collect();
    rows.sort_by(|a, b| a.center()[1].total_cmp(&b.center()[1]));
    rows
}

pub(super) fn target_label(world: &ApplicationWorld) -> &str {
    let panel = world.run_in_window("Customize", "Custompanel");
    let x = panel.minimum[0];
    let y = panel.center()[1];
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            (run.text.starts_with("Part|") || run.text.starts_with("Assembly|"))
                && (run.minimum[0] - x).abs() < 18.
                && (0. ..32.).contains(&(run.center()[1] - y))
        })
        .map(|run| run.text.as_str())
        .expect("expected the rendered custom panel target selector")
}

pub(super) fn expect_target_pending(world: &ApplicationWorld, pending: bool) {
    let label = target_label(world);
    assert_eq!(label.ends_with('*'), pending, "target selector {label:?}");
}

#[then("the target has a pending-change marker")]
fn target_remains_pending(world: &mut ApplicationWorld) {
    expect_target_pending(world, true);
}

#[then("Apply and cancels are enabled")]
#[then("Apply and cancels remain enabled")]
#[then("Apply and cancels are available")]
#[then("Apply and cancels are available in the customization dialog")]
fn edits_enabled(world: &mut ApplicationWorld) {
    assert_edit_controls(world, true);
}

#[then("Apply and cancels are disabled")]
#[then("Apply and cancels remain disabled")]
fn edits_disabled(world: &mut ApplicationWorld) {
    assert_edit_controls(world, false);
}

fn assert_edit_controls(world: &ApplicationWorld, enabled: bool) {
    for label in ["Apply", "Cancel"] {
        assert_eq!(
            world.run_in_window("Customize", label).alpha == u8::MAX,
            enabled,
            "{label}"
        );
    }
}

#[when("the user clicks Apply")]
#[when("the user clicks \"Apply\"")]
fn apply(world: &mut ApplicationWorld) {
    for window in ["Ruled Surface", "Stitch"] {
        if super::super::diagnostics::window_geometry(window).is_some() {
            let center = world.run_in_window(window, "Apply").center();
            world.click(super::super::diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
            world.pump(Vec::new());
            world.pump(Vec::new());
            return;
        }
    }
    click_in_customize(world, "Apply");
}

#[when("the user closes customization")]
#[when("the user closes customization without editing any commands")]
fn close(world: &mut ApplicationWorld) {
    close_customization_button(world);
}

#[when("the user reopens ribbon customization")]
fn reopen_from_ribbon(world: &mut ApplicationWorld) {
    if world.shows("Customize") {
        open_tools_customization(world);
    } else {
        open_panel_menu(world);
        choose_customize(world);
    }
}

#[when("the user closes and reopens Ribbon customization")]
fn close_and_reopen(world: &mut ApplicationWorld) {
    close(world);
    reopen_from_ribbon(world);
}

#[given(
    "the ribbon customization dialog has a staged \"2D Sketches\" addition to the initially empty \"Part | 3D Model\" custom panel"
)]
fn staged_on_model(world: &mut ApplicationWorld) {
    empty_model_customization(world);
    add_command(world, "2DSketches");
    expect_target_pending(world, true);
}

#[then("no custom panel appears in the visible ribbon")]
fn no_panel(world: &mut ApplicationWorld) {
    no_user_commands(world);
}

#[then("the \"Part | 3D Model\" custom panel list is empty")]
fn model_panel_empty(world: &mut ApplicationWorld) {
    model_target(world);
    target_is_empty(world);
}

#[given("the Part Tools and Part Manage custom panel lists are initially empty")]
#[given("the Part | Tools and Part | Manage custom targets are empty")]
fn empty_tools_and_manage(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Customize").is_none() {
        open_tools_customization(world);
    }
    target_is_empty(world);
    choose_target(world, "Manage");
    target_is_empty(world);
    choose_target(world, "Tools");
}

#[given("the Ribbon customization dialog shows the Part Tools target and All Commands")]
#[given("the Ribbon customization dialog shows Part Tools and All Commands")]
fn tools_all_commands(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools");
    all_commands(world);
}

#[when("the user selects the Part Manage target")]
#[when("the user selects \"Part | Manage\" as the target")]
#[when("the user selects the Part | Manage target")]
#[when("the user selects Part Manage as the target")]
fn select_manage(world: &mut ApplicationWorld) {
    choose_target(world, "Manage");
}

#[then("the target command list is empty")]
#[then("the Part Tools target is empty")]
#[then("the Part | Manage target list is empty")]
#[then("the Part | Tools custom target remains empty")]
#[then("the Part | Tools target remains empty")]
#[then("its target list is empty")]
fn empty_target_list(world: &mut ApplicationWorld) {
    target_is_empty(world);
}

#[given("the Part | Tools custom target is empty")]
fn empty_tools_panel(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Customize").is_some() {
        assert_eq!(target_label(world), "Part|Tools");
        target_is_empty(world);
        return;
    }
    world.click_text("Tools");
    no_user_commands(world);
}

#[when("the user selects the Part Tools target")]
#[when("the user returns to the Part Tools target")]
#[when("the user returns to the Part | Tools target")]
fn select_tools(world: &mut ApplicationWorld) {
    choose_target(world, "Tools");
}

pub(super) fn choose_target(world: &mut ApplicationWorld, tab: &str) {
    if !(world.shows("Part | Annotate") && world.shows("Part | Manage")) {
        let current = target_label(world).to_owned();
        click_in_customize(world, &current);
    }
    world.click_text(&format!("Part | {tab}"));
    world.pump(Vec::new());
    assert_eq!(
        target_label(world).trim_end_matches('*'),
        super::super::world::squeezed(&format!("Part | {tab}"))
    );
}

pub(super) fn add_command(world: &mut ApplicationWorld, name: &str) {
    let run = world.run_in_window("Customize", name);
    let source = world.run_in_window("Customize", "Availablecommands");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(run.center()[0] >= source.minimum[0] && run.center()[0] < target.minimum[0]);
    click_in_customize(world, name);
    transfer_to_target(world);
    expect_target_command(world, name);
}

fn expect_target_command(world: &ApplicationWorld, name: &str) {
    let run = world.run_in_window("Customize", name);
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(
        run.center()[0] >= target.minimum[0],
        "expected {name:?} in the target list"
    );
}

#[when("the user adds \"2D Sketches\" to the Part Tools custom panel")]
#[when("the user adds \"2D Sketches\" to the Part | Tools target without applying")]
#[when("the user adds \"2D Sketches\" to Part Tools")]
fn add_sketches_to_tools(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools");
    add_command(world, "2DSketches");
}

#[when("the user adds \"2D Sketches\" to the Part Manage custom panel")]
fn add_sketches_to_manage(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Manage");
    add_command(world, "2DSketches");
}

#[when("the user adds \"3D Annotations\" to its custom panel")]
fn add_annotations(world: &mut ApplicationWorld) {
    add_command(world, "3DAnnotations");
}

#[then("\"2D Sketches\" appears in the Part Tools custom panel list")]
#[then("\"2D Sketches\" appears in the Part Manage custom panel list")]
#[then("\"2D Sketches\" remains in its custom panel list")]
#[then("its staged 2D Sketches command remains present")]
#[then("\"2D Sketches\" is absent from the visible available commands list")]
fn sketches_in_selected_target(world: &mut ApplicationWorld) {
    expect_target_command(world, "2DSketches");
}

#[then("the Part Tools target still has a pending-change marker")]
#[then("the Part Tools target retains its pending-change marker")]
#[then("the target still shows a pending-change marker")]
fn tools_stays_pending(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world), "Part|Tools*");
}

#[then("the customization dialog remains open on the Part Manage target")]
fn stays_on_manage(world: &mut ApplicationWorld) {
    customization_remains_open(world);
    assert_eq!(target_label(world), "Part|Manage");
}

#[then("its list contains \"3D Annotations\" without a pending-change marker")]
fn annotations_applied(world: &mut ApplicationWorld) {
    expect_target_command(world, "3DAnnotations");
    expect_target_pending(world, false);
}

#[then("its list contains \"2D Sketches\" without a pending-change marker")]
fn sketches_applied(world: &mut ApplicationWorld) {
    expect_target_command(world, "2DSketches");
    expect_target_pending(world, false);
}

#[then("the Tools ribbon displays a User Commands panel")]
#[then("the User Commands panel appears on the Tools ribbon")]
#[then("the User Commands panel remains visible")]
fn tools_panel_visible(world: &mut ApplicationWorld) {
    world.expect_text("User Commands");
}

#[given("the Part Tools custom panel is initially empty")]
fn tools_panel_starts_empty(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_no_text("User Commands");
}

#[given("the ribbon customization dialog is open")]
fn tools_dialog_open(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Customize").is_none() {
        open_tools_customization(world);
    }
    customization_remains_open(world);
}

#[when("the user adds \"2D Sketches\" to the custom panel and clicks \"Apply\"")]
fn apply_sketches(world: &mut ApplicationWorld) {
    add_command(world, "2DSketches");
    apply(world);
}

#[when("the user adds \"3D Annotations\" to the custom panel and clicks \"Cancel\"")]
fn discard_annotations(world: &mut ApplicationWorld) {
    add_command(world, "3DAnnotations");
    cancel_customization(world);
}

#[then("the Part Tools custom panel contains \"2D Sketches\"")]
fn tools_has_sketches(world: &mut ApplicationWorld) {
    assert_eq!(target_label(world).trim_end_matches('*'), "Part|Tools");
    expect_target_command(world, "2DSketches");
}

#[then("\"3D Annotations\" is available in the source list")]
#[then("3D Annotations is available in the source list")]
fn annotations_available(world: &mut ApplicationWorld) {
    let command = world.run_in_window("Customize", "3DAnnotations");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(command.center()[0] < target.minimum[0]);
}

#[given("\"2D Sketches\" is selected in the available commands list")]
fn select_sketches(world: &mut ApplicationWorld) {
    select_available_sketches(world);
}

#[when("the user right-clicks \"2D Sketches\" in the available commands list")]
fn secondary_sketches(world: &mut ApplicationWorld) {
    secondary_in_customize(world, "2DSketches");
}

#[when("the user right-clicks \"3D Annotations\" in the available commands list")]
fn secondary_annotations(world: &mut ApplicationWorld) {
    secondary_in_customize(world, "3DAnnotations");
}

fn secondary_in_customize(world: &mut ApplicationWorld, name: &str) {
    let center = world.run_in_window("Customize", name).center();
    world.right_click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.hover([800, 900]);
}

#[then("\"3D Annotations\" is highlighted instead of \"2D Sketches\"")]
fn only_annotations_selected(world: &mut ApplicationWorld) {
    assert!(world.row_accented_beside("3D Annotations", "2D Sketches"));
    assert!(!world.row_accented_beside("2D Sketches", "<Separator>"));
}

#[given("\"2D Sketches\" has been staged in the target list")]
#[given("2D Sketches has been staged in the target list")]
#[given("\"2D Sketches\" is staged in the target list")]
#[given("the user has staged \"2D Sketches\" as the only command")]
fn sketches_staged(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
}

#[when("the user right-clicks the \"2D Sketches\" name in the target list")]
fn secondary_target_sketches(world: &mut ApplicationWorld) {
    expect_target_command(world, "2DSketches");
    secondary_in_customize(world, "2DSketches");
}

#[then("the target row is highlighted")]
fn target_row_selected(world: &mut ApplicationWorld) {
    let row = world.run_in_window("Customize", "2DSketches").clone();
    let x = row.maximum[0] + 8.;
    let y = row.center()[1];
    let selected = diagnostics::pointer_pixels([x, y]).map(|value| value as usize);
    let blank = diagnostics::pointer_pixels([x, y + 30.]).map(|value| value as usize);
    assert_ne!(world.pixel_at(selected), world.pixel_at(blank));
    assert_eq!(world.run_in_window("Customize", "<").alpha, u8::MAX);
}

#[test]
fn secondary_click_replaces_the_staged_target_selection() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_two_commands(&mut world);
    click_in_customize(&mut world, "2DSketches");
    secondary_in_customize(&mut world, "3DAnnotations");
    transfer_to_source(&mut world);
    expect_target_command(&world, "2DSketches");
    annotations_available(&mut world);
    cancel_customization(&mut world);
    world.close();
}
