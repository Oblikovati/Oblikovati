// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn popup_depth() -> i32 {
    let context =
        unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected live GUI context");
    context.OpenPopupStack.Size
}
fn template_point(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let details_x = world.run("Template details").minimum[0];
    let expected = label.replace(' ', "");
    let tile = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == expected && run.minimum[0] < details_x)
        .unwrap_or_else(|| panic!("expected template tile {label:?} in {}", world.text()));
    super::super::super::diagnostics::pointer_pixels(tile.center())
}
fn right_click_template(world: &mut ApplicationWorld, label: &str) {
    let point = template_point(world, label);
    world.right_click(point);
    world.pump(Vec::new());
}
#[test]
fn secondary_click_selects_inch_templates_without_opening_a_menu() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("English");
    choose_template_view(&mut world, "Medium Icons");
    let original_depth = popup_depth();
    right_click_template(&mut world, "Standard inch part");
    expect_selected_template(&world, "Standard inch part");
    world.expect_text("Length: inches (in)");
    assert_eq!(popup_depth(), original_depth);
    right_click_template(&mut world, INCH_SHEET_METAL);
    expect_selected_template(&world, INCH_SHEET_METAL);
    world.expect_text("Length: inches (in)");
    assert_eq!(popup_depth(), original_depth);
    let label = template_point(&world, INCH_SHEET_METAL);
    let sample = [label[0] as usize, label[1] as usize - 10];
    let hovered_selection = world.pixel_at(sample);
    world.hover([800, 560]);
    let idle_selection = world.pixel_at(sample);
    assert_eq!(
        hovered_selection, idle_selection,
        "expected selection to stay visible while hovered"
    );
    world.save_capture("secondary-template-selection");
    world.right_click([800, 560]);
    world.pump(Vec::new());
    expect_selected_template(&world, INCH_SHEET_METAL);
    assert_eq!(popup_depth(), original_depth);
    world.key(ViewportPhysicalKey::RightArrow, false);
    expect_selected_template(&world, INCH_SHEET_METAL);
    world.key(ViewportPhysicalKey::Escape, false);
    assert_eq!(world.active_id(), part);
    world.close();
}

#[test]
fn secondary_selection_works_in_list_and_details_views() {
    let mut world = ApplicationWorld::new();
    world.key(ViewportPhysicalKey::N, true);
    for view in ["List", "Details"] {
        choose_template_view(&mut world, view);
        let depth = popup_depth();
        right_click_template(&mut world, METRIC_SHEET_METAL);
        expect_selected_template(&world, METRIC_SHEET_METAL);
        assert_eq!(popup_depth(), depth);
        right_click_template(&mut world, METRIC_PART);
        expect_selected_template(&world, METRIC_PART);
        assert_eq!(popup_depth(), depth);
    }
    world.close();
}

#[test]
fn secondary_click_on_selected_drawing_retains_its_preview() {
    let mut world = ApplicationWorld::new();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric drawing with title block");
    world.click_text("Use Template");
    assert!(world.run_is_enabled("Create"));
    right_click_template(&mut world, "Metric drawing with title block");
    assert!(
        world.run_is_enabled("Create"),
        "expected the chosen drawing preview to remain"
    );
    let point = template_point(&world, "Metric drawing with title block");
    world.click(point);
    assert!(
        world.run_is_enabled("Create"),
        "expected the preview after a primary click"
    );
    world.close();
}
