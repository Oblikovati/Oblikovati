// SPDX-License-Identifier: GPL-2.0-only
//! Native navigation checks using the templates the chooser actually supplies.
use super::*;

#[then("the chooser closes and a new empty part becomes active")]
fn new_empty_part_after_chooser(world: &mut ApplicationWorld) {
    chooser_closed(world);
    new_empty_part(world);
}

#[then("a new document tab is visible alongside the original part")]
#[then("a new document tab appears alongside the original part")]
fn both_part_tabs_visible(world: &mut ApplicationWorld) {
    new_tab(world);
    let original = world.previous_document.expect("expected the original Part");
    assert_ne!(world.active_id(), original);
    let original_name = world.document_name(original);
    world.expect_tab(&original_name);
}

#[when("the user closes the new unedited part")]
fn close_new_part(world: &mut ApplicationWorld) {
    assert_ne!(Some(world.active_id()), world.previous_document);
    world.close_active_tab();
}

#[then("the original part becomes active")]
fn original_part_reactivated(world: &mut ApplicationWorld) {
    previous_document(world);
}

#[then("the Part group remains collapsed")]
fn part_group_stays_collapsed(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part"]);
    tiles_hidden(world);
}

#[then("Standard (mm) is shown as the selected template")]
fn part_selection_is_visible(world: &mut ApplicationWorld) {
    expect_part_details(world);
    let title = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    let other = basic_catalog::catalog_title(world, METRIC_SHEET_METAL, "Sheet Metal (mm)");
    let selected = named_navigation::gallery_tile(world, title);
    let plain = named_navigation::gallery_tile(world, other);
    let x = selected.maximum[0].max(plain.maximum[0]) as usize + 6;
    let selected_y = selected.center()[1] as usize;
    let plain_y = plain.center()[1] as usize;
    assert_ne!(
        world.pixel_at([x, selected_y]),
        world.pixel_at([x, plain_y]),
        "expected Standard (mm) to have the selection highlight"
    );
}

#[test]
fn collapsed_group_can_create_and_remains_collapsed_on_reopen() {
    let mut world = ApplicationWorld::new();
    let original = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric");
    world.click_text(METRIC_PART);
    world.click_text("Part");
    assert_part_template_tile(&world, METRIC_PART, false);
    expect_selected_template(&world, METRIC_PART);
    assert!(world.run_is_enabled("Create"));
    world.click_text("Create");
    let created = world.active_id();
    assert_ne!(created, original);
    assert_eq!(world.state().documents.len(), 2);
    world.close_active_tab();
    assert_eq!(world.active_id(), original);
    world.key(ViewportPhysicalKey::N, true);
    assert_part_template_tile(&world, METRIC_PART, false);
    expect_selected_template(&world, METRIC_PART);
    assert!(world.run_is_enabled("Create"));
    world.click_text("Part");
    assert_part_template_tile(&world, METRIC_PART, true);
    world.click_text("Cancel");
    world.close();
}

#[test]
fn enter_after_folder_highlight_creates_the_selected_template() {
    let mut world = ApplicationWorld::new();
    let original = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric");
    world.click_text(METRIC_PART);
    world.click_text("Metric");
    world.key(ViewportPhysicalKey::UpArrow, false);
    assert!(world.row_accented_beside("English", "Metric"));
    assert_part_template_tile(&world, METRIC_PART, true);
    expect_selected_template(&world, METRIC_PART);
    world.key(ViewportPhysicalKey::Enter, false);
    let created = world.active_id();
    assert_ne!(created, original);
    assert_eq!(world.state().documents.len(), 2);
    world.close_active_tab();
    assert_eq!(world.active_id(), original);
    world.close();
}

fn part_group_top(world: &ApplicationWorld) -> Option<f32> {
    let details_x = world.run("Template details").minimum[0];
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Part" && run.minimum[0] < details_x)
        .map(|run| run.center()[1])
}
#[test]
fn down_scrolls_the_gallery_without_changing_its_selection() {
    let mut world = ApplicationWorld::new();
    world
        .app()
        .owner
        .set_ui_scale_from_gui(oblikovati_model::ui_scale::UiScaleOptions::new(1.5, 1.).unwrap())
        .unwrap();
    world.pump(Vec::new());
    let original = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    choose_template_view(&mut world, "Medium Icons");
    world.click_text(METRIC_SHEET_METAL);
    let before = part_group_top(&world).expect("expected Part heading");
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.pump(Vec::new());
    let after = part_group_top(&world);
    assert!(
        after.is_none_or(|position| position < before),
        "Part heading did not scroll: {before} -> {after:?}"
    );
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::UpArrow, false);
    world.pump(Vec::new());
    let restored = part_group_top(&world).expect("expected restored Part heading");
    assert!(
        (restored - before).abs() < 1.,
        "expected scroll restored: {before} -> {restored}"
    );
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::Escape, false);
    assert_eq!(world.active_id(), original);
    world.close();
}

#[test]
fn arrow_and_endpoint_keys_navigate_the_selected_template_group() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    choose_template_view(&mut world, "Medium Icons");
    world.click_text(METRIC_SHEET_METAL);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::RightArrow, false);
    expect_selected_template(&world, METRIC_PART);
    world.key(ViewportPhysicalKey::LeftArrow, false);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::End, false);
    expect_selected_template(&world, METRIC_PART);
    world.key(ViewportPhysicalKey::Home, false);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::LeftArrow, false);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.click_text("Display");
    world.key(ViewportPhysicalKey::End, false);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.key(ViewportPhysicalKey::Escape, false);
    world.key(ViewportPhysicalKey::Escape, false);
    assert_eq!(world.active_id(), part);
    world.close();
}
