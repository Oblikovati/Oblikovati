// SPDX-License-Identifier: GPL-2.0-only
//! Exercise rejection and recovery through the native Extrude profile selector.
use super::*;

#[test]
fn rejected_extrusion_retains_draft_and_success_commits_once() {
    let (mut world, document) = requested_extrusion_fixture();
    let generation = document_generation(&mut world, document);
    set_extrusion_profile(&mut world, "1 Profile", "0", "99");
    world.click_text("OK");
    world.pump(Vec::new());
    world.expect_text("Extrusion failed");
    world.expect_text("Distance A");
    assert_no_feature(&mut world, document, generation);
    set_extrusion_profile(&mut world, "Profile 100", "99", "0");
    assert!(world.state().extrude_preview_active);
    world.click_text("OK");
    world.pump(Vec::new());
    world.expect_no_text("Distance A");
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert_eq!(session.snapshot().work_geometry.features().len(), 1);
    world.close();
}
fn set_extrusion_profile(world: &mut ApplicationWorld, summary: &str, old: &str, value: &str) {
    let selector = world
        .run_in_window("Extrude", &super::super::world::squeezed(summary))
        .center();
    world.click(super::super::diagnostics::pointer_pixels(selector));
    let popup = world.run("Profile index").list;
    let at = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == popup && run.text == old)
        .unwrap_or_else(|| panic!("expected profile index {old} in the selector"))
        .center();
    world.click(super::super::diagnostics::pointer_pixels(at));
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
    let popup = world.run("Profile index").list;
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.list == popup && run.text == value),
        "expected {value:?} in popup list {popup}, runs: {:?}",
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.list == popup)
            .map(|run| &run.text)
            .collect::<Vec<_>>()
    );
    let profile_label = world.run_in_window("Extrude", "Profiles").center();
    world.click(super::super::diagnostics::pointer_pixels(profile_label));
    world.expect_no_text("Profile index");
}
