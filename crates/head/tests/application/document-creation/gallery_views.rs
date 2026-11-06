// SPDX-License-Identifier: GPL-2.0-only
use super::*;
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
    world.run(METRIC_SHEET_METAL).center()[1] - world.run(METRIC_PART).center()[1]
}
#[when("I choose List in the template display menu")]
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
fn choose_medium_view(world: &mut ApplicationWorld) {
    choose_template_view(world, "Medium Icons");
}
#[then("the gallery displays medium-sized template icons")]
fn medium_tiles(world: &mut ApplicationWorld) {
    let spacing = tile_spacing(world);
    assert!(
        spacing > 30.,
        "tile spacing {spacing}: expected medium tiles"
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
    let spacing = tile_spacing(world);
    assert!(
        spacing > 50.,
        "tile spacing {spacing}: expected extra-large tiles"
    );
}
#[when("I choose Small icons in the template display menu")]
fn choose_small(world: &mut ApplicationWorld) {
    choose_template_view(world, "Small Icons");
}
#[then("the template gallery displays small icons")]
fn small_tiles(world: &mut ApplicationWorld) {
    let spacing = tile_spacing(world);
    assert!(
        spacing < 40.,
        "tile spacing {spacing}: expected small tiles"
    );
}
#[then("the template gallery displays medium-sized icons")]
fn medium_sized(world: &mut ApplicationWorld) {
    medium_tiles(world);
}
