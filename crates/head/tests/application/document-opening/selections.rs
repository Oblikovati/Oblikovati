// SPDX-License-Identifier: GPL-2.0-only
//! Document-opening scenarios on what is selected, how it opens, and thumbnails.
use super::*;

const EXPRESS: &str = "Load in Express Mode";
pub(super) const DEFER: &str = "Load in Defer Mode";

#[cfg(test)]
#[path = "assembly_placement_tests.rs"]
mod assembly_placement_tests;

fn documents(world: &mut ApplicationWorld) -> usize {
    world.state().documents.len()
}
pub(super) fn click_option(world: &mut ApplicationWorld, label: &str) {
    world.click_text(label);
    settle(world);
}
/// Close what the fixture opened by saving, as a file that is "not already open".
fn close_active(world: &mut ApplicationWorld) {
    world.close_active_tab();
    settle(world);
}
pub(super) fn switch_to_tiles(world: &mut ApplicationWorld) {
    world.click_text("Tiles");
    settle(world);
}
fn drawings_in_tiles(world: &mut ApplicationWorld) -> PathBuf {
    if let Some(folder) = published_drawings::staged_folder(world) {
        open_dialog(world);
        look_in(world, &folder);
        switch_to_tiles(world);
        return folder;
    }
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
    folder
}
pub(super) fn rectangle_over_drawings(world: &mut ApplicationWorld) {
    let [first, second] = drawing_names(world).map(|name| world.run(name).clone());
    let stride = second.minimum[0] - first.minimum[0];
    assert!(stride > 0. && (first.center()[1] - second.center()[1]).abs() < 4.);
    // Start beside the row so a Part underneath the first drawing is outside the rectangle.
    let start = [second.minimum[0] + stride - 34., second.center()[1] + 14.];
    world.drag(
        start.map(|coordinate| coordinate as i32),
        first.center().map(|coordinate| coordinate as i32),
    );
    settle(world);
}

pub(super) fn drawing_names(world: &ApplicationWorld) -> [&'static str; 2] {
    [
        published_drawings::first_drawing_name(world),
        published_drawings::second_drawing_name(world),
    ]
}
// Assembly options
#[given("a saved assembly with a preview is selected")]
fn assembly_selected(world: &mut ApplicationWorld) {
    cancel_dialog(world);
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Assembly, &folder, ASSEMBLY_FILE);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, ASSEMBLY_FILE);
}
#[given("the Options pane displays Load in Express Mode unchecked")]
fn express_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked(EXPRESS), "expected {EXPRESS} unchecked");
}
#[given("Model State, Design View and Position View controls are displayed")]
fn assembly_rows(world: &mut ApplicationWorld) {
    for row in ["Model State", "Design View", "Position View"] {
        assert!(holds_text(world, row), "expected {row}");
    }
}
#[when("I select Load in Express Mode")]
#[when("the user selects Load in Express Mode")]
#[when("I clear Load in Express Mode")]
#[when("the user clears Load in Express Mode")]
fn toggle_express(world: &mut ApplicationWorld) {
    click_option(world, EXPRESS);
}
#[then("Design View and Position View become unavailable")]
fn views_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Design View"), "Design View");
    assert!(
        !world.run_is_enabled("Position View: Primary"),
        "Position View"
    );
}
#[then("Model State remains available")]
fn state_on(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Model State: Primary"), "Model State");
}
#[then("Design View and Position View become available again")]
fn views_on(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Design View"), "Design View");
    assert!(
        world.run_is_enabled("Position View: Primary"),
        "Position View"
    );
}
#[then("the assembly preview remains displayed")]
fn assembly_preview(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}

#[cfg(test)]
#[path = "assembly_loading_tests.rs"]
mod assembly_loading_tests;

// Already open files
#[given("a saved part is open in the MCAD Software")]
pub(super) fn part_open(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    let id = saved_document(world, GuiTemplateKind::Part, &folder, PART_FILE);
    world.parts.push(id);
}
#[given("a different drawing document is active")]
pub(super) fn drawing_on_top(world: &mut ApplicationWorld) {
    world.other_document = Some(world.add_template(GuiTemplateKind::Drawing));
    world.observed_counts = vec![documents(world)];
    world.observed_document_windows = world.state().document_windows;
}
#[given("the open part's exact file location is known")]
pub(super) fn location_known(world: &mut ApplicationWorld) {
    assert!(
        models_folder(world).join(PART_FILE).exists(),
        "expected the saved part"
    );
}
#[when("I choose Open")]
#[when("the user chooses Open")]
fn press_open(world: &mut ApplicationWorld) {
    choose_open(world);
}
#[then("the Open dialog closes")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
}
#[then("the existing part document tab becomes active")]
pub(super) fn part_tab_active(world: &mut ApplicationWorld) {
    let part = *world.parts.last().expect("expected the saved part");
    assert_eq!(world.active_id(), part, "active document");
}
#[then("no additional document tab is created")]
pub(super) fn no_extra_tab(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0], "documents");
    assert_eq!(
        world.state().document_windows,
        world.observed_document_windows,
        "document windows"
    );
}
#[then("the drawing tab remains available")]
pub(super) fn drawing_tab_kept(world: &mut ApplicationWorld) {
    let drawing = world.other_document.expect("expected the original drawing");
    let name = world.document_name(drawing);
    world.expect_tab(&name);
}
// Thumbnails
#[given(
    "the Open dialog in the MCAD Software displays a saved drawing and a saved part as thumbnails"
)]
fn drawing_and_part_tiles(world: &mut ApplicationWorld) {
    if let Some(folder) = published_drawings::staged_folder(world) {
        open_dialog(world);
        look_in(world, &folder);
        switch_to_tiles(world);
        return;
    }
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(world, GuiTemplateKind::Part, &folder, PART_FILE);
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
}
#[given("the drawing thumbnail is selected with its sheet preview displayed")]
fn drawing_tile_selected(world: &mut ApplicationWorld) {
    let name = published_drawings::first_drawing_name(world);
    select_row(world, name);
    expect_name_and_preview(world, name);
}
#[given("the Options pane shows the drawing loading choices")]
fn drawing_choices(world: &mut ApplicationWorld) {
    world.expect_text(DEFER);
}
#[when("I select the part thumbnail")]
#[when("the user selects the part thumbnail")]
fn select_part_tile(world: &mut ApplicationWorld) {
    select_row(world, published_drawings::part_name(world));
}
#[then("the part thumbnail is highlighted")]
fn part_tile_lit(world: &mut ApplicationWorld) {
    assert!(
        row_lit(world, published_drawings::part_name(world)),
        "expected the tile's accent"
    );
}
#[then("the file name field identifies the part")]
fn name_is_part(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    let name = published_drawings::part_name(world);
    assert!(shown.contains(name), "file name {shown:?}: expected {name}");
}

#[given("the working folder contains `OpenDrawingA.odd` and `OpenPart.opd` as thumbnails")]
fn staged_part_and_drawing_tiles(world: &mut ApplicationWorld) {
    world.expect_no_text("Date modified");
    world.expect_text(published_drawings::first_drawing_name(world));
    world.expect_text(published_drawings::part_name(world));
}
#[then("the separate preview displays the part geometry")]
fn separate_preview(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
}
// The context menu
#[given("the Open dialog in the MCAD Software displays a saved part as a thumbnail")]
fn part_tile(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Part, &folder, PART_FILE);
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
}
#[given("the part is selected with its file name and preview displayed")]
fn part_tile_selected(world: &mut ApplicationWorld) {
    select_row(world, PART_FILE);
    expect_name_and_preview(world, PART_FILE);
}

#[given("`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is selected as a thumbnail")]
fn published_part_tile(world: &mut ApplicationWorld) {
    published_preview::preview_target_selected(world);
    world.expect_no_text("Date modified");
}

fn context_part(world: &ApplicationWorld) -> &str {
    published_preview::inspection_path(world)
        .and_then(|path| path.file_name()?.to_str())
        .unwrap_or(PART_FILE)
}
#[when("I right-click the part thumbnail")]
#[when("the user right-clicks the part thumbnail")]
fn right_click_tile(world: &mut ApplicationWorld) {
    let at = world.run(context_part(world)).center();
    world.right_click(super::super::diagnostics::pointer_pixels(at));
    settle(world);
}
#[then("a context menu appears with Open, Copy, Rename and Properties choices")]
fn tile_menu(world: &mut ApplicationWorld) {
    let menu = world
        .frame
        .runs
        .iter()
        .find(|row| row.text == "Rename" && world.frame.is_popup_row(row))
        .expect("expected the thumbnail context menu");
    let choices: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|row| row.list == menu.list)
        .map(|row| row.text.as_str())
        .collect();
    for choice in ["Open", "Copy", "Rename", "Properties"] {
        assert!(
            choices.contains(&choice),
            "missing {choice:?} in {choices:?}"
        );
    }
}
#[when("I click an empty area of the file list")]
#[when("the user clicks an empty area of the file list")]
fn click_empty_list(world: &mut ApplicationWorld) {
    use super::super::diagnostics::pointer_pixels;
    let gutter = file_list_gutter();
    world.click(pointer_pixels(gutter));
    settle(world);
}

fn file_list_gutter() -> [f32; 2] {
    let list = super::super::diagnostics::child_scroll("Open", "##folder-listing")
        .expect("expected the folder listing");
    // The tile grid leaves padding before the scrollbar, even on a full screen of rows.
    [list.content_right - 3., list.center[1]]
}
#[then("the selected-item count is cleared")]
fn no_selected_items(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "1 item selected"),
        "selection count remains"
    );
    let tile = world.run(context_part(world));
    let y = tile.center()[1];
    use super::super::diagnostics::pointer_pixels;
    let row = pointer_pixels([tile.maximum[0] + 8., y]).map(|value| value as usize);
    let plain = pointer_pixels([file_list_gutter()[0], y]).map(|value| value as usize);
    assert_eq!(
        world.pixel_at(row),
        world.pixel_at(plain),
        "part thumbnail remains selected"
    );
}
#[then("the dialog remains open with the part name and preview displayed")]
fn open_with_part(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    let name = context_part(world).to_owned();
    expect_name_and_preview(world, &name);
}
#[test]
fn file_row_context_menu_properties_remains_clickable() {
    let mut world = ApplicationWorld::new();
    part_tile(&mut world);
    part_tile_selected(&mut world);
    right_click_tile(&mut world);
    world.click_text("Properties");
    settle(&mut world);
    world.expect_text("File Properties");
    world.close();
}

#[cfg(target_os = "linux")]
#[path = "native_context.rs"]
mod native_context;
// Several drawings
#[given("the Open dialog in the MCAD Software displays two adjacent saved drawings as thumbnails")]
fn adjacent_tiles(world: &mut ApplicationWorld) {
    drawings_in_tiles(world);
}
#[given("each drawing has a saved sheet preview")]
fn each_preview(world: &mut ApplicationWorld) {
    for name in drawing_names(world) {
        select_row(world, name);
        expect_name_and_preview(world, name);
    }
}

#[given("the dialog folder contains adjacent `OpenDrawingA.odd` and `OpenDrawingB.odd` thumbnails")]
fn staged_adjacent_drawings(world: &mut ApplicationWorld) {
    let [first, second] = drawing_names(world).map(|name| world.run(name).center());
    assert!((first[1] - second[1]).abs() < 4. && first[0] < second[0]);
    world.expect_no_text("Date modified");
}
#[when("I drag a selection rectangle from empty list space across both thumbnails")]
#[when("the user drags a selection rectangle from empty list space across both thumbnails")]
#[when("I select both drawing thumbnails with a selection rectangle")]
#[when("the user selects both drawing thumbnails with a selection rectangle")]
fn drag_rectangle(world: &mut ApplicationWorld) {
    rectangle_over_drawings(world);
    both_lit(world);
    two_selected(world);
}
#[then("both drawing thumbnails are selected")]
fn both_lit(world: &mut ApplicationWorld) {
    for name in drawing_names(world) {
        assert!(row_lit(world, name), "expected selected tile {name}");
    }
}
#[then("the status area reports two selected items")]
fn two_selected(world: &mut ApplicationWorld) {
    assert!(holds_text(world, "2 items selected"), "expected the count");
}
#[then("the file name field contains both drawing names in quotes")]
fn quoted(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    for name in drawing_names(world) {
        assert!(
            shown.contains(&format!("\"{name}\"")),
            "file name {shown:?}"
        );
    }
}
#[then("the separate preview area displays a placeholder")]
fn several_placeholder(world: &mut ApplicationWorld) {
    expect_placeholder(world);
}
#[then("individual drawing loading options are not displayed")]
fn no_options(world: &mut ApplicationWorld) {
    world.expect_no_text(DEFER);
}
#[when("I select one of the drawing thumbnails")]
#[when("the user selects one of the drawing thumbnails")]
fn select_one(world: &mut ApplicationWorld) {
    let [first, _] = drawing_names(world);
    select_row(world, first);
}
#[then("only that drawing is selected")]
fn only_one(world: &mut ApplicationWorld) {
    let [first, second] = drawing_names(world);
    assert!(row_lit(world, first), "the chosen tile");
    assert!(!row_lit(world, second), "the other tile");
}
#[then("the file name field identifies that drawing")]
fn name_that(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    let [first, _] = drawing_names(world);
    assert!(
        shown.contains(first) && !shown.contains('"'),
        "file name {shown:?}"
    );
}
#[then("its sheet preview and drawing loading options are displayed")]
fn its_options(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
    world.expect_text(DEFER);
}
#[path = "multi_reopen.rs"]
mod multi_reopen;
