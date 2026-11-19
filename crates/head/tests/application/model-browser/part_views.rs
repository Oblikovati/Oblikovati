// SPDX-License-Identifier: GPL-2.0-only
//! Drive the part-view copy and activation journey through the native window (#4118).
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::contract::PartDesignView as _;

#[given("the View representation branch is expanded with Primary active")]
pub(super) fn expanded_primary_view(world: &mut ApplicationWorld) {
    let branch = view_branch(world).clone();
    world.click([branch.minimum[0] as i32 - 10, branch.center()[1] as i32]);
    world.pump(Vec::new());
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|row| row.text.trim() == "Primary"
                && row.list == branch.list
                && row.center()[1] > branch.center()[1]),
        "expected an actual Primary row beneath the View branch"
    );
    assert!(world.checked("Primary"));
}

#[when("the user chooses Copy from the Primary view representation context menu")]
pub(super) fn copy_primary_view(world: &mut ApplicationWorld) {
    right_click_row(world, "Primary");
    choose_menu_row(world, "Copy");
}

#[then("a new view representation appears with its generated name selected for editing")]
fn copied_name_selected(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    assert!(world.text_is_selected("View1"));
}

#[then("Primary remains active")]
#[then("Primary is marked active")]
fn primary_view_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .part_design_views
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Primary"
    );
    assert!(world.checked("Primary"));
}

#[when("the user enters \"Capture View\" and presses Enter")]
pub(super) fn rename_copied_view(world: &mut ApplicationWorld) {
    world.type_text("Capture View");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("the new view representation has the name \"Capture View\"")]
fn copied_view_renamed(world: &mut ApplicationWorld) {
    world.expect_text("Capture View");
}

#[when("the user double-clicks \"Capture View\"")]
pub(super) fn activate_copied_view(world: &mut ApplicationWorld) {
    world.double_click(world.run("Capture View").center().map(|value| value as i32));
    world.pump(Vec::new());
}

#[then("\"Capture View\" is marked active")]
pub(super) fn copied_view_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .part_design_views
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Capture View"
    );
    assert!(world.checked("Capture View"));
}

#[then("the View branch identifies \"Capture View\"")]
fn copied_view_branch_label(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Capture View");
}

#[when("the user double-clicks Primary")]
fn activate_primary_view(world: &mut ApplicationWorld) {
    world.double_click(world.run("Primary").center().map(|value| value as i32));
    world.pump(Vec::new());
}

#[then("the View branch identifies Primary")]
fn primary_view_branch_label(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Primary");
}

pub(super) fn expect_branch_label(world: &ApplicationWorld, name: &str) {
    let branch = view_branch(world);
    let active = world.run(&format!("<{name}>"));
    assert!((active.center()[1] - branch.center()[1]).abs() < 3.);
}

pub(super) fn view_branch(world: &ApplicationWorld) -> &super::super::diagnostics::RenderedRun {
    let representations = world.run("Representations");
    world
        .frame
        .runs
        .iter()
        .filter(|row| {
            row.text.trim() == "View"
                && row.list == representations.list
                && row.center()[1] > representations.center()[1]
        })
        .min_by(|first, second| first.center()[1].total_cmp(&second.center()[1]))
        .expect("expected View inside the browser's Representations branch")
}

#[given("The MCAD Software has an empty part with a copied view active")]
fn empty_part_with_active_copy(world: &mut ApplicationWorld) {
    world.add_part();
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_copied_view(world);
    activate_copied_view(world);
}

#[given("Publish is checked in the copied view context menu")]
fn published_view_menu_open(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(world.checked("Publish"));
}

#[when("the user chooses Publish from the copied view context menu")]
#[when("the user chooses Publish from the copied view context menu again")]
fn toggle_view_publication(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Publish");
}

#[then("Publish is unchecked when the context menu is reopened")]
fn view_publication_unchecked(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(!world.checked("Publish"));
}

#[then("Publish is checked when the context menu is reopened")]
fn view_publication_checked(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(world.checked("Publish"));
}

#[then("the copied view remains active")]
fn published_copy_remains_active(world: &mut ApplicationWorld) {
    copied_view_active(world);
}

#[given("Primary is the active view representation")]
fn primary_active_before_copy(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .part_design_views
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Primary"
    );
}

#[given("an inactive copied view representation exists")]
fn inactive_view_copy(world: &mut ApplicationWorld) {
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_copied_view(world);
    primary_view_active(world);
}

#[given("its camera autosave option is checked")]
fn copied_view_autosave_enabled(world: &mut ApplicationWorld) {
    let state = world.state();
    let views = state.browser.unwrap().part_design_views.unwrap();
    assert!(
        views
            .list()
            .iter()
            .find(|view| view.name() == "Capture View")
            .unwrap()
            .autosave_camera()
    );
}

#[when("the user opens the copied view's context menu and expands Camera View")]
#[when("the user reopens the Camera View submenu")]
fn open_camera_submenu(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    let at = super::super::diagnostics::pointer_pixels(world.run("Camera View").center());
    world.hover(at);
    world.pump(Vec::new());
}

#[then("Save Current Camera is unavailable")]
fn inactive_camera_save_unavailable(world: &mut ApplicationWorld) {
    let save = world.run("Save Current Camera");
    let restore = world.run("Restore Saved Camera");
    let brightness = |run: &super::super::diagnostics::RenderedRun| {
        run.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(run.alpha)
    };
    assert!(brightness(save) < brightness(restore));
}

#[then("Restore Saved Camera is available")]
fn inactive_camera_restore_available(world: &mut ApplicationWorld) {
    world.expect_text("Restore Saved Camera");
}

#[then("Autosave Camera is checked")]
fn camera_autosave_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Autosave Camera"));
}

#[then("Autosave Camera is unchecked")]
fn camera_autosave_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Autosave Camera"));
}

#[when("the user chooses Autosave Camera")]
#[when("the user chooses Autosave Camera again")]
fn toggle_camera_autosave(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Autosave Camera");
}

#[when("the user dismisses the menus")]
fn dismiss_view_menus(world: &mut ApplicationWorld) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.pump(Vec::new());
}

#[then("Primary remains the active view")]
fn primary_stays_active(world: &mut ApplicationWorld) {
    primary_view_active(world);
}

#[given("a copied view representation is active")]
pub(super) fn activate_copy_for_camera(world: &mut ApplicationWorld) {
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_copied_view(world);
    activate_copied_view(world);
}

#[given("the orientation indicator shows Front")]
fn camera_starts_front(world: &mut ApplicationWorld) {
    super::super::viewport_orientation::press_part(world, |parts| parts.center);
    super::super::viewport_orientation::expect_control_shows(world, "Front");
}

#[given("Autosave Camera is off for the active view")]
fn disable_active_camera_autosave(world: &mut ApplicationWorld) {
    open_camera_submenu(world);
    assert!(world.checked("Autosave Camera"));
    toggle_camera_autosave(world);
}

#[when("the user chooses Save Current Camera from the active view's Camera View submenu")]
fn save_active_view_camera(world: &mut ApplicationWorld) {
    open_camera_submenu(world);
    choose_menu_row(world, "Save Current Camera");
}

#[when("the user changes the orientation to Top using the orientation control")]
fn camera_turns_to_top(world: &mut ApplicationWorld) {
    super::super::viewport_orientation::press_part(world, |parts| {
        [parts.center[0], parts.center[1] - parts.half * 0.85]
    });
}

#[then("the orientation indicator shows Top")]
fn camera_indicates_top(world: &mut ApplicationWorld) {
    super::super::viewport_orientation::expect_control_shows(world, "Top");
}

#[when("the user chooses Restore Saved Camera from the active view's Camera View submenu")]
fn restore_active_view_camera(world: &mut ApplicationWorld) {
    open_camera_submenu(world);
    choose_menu_row(world, "Restore Saved Camera");
}

#[then("the orientation indicator shows Front")]
fn camera_indicates_front(world: &mut ApplicationWorld) {
    super::super::viewport_orientation::expect_control_shows(world, "Front");
}

#[given("its annotation scale is Auto")]
fn active_view_scale_is_auto(world: &mut ApplicationWorld) {
    let state = world.state();
    let views = state.browser.unwrap().part_design_views.unwrap();
    assert_eq!(
        views.active().unwrap().annotation_scale().mode,
        oblikovati_api::types::PART_ANNOTATION_SCALE_AUTO
    );
}

#[when("the user opens the active view's context menu and expands Annotation Scale")]
#[when("the user opens the active view's Annotation Scale submenu")]
#[when("the user reopens the active view's Annotation Scale submenu")]
#[when("the user reopens the Annotation Scale submenu")]
fn open_annotation_scale_menu(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    let at = super::super::diagnostics::pointer_pixels(world.run("Annotation Scale").center());
    world.hover(at);
    world.pump(Vec::new());
}

#[then("Auto is checked")]
fn annotation_auto_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Auto"));
}
#[then("Auto is not checked")]
fn annotation_auto_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Auto"));
}
#[then("1:1 is checked")]
fn annotation_one_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("1:1"));
}
#[then("1:1 is not checked")]
fn annotation_one_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("1:1"));
}
#[then("fixed ratios and Custom are available")]
fn annotation_choices_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("1:1"));
    assert!(world.run_is_enabled("Custom"));
}
#[when("the user chooses 1:1")]
fn choose_one_to_one(world: &mut ApplicationWorld) {
    choose_menu_row(world, "1:1");
}
#[when("the user chooses Auto")]
#[when("the user selects Auto")]
fn choose_annotation_auto(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Auto");
}

#[given("a copied view representation is active with annotation scale Auto")]
fn active_copy_with_auto_scale(world: &mut ApplicationWorld) {
    activate_copy_for_camera(world);
    active_view_scale_is_auto(world);
}
#[when("the user chooses Custom")]
fn choose_custom_scale(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Custom");
}
#[then("the Edit Annotation Scale dialog opens with Auto selected")]
fn custom_scale_dialog_is_open(world: &mut ApplicationWorld) {
    world.expect_text("Edit Annotation Scale");
    assert!(world.text_is_selected("Auto"));
}
#[then("OK is disabled")]
fn custom_scale_ok_disabled(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"));
}
#[when(expr = "the user replaces the scale with {string}")]
fn replace_custom_scale(world: &mut ApplicationWorld, value: String) {
    select_all_text(world);
    world.type_text(&value);
    world.pump(Vec::new());
}
#[then(expr = "the scale field displays {string}")]
fn custom_scale_text(world: &mut ApplicationWorld, value: String) {
    world.expect_text(&value);
}
#[when("the user confirms the dialog")]
fn confirm_custom_scale(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
}
#[then("Custom is checked")]
fn custom_scale_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Custom"));
}
#[then("the Custom entry displays the equivalent decimal value 1.50")]
fn custom_scale_decimal(world: &mut ApplicationWorld) {
    world.expect_text("1.50");
}
#[then("the annotation scale submenu closes")]
fn scale_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("1:1");
}

#[given("the Edit Annotation Scale dialog is open")]
fn open_custom_scale_dialog(world: &mut ApplicationWorld) {
    open_annotation_scale_menu(world);
    choose_custom_scale(world);
}
#[then("the scale text is red")]
fn invalid_scale_is_red(world: &mut ApplicationWorld) {
    let color = scale_input_run(world).color;
    assert!(
        u16::from(color[0]) > u16::from(color[1]) * 2,
        "expected red scale input, got {color:?}"
    );
}
#[then("the scale text returns to its normal color")]
fn valid_scale_is_not_red(world: &mut ApplicationWorld) {
    let color = scale_input_run(world).color;
    assert!(
        u16::from(color[0]) < u16::from(color[1]) * 2,
        "expected normal scale input, got {color:?}"
    );
}
fn scale_input_run(world: &ApplicationWorld) -> &super::super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| ["0", "1:0", "abc", "3:2"].contains(&run.text.as_str()))
        .expect("expected the entered scale text")
}

/// Select the focused input on the platform's standard shortcut.
pub(super) fn select_all_text(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{
        ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey,
    };
    let modifiers = ViewportModifiers {
        control: !cfg!(target_os = "macos"),
        super_key: cfg!(target_os = "macos"),
        ..Default::default()
    };
    let key = |pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::A,
            layout_key: ViewportPhysicalKey::A,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![key(true, modifiers), key(false, Default::default())]);
}
