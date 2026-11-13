// SPDX-License-Identifier: GPL-2.0-only
//! Document-opening scenarios on what is selected, how it opens, and thumbnails.
use super::*;

const EXPRESS: &str = "Load in Express Mode";
const DEFER: &str = "Load in Defer Mode";
const SKIP: &str = "Skip all unresolved files";

#[test]
fn assembly_ribbon_places_an_open_component_and_replays_it() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let fixture = crate::scene_fixture::selected_prism_document();
    let part = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap()
    .with_parameters(&fixture.parameters, Default::default())
    .unwrap()
    .with_sketches(&fixture.sketches, 1, 104)
    .unwrap()
    .with_work_geometry(&fixture.work_geometry, 1)
    .unwrap();
    std::fs::write(folder.join("place-part.opd"), part.encode().unwrap()).unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "place-part.opd");
    choose_open(&mut world);
    let component = world.active_id();
    let name = world
        .state()
        .documents
        .iter()
        .find(|row| row.id == component)
        .unwrap()
        .name
        .clone();
    let assembly = saved_document(
        &mut world,
        GuiTemplateKind::Assembly,
        &folder,
        "place-root.oad",
    );
    world.click_text("Place");
    world.expect_text("Choose an open saved component");
    world.click_text("Cancel");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        0
    );
    world.click_text("Place");
    world.click_text("Choose an open saved component");
    world.click_text(&name);
    let button = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Place")
        .max_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .unwrap()
        .center();
    world.click(super::super::diagnostics::pointer_pixels(button));
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    assert_eq!(world.active_id(), assembly);
    world.key(ViewportPhysicalKey::Z, true);
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        0
    );
    world.key(ViewportPhysicalKey::Y, true);
    settle(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    world.key(ViewportPhysicalKey::S, true);
    settle(&mut world);
    world.close_active_tab();
    settle(&mut world);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "place-root.oad");
    choose_open(&mut world);
    settle(&mut world);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 1);
    assert!(status.scene_errors.is_empty(), "{:?}", status.scene_errors);
    world.close();
}

fn documents(world: &mut ApplicationWorld) -> usize {
    world.state().documents.len()
}
fn click_option(world: &mut ApplicationWorld, label: &str) {
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
    let first = world.run(DRAWING_FILE).clone();
    let second = world.run(SECOND_DRAWING_FILE).clone();
    let top = first.center()[1].min(second.center()[1]) as i32;
    let bottom = first.center()[1].max(second.center()[1]) as i32;
    let left = first.minimum[0].min(second.minimum[0]) as i32;
    let right = first.maximum[0].max(second.maximum[0]) as i32;
    // From the empty space under the last tile, up across both.
    world.drag([left, bottom + 80], [right, top]);
    settle(world);
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

#[test]
fn express_open_refuses_mouse_and_enter_until_full_mode_is_selected() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let folder = models_folder(&mut world);
    saved_document(
        &mut world,
        GuiTemplateKind::Assembly,
        &folder,
        ASSEMBLY_FILE,
    );
    close_active(&mut world);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    toggle_express(&mut world);
    world.expect_text("Express assembly loading is not available");
    choose_open(&mut world);
    world.key(ViewportPhysicalKey::Enter, false);
    let center = world.run(ASSEMBLY_FILE).center().map(|value| value as i32);
    world.pump(super::super::pointer_click_events(center, 2));
    settle(&mut world);
    world.expect_text("Look in");
    assert_eq!(world.active_id(), part);
    toggle_express(&mut world);
    choose_open(&mut world);
    world.expect_no_text("Look in");
    assert_ne!(world.active_id(), part);
    world.close();
}

#[test]
fn populated_assembly_reports_a_missing_component() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let folder = models_folder(&mut world);
    std::fs::write(folder.join(ASSEMBLY_FILE), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}]}\n").unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assert_ne!(world.active_id(), part);
    let status = world.app().window.native.scene_status().unwrap();
    assert!(
        status
            .scene_errors
            .iter()
            .any(|error| error.contains("part.opd"))
    );
    assert_eq!(status.visible_bodies, 0);
    world.expect_text("Part:1");
    world.expect_text("[unresolved]");
    world.close();
}

#[test]
fn opening_an_assembly_displays_its_referenced_part() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let fixture = crate::scene_fixture::selected_prism_document();
    let part = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap()
    .with_parameters(&fixture.parameters, Default::default())
    .unwrap()
    .with_sketches(&fixture.sketches, 1, 104)
    .unwrap()
    .with_work_geometry(&fixture.work_geometry, 1)
    .unwrap();
    std::fs::write(folder.join("part.opd"), part.encode().unwrap()).unwrap();
    std::fs::write(folder.join("nested.oad"), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:2', component: part.opd}]}\n").unwrap();
    std::fs::write(folder.join(ASSEMBLY_FILE), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}, {name: 'Nested:1', component: nested.oad, transform: [1, 0, 0, 5, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]}, {name: 'Suppressed:1', component: part.opd, suppressed: true}]}\n").unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    settle(&mut world);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 2);
    assert!(status.scene_errors.is_empty(), "{:?}", status.scene_errors);
    world.expect_text("Part:1");
    world.expect_text("Nested:1");
    world.expect_text("Part:2");
    let brightness = |run: &super::super::diagnostics::RenderedRun| {
        run.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(run.alpha)
    };
    let active = brightness(world.run("Part:1"));
    let suppressed = brightness(world.run("Suppressed:1"));
    assert!(suppressed < active);
    world.click_text("Part:2");
    let selected = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(selected.len(), 2);
    assert_selected_and_unselected_solids(&mut world, true);
    world.click_text("Nested:1");
    let parent = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(parent.len(), 1);
    assert_selected_and_unselected_solids(&mut world, true);
    world.expect_text("Part:2");
    world.click_text("Part:1");
    let other = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(other.len(), 1);
    assert_ne!(selected, other);
    assert_ne!(parent, other);
    assert_selected_and_unselected_solids(&mut world, false);
    world.save_capture("assembly-referenced-part");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let (minimum, maximum) = super::super::solid_visibility::visible_fixture_bounds(
        &capture,
        &super::super::diagnostics::window_occlusions(),
    )
    .expect("expected the unselected green occurrence");
    let point = std::array::from_fn(|axis| ((minimum[axis] + maximum[axis]) / 2) as i32);
    world.click(point);
    assert_eq!(
        world.state().browser.as_ref().unwrap().selected_occurrence,
        Some(selected)
    );
    assert_selected_and_unselected_solids(&mut world, true);
    world.save_capture("assembly-picked-occurrence");
    world.click([400, 700]);
    assert!(
        world
            .state()
            .browser
            .as_ref()
            .unwrap()
            .selected_occurrence
            .is_none()
    );
    let before = green_assembly_bounds(&mut world);
    occurrence_menu(&mut world, "Part:1", "Suppress");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    let after = green_assembly_bounds(&mut world);
    assert!(
        after.0[0] > before.0[0],
        "expected the left placement to disappear"
    );
    world.save_capture("assembly-suppressed-occurrence");
    world.key(ViewportPhysicalKey::Z, true);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.key(ViewportPhysicalKey::Y, true);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    occurrence_menu(&mut world, "Part:1", "Unsuppress");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.click_text("Save");
    close_active(&mut world);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.click_text("Search");
    world.pump(Vec::new());
    super::super::workspace_layout::browser_tree::type_in_search(&mut world, "Part:2");
    world.expect_text("Nested:1");
    world.expect_text("Part:2");
    world.expect_no_text("Part:1");
    world.close();
}
fn occurrence_menu(world: &mut ApplicationWorld, name: &str, command: &str) {
    let center = world.run(name).center();
    world.right_click(super::super::diagnostics::pointer_pixels(center));
    world.click_text(command);
}
fn green_assembly_bounds(world: &mut ApplicationWorld) -> ([usize; 2], [usize; 2]) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    super::super::solid_visibility::visible_fixture_bounds(
        &capture,
        &super::super::diagnostics::window_occlusions(),
    )
    .expect("expected a visible assembly component")
}
fn assert_selected_and_unselected_solids(world: &mut ApplicationWorld, selected_right: bool) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let mut colors = [0usize; 2];
    let mut x_totals = [0usize; 2];
    for (index, pixel) in capture.rgba.chunks_exact(4).enumerate() {
        if index % width < 320 || index / width < 200 {
            continue;
        }
        if pixel[1] > pixel[0].saturating_add(30) && pixel[1] > pixel[2].saturating_add(30) {
            colors[0] += 1;
            x_totals[0] += index % width;
        }
        if pixel[2] > pixel[0].saturating_add(30) && pixel[2] > pixel[1].saturating_add(30) {
            colors[1] += 1;
            x_totals[1] += index % width;
        }
    }
    assert!(
        colors.iter().all(|count| *count > 100),
        "expected green and blue solids, got {colors:?}"
    );
    let green_x = x_totals[0] / colors[0];
    let blue_x = x_totals[1] / colors[1];
    assert_eq!(
        blue_x > green_x,
        selected_right,
        "blue at {blue_x}, green at {green_x}"
    );
}
// Already open files
#[given("a saved part is open in the MCAD Software")]
fn part_open(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    let id = saved_document(world, GuiTemplateKind::Part, &folder, PART_FILE);
    world.parts.push(id);
}
#[given("a different drawing document is active")]
fn drawing_on_top(world: &mut ApplicationWorld) {
    world.add_template(GuiTemplateKind::Drawing);
    world.observed_counts = vec![documents(world)];
}
#[given("the open part's exact file location is known")]
fn location_known(world: &mut ApplicationWorld) {
    assert!(
        models_folder(world).join(PART_FILE).exists(),
        "expected the saved part"
    );
}
#[when("I press Control+O and choose the already open part at that location")]
#[when("the user presses Control+O and chooses the already open part at that location")]
fn choose_open_part(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, PART_FILE);
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
fn part_tab_active(world: &mut ApplicationWorld) {
    let part = *world.parts.last().expect("expected the saved part");
    assert_eq!(world.active_id(), part, "active document");
}
#[then("no additional document tab is created")]
fn no_extra_tab(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0], "documents");
}
#[then("the drawing tab remains available")]
fn drawing_tab_kept(world: &mut ApplicationWorld) {
    assert!(documents(world) >= 2, "expected the drawing still open");
}
// Drawing options
#[given("a saved drawing is selected with its Options pane displayed")]
fn drawing_selected(world: &mut ApplicationWorld) {
    cancel_dialog(world);
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, DRAWING_FILE);
}
#[given("Load in Defer Mode is unchecked")]
#[given("Load in Defer Mode is unchecked for the first drawing")]
#[then("Load in Defer Mode is unchecked")]
#[then("Load in Defer Mode is unchecked for the second drawing")]
fn defer_off(world: &mut ApplicationWorld) {
    assert!(!world.checked(DEFER), "expected {DEFER} unchecked");
}
#[given("Skip all unresolved files is checked")]
fn skip_on(world: &mut ApplicationWorld) {
    click_option(world, SKIP);
    assert!(world.checked(SKIP), "expected {SKIP} checked");
}
#[when("I select Load in Defer Mode")]
#[when("the user selects Load in Defer Mode")]
#[when("I clear Load in Defer Mode")]
#[when("the user clears Load in Defer Mode")]
fn toggle_defer(world: &mut ApplicationWorld) {
    click_option(world, DEFER);
}
#[then("Skip all unresolved files is unavailable")]
fn skip_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled(SKIP), "expected {SKIP} unavailable");
}
#[then("Skip all unresolved files is available and unchecked")]
fn skip_clear(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled(SKIP), "expected {SKIP} available");
    assert!(!world.checked(SKIP), "expected {SKIP} unchecked");
}
#[then("the selected drawing and its preview remain displayed")]
fn drawing_kept(world: &mut ApplicationWorld) {
    expect_name_and_preview(world, DRAWING_FILE);
}
#[given("two saved drawings are visible in the same folder")]
fn two_drawings(world: &mut ApplicationWorld) {
    cancel_dialog(world);
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
}
#[given("the first drawing is selected in the Options pane")]
fn first_selected(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
}
#[when("I select the second drawing")]
#[when("the user selects the second drawing")]
fn select_second(world: &mut ApplicationWorld) {
    select_row(world, SECOND_DRAWING_FILE);
}
#[then("the second drawing is selected and its file name is displayed")]
fn second_shown(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(shown.contains(SECOND_DRAWING_FILE), "file name {shown:?}");
}
#[when("I select the first drawing again")]
#[when("the user selects the first drawing again")]
fn select_first_again(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
}
#[then("the drawing remains selected without being opened")]
fn still_selected(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    let shown = file_name_shown(world);
    assert!(shown.contains(DRAWING_FILE), "file name {shown:?}");
}
// Thumbnails
#[given(
    "the Open dialog in the MCAD Software displays a saved drawing and a saved part as thumbnails"
)]
fn drawing_and_part_tiles(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(world, GuiTemplateKind::Part, &folder, PART_FILE);
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
}
#[given("the drawing thumbnail is selected with its sheet preview displayed")]
fn drawing_tile_selected(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
    expect_name_and_preview(world, DRAWING_FILE);
}
#[given("the Options pane shows the drawing loading choices")]
fn drawing_choices(world: &mut ApplicationWorld) {
    world.expect_text(DEFER);
}
#[when("I select the part thumbnail")]
#[when("the user selects the part thumbnail")]
fn select_part_tile(world: &mut ApplicationWorld) {
    select_row(world, PART_FILE);
}
#[then("the part thumbnail is highlighted")]
fn part_tile_lit(world: &mut ApplicationWorld) {
    assert!(row_lit(world, PART_FILE), "expected the tile's accent");
}
#[then("the file name field identifies the part")]
fn name_is_part(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(shown.contains(PART_FILE), "file name {shown:?}");
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
#[when("I right-click the part thumbnail")]
#[when("the user right-clicks the part thumbnail")]
fn right_click_tile(world: &mut ApplicationWorld) {
    let at = world.run(PART_FILE).center().map(|value| value as i32);
    world.right_click(at);
    settle(world);
}
#[then("a context menu appears with Open, Copy, Rename and Properties choices")]
fn tile_menu(world: &mut ApplicationWorld) {
    for row in ["Copy", "Rename", "Properties"] {
        world.expect_text(row);
    }
}
#[when("I click an empty area of the file list")]
#[when("the user clicks an empty area of the file list")]
fn click_empty_list(world: &mut ApplicationWorld) {
    let tile = world.run(PART_FILE).clone();
    world.click([tile.center()[0] as i32 + 120, tile.center()[1] as i32 + 90]);
    settle(world);
}
#[then("the selected-item count is cleared")]
fn no_selected_items(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "1 item selected"),
        "selection count remains"
    );
    assert!(
        !row_lit(world, PART_FILE),
        "part thumbnail remains selected"
    );
}
#[then("the dialog remains open with the part name and preview displayed")]
fn open_with_part(world: &mut ApplicationWorld) {
    world.expect_text("Look in");
    expect_name_and_preview(world, PART_FILE);
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
// Several drawings
#[given("the Open dialog in the MCAD Software displays two adjacent saved drawings as thumbnails")]
fn adjacent_tiles(world: &mut ApplicationWorld) {
    drawings_in_tiles(world);
}
#[given("each drawing has a saved sheet preview")]
fn each_preview(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
    expect_name_and_preview(world, DRAWING_FILE);
    select_row(world, SECOND_DRAWING_FILE);
    expect_name_and_preview(world, SECOND_DRAWING_FILE);
}
#[when("I drag a selection rectangle from empty list space across both thumbnails")]
#[when("the user drags a selection rectangle from empty list space across both thumbnails")]
#[when("I select both drawing thumbnails with a selection rectangle")]
#[when("the user selects both drawing thumbnails with a selection rectangle")]
fn drag_rectangle(world: &mut ApplicationWorld) {
    rectangle_over_drawings(world);
    both_lit(world);
}
#[then("both drawing thumbnails are selected")]
fn both_lit(world: &mut ApplicationWorld) {
    assert!(row_lit(world, DRAWING_FILE), "first tile");
    assert!(row_lit(world, SECOND_DRAWING_FILE), "second tile");
}
#[then("the status area reports two selected items")]
fn two_selected(world: &mut ApplicationWorld) {
    assert!(holds_text(world, "2 items selected"), "expected the count");
}
#[then("the file name field contains both drawing names in quotes")]
fn quoted(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    for name in [DRAWING_FILE, SECOND_DRAWING_FILE] {
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
    select_row(world, DRAWING_FILE);
}
#[then("only that drawing is selected")]
fn only_one(world: &mut ApplicationWorld) {
    assert!(row_lit(world, DRAWING_FILE), "the chosen tile");
    assert!(!row_lit(world, SECOND_DRAWING_FILE), "the other tile");
}
#[then("the file name field identifies that drawing")]
fn name_that(world: &mut ApplicationWorld) {
    let shown = file_name_shown(world);
    assert!(
        shown.contains(DRAWING_FILE) && !shown.contains('"'),
        "file name {shown:?}"
    );
}
#[then("its sheet preview and drawing loading options are displayed")]
fn its_options(world: &mut ApplicationWorld) {
    world.expect_no_text("No preview image");
    world.expect_text(DEFER);
}
// Reopening several
#[given("two saved drawings are already open in separate tabs in the MCAD Software")]
fn two_open(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    world.observed_counts = vec![documents(world)];
}
#[given("one of those drawings is active")]
fn one_active(world: &mut ApplicationWorld) {
    assert!(world.state().document.is_some(), "expected one active");
}
#[given("the Open dialog displays their exact saved files")]
fn dialog_shows_files(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
    switch_to_tiles(world);
}
#[when("I select both drawings and choose Open")]
#[when("the user selects both drawings and chooses Open")]
fn select_both_open(world: &mut ApplicationWorld) {
    rectangle_over_drawings(world);
    choose_open(world);
}
#[then("both existing drawing tabs remain available")]
#[then("no duplicate tab is created for either drawing")]
fn tabs_kept(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0], "documents");
}
#[then("one of those drawing tabs becomes active")]
fn one_tab_active(world: &mut ApplicationWorld) {
    assert!(world.state().document.is_some(), "expected one active");
}
#[given("a saved drawing is already open in a tab in the MCAD Software")]
fn one_open(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(world, GuiTemplateKind::Drawing, &folder, DRAWING_FILE);
}
#[given("a second saved drawing with available references is not open")]
fn second_closed(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    saved_document(
        world,
        GuiTemplateKind::Drawing,
        &folder,
        SECOND_DRAWING_FILE,
    );
    close_active(world);
    world.observed_counts = vec![documents(world)];
}
#[given("the Open dialog displays the exact files for both drawings")]
fn dialog_both_files(world: &mut ApplicationWorld) {
    dialog_shows_files(world);
}
#[then("the previously open drawing retains its existing tab without a duplicate")]
fn no_duplicate(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0] + 1, "documents");
}
#[then("the second drawing opens in a new tab")]
fn second_new_tab(world: &mut ApplicationWorld) {
    assert_eq!(documents(world), world.observed_counts[0] + 1, "documents");
}
#[then("the second drawing is active with its sheet displayed")]
fn second_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let active = state.document.expect("expected an active document");
    let path = state.document_storage[&active]
        .path
        .clone()
        .unwrap_or_default();
    assert!(path.ends_with(SECOND_DRAWING_FILE), "active file {path:?}");
    world.expect_text("Sheet 1 of 1");
}
