// SPDX-License-Identifier: GPL-2.0-only
//! Captured template catalog observations require separately supplied template assets.
use super::*;

#[given("the new-document template chooser shows the metric folder in grouped Details view")]
fn metric_details_chooser(world: &mut ApplicationWorld) {
    open_chooser(world);
    metric_selected(world);
    grouped_by_default(world);
    choose_template_view(world, "Details");
    world.expect_text("Name");
}
pub(super) fn require_named_catalog(case: &str) {
    super::super::super::gap_runner::block_fixture(&format!(
        "{case}: captured named templates and their folder configuration are unavailable; see document-creation/BLOCKERS.md"
    ));
}
#[given("the previously selected standard inch part is displayed in the details pane")]
fn selected_captured_inch(world: &mut ApplicationWorld) {
    super::named_catalog::inch_details(world);
}
#[given("the gallery is at its leftmost horizontal position")]
fn captured_gallery_view(_: &mut ApplicationWorld) {
    require_named_catalog("captured narrow gallery and template order");
}
#[given("Standard (mm) is selected in the expanded Part group")]
fn captured_collapsed_selection(_: &mut ApplicationWorld) {
    require_named_catalog("selected Standard (mm) part template");
}
#[test]
fn missing_named_catalog_is_a_fixture_block() {
    let error = std::panic::catch_unwind(|| require_named_catalog("captured catalog"))
        .expect_err("the captured catalog must not be inferred");
    let message = error.downcast_ref::<String>().expect("fixture diagnostic");
    assert!(message.contains("BDD fixture blocked: captured catalog"));
    assert!(message.contains("templates and their folder configuration are unavailable"));
}
