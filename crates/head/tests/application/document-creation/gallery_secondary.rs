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

#[given("the Part group contains Standard (in) and Sheet Metal (in)")]
fn inch_part_tiles(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.active_id());
    world.expect_text("Standard inch part");
    world.expect_text(INCH_SHEET_METAL);
    assert_eq!(popup_depth(), 1, "expected only the chooser popup");
}
#[when("the user right-clicks the Standard (in) part template")]
fn secondary_standard(world: &mut ApplicationWorld) {
    right_click_template(world, "Standard inch part");
}
#[then("Standard (in) is selected and its details show inch units")]
fn standard_inch_details(world: &mut ApplicationWorld) {
    expect_selected_template(world, "Standard inch part");
    world.expect_text("Length: inches (in)");
}
#[then("no context menu is visible")]
fn no_secondary_menu(_world: &mut ApplicationWorld) {
    assert_eq!(popup_depth(), 1, "expected only the chooser popup");
}
#[when("the user right-clicks the Sheet Metal (in) part template")]
fn secondary_sheet_metal(world: &mut ApplicationWorld) {
    right_click_template(world, INCH_SHEET_METAL);
}
#[then("Sheet Metal (in) is selected and its details show inch units")]
#[then("Sheet Metal (in) remains selected and its details remain displayed")]
fn sheet_metal_inch_details(world: &mut ApplicationWorld) {
    expect_selected_template(world, INCH_SHEET_METAL);
    world.expect_text("Length: inches (in)");
}
#[when("the user right-clicks empty space in the Part gallery")]
fn secondary_empty_gallery(world: &mut ApplicationWorld) {
    let right = world.run("Template details").minimum[0];
    let bottom = world.run("Create").minimum[1];
    let point = super::super::super::diagnostics::pointer_pixels([right - 25., bottom - 25.]);
    world.right_click(point);
    world.pump(Vec::new());
}
#[then("the chooser closes and the original part remains active")]
fn original_part_after_chooser(world: &mut ApplicationWorld) {
    world.expect_no_text("Template folders");
    assert_eq!(Some(world.active_id()), world.previous_document);
}

#[test]
fn selected_builtin_part_shows_its_unassigned_material() {
    let mut world = ApplicationWorld::new();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text(METRIC_PART);
    expect_selected_template(&world, METRIC_PART);
    world.expect_text("Material: Unassigned");
    world.save_capture("template-unassigned-material");
    world.close();
}

#[test]
fn material_source_failure_is_visible_only_in_the_chooser() {
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateMaterial, GuiTemplateMaterialEntry};
    let mut world = ApplicationWorld::new();
    world.app().window.template_material_override = Some(GuiTemplateMaterialEntry {
        template: GuiDocumentTemplate::default(),
        material: GuiTemplateMaterial::Unavailable {
            cause: "material \"fixture-steel\": fixture catalog unavailable".into(),
        },
    });
    world.app().window.template_size_issue_override = Some("fixture size failure".into());
    world.pump(Vec::new());
    world.expect_no_text("fixture catalog unavailable");
    world.expect_no_text("fixture size failure");
    world.key(ViewportPhysicalKey::N, true);
    world.click_text(METRIC_PART);
    let visible: String = world
        .text()
        .chars()
        .filter(|ch| !ch.is_whitespace())
        .collect();
    assert!(
        visible.contains("Materialunavailable:material\"fixture-steel\":fixturecatalogunavailable")
    );
    assert!(visible.contains("Templatesizesunavailable:fixturesizefailure"));
    world.save_capture("template-material-failure");
    world.close();
}
