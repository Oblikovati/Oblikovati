// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[when("I click the close button on the second part's document tab")]
#[when("the user clicks the close button on the second part's document tab")]
fn close_second_mark(world: &mut ApplicationWorld) {
    world.click_close_mark(second(world));
}
fn expect_second_gone_without_prompt(world: &mut ApplicationWorld) {
    let id = second(world);
    assert!(
        !world.document_is_open(id),
        "expected the second part closed"
    );
    world.expect_no_text("before closing");
}
#[then("the second part's tab disappears")]
fn second_tab_gone(world: &mut ApplicationWorld) {
    expect_second_gone_without_prompt(world);
    let path = world
        .saved_path
        .as_ref()
        .expect("expected the saved part path");
    let current = std::fs::read(path).expect("expected the saved part file");
    assert_eq!(
        Some(current),
        world.bytes_before_save,
        "closing changed the saved file"
    );
}
#[then("no save prompt is displayed")]
fn no_prompt(world: &mut ApplicationWorld) {
    world.expect_no_text("before closing");
}
#[given("other documents are also open")]
fn others_open(world: &mut ApplicationWorld) {
    // Opened first, so the active part is neither the first nor the last tab that closes back to it.
    let active = world.active_id();
    world.add_part();
    world.activate_through_tab(active);
    world.previous_document = Some(active);
}
#[when("I create a new empty part from a standard metric template")]
fn create_from_template(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric");
    world.click_text("Standard metric part");
    world.click_text("Create");
}
#[then("the new part becomes active and has its own document tab")]
fn new_part_active(world: &mut ApplicationWorld) {
    let id = world.active_id();
    assert_ne!(Some(id), world.previous_document);
    world.closed_document = Some(id);
    let name = world.document_name(id);
    world.expect_tab(&name);
}
#[when("I click the close button on the new part's tab without editing it")]
#[when("I click the close button on the new part's document tab")]
#[when("I click the close button on the part's document tab")]
fn close_active_mark(world: &mut ApplicationWorld) {
    world.closed_document = Some(world.active_id());
    world.close_active_tab();
}
#[then("the new part's tab disappears without a save prompt")]
fn new_part_gone(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected a closed document");
    assert!(!world.document_is_open(id), "expected the new part closed");
    world.expect_no_text("before closing");
}
#[then("the previously active part is displayed again")]
#[then("the previously active part is displayed")]
fn previous_again(world: &mut ApplicationWorld) {
    let id = world.previous_document.expect("expected a prior document");
    world.expect_active(id);
}
#[then("its model browser and viewport are restored")]
fn restored(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[given("a new part is open in the MCAD Software")]
fn new_part_open(world: &mut ApplicationWorld) {
    world.add_part();
}
#[given("I am editing a sketch in the part")]
#[given("I am editing that sketch")]
fn editing_sketch(world: &mut ApplicationWorld) {
    if world.state().authoring != GuiAuthoringContext::Sketch {
        world.start_sketch();
    }
}
#[given("the environment-exit message is enabled")]
fn exit_enabled(world: &mut ApplicationWorld) {
    world
        .app()
        .owner
        .set_prompt_frequency_from_gui(
            ApplicationPrompt::ExitEnvironment,
            PromptFrequency::AlwaysShow,
        )
        .unwrap();
}
#[then("a dialog asks whether to exit the sketch environment and continue")]
fn exit_asked(world: &mut ApplicationWorld) {
    world.expect_text(EXIT_QUESTION);
}
#[then("the part remains open")]
fn part_open(world: &mut ApplicationWorld) {
    // The part a close asked about, else the part of a drawing and part pair.
    let id = world
        .closed_document
        .or(world.other_document)
        .expect("expected the part");
    world.expect_no_text(EXIT_QUESTION);
    world.expect_no_text("before closing");
    assert!(world.document_is_open(id));
}
#[then("the sketch editing environment remains active")]
fn sketch_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    // The status line counts dimensions only while a sketch is under edit; a ribbon tab may hide
    // Finish Sketch.
    assert!(
        world.status_shows("dimension") || world.status_shows("Fully Constrained"),
        "expected the sketch's count in status {:?}",
        world.status_text()
    );
}
#[given("a new part contains an unsaved sketch in the MCAD Software")]
fn part_with_open_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
}
#[when("I choose OK to exit the sketch environment and continue")]
fn exit_ok(world: &mut ApplicationWorld) {
    world.click_text("OK");
}
#[then("a save-changes prompt identifies the part")]
fn prompt_names_part(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    let name = world.document_name(id);
    world.expect_text(&format!("Save changes to {name}"));
}
#[then("the prompt offers Yes, No and Cancel")]
fn yes_no_cancel(world: &mut ApplicationWorld) {
    for label in ["Yes", "No", "Cancel"] {
        world.expect_text(label);
    }
}
#[then("its sketch remains listed in the model browser")]
#[then("its sketch is listed in the model browser")]
fn sketch_listed(world: &mut ApplicationWorld) {
    world.expect_browser_sketch();
}
#[then("the part modeling environment is active")]
fn modeling_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    world.expect_text("Start 2D Sketch");
}
#[given("a new part with an unsaved sketch is active in the MCAD Software")]
fn new_part_with_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
}
#[given("the part is in the modeling environment")]
fn in_modeling(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
}
#[then("a Save As dialog opens with a suggested file name")]
fn suggested_name(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    let name = format!("{}.opd", world.document_name(id));
    world.expect_text("Save in");
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected a selected proposal"
    );
}
#[when("I cancel the Save As dialog")]
fn cancel_save_as(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the part remains open and active")]
fn open_and_active(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    world.expect_no_text("Save in");
    world.expect_active(id);
}
#[given("another part was active before the new part was created")]
fn previous_part(world: &mut ApplicationWorld) {
    // Given order puts the new part first; a second open part keeps both facts true.
    let new_part = world.active_id();
    let other = world.add_part();
    world.activate_through_tab(new_part);
    world.previous_document = Some(other);
}
#[when("I accept the suggested file name in the Save As dialog")]
fn accept_suggested(world: &mut ApplicationWorld) {
    world.accept_save_as();
}
#[then("the part file exists at the selected location")]
fn file_exists(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved file");
    assert!(path.exists(), "saved file {path:?}: expected it on disk");
}
#[then("the new part's tab disappears")]
fn new_tab_gone(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected a closed document");
    assert!(
        !world.document_is_open(id),
        "document {id}: expected it closed"
    );
}
#[when("I choose Close")]
#[when("the user chooses Close")]
fn choose_close(world: &mut ApplicationWorld) {
    world.click_text("Close");
}
