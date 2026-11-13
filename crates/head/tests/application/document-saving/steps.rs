// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
#[path = "dialog_navigation.rs"]
mod dialog_navigation;
const SAVE_IN: &str = "Save in";
const ICONIC: &str = "Active Component Iso View on Save";
const WINDOW: &str = "Active Window on Save";
const IMPORT: &str = "Import From File";
const NO_PREVIEW: &str = "No preview image";
const IMAGE_CHOOSER: &str = "Select preview image";
const EXISTING_PART: &str = "existing part.opd";
const SUBFOLDER: &str = "src/";
/// A folder inside `src`: the list shows folders whatever its file type filter.
const NESTED_FOLDER: &str = "application/";
const VIEWPORT_CENTRE: [usize; 2] = [700, 450];

fn modified_part(world: &mut ApplicationWorld) -> u64 {
    let id = world.add_part();
    world.add_finished_sketch();
    id
}
#[given("a new unsaved part is active in the MCAD Software")]
#[given("a new unsaved empty part is active in the MCAD Software")]
fn new_part(world: &mut ApplicationWorld) {
    world.add_part();
}
#[given("the proposed file name does not already exist in the selected folder")]
#[given("the proposed file name does not exist in the selected folder")]
fn free_name(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    assert_eq!(std::fs::read_dir(folder).unwrap().count(), 0);
}
#[when("I press Control+S")]
#[when("I press Control+S again in the same session")]
fn press_save(world: &mut ApplicationWorld) {
    world.press_save_shortcut();
}
#[then("a Save As dialog opens")]
#[then("the Save As dialog opens")]
fn save_as_opens(world: &mut ApplicationWorld) {
    world.expect_text(SAVE_IN);
    world.expect_text("Save As");
}
#[then("a proposed document name is selected in the file name field")]
fn proposed_selected(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = format!("{}.opd", world.document_name(id));
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected a selected proposal"
    );
    world.proposed_name = Some(name);
}
#[then("the Save As dialog reopens with the same proposed document name selected")]
fn same_proposal_selected(world: &mut ApplicationWorld) {
    save_as_opens(world);
    let name = world
        .proposed_name
        .clone()
        .expect("expected the first proposed name");
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected the same selected proposal"
    );
}
#[then("a location, file type and preview options are displayed")]
fn dialog_contents(world: &mut ApplicationWorld) {
    for shown in [
        SAVE_IN,
        "Part Files (*.opd)",
        "Save Preview Picture",
        ICONIC,
        WINDOW,
        IMPORT,
    ] {
        world.expect_text(shown);
    }
}
#[when("I cancel the dialog")]
#[when("I cancel the dialog again")]
#[when("I cancel Save As")]
#[when("the user cancels Save As")]
fn cancel_dialog(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the Save As dialog closes")]
fn save_as_closed(world: &mut ApplicationWorld) {
    world.expect_no_text(SAVE_IN);
}
#[then("the unsaved part remains active")]
fn unsaved_remains(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_no_text(SAVE_IN);
    world.expect_active(id);
    assert!(world.state().document_storage[&id].path.is_none());
}
#[when("I accept the proposed file name in the Save As dialog")]
#[when("I accept the proposed file name in Save As")]
fn accept_proposed(world: &mut ApplicationWorld) {
    world.accept_save_as();
}
#[then("the dialog closes")]
fn dialog_closes(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.expect_no_text(SAVE_IN);
    world.expect_no_text("General Properties");
    assert!(super::diagnostics::window_geometry("Customize").is_none());
}
#[then("the active document tab displays the saved file name")]
fn tab_has_file_name(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved file");
    let stem = path.file_stem().unwrap().to_str().unwrap();
    world.expect_tab(stem);
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
}
#[when("I close that document tab")]
#[when("I close the document tab")]
fn close_tab(world: &mut ApplicationWorld) {
    world.closed_document = Some(world.active_id());
    world.close_active_tab();
}
#[when("I open the saved file from the same folder")]
#[when("I reopen the newly saved part in the same session")]
#[when("I reopen the saved part in the same session")]
#[when("I reopen the saved part file")]
fn reopen(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved file");
    if let Some(id) = world.closed_document {
        assert!(
            !world.document_is_open(id),
            "expected the closed document gone"
        );
    }
    world.reopen_through_dialog(&path);
}
#[then("the saved part becomes active")]
#[then("the part becomes active")]
#[then("the reopened part is active")]
fn reopened_active(world: &mut ApplicationWorld) {
    let id = world.active_id();
    // Without a saved file, any part the scenario opened may be the one a step brought forward.
    let Some(path) = world.saved_path.clone() else {
        assert!(
            world.parts.contains(&id),
            "active {id}: expected one of {:?}",
            world.parts
        );
        return;
    };
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        path.to_str()
    );
    world.expect_active(id);
}
#[then("its model browser displays the origin and end-of-part marker")]
fn browser_markers(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_text("End of Part");
}
#[given("a saved empty part is open in the MCAD Software")]
fn saved_part(world: &mut ApplicationWorld) {
    let id = world.add_part();
    let path = world.save_part_to_file(id, "saved part.opd");
    world.saved_path = Some(path);
}
#[when("I create a planar sketch with a rectangular profile")]
fn create_rectangle(world: &mut ApplicationWorld) {
    world.start_sketch();
    world.draw_rectangle();
}
#[when("I finish the sketch")]
fn finish(world: &mut ApplicationWorld) {
    world.finish_sketch();
    let path = world.saved_path.as_ref().expect("expected a saved file");
    world.bytes_before_save = Some(std::fs::read(path).unwrap());
}
#[then("the existing document remains open without a Save As dialog")]
fn no_dialog(world: &mut ApplicationWorld) {
    world.expect_no_text(SAVE_IN);
    let id = world.active_id();
    world.expect_active(id);
    assert!(!world.state().document_storage[&id].unsaved_edits);
    let path = world.saved_path.as_ref().expect("expected a saved file");
    assert_ne!(
        Some(std::fs::read(path).unwrap()),
        world.bytes_before_save,
        "Control+S: expected the sketch written to {path:?}"
    );
}
#[then("the model browser contains the sketch")]
#[then("its sketch remains in the model browser")]
#[then("its model browser contains the finished sketch")]
fn browser_sketch(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 1);
    world.expect_browser_sketch();
}
#[then("the rectangular sketch profile is visible in the viewport")]
fn rectangle_visible(world: &mut ApplicationWorld) {
    world.expect_drawn_region();
}
#[given("the Save As dialog is open for a new part in the MCAD Software")]
fn dialog_open(world: &mut ApplicationWorld) {
    world.add_part();
    world.press_save_shortcut();
    world.expect_text(SAVE_IN);
}
#[given("Save Preview Picture is selected")]
fn preview_selected(world: &mut ApplicationWorld) {
    // The source choices exist only while the checkbox is set.
    world.expect_text(ICONIC);
}
#[when("I clear Save Preview Picture")]
#[when("I select Save Preview Picture again")]
fn toggle_preview(world: &mut ApplicationWorld) {
    world.click_text("Save Preview Picture");
}
#[then("the preview source choices are hidden")]
fn sources_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text(ICONIC);
    world.expect_no_text(WINDOW);
}
#[then("the preview area displays a placeholder")]
#[then("the preview area still displays a placeholder")]
fn placeholder(world: &mut ApplicationWorld) {
    world.expect_text(NO_PREVIEW);
}
#[then("the preview source choices reappear")]
fn sources_back(world: &mut ApplicationWorld) {
    world.expect_text(ICONIC);
    world.expect_text(WINDOW);
    world.expect_no_text(NO_PREVIEW);
}
#[when("I choose Active Window on Save")]
fn choose_window(world: &mut ApplicationWorld) {
    world.click_text(WINDOW);
}
#[then("that preview source is selected")]
fn window_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected(WINDOW), "expected {WINDOW:?} selected");
}
#[then("Active Component Iso View on Save is not selected")]
fn iso_not_selected(world: &mut ApplicationWorld) {
    assert!(!world.radio_selected(ICONIC), "expected {ICONIC:?} cleared");
}
#[when("I choose Active Component Iso View on Save")]
fn choose_iso(world: &mut ApplicationWorld) {
    world.click_text(ICONIC);
    assert!(world.radio_selected(ICONIC), "expected {ICONIC:?} selected");
}
#[when("I choose Import From File as the preview source")]
fn choose_import(world: &mut ApplicationWorld) {
    world.click_text(IMPORT);
}
#[then("a browse control appears beside that option")]
fn browse_beside(world: &mut ApplicationWorld) {
    let option = world.run(IMPORT).clone();
    let browse = world.run("Browse...").clone();
    assert!(
        browse.minimum[0] > option.maximum[0]
            && (browse.center()[1] - option.center()[1]).abs() < 4.,
        "browse {browse:?}: expected a control beside {option:?}"
    );
}
#[when("I choose the browse control")]
fn choose_browse(world: &mut ApplicationWorld) {
    world.click_text("Browse...");
}
#[then("an image file chooser opens above the Save As dialog")]
fn chooser_above(world: &mut ApplicationWorld) {
    let chooser = world.run(IMAGE_CHOOSER).list;
    let dialog = world.run(SAVE_IN).list;
    assert!(
        chooser > dialog,
        "image chooser: expected it drawn above Save As"
    );
}
#[when("I cancel the image file chooser")]
fn cancel_chooser(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the Save As dialog remains open")]
fn dialog_remains(world: &mut ApplicationWorld) {
    world.expect_no_text(IMAGE_CHOOSER);
    world.expect_text(SAVE_IN);
}
#[then("Import From File remains selected")]
fn import_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected(IMPORT), "expected {IMPORT:?} selected");
}
#[when("I open the file type list")]
fn open_types(world: &mut ApplicationWorld) {
    world.click_text("Part Files (*.opd)");
}
#[then("the list offers the native part file type and an All Files filter")]
fn types_offered(world: &mut ApplicationWorld) {
    world.expect_text("Part Files (*.opd)");
    world.expect_text("All Files (*.*)");
    world.expect_text(SAVE_IN);
}
#[given("an unsaved part containing a finished empty sketch is active in the MCAD Software")]
#[given("an unsaved modified part is active in the MCAD Software")]
fn unsaved_with_sketch(world: &mut ApplicationWorld) {
    // The other document lets a closing part hand focus to something.
    let other = world.add_part();
    world.other_document = Some(other);
    modified_part(world);
}
#[given("an unsaved part containing a finished sketch is active in the MCAD Software")]
fn unsaved_with_rectangle(world: &mut ApplicationWorld) {
    let other = world.add_part();
    world.other_document = Some(other);
    world.add_part();
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
}
#[given("another document is open")]
fn another_open(world: &mut ApplicationWorld) {
    assert!(world.other_document.is_some(), "expected a second document");
}
#[given("save-change prompts are configured to always show")]
fn prompts_always(world: &mut ApplicationWorld) {
    let owner = &mut world.app().owner;
    owner
        .set_prompt_frequency_from_gui(ApplicationPrompt::SaveChanges, PromptFrequency::AlwaysShow)
        .unwrap();
}
#[then("a save-changes prompt offers Yes, No and Cancel")]
fn prompt_buttons(world: &mut ApplicationWorld) {
    for label in ["Yes", "No", "Cancel"] {
        world.expect_text(label);
    }
    world.expect_text("before closing");
    world.expect_text("Always show this message");
}
#[when("I choose Cancel")]
#[when("I cancel the save-changes prompt")]
#[when("I choose Cancel at the save-changes prompt")]
fn prompt_cancel(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the part remains active")]
fn part_remains(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_no_text("before closing");
    world.expect_active(id);
    if world.closed_document.is_some() {
        assert_eq!(world.closed_document, Some(id));
    }
}
#[when("I choose Yes in the save-changes prompt")]
#[when("I choose Yes to save changes")]
fn prompt_yes(world: &mut ApplicationWorld) {
    world.click_text("Yes");
}
#[when("I open the prompt-frequency list")]
fn open_frequency(world: &mut ApplicationWorld) {
    world.click_text("Always show this message");
}
#[then(
    "the list offers always show, once per operation, suppress for this session and suppress permanently"
)]
fn frequency_offers(world: &mut ApplicationWorld) {
    for label in [
        "Always show this message",
        "Show once per operation",
        "Suppress for this session",
        "Suppress permanently",
    ] {
        world.expect_text(label);
    }
}
#[when("I dismiss the list without changing its selection")]
fn dismiss_frequency(world: &mut ApplicationWorld) {
    world.escape();
    world.expect_no_text("Show once per operation");
    world.expect_text("before closing");
    assert_eq!(
        world.state().prompts.save_changes,
        PromptFrequency::AlwaysShow
    );
}
#[then("the document closes")]
fn document_closes(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected a closed document");
    world.expect_no_text(SAVE_IN);
    assert!(
        !world.document_is_open(id),
        "document {id}: expected it closed"
    );
}
#[then("another open document becomes active")]
fn other_active(world: &mut ApplicationWorld) {
    let other = world.other_document.expect("expected another document");
    world.expect_active(other);
}
#[given("a part with a rectangular sketch is saved in the MCAD Software")]
fn saved_sketch_part(world: &mut ApplicationWorld) {
    let id = world.add_part();
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
    world.pump(Vec::new());
    let path = world.save_part_to_file(id, "saved part.opd");
    world.saved_path = Some(path);
}
#[given("the part uses the observed preview configuration for a sketch thumbnail")]
fn stored_picture(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved part");
    let picture = oblikovati_persistence::package::preview_picture_of_file(&path);
    assert!(
        picture.is_some(),
        "saved file {path:?}: expected a picture in it"
    );
}
