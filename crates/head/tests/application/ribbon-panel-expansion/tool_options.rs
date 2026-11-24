// SPDX-License-Identifier: GPL-2.0-only
//! Open and dismiss the Options expansion and cancel a ribbon layout reset.
use super::super::ribbon_display::{activate_display_control, cycling};
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};
use cucumber::{given, then, when};

const OPTIONS: &str = "##ribbon-additional-ToolOptions";
const RESET: &str = "Reset Ribbon";

#[given("`fixtures/cad/blank-metric-part.opd` is open in the MCAD Software")]
pub(in super::super) fn open_blank_metric_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/blank-metric-part.opd");
    world.open_fixture(&path);
    let id = world.active_id();
    let storage = &world.state().document_storage[&id];
    assert_eq!(
        storage.path.as_deref().map(std::path::Path::new),
        Some(path.as_path())
    );
    world.previous_document = Some(id);
}

#[given(
    "`fixtures/cad/blank-metric-part.opd` is open with the full 3D Model ribbon displayed in the MCAD Software"
)]
fn open_blank_metric_part_with_full_ribbon(world: &mut ApplicationWorld) {
    open_blank_metric_part(world);
    world.expect_text("Extrude");
    world.expect_text("Start 2D Sketch");
}

#[given("the Options additional command area is closed")]
#[then("the Options additional command area closes")]
#[then("the Options additional-command area is closed")]
#[then("the Options additional command area is no longer visible")]
#[then("the Options additional command area is not automatically reopened")]
fn options_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(OPTIONS).is_none());
}

#[given("the Options panel displays its additional commands")]
#[when("the user opens the Options additional command area")]
fn open_options(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Options");
    assert!(diagnostics::window_geometry(OPTIONS).is_some());
}

#[then("it appears beneath the Options panel over the viewport")]
fn options_below_panel(world: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(OPTIONS).expect("expected the Options pane");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    let title = world.run("Options");
    assert!(position[1] >= ribbon[1] + size[1]);
    assert!(position[0] <= title.minimum[0]);
}

#[then("Reset UI Layout is displayed")]
fn reset_is_displayed(world: &mut ApplicationWorld) {
    assert!(world.window_text(OPTIONS).contains("ResetUILayout"));
    assert!(world.run_is_enabled("Reset UI Layout"));
}

#[then("iLogic Configuration is displayed")]
fn ilogic_is_displayed(world: &mut ApplicationWorld) {
    assert!(world.window_text(OPTIONS).contains("iLogicConfiguration"));
    assert!(!world.run_is_enabled("iLogic Configuration"));
}

#[then("Reset UI Layout and iLogic Configuration are displayed")]
fn options_commands_are_displayed(world: &mut ApplicationWorld) {
    reset_is_displayed(world);
    ilogic_is_displayed(world);
}

#[when("the user activates Reset UI Layout")]
fn request_reset(world: &mut ApplicationWorld) {
    world.click_text("Reset UI Layout");
}

#[then("a confirmation dialog warns that the UI layout will be reset to its default")]
fn reset_warning(world: &mut ApplicationWorld) {
    let text = world.window_text(RESET);
    assert!(text.contains("ribbonwillreturntoitsdefaultlayout"));
}

#[then("the dialog offers Yes and No")]
fn reset_choices(world: &mut ApplicationWorld) {
    let text = world.window_text(RESET);
    assert!(text.contains("Yes") && text.contains("No"));
}

#[when("the user clicks No")]
fn cancel_reset(world: &mut ApplicationWorld) {
    world.click_text("No");
}

#[then("the confirmation dialog closes")]
fn reset_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(RESET).is_none());
}

#[given("the full Tools ribbon uses the Cycle Through All display policy")]
fn full_tools_cycling(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    cycling(world);
    world.expect_text("Application Options");
}

#[given("the Options additional command area is open and pinned")]
fn pin_options(world: &mut ApplicationWorld) {
    open_options(world);
    let pin = world.run_in_window(OPTIONS, "Pin").center();
    world.click(diagnostics::pointer_pixels(pin));
}

#[given("clicking outside the panel leaves the additional area open")]
fn pinned_options_stays_open(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
    reset_is_displayed(world);
}

#[then("the ribbon shows compact panel buttons")]
fn options_group_button(world: &mut ApplicationWorld) {
    world.run_in_window("##application-ribbon", "Options");
    world.expect_no_text("Application Options");
    world.expect_no_text("Document Settings");
}

#[when("the user continues the display cycle to restore the full ribbon")]
fn restore_full_tools(world: &mut ApplicationWorld) {
    for _ in 0..3 {
        activate_display_control(world);
    }
}

#[when("the user opens the Options additional command area and clicks outside it")]
fn reopened_options_dismisses(world: &mut ApplicationWorld) {
    open_options(world);
    world.click([1000, 600]);
    options_closed(world);
}

#[cfg(target_os = "linux")]
#[test]
fn x11_options_layout_reset_can_be_cancelled() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let part = world.active_id();
    world.click_text("Tools");
    open_options(&mut world);
    world.save_capture("tools-options-expanded");
    click_native_command(&mut world, OPTIONS, "ResetUILayout");
    reset_warning(&mut world);
    reset_choices(&mut world);
    world.save_capture("tools-options-reset-confirmation");
    click_native_command(&mut world, RESET, "No");
    reset_closed(&mut world);
    options_closed(&mut world);
    world.expect_active(part);
    world.expect_text("Application Options");
    world.close();
}

#[cfg(target_os = "linux")]
fn click_native_command(world: &mut ApplicationWorld, window: &str, label: &str) {
    let at = diagnostics::pointer_pixels(world.run_in_window(window, label).center());
    super::super::x11_input::click(at);
    world.pump(Vec::new());
}
