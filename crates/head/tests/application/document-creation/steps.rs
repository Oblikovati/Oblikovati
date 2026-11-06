// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use super::documents::{VIEWPORT_X, VIEWPORT_Y};
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
#[path = "gallery_views.rs"]
mod gallery_views;
const METRIC_PART: &str = "Standard metric part";
const METRIC_SHEET_METAL: &str = "Metric sheet-metal part";
const INCH_SHEET_METAL: &str = "Inch sheet-metal part";
const METRIC_ASSEMBLY: &str = "Standard metric assembly";
const METRIC_DRAWING: &str = "Metric drawing with title block";
#[given("a document is open in the MCAD Software")]
#[given("a part document is active in the MCAD Software")]
fn open_document(world: &mut ApplicationWorld) {
    let id = world.add_part();
    world.previous_document = Some(id);
}
#[when("I press Control+N")]
#[given("the new-document template chooser is open")]
#[given("the new-document template chooser is open in the MCAD Software")]
fn open_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
}
#[then("the new-document template chooser opens")]
fn chooser_open(world: &mut ApplicationWorld) {
    world.expect_text("Template folders");
}
#[then("templates are grouped by document type")]
fn grouped_by_type(world: &mut ApplicationWorld) {
    world.expect_text("Part");
}
#[when("I select the metric template folder")]
fn metric_folder(world: &mut ApplicationWorld) {
    world.click_text("Metric");
}
#[when("I select the English template folder")]
fn english_folder(world: &mut ApplicationWorld) {
    world.click_text("English");
}
#[when("I select the standard metric part template")]
fn select_metric(world: &mut ApplicationWorld) {
    world.click_text("Standard metric part");
}
#[when("I select the standard inch part template")]
fn select_inch(world: &mut ApplicationWorld) {
    world.click_text("Standard inch part");
}
#[then("the template details identify a part with millimeter units")]
fn metric_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Part");
    world.expect_text("millimeters (mm)");
}
#[then("the template details identify an assembly with millimeter units")]
fn assembly_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Assembly");
    world.expect_text("millimeters (mm)");
}
#[then("the template details identify a part with inch units")]
fn inch_details(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Part");
    world.expect_text("inches (in)");
}
#[when("I cancel the chooser")]
fn cancel_chooser(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Escape, false);
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
    let state = world.app().owner.gui_command_state().unwrap();
    let browser = state.browser.unwrap();
    assert!(browser.features.is_empty() && browser.sketches.is_empty());
    world.expect_blank_region(VIEWPORT_X, VIEWPORT_Y);
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
fn select_assembly(world: &mut ApplicationWorld) {
    world.click_text(METRIC_ASSEMBLY);
}
#[when("I select the metric sheet-metal part template")]
#[given("a sheet-metal part template is selected")]
fn select_sheet_metal(world: &mut ApplicationWorld) {
    world.click_text(METRIC_SHEET_METAL);
}
#[given("the metric sheet-metal template is selected")]
fn sheet_metal_selected(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.click_text(METRIC_SHEET_METAL);
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
    world.expect_blank_region(VIEWPORT_X, VIEWPORT_Y);
}
#[then("its model browser shows representations, relationships and origin entries")]
fn assembly_browser(world: &mut ApplicationWorld) {
    for entry in ["Representations", "Relationships", "Origin"] {
        world.expect_text(entry);
    }
}
fn expect_muted_command(world: &mut ApplicationWorld, label: &str) {
    world.expect_text(label);
    let before = world.state();
    world.click_text(label);
    assert_eq!(
        world.state().documents.len(),
        before.documents.len(),
        "muted command {label:?}: expected no effect"
    );
}
#[then("the assembly ribbon offers commands to place and create components")]
fn assembly_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Assemble");
    expect_muted_command(world, "Place");
    expect_muted_command(world, "Create");
}
#[given("a drawing template with a border and title block is available")]
fn drawing_available(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_DRAWING);
}
#[when("I select the drawing template")]
fn select_drawing(world: &mut ApplicationWorld) {
    world.click_text(METRIC_DRAWING);
}
#[when("I choose its Use Template preview")]
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
    expect_muted_command(world, "Base View");
    expect_muted_command(world, "Projected View");
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
    expect_muted_command(world, "Flange");
    expect_muted_command(world, "Flat Pattern");
}
#[given("the metric part template group is expanded")]
fn part_group_expanded(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.expect_text(METRIC_PART);
}
#[when("I collapse the part template group")]
#[when("I expand the part template group")]
fn toggle_part_group(world: &mut ApplicationWorld) {
    world.click_text("Part");
}
#[then("its template tiles are hidden")]
fn tiles_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text(METRIC_PART);
}
#[then("the assembly and drawing groups remain visible")]
fn other_groups(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_ASSEMBLY);
    world.expect_text(METRIC_DRAWING);
}
#[then("the selected template's details remain displayed")]
fn details_remain(world: &mut ApplicationWorld) {
    // Only the details pane can still name a tile whose group is hidden.
    world.expect_text(METRIC_SHEET_METAL);
    world.expect_text("Document type: Part");
}
#[then("its template tiles are visible again")]
fn tiles_back(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_PART);
}
#[then("the sheet-metal template remains selected")]
fn sheet_metal_remains(world: &mut ApplicationWorld) {
    world.expect_text("Document type: Part");
    world.expect_text("millimeters (mm)");
    let runs = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text.replace(' ', "") == METRIC_SHEET_METAL.replace(' ', ""))
        .count();
    assert_eq!(runs, 2, "expected the tile and the details pane");
}
#[then("the gallery shows the English folder's templates")]
fn english_gallery(world: &mut ApplicationWorld) {
    world.expect_text("Standard inch part");
    world.expect_text(INCH_SHEET_METAL);
    world.expect_no_text(METRIC_PART);
}
#[then("the details still identify the previously selected metric sheet-metal template")]
fn details_still_metric(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_SHEET_METAL);
    world.expect_text("millimeters (mm)");
}
#[when("I select the inch sheet-metal template")]
fn select_inch_sheet_metal(world: &mut ApplicationWorld) {
    world.click_text(INCH_SHEET_METAL);
}
#[when("I select the metric presentation template")]
fn select_presentation(world: &mut ApplicationWorld) {
    world.click_text("Standard metric presentation");
}
#[then("an Insert dialog opens for an assembly file")]
fn insert_opens(world: &mut ApplicationWorld) {
    world.expect_text("Insert");
    world.expect_text("Assembly Files (*.oad)");
    world.expect_text("Look in");
}
#[when("I cancel the Insert dialog")]
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
fn open_display_menu(world: &mut ApplicationWorld) {
    world.click_text("Display");
    // A popup in a modal lays out one frame later; the frame budget of a pump can end before it.
    world.pump(Vec::new());
}
#[then("the menu offers icon sizes, List, Details, and Grouped by Type")]
fn display_menu(world: &mut ApplicationWorld) {
    for entry in [
        "Large Icons",
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
