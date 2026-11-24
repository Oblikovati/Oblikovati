// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use super::documents::{VIEWPORT_X, VIEWPORT_Y};
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
#[path = "basic_catalog.rs"]
mod basic_catalog;
#[path = "gallery_views.rs"]
mod gallery_views;
const METRIC_PART: &str = "Standard metric part";
const METRIC_SHEET_METAL: &str = "Metric sheet-metal part";
const INCH_SHEET_METAL: &str = "Inch sheet-metal part";
const METRIC_ASSEMBLY: &str = "Standard metric assembly";
const METRIC_DRAWING: &str = "Metric drawing with title block";

fn open_fixture_project(world: &mut ApplicationWorld, path: &str) {
    let path = super::scenario_root().join(path).canonicalize().unwrap();
    world.app().owner.load_desktop_project(&path).unwrap();
    let state = world.state();
    assert!(
        state
            .projects
            .iter()
            .any(|project| { project.active && std::path::Path::new(&project.location) == path }),
        "expected the fixture project {path:?} active"
    );
}
#[given("a document is open in the MCAD Software")]
#[given("a part document is active in the MCAD Software")]
fn open_document(world: &mut ApplicationWorld) {
    let id =
        if super::scenario_capture::is_main_window_frame(world.scenario_feature_path.as_deref()) {
            super::main_window_frame::open_frame_part(world)
        } else {
            world.add_part()
        };
    world.previous_document = Some(id);
}
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open with `Sketch1` visible in the MCAD Software"
)]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is active with `Sketch1` visible in the MCAD Software"
)]
fn open_part_with_visible_sketch(world: &mut ApplicationWorld) {
    use super::workspace_layout::native_part::{fixture_sketch, open_named_extrusion};

    open_named_extrusion(world);
    let sketch = fixture_sketch(world);
    assert_eq!(sketch.name, "Sketch1");
    assert!(sketch.visible, "expected Sketch1 visible in the saved Part");
    let document = world.active_id();
    world.parts.push(document);
}
#[when("I press Control+N")]
#[when("the user presses Control+N")]
#[given("the user presses Control+N to open the new-document template chooser")]
fn open_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
}
#[given("the new-document template chooser is open")]
#[given("the new-document template chooser is open in the MCAD Software")]
fn ensure_chooser_open(world: &mut ApplicationWorld) {
    if !world.shows("Template folders") {
        open_chooser(world);
    }
    chooser_open(world);
}
#[then("the new-document template chooser opens")]
fn chooser_open(world: &mut ApplicationWorld) {
    world.expect_text("Template folders");
}
#[then("templates are grouped by document type")]
fn grouped_by_type(world: &mut ApplicationWorld) {
    gallery_views::expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
}
#[when("I select the metric template folder")]
#[when("the user selects the metric template folder")]
fn metric_folder(world: &mut ApplicationWorld) {
    world.click_text("Metric");
}
#[when("I select the English template folder")]
#[when("the user selects the English template folder")]
fn english_folder(world: &mut ApplicationWorld) {
    world.click_text("English");
}
#[when("I select the standard metric part template")]
#[when("the user selects the standard metric part template")]
fn select_metric(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, METRIC_PART, "Standard (mm)");
}
#[when("I select the standard inch part template")]
#[when("the user selects the standard inch part template")]
fn select_inch(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, "Standard inch part", "Standard (in)");
}
#[then("the template details identify a part with millimeter units")]
fn metric_details(world: &mut ApplicationWorld) {
    gallery_views::expect_selected_template(world, "Document type: Part");
    gallery_views::expect_selected_template(world, "Length: millimeters (mm)");
}
#[then("the template details identify an assembly with millimeter units")]
fn assembly_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Assembly");
    world.expect_text("millimeters (mm)");
}
#[then("the template details identify a part with inch units")]
fn inch_details(world: &mut ApplicationWorld) {
    gallery_views::expect_selected_template(world, "Document type: Part");
    gallery_views::expect_selected_template(world, "Length: inches (in)");
}
#[when("I cancel the chooser")]
#[when("the user cancels the chooser")]
fn cancel_chooser(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the chooser closes")]
fn chooser_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Template folders");
}
#[then("the previously active document remains displayed")]
fn previous_document(world: &mut ApplicationWorld) {
    assert_eq!(
        world.app().owner.gui_command_state().unwrap().document,
        world.previous_document
    );
}
#[when("I choose Create")]
#[when("the user chooses Create")]
fn create_part(world: &mut ApplicationWorld) {
    world.click_text("Create");
}
#[then("a new part document becomes active")]
fn new_part(world: &mut ApplicationWorld) {
    let state = world.app().owner.gui_command_state().unwrap();
    assert!(state.document.is_some() && state.document != world.previous_document);
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
}
#[then("its viewport contains no visible model geometry")]
fn empty_part(world: &mut ApplicationWorld) {
    super::viewport_background::expect_empty_part(world);
}
#[then("its model browser displays an origin and an end-of-part marker")]
fn browser_origin(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_text("End of Part");
}
#[then("a new document tab is visible")]
fn new_tab(world: &mut ApplicationWorld) {
    let id = world.active_id();
    assert!(
        world
            .state()
            .documents
            .iter()
            .any(|row| row.id == id && row.visible)
    );
    let name = world.document_name(id);
    world.expect_tab(&name);
}
#[when("I select the standard metric assembly template")]
#[when("the user selects the standard metric assembly template")]
fn select_assembly(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, METRIC_ASSEMBLY, "Standard Assembly");
}
#[when("I select the metric sheet-metal part template")]
#[when("the user selects the metric sheet-metal part template")]
#[given("a sheet-metal part template is selected")]
fn select_sheet_metal(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, METRIC_SHEET_METAL, "Sheet Metal (mm)");
}
#[given("the metric sheet-metal template is selected")]
fn sheet_metal_selected(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    select_sheet_metal(world);
}
#[given("its details show millimeter units")]
fn details_in_millimeters(world: &mut ApplicationWorld) {
    world.expect_text("millimeters (mm)");
}
fn expect_new_document(world: &mut ApplicationWorld, kind: oblikovati_api::types::DocumentType) {
    let state = world.state();
    assert!(state.document.is_some() && state.document != world.previous_document);
    assert_eq!(state.document_type, Some(kind));
}
#[then("a new assembly document becomes active")]
fn new_assembly(world: &mut ApplicationWorld) {
    expect_new_document(world, oblikovati_api::types::DOCUMENT_ASSEMBLY);
}
#[then("a new drawing document becomes active")]
fn new_drawing(world: &mut ApplicationWorld) {
    expect_new_document(world, oblikovati_api::types::DOCUMENT_DRAWING);
}
#[then("a new presentation document becomes active")]
fn new_presentation(world: &mut ApplicationWorld) {
    expect_new_document(world, oblikovati_api::types::DOCUMENT_PRESENTATION);
}
#[then("its viewport contains no visible components")]
fn no_components(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected an Assembly browser");
    assert!(
        browser.occurrences.is_empty(),
        "expected no Assembly components"
    );
    super::viewport_background::expect_empty_viewport(world);
}
#[then("its model browser shows representations, relationships and origin entries")]
fn assembly_browser(world: &mut ApplicationWorld) {
    for entry in ["Representations", "Relationships", "Origin"] {
        world.expect_text(entry);
    }
}
#[then("the assembly ribbon offers commands to place and create components")]
fn assembly_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Assemble");
    world.expect_text("Place");
    world.expect_text("Create");
}
#[given("a drawing template with a border and title block is available")]
fn drawing_available(world: &mut ApplicationWorld) {
    let title = basic_catalog::catalog_title(world, METRIC_DRAWING, "Standard Drawing");
    world.expect_text(title);
}
#[when("I select the drawing template")]
#[when("the user selects the drawing template")]
fn select_drawing(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, METRIC_DRAWING, "Standard Drawing");
}
#[when("I choose its Use Template preview")]
#[when("the user chooses its Use Template preview")]
fn use_template(world: &mut ApplicationWorld) {
    world.click_text("Use Template");
}
#[then("its viewport displays a sheet with a border and title block")]
fn sheet_shown(world: &mut ApplicationWorld) {
    let centre = [
        (VIEWPORT_X[0] + VIEWPORT_X[1]) / 2,
        (VIEWPORT_Y[0] + VIEWPORT_Y[1]) / 2,
    ];
    let light = world.pixel_at(centre);
    assert!(
        light[..3].iter().all(|channel| *channel > 200),
        "sheet centre {light:?}: expected a light sheet"
    );
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_text(&format!("Title: {name}"));
    world.expect_text("Scale 1:1");
    let title = world.run("Scale 1:1").clone();
    let extent = world.extent();
    assert!(
        title.center()[0] > extent[0] as f32 / 2. && title.center()[1] > extent[1] as f32 / 2.,
        "title block {title:?}: expected the lower right of the sheet"
    );
}
#[then("the drawing browser lists the sheet and its border")]
fn drawing_browser(world: &mut ApplicationWorld) {
    world.expect_text("Sheet:1");
    world.expect_text("Border");
}
#[then("the ribbon offers drawing view commands")]
fn drawing_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.expect_text("Projected View");
    assert!(world.run_is_enabled("Base View"));
}
#[then("the browser shows a folded model and its origin")]
fn folded_browser(world: &mut ApplicationWorld) {
    world.expect_text("Folded Model");
    world.expect_text("Origin");
    world.expect_text("End of Folded");
}
#[then("the sheet-metal ribbon is active")]
fn sheet_metal_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Sheet Metal");
}
#[given("the metric part template group is expanded")]
fn part_group_expanded(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    assert_metric_part_tiles(world, true);
}
#[when("I collapse the part template group")]
#[when("I expand the part template group")]
#[when("the user collapses the part template group")]
#[when("the user expands the part template group")]
fn toggle_part_group(world: &mut ApplicationWorld) {
    world.click_text("Part");
}
fn assert_part_template_tile(world: &ApplicationWorld, title: &str, visible: bool) {
    let details_x = world.run("Template details").minimum[0];
    let expected = title.replace(' ', "");
    let found = world
        .frame
        .runs
        .iter()
        .any(|run| run.minimum[0] < details_x && run.text == expected);
    assert_eq!(
        found,
        visible,
        "template {title:?} visibility in {:?}",
        world.text()
    );
}

#[then("its template tiles are hidden")]
fn tiles_hidden(world: &mut ApplicationWorld) {
    assert_metric_part_tiles(world, false);
}

fn assert_metric_part_tiles(world: &mut ApplicationWorld, visible: bool) {
    let titles = basic_catalog::metric_part_titles(world);
    assert!(
        !titles.is_empty(),
        "expected the Metric Part group to contain templates"
    );
    for title in titles {
        assert_part_template_tile(world, &title, visible);
    }
}
#[then("the assembly and drawing groups remain visible")]
fn other_groups(world: &mut ApplicationWorld) {
    gallery_views::expect_group_headings(world, &["Assembly", "Drawing"]);
    let assembly = basic_catalog::catalog_title(world, METRIC_ASSEMBLY, "Standard Assembly");
    let drawing = basic_catalog::catalog_title(world, METRIC_DRAWING, "Standard Drawing");
    assert_part_template_tile(world, assembly, true);
    assert_part_template_tile(world, drawing, true);
}
#[then("the selected template's details remain displayed")]
fn details_remain(world: &mut ApplicationWorld) {
    let name = basic_catalog::catalog_title(world, METRIC_SHEET_METAL, "Sheet Metal (mm)");
    gallery_views::expect_selected_template(world, name);
    metric_details(world);
}
#[then("its template tiles are visible again")]
fn tiles_back(world: &mut ApplicationWorld) {
    assert_metric_part_tiles(world, true);
}
#[then("the sheet-metal template remains selected")]
fn sheet_metal_remains(world: &mut ApplicationWorld) {
    let name = basic_catalog::catalog_title(world, METRIC_SHEET_METAL, "Sheet Metal (mm)");
    assert_part_template_tile(world, name, true);
    details_remain(world);
}
#[then("the gallery shows the English folder's templates")]
fn english_gallery(world: &mut ApplicationWorld) {
    let part = basic_catalog::catalog_title(world, "Standard inch part", "Standard (in)");
    let sheet = basic_catalog::catalog_title(world, INCH_SHEET_METAL, "Sheet Metal (in)");
    let metric = basic_catalog::catalog_title(world, METRIC_PART, "Standard (mm)");
    assert_part_template_tile(world, part, true);
    assert_part_template_tile(world, sheet, true);
    assert_part_template_tile(world, metric, false);
}
#[then("the details still identify the previously selected metric sheet-metal template")]
fn details_still_metric(world: &mut ApplicationWorld) {
    details_remain(world);
}
#[when("I select the inch sheet-metal template")]
#[when("the user selects the inch sheet-metal template")]
fn select_inch_sheet_metal(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(world, INCH_SHEET_METAL, "Sheet Metal (in)");
}
#[when("I select the metric presentation template")]
#[when("the user selects the metric presentation template")]
fn select_presentation(world: &mut ApplicationWorld) {
    basic_catalog::select_catalog_template(
        world,
        "Standard metric presentation",
        "Standard Presentation",
    );
}
#[then("an Insert dialog opens for an assembly file")]
fn insert_opens(world: &mut ApplicationWorld) {
    world.expect_text("Insert");
    world.expect_text("Assembly Files (*.oad)");
    world.expect_text("Look in");
}
#[when("I cancel the Insert dialog")]
#[when("the user cancels the Insert dialog")]
fn cancel_insert(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.expect_no_text("Look in");
}
#[then("the presentation remains open with an empty viewport")]
fn presentation_empty(world: &mut ApplicationWorld) {
    expect_new_document(world, oblikovati_api::types::DOCUMENT_PRESENTATION);
    world.expect_blank_region([400, 1000], [260, 600]);
}
#[then("the browser shows a scene")]
fn scene_browser(world: &mut ApplicationWorld) {
    world.expect_text("Scene1");
}
#[then("the workspace displays a storyboard timeline and snapshot views panel")]
fn presentation_panels(world: &mut ApplicationWorld) {
    world.expect_text("Storyboard");
    world.expect_text("Snapshot Views");
}
#[then("the presentation ribbon offers Insert Model")]
fn insert_model(world: &mut ApplicationWorld) {
    world.click_text("Insert Model");
    world.expect_text("Look in");
    world.click_text("Cancel");
    world.expect_no_text("Look in");
}
#[when("I open the template display menu")]
#[when("the user opens the template display menu")]
fn open_display_menu(world: &mut ApplicationWorld) {
    world.click_text("Display");
    // A popup in a modal lays out one frame later; the frame budget of a pump can end before it.
    world.pump(Vec::new());
}
#[then("the menu offers icon sizes, List, Details, and Grouped by Type")]
fn display_menu(world: &mut ApplicationWorld) {
    for entry in [
        "Extra Large Icons",
        "Small Icons",
        "List",
        "Details",
        "Grouped by Type",
    ] {
        world.expect_text(entry);
    }
    assert!(
        world.checked("Grouped by Type"),
        "expected grouping on by default"
    );
}
#[when("I choose Details")]
#[when("the user chooses Details")]
fn choose_details(world: &mut ApplicationWorld) {
    world.click_text("Details");
}
#[then("the template gallery displays rows with Name, Size, and Type columns")]
fn details_rows(world: &mut ApplicationWorld) {
    for heading in ["Name", "Size", "Type"] {
        world.expect_text(heading);
    }
    let row = world.run(METRIC_PART).clone();
    let kind = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Part" && (run.center()[1] - row.center()[1]).abs() < 3.)
        .cloned();
    assert!(
        kind.is_some(),
        "row {row:?}: expected its Type column to read Part"
    );
}
#[then("the template details pane remains visible")]
fn details_pane(world: &mut ApplicationWorld) {
    world.expect_text("Template details");
}
#[given("template grouping by document type is enabled")]
fn grouping_on(_: &mut ApplicationWorld) {
    // Grouping is the chooser's own default, read once it opens.
}
#[given("the template details are blank")]
fn details_blank(world: &mut ApplicationWorld) {
    world.expect_no_text("Document type:");
}

#[cfg(test)]
#[path = "ribbon_regression.rs"]
mod ribbon_regression;
