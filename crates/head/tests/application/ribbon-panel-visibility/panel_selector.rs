// SPDX-License-Identifier: GPL-2.0-only
//! Read the direct selector and ribbon captions from completed native draw packets.
use super::super::diagnostics::window_geometry;
use super::*;
fn ribbon_bottom() -> f32 {
    let (position, size) = window_geometry("##application-ribbon").unwrap();
    position[1] + size[1]
}
fn selector_is_open(world: &ApplicationWorld) -> bool {
    if window_geometry("##ribbon-column").is_some() {
        return super::side_selector::popup_depth() > 0;
    }
    let row = world.run(PARAMETERS).center();
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    row[0] > size[0] * 0.5 && row[1] > ribbon_bottom()
}
pub(super) fn caption_is_visible(world: &ApplicationWorld) -> bool {
    let (position, size) = window_geometry("##application-ribbon").unwrap();
    world.frame.runs.iter().any(|row| {
        let center = row.center();
        row.text == PARAMETERS
            && (0..2).all(|axis| {
                center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
            })
    })
}
#[when("the user clicks the panel selector at the right end of the ribbon")]
#[when("the user clicks the ribbon panel selector")]
#[when("the user clicks the ribbon panel selector again")]
#[when("the user opens the ribbon panel selector")]
#[when("the user clicks the direct panel visibility selector")]
#[when("I click the direct panel selector")]
fn open_selector(world: &mut ApplicationWorld) {
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    world.click([size[0] as i32 - 22, 45]);
    world.pump(Vec::new());
}
#[then("the 3D Model panel visibility list opens")]
fn model_panel_list_open(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    for panel in ["Sketch", "Create", "Modify"] {
        assert!(world.checked(panel), "expected {panel} checked");
    }
}
#[then("the panel visibility list closes")]
fn model_panel_list_closed(_world: &mut ApplicationWorld) {
    assert_eq!(
        super::side_selector::popup_depth(),
        0,
        "expected selector closed"
    );
}
#[then("the full 3D Model ribbon remains visible")]
fn full_model_ribbon_visible(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Extrude");
}
#[given("the full 3D Model ribbon shows the Modify panel")]
#[given("the expanded 3D Model ribbon shows the Modify panel")]
#[then("the Modify panel remains visible")]
fn modify_panel_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[then("the panel list opens with Modify checked")]
fn list_modify_checked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(world.checked("Modify"), "expected Modify checked");
}
#[then("the panel list opens with Parameters unchecked")]
fn list_unchecked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
}
#[when("the user selects Parameters in the list")]
#[when("the user selects Parameters again")]
#[when("I select Parameters in the panel list")]
#[when("the user selects Parameters in the panel list")]
fn toggle_selector_parameter(world: &mut ApplicationWorld) {
    world.click_text(PARAMETERS);
    world.pump(Vec::new());
}
#[then("the panel list remains open with Parameters checked")]
fn open_parameters_checked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(world.checked(PARAMETERS), "expected Parameters checked");
}
#[then("the panel list remains open with Parameters unchecked")]
fn open_parameters_unchecked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
}
#[then("Parameters is checked and its panel appears in the ribbon")]
fn checked_and_visible(world: &mut ApplicationWorld) {
    assert!(world.checked(PARAMETERS), "expected Parameters checked");
    assert!(
        caption_is_visible(world),
        "expected the actual ribbon panel caption"
    );
}
#[then("Parameters is unchecked and its panel disappears")]
fn unchecked_and_hidden(world: &mut ApplicationWorld) {
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
    assert!(
        !caption_is_visible(world),
        "expected no ribbon panel caption"
    );
}
#[then("the panel list remains open")]
fn list_stays_open(world: &mut ApplicationWorld) {
    assert!(
        selector_is_open(world),
        "expected the panel list to stay open"
    );
}
#[when("the user presses Escape")]
#[when("the user presses Escape again")]
fn escape_selector(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    if world.application_close_requested {
        super::super::x11_input::press_escape();
        world.pump(Vec::new());
        world.application_close_requested = false;
        return;
    }
    world.escape();
    world.pump(Vec::new());
}
#[then("the panel list closes")]
fn list_closes(world: &mut ApplicationWorld) {
    assert!(
        !selector_is_open(world),
        "expected the direct panel list closed"
    );
}
#[given("Modify is visible and Parameters is hidden")]
#[then("Modify remains visible and Parameters remains hidden")]
fn model_panel_choices(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    assert!(!caption_is_visible(world));
}
#[then("the panel list shows Modify checked and Parameters unchecked")]
fn initial_panel_choices(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world));
    assert!(world.checked("Modify"));
    assert!(!world.checked(PARAMETERS));
}
#[when("the user selects the View ribbon tab")]
fn select_view(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    {
        let point = super::super::diagnostics::pointer_pixels(world.run("View").center());
        super::super::x11_input::click(point);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.click_text("View");
}
#[when("the user returns to the 3D Model ribbon tab")]
fn select_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[then("the View ribbon is displayed")]
fn view_ribbon_visible(world: &mut ApplicationWorld) {
    world.expect_text("Navigate");
}
#[then("the panel list remains closed")]
fn panel_list_remains_closed(world: &mut ApplicationWorld) {
    list_closes(world);
}
#[when("the user clears Modify in the panel list")]
#[when("the user selects Modify in the panel list")]
fn toggle_modify(world: &mut ApplicationWorld) {
    world.click_text("Modify");
}
#[then("the Modify panel and its expanded command area disappear")]
fn hidden_modify_and_expansion(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(!modify_caption_visible(world));
}
#[then("the panel list remains open with Modify unchecked")]
fn open_modify_unchecked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world));
    assert!(!world.checked("Modify"));
}
#[then("the Modify panel returns to the ribbon")]
fn modify_back(world: &mut ApplicationWorld) {
    assert!(modify_caption_visible(world));
}
fn modify_caption_visible(world: &ApplicationWorld) -> bool {
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    world.frame.runs.iter().any(|run| {
        run.text.contains("Modify")
            && run.center()[0] < size[0] * 0.5
            && run.center()[1] < ribbon_bottom()
    })
}
#[then("its additional command area is not open")]
fn no_additional(_: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-additional-PartModify").is_none());
}
#[then("the panel list remains open with Modify checked")]
fn open_modify_checked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world));
    assert!(world.checked("Modify"));
}

#[test]
fn native_panel_selector_uses_arrow_and_activation_keys() {
    use oblikovati_viewport::ViewportPhysicalKey::{DownArrow, Enter, Space};
    for activate in [Space, Enter] {
        let mut world = ApplicationWorld::new();
        world.add_part();
        open_selector(&mut world);
        world.key(DownArrow, false);
        world.key(DownArrow, false);
        assert!(world.checked("Create"));
        world.key(activate, false);
        assert!(!world.checked("Create"));
        assert!(selector_is_open(&world));
        world.key(activate, false);
        assert!(world.checked("Create"));
        assert!(selector_is_open(&world));
        world.close();
    }
}
#[test]
fn display_control_dismisses_selector_and_minimizes_on_the_same_click() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::super::ribbon_display::click_display_menu(&mut world);
    world.click_text("Minimize to Tabs");
    world.click_text("Ribbon Display");
    world.expect_text("Start 2D Sketch");
    open_selector(&mut world);
    assert!(world.checked("Modify"));
    assert!(!world.checked(PARAMETERS));
    world.click_text("Ribbon Display");
    assert!(!selector_is_open(&world));
    world.expect_no_text("Start 2D Sketch");
    world.click_text("3D Model");
    world.expect_text("Start 2D Sketch");
    open_selector(&mut world);
    assert!(world.checked("Modify"));
    assert!(!world.checked(PARAMETERS));
    world.click_text(PARAMETERS);
    assert!(world.checked(PARAMETERS));
    world.expect_text("Start 2D Sketch");
    world.click_text(PARAMETERS);
    assert!(!world.checked(PARAMETERS));
    world.expect_text("Start 2D Sketch");
    world.click_text("Ribbon Display");
    world.expect_text("Start 2D Sketch");
    assert!(!selector_is_open(&world));
    open_selector(&mut world);
    assert!(world.checked("Modify"));
    assert!(!world.checked(PARAMETERS));
    world.close();
}
