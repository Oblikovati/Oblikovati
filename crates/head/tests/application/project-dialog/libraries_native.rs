// SPDX-License-Identifier: GPL-2.0-only
//! New projects retain transferred reference library roots across saves.
use super::*;
use oblikovati_persistence::project_file::ProjectConfiguration;
use std::path::{Path, PathBuf};

#[test]
fn wizard_saves_transferred_libraries_in_search_order() {
    let mut world = ApplicationWorld::new();
    point_home_at_scratch(&mut world);
    let root = world.scratch_folder();
    let libraries = register_library_source(&mut world, &root);
    open_wizard_on_file_page(&mut world);
    browse_folder(&mut world);
    select_documents(&mut world);
    accept_folder(&mut world);
    choose_button(&mut world, "Next");
    world.expect_text("Select Libraries");
    assert!(!world.run_is_enabled(">>"));
    transfer(&mut world, &libraries[0], ">>");
    transfer(&mut world, &libraries[1], ">>");
    transfer(&mut world, &libraries[0], "<<");
    choose_button(&mut world, "Back");
    choose_button(&mut world, "Next");
    transfer(&mut world, &libraries[0], ">>");
    choose_button(&mut world, "Finish");
    world.expect_no_text("Select Libraries");
    assert_stored_libraries(&root, [&libraries[1], &libraries[0]]);
    world.close();
}

fn register_library_source(world: &mut ApplicationWorld, root: &Path) -> [PathBuf; 2] {
    let source = root.join("S");
    let libraries = [source.join("LibA"), source.join("LibB")];
    for path in &libraries {
        std::fs::create_dir_all(path).unwrap();
    }
    let project = source.join("source.opj");
    std::fs::write(
        &project,
        "schemaVersion: 1\nname: Source\nworkspace: .\nlibraries: [LibA, LibB, ./LibA]\n",
    )
    .unwrap();
    world.app().owner.load_desktop_project(&project).unwrap();
    world.pump(Vec::new());
    libraries
}

fn transfer(world: &mut ApplicationWorld, root: &Path, button: &str) {
    choose_button(world, root.to_str().unwrap());
    assert!(
        world.run_is_enabled(button),
        "expected {button} for {root:?}"
    );
    choose_button(world, button);
}

fn assert_stored_libraries(root: &Path, expected: [&PathBuf; 2]) {
    let path = root.join("Documents/New Project.opj");
    let config = ProjectConfiguration::decode(&std::fs::read(&path).unwrap()).unwrap();
    let bound = config.bind(&path).unwrap();
    assert_eq!(bound.lookup.library_roots(), expected.map(PathBuf::clone));
    assert_eq!(bound.lookup.search_paths().next(), path.parent());
}
