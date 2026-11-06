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
