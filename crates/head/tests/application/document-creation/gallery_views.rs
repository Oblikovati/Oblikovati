// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[path = "captured_catalog.rs"]
mod captured_catalog;
#[path = "chooser_details.rs"]
mod chooser_details;
#[path = "chooser_navigation_regressions.rs"]
mod chooser_navigation_regressions;
#[path = "gallery_folder_keyboard.rs"]
mod gallery_folder_keyboard;
#[path = "gallery_icons.rs"]
mod gallery_icons;
#[path = "gallery_secondary.rs"]
mod gallery_secondary;
#[path = "named_catalog.rs"]
mod named_catalog;
#[path = "named_navigation.rs"]
mod named_navigation;
#[path = "project_inspection.rs"]
mod project_inspection;
#[path = "template_content.rs"]
mod template_content;
#[given("the new-document chooser shows the metric folder grouped by type using Medium icons")]
#[given("the template chooser shows the metric folder grouped by type using Medium icons")]
fn metric_medium_chooser(world: &mut ApplicationWorld) {
    open_chooser(world);
    if !world.state().project_templates.is_empty() {
        named_navigation::metric_icons(world);
        return;
    }
    metric_selected(world);
    grouped_by_default(world);
    choose_template_view(world, "Medium Icons");
    medium_tiles(world);
}

#[given("the template chooser shows the metric folder grouped by type")]
#[given("the template chooser shows the metric folder and its grouped templates")]
fn metric_grouped_chooser(world: &mut ApplicationWorld) {
    open_chooser(world);
    metric_selected(world);
    grouped_by_default(world);
}
#[given("the template chooser shows the English folder grouped by type using Medium icons")]
fn english_medium_chooser(world: &mut ApplicationWorld) {
    open_chooser(world);
    world.click_text("English");
    grouped_by_default(world);
    choose_template_view(world, "Medium Icons");
    world.expect_text("Standard inch part");
}
#[given("the metric template folder is selected")]
fn metric_selected(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    metric_gallery(world);
}
fn expect_part_details(world: &mut ApplicationWorld) {
    let title = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    expect_selected_template(world, title);
    metric_details(world);
}
#[given("a standard metric part template's details are displayed")]
fn part_details_shown(world: &mut ApplicationWorld) {
    select_metric(world);
    expect_part_details(world);
}
#[given("templates are grouped by type")]
fn grouped_by_default(world: &mut ApplicationWorld) {
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
}
pub(super) fn expect_group_headings(world: &mut ApplicationWorld, headings: &[&str]) {
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
    let templates = basic_catalog::metric_part_titles(world);
    for title in templates {
        assert!(
            gallery_has_template(world, &title),
            "expected {title} in the flat gallery"
        );
    }
    other_group_templates(world);
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
fn first_two_part_titles(world: &mut ApplicationWorld) -> (String, String) {
    let mut titles = basic_catalog::metric_part_titles(world);
    titles.sort_by_cached_key(|title| title.to_lowercase());
    let [first, second, ..] = titles.as_slice() else {
        panic!("expected at least two Metric Part templates");
    };
    (first.clone(), second.clone())
}
fn tile_spacing(world: &mut ApplicationWorld) -> f32 {
    let (first, second) = first_two_part_titles(world);
    (world.run(&first).center()[1] - world.run(&second).center()[1]).abs()
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
    select_assembly(world);
}
fn expect_assembly_details(world: &mut ApplicationWorld) {
    let title = basic_catalog::catalog_title(world, METRIC_ASSEMBLY, "Standard Assembly");
    expect_selected_template(world, title);
    expect_selected_template(world, "Document type: Assembly");
    expect_selected_template(world, "Length: millimeters (mm)");
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
fn expect_icon_row(world: &mut ApplicationWorld, expected_size: f32) {
    let (left, right) = first_two_part_titles(world);
    let first = world.run(&left).center();
    let second = world.run(&right).center();
    assert!(
        second[0] > first[0] && (second[1] - first[1]).abs() < 3.,
        "expected template icons in a horizontal row"
    );
    let size = gallery_icons::template_icon_size(world, &left);
    assert!(
        (size[0] - expected_size).abs() < 1. && (size[1] - expected_size).abs() < 1.,
        "icon size {size:?}: expected {expected_size}"
    );
}
#[when("I choose Back in the chooser navigation bar")]
#[when("the user chooses Back in the chooser navigation bar")]
fn chooser_back(world: &mut ApplicationWorld) {
    world.click_text("Back");
}
#[when("I choose Forward in the chooser navigation bar")]
#[when("the user chooses Forward in the chooser navigation bar")]
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
    let inch = basic_catalog::catalog_title(world, "Standard inch part", "Standard (in)");
    let metric = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    assert!(gallery_has_template(world, inch));
    assert!(!gallery_has_template(world, metric));
}
#[then("the gallery displays its metric templates")]
fn metric_gallery(world: &mut ApplicationWorld) {
    let inch = basic_catalog::catalog_title(world, "Standard inch part", "Standard (in)");
    let metric = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    assert!(gallery_has_template(world, metric));
    assert!(!gallery_has_template(world, inch));
}
fn gallery_has_template(world: &ApplicationWorld, name: &str) -> bool {
    let details = world.run("Template details").minimum[0];
    let expected = name.replace(' ', "");
    world
        .frame
        .runs
        .iter()
        .any(|run| run.minimum[0] < details && run.text.replace(' ', "") == expected)
}
#[given("the metric folder and standard metric assembly template are selected")]
#[given("the standard metric assembly template details are displayed")]
fn assembly_details_shown(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    select_assembly(world);
    expect_assembly_details(world);
}
#[when("I reopen the new-document chooser during the same session")]
#[when("the user reopens the new-document chooser during the same session")]
fn reopen_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.pump(Vec::new());
}
#[then("the standard metric assembly template is selected")]
fn assembly_tile_selected(world: &mut ApplicationWorld) {
    let title = basic_catalog::catalog_title(world, METRIC_ASSEMBLY, "Standard Assembly");
    assert!(gallery_has_template(world, title));
    expect_selected_template(world, title);
}
#[then("its details identify an assembly with millimeter units")]
fn its_assembly_details(world: &mut ApplicationWorld) {
    expect_assembly_details(world);
}
#[given("the template root has no templates directly in it")]
fn root_has_no_templates(world: &mut ApplicationWorld) {
    basic_catalog::catalog_root_is_empty(world);
    world.expect_text("Templates");
}
#[when("I select Home in the chooser navigation bar")]
#[when("the user selects Home in the chooser navigation bar")]
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
    let templates = basic_catalog::metric_part_titles(world);
    assert!(
        templates
            .iter()
            .all(|title| !gallery_has_template(world, title))
    );
    let drawing = basic_catalog::catalog_title(world, METRIC_DRAWING, "Standard Drawing");
    let assembly = basic_catalog::catalog_title(world, METRIC_ASSEMBLY, "Standard Assembly");
    assert!(!gallery_has_template(world, drawing));
    assert!(!gallery_has_template(world, assembly));
}
#[then("the metric assembly template details remain displayed")]
fn assembly_details_remain(world: &mut ApplicationWorld) {
    expect_assembly_details(world);
}
#[then("the metric template folder and its gallery are displayed again")]
fn metric_folder_back(world: &mut ApplicationWorld) {
    metric_folder_selected(world);
    metric_gallery(world);
}
#[given("the metric folder is selected and templates are grouped by type")]
fn metric_grouped(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    grouped_by_default(world);
}
#[when("I choose Extra large icons in the template display menu")]
#[when("the user chooses Extra large icons in the template display menu")]
fn choose_extra_large(world: &mut ApplicationWorld) {
    choose_template_view(world, "Extra Large Icons");
}
#[then("the template gallery displays extra-large tiles")]
fn extra_large_tiles(world: &mut ApplicationWorld) {
    expect_icon_row(world, 56.);
}
#[when("I choose Small icons in the template display menu")]
#[when("the user chooses Small icons in the template display menu")]
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
#[when("the user double-clicks the standard metric part template")]
fn double_click_part(world: &mut ApplicationWorld) {
    let title = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    let center = world.run(title).center().map(|value| value as i32);
    world.pump(super::super::pointer_click_events(center, 2));
    world.pump(Vec::new());
}
#[when("I press Enter")]
#[when("the user presses Enter")]
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
    let state = world.state();
    let document = state.document.expect("expected the original Part");
    if state.document_storage[&document].path.is_some() {
        super::super::workspace_layout::browser_feature::expect_solid_visible(world);
    } else {
        world.expect_blank_region(VIEWPORT_X, VIEWPORT_Y);
    }
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
    metric_gallery(world);
    other_group_templates(world);
}

fn other_group_templates(world: &mut ApplicationWorld) {
    for (built_in, native) in [
        (METRIC_ASSEMBLY, "Standard Assembly"),
        (METRIC_DRAWING, "Standard Drawing"),
    ] {
        let title = basic_catalog::catalog_title(world, built_in, native);
        assert!(
            gallery_has_template(world, title),
            "expected {title} in the gallery"
        );
    }
}

pub(super) fn expect_selected_template(world: &ApplicationWorld, title: &str) {
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
