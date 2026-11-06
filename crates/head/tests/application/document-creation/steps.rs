// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
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
    world.expect_blank_region([400, 1400], [250, 650]);
}
#[then("its model browser displays an origin and an end-of-part marker")]
fn browser_origin(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_text("End of Part");
}
#[then("a new document tab is visible")]
fn new_tab(world: &mut ApplicationWorld) {
    let state = world.app().owner.gui_command_state().unwrap();
    assert!(
        state
            .documents
            .iter()
            .any(|document| Some(document.id) == state.document && document.visible)
    );
    world.expect_text("Part");
}
