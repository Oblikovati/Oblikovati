// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[path = "chooser_details/type_visibility.rs"]
mod type_visibility;
use type_visibility::expect_full_type_value;
#[test]
fn details_view_retains_controls_in_a_narrow_window_with_larger_text() {
    let mut world = ApplicationWorld::new();
    open_narrow_scaled_details(&mut world);
    world.expect_text("Template details");
    world.save_capture("template-details-narrow-large-text");
    world.drag([300, 503], [540, 503]);
    world.expect_text("Size");
    world.expect_text("Type");
    expect_full_type_value(&world, "Presentation");
    expect_full_type_value(&world, "Assembly");
    world.save_capture("template-details-narrow-scrolled");
    world.close();
}
fn open_narrow_scaled_details(world: &mut ApplicationWorld) {
    const TEXT_SCALE: f32 = 1.5;
    super::super::super::super::verify_gui_resize_recovery(
        &mut world.app().window.native,
        [800, 600],
    );
    world
        .app()
        .owner
        .set_ui_scale_from_gui(
            oblikovati_model::ui_scale::UiScaleOptions::new(TEXT_SCALE, 1.).unwrap(),
        )
        .unwrap();
    world.pump(Vec::new());
    open_chooser(world);
    world.click_text("Display");
    world.pump(Vec::new());
    world.click_text("Details");
    world.pump(Vec::new());
}
#[then("the previously selected template details remain displayed")]
#[then("the selected template details remain displayed")]
fn previous_details(world: &mut ApplicationWorld) {
    world.expect_text("Template details");
}
fn expect_part_tiles_ordered(world: &mut ApplicationWorld, plain_first: bool) {
    let plain = world.run(METRIC_PART).center()[1];
    let sheet = world.run(METRIC_SHEET_METAL).center()[1];
    assert_eq!(
        plain < sheet,
        plain_first,
        "plain part at {plain}, sheet metal at {sheet}: the smaller file is the plain part"
    );
}
fn open_details_view(world: &mut ApplicationWorld) {
    choose_template_view(world, "Details");
}
#[given("the gallery is grouped by template type and displayed in Details view")]
#[given("the metric gallery is grouped by template type in Details view")]
fn details_view_grouped(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    open_details_view(world);
    grouped_by_default(world);
}
#[given("the standard metric part template details are displayed")]
fn metric_part_details_shown(world: &mut ApplicationWorld) {
    world.click_text(METRIC_PART);
    expect_part_details(world);
}
fn click_size_heading(world: &mut ApplicationWorld) {
    world.click_text("Size");
    world.pump(Vec::new());
}
#[when("I click the Size column header")]
#[when("the user clicks the Size column header")]
#[when("I click the Size column header again")]
fn click_size(world: &mut ApplicationWorld) {
    click_size_heading(world);
}
#[given("the Size column is not the active sort column")]
fn size_not_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world.run("Size").text,
        "Size",
        "expected an unsorted Size heading"
    );
}
#[then("the visible part templates are ordered by increasing file size")]
fn ordered_increasing(world: &mut ApplicationWorld) {
    expect_part_tiles_ordered(world, true);
}
#[given("Size is the active sort with visible part templates ordered by increasing file size")]
fn sorted_ascending(world: &mut ApplicationWorld) {
    click_size_heading(world);
    expect_part_tiles_ordered(world, true);
}
#[then("the visible part templates are ordered by decreasing file size")]
fn ordered_decreasing(world: &mut ApplicationWorld) {
    expect_part_tiles_ordered(world, false);
}
#[then("the standard metric part template details remain displayed")]
fn part_details_remain(world: &mut ApplicationWorld) {
    // The root's gallery has no tile, so only the details pane can name it there.
    world.expect_text(METRIC_PART);
    world.expect_text("Document type: Part");
    world.expect_text("millimeters (mm)");
}
#[then("the gallery remains grouped by template type")]
fn still_grouped(world: &mut ApplicationWorld) {
    grouped_by_default(world);
}
#[given("the visible part templates are sorted by decreasing file size")]
fn sorted_decreasing(world: &mut ApplicationWorld) {
    click_size_heading(world);
    click_size_heading(world);
    expect_part_tiles_ordered(world, false);
}
#[when("I reopen the chooser in the same session")]
#[when("the user reopens the chooser in the same session")]
fn reopen_same_session(world: &mut ApplicationWorld) {
    reopen_chooser(world);
}
#[then("the metric template folder and medium icon view are displayed")]
fn metric_medium(world: &mut ApplicationWorld) {
    metric_folder_selected(world);
    medium_tiles(world);
}
#[when("I choose Details in the template display menu")]
#[when("the user chooses Details in the template display menu")]
fn choose_details_view(world: &mut ApplicationWorld) {
    open_details_view(world);
}
#[then("the Size column indicates descending order")]
fn size_descending(world: &mut ApplicationWorld) {
    world.expect_text("Size v");
}
#[given("the standard metric part template is selected")]
fn metric_part_selected(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    world.click_text(METRIC_PART);
}
#[given("the template root contains no direct template entries")]
fn root_empty(world: &mut ApplicationWorld) {
    world.expect_text("Templates");
}
#[when("I choose Home in the template chooser")]
#[when("the user chooses Home in the template chooser")]
fn home_in_chooser(world: &mut ApplicationWorld) {
    chooser_home(world);
}
#[then("the root is selected and its template groups are empty")]
fn root_groups_empty(world: &mut ApplicationWorld) {
    root_selected(world);
    expect_group_headings(world, &["Part", "Assembly", "Drawing"]);
    world.expect_no_text(METRIC_ASSEMBLY);
    world.expect_no_text(METRIC_SHEET_METAL);
}
const INCH_DRAWING: &str = "Inch drawing with title block";
fn create_button_alpha(world: &ApplicationWorld) -> u8 {
    world.run("Create").alpha
}
#[given("no template is selected")]
fn none_selected(world: &mut ApplicationWorld) {
    world.expect_text("Select a template.");
}
#[given("the selected template root has no direct template entries")]
fn root_selected_no_entries(world: &mut ApplicationWorld) {
    chooser_home(world);
    world.expect_no_text(METRIC_PART);
}
#[given("Create is disabled")]
#[then("Create is disabled")]
#[then("Create remains disabled")]
fn create_disabled(world: &mut ApplicationWorld) {
    let alpha = create_button_alpha(world);
    assert!(alpha < 200, "Create text alpha {alpha}: expected muted");
}
#[then("template tiles are displayed")]
fn tiles_displayed(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_ASSEMBLY);
}
#[then("its assembly details are displayed")]
fn assembly_details_displayed(world: &mut ApplicationWorld) {
    expect_assembly_details(world);
}
#[then("Create is enabled")]
fn create_enabled(world: &mut ApplicationWorld) {
    let alpha = create_button_alpha(world);
    assert!(alpha > 200, "Create text alpha {alpha}: expected full");
}
#[given("the standard metric assembly template is selected")]
fn assembly_chosen(world: &mut ApplicationWorld) {
    assembly_details_shown(world);
}
#[when("I select the English template folder without selecting another template")]
#[when("the user selects the English template folder without selecting another template")]
fn english_only(world: &mut ApplicationWorld) {
    world.click_text("English");
}
#[then("the gallery displays the English folder templates")]
fn english_folder_gallery(world: &mut ApplicationWorld) {
    inch_gallery(world);
}
#[then("the details still identify the metric assembly template")]
fn still_metric_assembly(world: &mut ApplicationWorld) {
    world.expect_text(METRIC_ASSEMBLY);
    expect_assembly_details(world);
}
#[then("a new empty assembly document becomes active")]
fn new_empty_assembly(world: &mut ApplicationWorld) {
    new_assembly(world);
    world.expect_blank_region(VIEWPORT_X, VIEWPORT_Y);
}
#[then("its browser displays representations, relationships and origin entries")]
fn its_assembly_browser(world: &mut ApplicationWorld) {
    assembly_browser(world);
}
#[given("an inch drawing template with a populated assembly layout preview is available")]
fn inch_drawing_available(world: &mut ApplicationWorld) {
    world.click_text("English");
    world.expect_text(INCH_DRAWING);
}
#[when("I select the inch drawing template")]
#[when("the user selects the inch drawing template")]
fn select_inch_drawing(world: &mut ApplicationWorld) {
    world.click_text(INCH_DRAWING);
}
#[then("the chooser displays the template sheet and available layout previews")]
fn layout_previews(world: &mut ApplicationWorld) {
    world.expect_text("Use Template");
    world.expect_text("Assembly Layout");
}
#[when("I select the populated assembly layout preview")]
#[when("the user selects the populated assembly layout preview")]
fn select_assembly_layout(world: &mut ApplicationWorld) {
    world.click_text("Assembly Layout");
}
#[then("the template chooser closes")]
fn template_chooser_closed(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.expect_no_text("Template folders");
}
#[then("a component-selection dialog opens for a drawing view")]
fn component_dialog_open(world: &mut ApplicationWorld) {
    world.expect_text("Select Component");
}
#[then("the dialog displays a document name field and a browse control")]
fn component_controls(world: &mut ApplicationWorld) {
    world.expect_text("Document name");
    world.expect_text("Browse...");
}
#[when("I cancel component selection")]
#[when("the user cancels component selection")]
fn cancel_component(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
}
#[then("the component-selection dialog closes")]
fn component_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Select Component");
}
#[then("the original part document remains active")]
fn original_part_active(world: &mut ApplicationWorld) {
    previous_document(world);
}
#[then("no new drawing tab is displayed")]
fn no_drawing_tab(world: &mut ApplicationWorld) {
    assert_eq!(world.state().documents.len(), world.parts.len());
    world.expect_no_text("Drawing1");
}
const LAYOUT_FILE: &str = "bracket.opd";
#[given("a populated drawing layout is being created in the MCAD Software")]
fn layout_being_created(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
    open_chooser(world);
    world.click_text("Metric");
    world.click_text(METRIC_DRAWING);
    world.click_text("Assembly Layout");
    world.click_text("Create");
    world.pump(Vec::new());
    world.expect_text("Select Component");
}
#[given("the component-selection dialog displays a document name")]
#[given("the component-selection dialog displays an initial component")]
fn component_name_shown(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_text(&name);
}
#[when("I choose Browse beside the document name")]
#[when("the user chooses Browse beside the document name")]
fn browse_component(world: &mut ApplicationWorld) {
    world.click_text("Browse...");
    world.pump(Vec::new());
}
#[then("a file-selection dialog opens above component selection")]
fn file_dialog_above(world: &mut ApplicationWorld) {
    let chooser = world.run("Select Component File").list;
    let component = world.run("Document name").list;
    assert!(
        chooser > component,
        "file selection: expected it above component selection"
    );
}
#[when("I cancel the file-selection dialog")]
#[when("the user cancels the file-selection dialog")]
fn cancel_file_dialog(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
}
#[then("the component-selection dialog is displayed again")]
fn component_again(world: &mut ApplicationWorld) {
    world.expect_text("Select Component");
    world.expect_no_text("Look in");
}
#[then("its previous document name remains displayed")]
fn component_name_remains(world: &mut ApplicationWorld) {
    component_name_shown(world);
}
#[then("the original document remains active")]
fn original_document_active(world: &mut ApplicationWorld) {
    previous_document(world);
}
#[given("a different part file that needs updating is available")]
fn different_part_file(world: &mut ApplicationWorld) {
    let path = world.scratch_folder().join(LAYOUT_FILE);
    std::fs::write(path, b"listed by this scenario, never opened").unwrap();
}
#[when("I select the different part file and choose Open")]
fn select_different_file(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    world.enter_in_field("Look in", folder.to_str().unwrap());
    world.click_text(LAYOUT_FILE);
    world.click_text("Open");
    world.pump(Vec::new());
}
#[then("the component-selection dialog displays the selected file name")]
fn component_shows_file(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    world.expect_text("Select Component");
    world.expect_text(LAYOUT_FILE);
}

#[then("it displays a file list, a file name field and Open and Cancel controls")]
fn component_file_chooser_controls(world: &mut ApplicationWorld) {
    for label in ["Name", "File name", "Open", "Cancel"] {
        world.expect_text(label);
    }
}

#[test]
fn component_browser_keeps_selected_file_name() {
    let mut world = ApplicationWorld::new();
    layout_being_created(&mut world);
    different_part_file(&mut world);
    browse_component(&mut world);
    select_different_file(&mut world);
    component_shows_file(&mut world);
    world.close();
}

#[test]
fn reopened_drawing_displays_its_authored_active_sheet() {
    use oblikovati_api::{client::Client, wire::OpenDocumentArgs};
    use oblikovati_model::drawing::{DrawingDocument, DrawingSheetRecipe};
    use oblikovati_persistence::package::DocumentPackage;
    let mut world = ApplicationWorld::new();
    let mut drawing = DrawingDocument::new();
    let mut sheet = DrawingSheetRecipe::a3_landscape("Portrait");
    sheet.orientation = oblikovati_api::types::SHEET_PORTRAIT;
    drawing.add_sheet(sheet).unwrap();
    let bytes = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap()
        .encode()
        .unwrap();
    let file = world.scratch_folder().join("portrait.odd");
    std::fs::write(&file, bytes).unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: file.to_string_lossy().into_owned(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_text("Sheet 2 of 2");
    let capture = super::super::super::super::capture_framebuffer(&mut world.app().window.native);
    let pixel = |x: usize, y: usize| {
        let at = (y * capture.extent[0] as usize + x) * 4;
        &capture.rgba[at..at + 3]
    };
    let width = capture.extent[0] as usize;
    let height = capture.extent[1] as usize;
    assert!(
        pixel(width * 59 / 100, height * 45 / 100)
            .iter()
            .all(|channel| *channel > 200),
        "expected portrait paper at its center"
    );
    assert!(
        pixel(width * 35 / 100, height * 45 / 100)
            .iter()
            .all(|channel| *channel < 200),
        "expected workspace beside portrait paper"
    );
    world.close();
}

#[path = "chooser_details/drawing_projection.rs"]
mod drawing_projection;
