// SPDX-License-Identifier: GPL-2.0-only
//! Desktop BDD bindings for Part Import scenarios (import-01..import-12).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_model::feature::imported_model::ImportMode;
use setup::{
    IMPORT_WINDOW, PICKER_WINDOW, STL_OPTIONS_WINDOW, click_popup_item, click_window_control,
    open_disposable_import_fixture, open_picker_and_select, reference_feature_name,
    right_click_reference_entry, wait_for_import_ready,
};

#[given(
    expr = "a disposable copy of the native fixture {string} is open in the 3D Model environment"
)]
fn disposable_import_fixture_in_3d_model(world: &mut ApplicationWorld, relative: String) {
    open_disposable_import_fixture(world, &relative);
}

#[then("the MCAD Software opens an Import file picker with a file type filter.")]
fn opens_import_picker_with_filter(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(PICKER_WINDOW).is_some(),
        "expected Import file picker {PICKER_WINDOW}, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(PICKER_WINDOW));
    assert!(
        text.contains("Filesoftype") || text.contains("CADandMeshFiles") || text.contains("STL"),
        "expected file type filter in {PICKER_WINDOW}, saw {text}"
    );
}

#[given(expr = "the user opens Import and selects {string} in the picker")]
fn user_opens_import_and_selects(world: &mut ApplicationWorld, relative: String) {
    open_picker_and_select(world, &relative);
}

#[when("the user clicks Open")]
fn user_clicks_open_in_picker(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Open");
}

#[then(
    "its geometry appears in the part and a named entry appears under 3rd Party in the browser."
)]
fn stl_geometry_and_third_party_appear(world: &mut ApplicationWorld) {
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected imported STL geometry triangles in viewport"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .third_party_entries
            .iter()
            .any(|entry| entry.contains("tetrahedron")),
        "expected named tetrahedron entry under 3rd Party, saw {:?}",
        browser.third_party_entries
    );
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("3rdParty") && model.contains("tetrahedron"),
        "expected 3rd Party folder with tetrahedron in Model browser, saw {model}"
    );
}

#[when("the user clicks Options")]
fn user_clicks_options(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Options");
}

#[then(
    "STL Import Options displays the import units, template units, bounding box, translation report and color controls."
)]
fn stl_options_displays_all_controls(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_some(),
        "expected {STL_OPTIONS_WINDOW}, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(STL_OPTIONS_WINDOW));
    for needle in [
        "ImportUnits",
        "TemplateUnits",
        "BoundingBox",
        "TranslationReport",
        "Color",
        "RGB",
    ] {
        assert!(
            text.contains(needle),
            "expected {needle:?} in {STL_OPTIONS_WINDOW}, saw {text}"
        );
    }
}

#[given("the user opens STL Import Options and enables Color")]
fn user_opens_stl_options_and_enables_color(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Options");
    assert!(super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_some());
    click_window_control(world, STL_OPTIONS_WINDOW, "Color");
}

#[when("the user clears Color")]
fn user_clears_color(world: &mut ApplicationWorld) {
    click_window_control(world, STL_OPTIONS_WINDOW, "Color");
}

#[then("the RGB selector is disabled.")]
fn rgb_selector_is_disabled(world: &mut ApplicationWorld) {
    assert!(
        world.run_in_window(STL_OPTIONS_WINDOW, "RGB").alpha < 200,
        "expected RGB selector disabled when Color is cleared"
    );
}

#[given(expr = "the user imports {string} into the part")]
fn user_imports_stl_into_part(world: &mut ApplicationWorld, relative: String) {
    open_picker_and_select(world, &relative);
    click_window_control(world, PICKER_WINDOW, "Open");
    stl_geometry_and_third_party_appear(world);
}

#[given("the user reopens Import, selects that STL and opens its options")]
fn user_reopens_import_and_opens_options(world: &mut ApplicationWorld) {
    open_picker_and_select(world, "fixtures/cad/import/tetrahedron.stl");
    click_window_control(world, PICKER_WINDOW, "Options");
    assert!(super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_some());
}

#[when("the user cancels the options and cancels the picker")]
fn user_cancels_options_and_picker(world: &mut ApplicationWorld) {
    click_window_control(world, STL_OPTIONS_WINDOW, "Cancel");
    assert!(super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_none());
    assert!(
        super::super::diagnostics::window_geometry(PICKER_WINDOW).is_some(),
        "expected Import picker to return after canceling STL options"
    );
    click_window_control(world, PICKER_WINDOW, "Cancel");
}

#[then("the MCAD Software returns to the part containing the earlier import.")]
fn returns_to_part_with_earlier_import(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(PICKER_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.third_party_entries.len(), 1);
}

#[given("STL Import Options is open")]
fn stl_import_options_is_open(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Options");
    assert!(super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).is_some());
}

#[when("the user opens Import Units")]
fn user_opens_import_units(world: &mut ApplicationWorld) {
    let row = world
        .run_in_window(STL_OPTIONS_WINDOW, "ImportUnits")
        .clone();
    let (origin, size) = super::super::diagnostics::window_geometry(STL_OPTIONS_WINDOW).unwrap();
    let combo =
        super::super::diagnostics::pointer_pixels([origin[0] + size[0] * 0.72, row.center()[1]]);
    world.click(combo);
    world.pump(Vec::new());
}

#[then("the list offers Template Units, inch, foot, centimeter, millimeter, meter and micron.")]
fn import_units_lists_all_seven(world: &mut ApplicationWorld) {
    let popup_texts: Vec<String> = world
        .frame
        .runs
        .iter()
        .filter(|run| world.frame.is_popup_row(run))
        .map(|run| super::super::world::squeezed(&run.text))
        .collect();
    for expected in [
        "TemplateUnits",
        "inch",
        "foot",
        "centimeter",
        "millimeter",
        "meter",
        "micron",
    ] {
        assert!(
            popup_texts.iter().any(|item| item == expected),
            "expected {expected:?} in Import Units dropdown, saw {popup_texts:?}"
        );
    }
}

#[when("the user clicks Open and chooses Convert Model")]
fn user_clicks_open_and_chooses_convert(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Open");
    assert!(super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_some());
    click_window_control(world, IMPORT_WINDOW, "ConvertModel");
}

#[then("the import panel exposes object filters and Part Options with a Surfaces choice.")]
fn import_panel_exposes_filters_and_surfaces(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(IMPORT_WINDOW));
    for needle in [
        "ObjectFilters",
        "SolidBodies",
        "SurfaceBodies",
        "PartOptions",
        "Surfaces",
    ] {
        assert!(
            text.contains(needle),
            "expected {needle:?} in Import panel, saw {text}"
        );
    }
}

#[given("the user opens the file and selects Convert Model")]
fn user_opens_file_and_selects_convert(world: &mut ApplicationWorld) {
    user_clicks_open_and_chooses_convert(world);
}

#[given("the destination part initially has one solid body")]
fn destination_has_one_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
}

#[when("the user opens Select and clicks Load Model")]
fn user_opens_select_and_clicks_load_model(world: &mut ApplicationWorld) {
    click_window_control(world, IMPORT_WINDOW, "Select");
    click_window_control(world, IMPORT_WINDOW, "LoadModel");
    wait_for_import_ready(world);
}

#[then("the tree shows Solid1 and the viewport shows a preview")]
fn tree_shows_solid1_and_viewport_preview(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(IMPORT_WINDOW));
    assert!(
        text.contains("Solid1"),
        "expected Solid1 in Import Select tree, saw {text}"
    );
    assert!(
        world.state().extrude_preview_active,
        "expected live viewport preview after Load Model"
    );
}

#[then("Solid1:1 appears in the browser and the part shows two solid bodies.")]
fn solid1_1_appears_and_two_solids(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Solid1:1"),
        "expected Solid1:1 in browser, saw {:?}",
        browser.features
    );
    assert_eq!(browser.bodies.len(), 2, "expected 2 solid bodies");
}

#[given("the destination part has two solid bodies")]
fn destination_has_two_solids(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 2);
}

#[when("the user clicks Open and chooses Reference Model")]
fn user_clicks_open_and_chooses_reference(world: &mut ApplicationWorld) {
    click_window_control(world, PICKER_WINDOW, "Open");
    assert!(super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_some());
    click_window_control(world, IMPORT_WINDOW, "ReferenceModel");
    wait_for_import_ready(world);
}

#[then(
    "the part browser shows a named reference entry and the solid body count increases from two to three."
)]
fn browser_shows_reference_entry_three_solids(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|f| f.kind == "Imported Reference" && f.name.contains("single-solid")),
        "expected named reference entry in browser, saw {:?}",
        browser.features
    );
    assert_eq!(browser.bodies.len(), 3, "expected 3 solid bodies");
}

#[given(expr = "the user imports {string} using Reference Model")]
fn user_imports_step_as_reference(world: &mut ApplicationWorld, relative: String) {
    open_picker_and_select(world, &relative);
    user_clicks_open_and_chooses_reference(world);
    click_window_control(world, IMPORT_WINDOW, "OK");
    browser_shows_reference_entry_three_solids(world);
}

#[given("the imported reference entry is present in the browser")]
fn imported_reference_entry_present(world: &mut ApplicationWorld) {
    let _ = reference_feature_name(world);
}

#[given("the imported reference remains linked")]
fn imported_reference_remains_linked(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .import_edits
            .values()
            .any(|seed| seed.import_mode == ImportMode::ReferenceModel
                && !seed.link_suppressed
                && !seed.link_broken)
    );
}

#[when("the user right-clicks the entry")]
#[when("the user reopens the reference entry context menu")]
fn user_right_clicks_reference_entry(world: &mut ApplicationWorld) {
    right_click_reference_entry(world);
}

#[then("its context menu offers Edit Import, Suppress Link and Break Link.")]
fn context_menu_offers_reference_commands(world: &mut ApplicationWorld) {
    let items: Vec<String> = world
        .frame
        .runs
        .iter()
        .filter(|run| world.frame.is_popup_row(run))
        .map(|run| super::super::world::squeezed(&run.text))
        .collect();
    for expected in ["EditImport", "SuppressLink", "BreakLink"] {
        assert!(
            items.iter().any(|item| item == expected),
            "expected {expected:?} in reference context menu, saw {items:?}"
        );
    }
}

#[when("the user chooses Edit Import from its context menu")]
fn user_chooses_edit_import(world: &mut ApplicationWorld) {
    right_click_reference_entry(world);
    click_popup_item(world, "Edit Import");
}

#[then("the Import panel reopens for that file and displays its reference options")]
fn import_panel_reopens_with_reference_options(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_some(),
        "expected Import panel to reopen"
    );
    let text = super::super::world::squeezed(&world.window_text(IMPORT_WINDOW));
    assert!(
        text.contains("single-solid") && text.contains("ReferenceModel"),
        "expected reference options for single-solid.stp, saw {text}"
    );
}

#[then("the panel closes and the reference entry remains in the browser.")]
fn panel_closes_and_reference_remains(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(IMPORT_WINDOW).is_none());
    let _ = reference_feature_name(world);
}

#[when("the user chooses Suppress Link from its browser context menu")]
fn user_chooses_suppress_link(world: &mut ApplicationWorld) {
    right_click_reference_entry(world);
    click_popup_item(world, "Suppress Link");
}

#[then("a crossed-link indicator appears and the geometry remains visible")]
fn crossed_link_indicator_appears_geometry_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .import_edits
            .values()
            .any(|seed| seed.link_suppressed),
        "expected link_suppressed state on reference import"
    );
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        model.contains("LinkSuppressed"),
        "expected crossed-link indicator in Model browser, saw {model}"
    );
    assert_eq!(browser.bodies.len(), 3);
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[then("it offers Unsuppress Link and disables Edit Import")]
fn menu_offers_unsuppress_and_disables_edit(world: &mut ApplicationWorld) {
    let unsuppress = world.frame.runs.iter().find(|run| {
        world.frame.is_popup_row(run)
            && super::super::world::squeezed(&run.text) == "UnsuppressLink"
    });
    assert!(unsuppress.is_some(), "expected Unsuppress Link menu item");
    let edit_item = world
        .frame
        .runs
        .iter()
        .find(|run| {
            world.frame.is_popup_row(run)
                && super::super::world::squeezed(&run.text) == "EditImport"
        })
        .expect("expected Edit Import menu item");
    assert!(
        edit_item.alpha < 200,
        "expected Edit Import disabled while link is suppressed"
    );
}

#[when("the user chooses Unsuppress Link")]
fn user_chooses_unsuppress_link(world: &mut ApplicationWorld) {
    click_popup_item(world, "Unsuppress Link");
}

#[then("the crossed-link indicator disappears.")]
fn crossed_link_indicator_disappears(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .import_edits
            .values()
            .all(|seed| !seed.link_suppressed),
        "expected link_suppressed cleared after Unsuppress Link"
    );
    let model = super::super::world::squeezed(&world.window_text("Model"));
    assert!(
        !model.contains("LinkSuppressed"),
        "expected crossed-link indicator removed, saw {model}"
    );
}
