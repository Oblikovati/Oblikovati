// SPDX-License-Identifier: GPL-2.0-only
//! Apply a project while the chooser has an old template selected.
use super::{ApplicationWorld, choose_button, choose_projects, holds_text};

#[test]
fn applying_a_project_updates_the_open_template_chooser() {
    let mut world = ApplicationWorld::new();
    install_project(&mut world, "Original", "Metric", "Metric shop");
    install_project(&mut world, "Alternate", "English", "English shop");
    let document = world.add_part();
    world.key(oblikovati_viewport::ViewportPhysicalKey::N, true);
    choose_button(&mut world, "Metric shop");
    choose_projects(&mut world);
    choose_button(&mut world, "Alternate");
    assert_active_project(&mut world, "Original", document);
    assert!(world.run_is_enabled("Apply"));
    choose_button(&mut world, "Apply");
    assert_active_project(&mut world, "Alternate", document);
    assert!(!world.run_is_enabled("Apply"));
    choose_button(&mut world, "Done");
    assert!(holds_text(&world, "Project: Alternate"));
    world.expect_text("English shop");
    world.expect_text("Select a template.");
    world.expect_no_text("Metric shop");
    assert!(!world.run_is_enabled("Create"));
    world.close();
}

fn install_project(world: &mut ApplicationWorld, name: &str, units: &str, title: &str) {
    let root = world.scratch_folder();
    let templates = root.join(name).join(units);
    std::fs::create_dir_all(&templates).unwrap();
    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/main-window-frame-start.opd");
    std::fs::copy(source, templates.join(format!("{title}.opd"))).unwrap();
    let project = root.join(format!("{name}.opj"));
    std::fs::write(
        &project,
        format!("schemaVersion: 1\nname: {name}\nworkspace: .\ntemplates: {name}\n"),
    )
    .unwrap();
    world.app().owner.load_desktop_project(&project).unwrap();
    world.pump(Vec::new());
}

fn assert_active_project(world: &mut ApplicationWorld, name: &str, document: u64) {
    let state = world.state();
    assert_eq!(state.document, Some(document));
    assert_eq!(
        state
            .projects
            .iter()
            .find(|project| project.active)
            .unwrap()
            .name,
        name
    );
    assert!(holds_text(world, &format!("(active) {name}")));
}
