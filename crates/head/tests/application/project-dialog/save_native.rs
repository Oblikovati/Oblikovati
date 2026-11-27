// SPDX-License-Identifier: GPL-2.0-only
//! Saving project settings uses the native dialog and preserves external edits.
use super::*;
use oblikovati_persistence::project_file::ProjectConfiguration;

#[test]
fn save_project_settings_checks_for_conflicts_and_refreshes_the_chooser() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    open_projects_dialog(&mut world);
    let root = world.scratch_folder();
    let path = root.join("gearbox.opj");
    let original = std::fs::read(&path).unwrap();
    choose_button(&mut world, "Edit Settings");
    assert!(!world.run_is_enabled("Save"));
    set_field(&mut world, "Project name", "Discarded");
    choose_button(&mut world, "Cancel Edit");
    assert_eq!(std::fs::read(&path).unwrap(), original);

    install_template(&root);
    edit_project_fields(&mut world);
    let external = b"schemaVersion: 1\nname: External change\nworkspace: work\n";
    std::fs::write(&path, external).unwrap();
    choose_button(&mut world, "Save");
    assert!(holds_text(&world, "changed on disk since it was loaded"));
    assert_eq!(world.state().projects[0].name, PROJECT_NAME);
    assert_eq!(std::fs::read(&path).unwrap(), external);

    std::fs::write(&path, original).unwrap();
    choose_button(&mut world, "Save");
    assert_saved(&mut world, &path);
    choose_button(&mut world, "Done");
    assert_eq!(world.state().document, Some(document));
    world.expect_text("Shop template");
    assert!(holds_text(&world, "Project: Updated Gearbox"));
    world.close();
}

fn edit_project_fields(world: &mut ApplicationWorld) {
    choose_button(world, "Edit Settings");
    for (label, value) in [
        ("Project name", "Updated Gearbox"),
        ("Workspace folder", "new-work"),
        ("Templates folder", "new-templates"),
        ("Design data folder", "new-design-data"),
    ] {
        set_field(world, label, value);
    }
    choose_button(world, "Add Library");
    set_field(world, "Reference folder", "new-library");
}

fn set_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let run = world.run(label);
    let point = [run.minimum[0] - 46., run.center()[1]];
    world.click(super::super::diagnostics::pointer_pixels(point));
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(value);
}

fn install_template(root: &std::path::Path) {
    let templates = root.join("new-templates/Metric");
    std::fs::create_dir_all(&templates).unwrap();
    std::fs::create_dir_all(root.join("new-design-data")).unwrap();
    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/main-window-frame-start.opd");
    std::fs::copy(source, templates.join("Shop template.opd")).unwrap();
}

fn assert_saved(world: &mut ApplicationWorld, path: &std::path::Path) {
    world.expect_no_text("Cancel Edit");
    assert_eq!(world.state().projects[0].name, "Updated Gearbox");
    let saved = ProjectConfiguration::decode(&std::fs::read(path).unwrap()).unwrap();
    assert_eq!(saved.name, "Updated Gearbox");
    assert_eq!(saved.workspace, "new-work");
    assert_eq!(saved.templates, "new-templates");
    assert_eq!(saved.design_data, "new-design-data");
    assert_eq!(saved.libraries, ["new-library"]);
}
