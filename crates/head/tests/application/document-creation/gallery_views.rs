// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[path = "chooser_details.rs"]
mod chooser_details;
#[path = "gallery_icons.rs"]
mod gallery_icons;
#[path = "gallery_secondary.rs"]
mod gallery_secondary;
#[given("the metric template folder is selected")]
fn metric_selected(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.expect_text(METRIC_PART);
}
fn expect_part_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Part");
    world.expect_text("millimeters (mm)");
    let tile_and_details = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == METRIC_PART.replace(' ', ""))
        .count();
    assert_eq!(
        tile_and_details, 2,
        "expected the tile and the details pane"
    );
}
#[given("a standard metric part template's details are displayed")]
fn part_details_shown(world: &mut ApplicationWorld) {
    world.click_text(METRIC_PART);
    expect_part_details(world);
}
#[given("templates are grouped by type")]
fn grouped_by_default(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
}
fn expect_group_headings(world: &mut ApplicationWorld, headings: &[&str]) {
    for heading in headings {
        let shown = world.frame.runs.iter().any(|run| run.text == *heading);
        assert!(
            shown,
            "rendered text {:?}: expected the heading {heading:?}",
            world.text()
        );
    }
}
fn toggle_grouping(world: &mut ApplicationWorld) {
    world.click_text("Display");
    world.pump(Vec::new());
    world.click_text("Grouped by Type");
    world.pump(Vec::new());
}
#[when("I clear Grouped by Type in the template display menu")]
#[when("I select Grouped by Type in the template display menu")]
#[when("the user clears Grouped by Type in the template display menu")]
#[when("the user selects Grouped by Type in the template display menu")]
fn change_grouping(world: &mut ApplicationWorld) {
    toggle_grouping(world);
}
#[then("the template gallery displays a flat collection without type group headings")]
fn flat_gallery(world: &mut ApplicationWorld) {
    let headings = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            matches!(
                run.text.as_str(),
                "Part" | "Assembly" | "Drawing" | "Presentation"
            )
        })
        .count();
    assert_eq!(
        headings,
        0,
        "rendered text {:?}: expected no group headings",
        world.text()
    );
    for tile in [
        METRIC_PART,
        METRIC_SHEET_METAL,
        METRIC_ASSEMBLY,
        METRIC_DRAWING,
    ] {
        world.expect_text(tile);
    }
}
#[then("the details pane continues to display the metric part template information")]
fn details_continue(world: &mut ApplicationWorld) {
    expect_part_details(world);
}
#[then("the gallery displays part, assembly and drawing group headings")]
fn headings_back(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
}
fn choose_template_view(world: &mut ApplicationWorld, view: &str) {
    world.click_text("Display");
    world.pump(Vec::new());
    world.click_text(view);
    world.pump(Vec::new());
}
/// The vertical distance between the part group's two tiles.
fn tile_spacing(world: &ApplicationWorld) -> f32 {
    (world.run(METRIC_SHEET_METAL).center()[1] - world.run(METRIC_PART).center()[1]).abs()
}
#[when("I choose List in the template display menu")]
#[when("the user chooses List in the template display menu")]
fn choose_list_view(world: &mut ApplicationWorld) {
    choose_template_view(world, "List");
}
#[then("the gallery displays compact template entries under type headings")]
fn compact_entries(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
    let spacing = tile_spacing(world);
    assert!(
        spacing < 24.,
        "tile spacing {spacing}: expected compact rows"
    );
}
#[when("I select the standard metric assembly template in the list")]
#[when("the user selects the standard metric assembly template in the list")]
fn select_assembly_in_list(world: &mut ApplicationWorld) {
    world.click_text(METRIC_ASSEMBLY);
}
fn expect_assembly_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Assembly");
    world.expect_text("millimeters (mm)");
}
#[then("the details pane identifies an assembly with millimeter units")]
#[then("the details pane continues to identify the metric assembly template")]
fn details_identify_assembly(world: &mut ApplicationWorld) {
    expect_assembly_details(world);
}
#[when("I choose Medium icons in the template display menu")]
#[when("the user chooses Medium icons in the template display menu")]
fn choose_medium_view(world: &mut ApplicationWorld) {
    choose_template_view(world, "Medium Icons");
}
#[then("the gallery displays medium-sized template icons")]
fn medium_tiles(world: &mut ApplicationWorld) {
    expect_icon_row(world, 40.);
}
fn expect_icon_row(world: &ApplicationWorld, expected_size: f32) {
    let first = world.run(METRIC_SHEET_METAL).center();
    let second = world.run(METRIC_PART).center();
    assert!(
        second[0] > first[0] && (second[1] - first[1]).abs() < 3.,
        "expected template icons in a horizontal row"
    );
    let size = gallery_icons::template_icon_size(world, METRIC_SHEET_METAL);
    assert!(
        (size[0] - expected_size).abs() < 1. && (size[1] - expected_size).abs() < 1.,
        "icon size {size:?}: expected {expected_size}"
    );
}
#[when("I choose Back in the chooser navigation bar")]
fn chooser_back(world: &mut ApplicationWorld) {
    world.click_text("Back");
}
#[when("I choose Forward in the chooser navigation bar")]
fn chooser_forward(world: &mut ApplicationWorld) {
    world.click_text("Forward");
}
#[then("the English folder is selected")]
fn english_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_accented_beside("English", "Metric"),
        "expected English accented"
    );
}
#[then("the metric folder is selected")]
fn metric_folder_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_accented_beside("Metric", "English"),
        "expected Metric accented"
    );
}
#[then("the gallery displays its inch templates")]
fn inch_gallery(world: &mut ApplicationWorld) {
    world.expect_text("Standard inch part");
    world.expect_no_text(METRIC_PART);
}
#[then("the gallery displays its metric templates")]
fn metric_gallery(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_PART);
    world.expect_no_text("Standard inch part");
}
#[given("the metric folder and standard metric assembly template are selected")]
#[given("the standard metric assembly template details are displayed")]
fn assembly_details_shown(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.click_text(METRIC_ASSEMBLY);
    expect_assembly_details(world);
}
#[when("I reopen the new-document chooser during the same session")]
fn reopen_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.pump(Vec::new());
}
#[then("the standard metric assembly template is selected")]
fn assembly_tile_selected(world: &mut ApplicationWorld) {
    let tile = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == METRIC_ASSEMBLY.replace(' ', ""))
        .count();
    assert_eq!(tile, 2, "expected the tile and the details pane");
}
#[then("its details identify an assembly with millimeter units")]
fn its_assembly_details(world: &mut ApplicationWorld) {
    expect_assembly_details(world);
}
#[given("the template root has no templates directly in it")]
fn root_has_no_templates(world: &mut ApplicationWorld) {
    world.expect_text("Templates");
}
#[when("I select Home in the chooser navigation bar")]
fn chooser_home(world: &mut ApplicationWorld) {
    world.click_text("Home");
}
#[then("the template root is selected in the folder tree")]
fn root_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_accented_beside("Templates", "English"),
        "expected the root accented"
    );
}
#[then("the gallery type sections contain no template entries")]
fn empty_sections(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
    world.expect_no_text(METRIC_PART);
    world.expect_no_text(METRIC_DRAWING);
}
#[then("the metric assembly template details remain displayed")]
fn assembly_details_remain(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_ASSEMBLY);
    expect_assembly_details(world);
}
#[then("the metric template folder and its gallery are displayed again")]
fn metric_folder_back(world: &mut ApplicationWorld) {
    metric_folder_selected(world);
    world.expect_text(METRIC_PART);
}
#[given("the metric folder is selected and templates are grouped by type")]
fn metric_grouped(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    grouped_by_default(world);
}
#[when("I choose Extra large icons in the template display menu")]
fn choose_extra_large(world: &mut ApplicationWorld) {
    choose_template_view(world, "Extra Large Icons");
}
#[then("the template gallery displays extra-large tiles")]
fn extra_large_tiles(world: &mut ApplicationWorld) {
    expect_icon_row(world, 56.);
}
#[when("I choose Small icons in the template display menu")]
fn choose_small(world: &mut ApplicationWorld) {
    choose_template_view(world, "Small Icons");
}
#[then("the template gallery displays small icons")]
fn small_tiles(world: &mut ApplicationWorld) {
    expect_icon_row(world, 18.);
}
#[then("the template gallery displays medium-sized icons")]
fn medium_sized(world: &mut ApplicationWorld) {
    medium_tiles(world);
}
#[when("I double-click the standard metric part template")]
fn double_click_part(world: &mut ApplicationWorld) {
    let center = world.run(METRIC_PART).center().map(|value| value as i32);
    world.pump(super::super::pointer_click_events(center, 2));
    world.pump(Vec::new());
}
#[when("I press Enter")]
fn press_enter(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("a new empty part document becomes active")]
fn new_empty_part(world: &mut ApplicationWorld) {
    expect_new_document(world, oblikovati_api::types::DOCUMENT_PART);
    empty_part(world);
}
#[then("the original part remains active")]
fn original_active(world: &mut ApplicationWorld) {
    previous_document(world);
}
#[then("its viewport and browser are displayed")]
fn viewport_and_browser_shown(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_blank_region(VIEWPORT_X, VIEWPORT_Y);
}
#[then("no new document tab is created")]
fn no_new_tab(world: &mut ApplicationWorld) {
    assert_eq!(world.state().documents.len(), world.parts.len());
}

#[test]
fn name_sort_reverses_group_order_without_changing_selected_template() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("English");
    world.click_text("Standard inch part");
    world.click_text("Metric");
    choose_template_view(&mut world, "Details");
    assert!(world.run(METRIC_SHEET_METAL).center()[1] < world.run(METRIC_PART).center()[1]);
    world.expect_text("Standard inch part");
    world.expect_text("Length: inches (in)");
    world.click_text("Name");
    world.pump(Vec::new());
    assert!(world.run(METRIC_PART).center()[1] < world.run(METRIC_SHEET_METAL).center()[1]);
    world.expect_text("Standard inch part");
    world.expect_text("Length: inches (in)");
    world.click_text("Name");
    world.pump(Vec::new());
    assert!(world.run(METRIC_SHEET_METAL).center()[1] < world.run(METRIC_PART).center()[1]);
    world.click_text("Cancel");
    assert_eq!(world.active_id(), part);
    world.close();
}

#[given("the selected template folder contains templates")]
fn selected_folder_contains_templates(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_PART);
    world.expect_text(METRIC_ASSEMBLY);
}

fn expect_selected_template(world: &ApplicationWorld, title: &str) {
    let details_x = world.run("Template details").minimum[0];
    let expected = title.replace(' ', "");
    let shown: String = world
        .frame
        .runs
        .iter()
        .filter(|run| run.minimum[0] >= details_x)
        .map(|run| run.text.as_str())
        .collect();
    assert!(
        shown.contains(&expected),
        "expected {title:?} in the details pane: {}",
        world.text()
    );
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

#[given("the selected template folder is not the English folder")]
fn selected_folder_is_not_english(world: &mut ApplicationWorld) {
    assert!(
        world.row_accented_beside("Metric", "English"),
        "expected Metric selected"
    );
}

#[test]
fn double_clicking_a_medium_icon_creates_the_selected_part() {
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    world.key(ViewportPhysicalKey::N, true);
    choose_template_view(&mut world, "Medium Icons");
    let center = super::super::diagnostics::pointer_pixels(world.run(METRIC_PART).center());
    world.pump(super::super::pointer_click_events(center, 2));
    world.pump(Vec::new());
    assert_eq!(world.state().documents.len(), 2);
    assert_ne!(world.active_id(), first);
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.close();
}

#[test]
fn keyboard_selection_scrolls_narrow_icon_gallery_to_each_endpoint() {
    let mut world = ApplicationWorld::new();
    super::super::super::verify_gui_resize_recovery(&mut world.app().window.native, [800, 600]);
    world
        .app()
        .owner
        .set_ui_scale_from_gui(oblikovati_model::ui_scale::UiScaleOptions::new(1.5, 1.).unwrap())
        .unwrap();
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::N, true);
    choose_template_view(&mut world, "Medium Icons");
    world.click_text(METRIC_SHEET_METAL);
    expect_selected_template(&world, METRIC_SHEET_METAL);
    let first_y = world.run(METRIC_SHEET_METAL).center()[1];
    world.key(ViewportPhysicalKey::End, false);
    world.pump(Vec::new());
    expect_selected_template(&world, METRIC_PART);
    world.save_capture("template-icon-scroll-end");
    world.expect_text(METRIC_PART);
    assert!(
        (world.run(METRIC_PART).center()[1] - first_y).abs() < 3.,
        "expected the same vertical position"
    );
    world.key(ViewportPhysicalKey::Home, false);
    world.pump(Vec::new());
    expect_selected_template(&world, METRIC_SHEET_METAL);
    world.expect_text(METRIC_SHEET_METAL);
    world.close();
}
