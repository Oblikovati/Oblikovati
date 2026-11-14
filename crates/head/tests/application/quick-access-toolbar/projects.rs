// SPDX-License-Identifier: GPL-2.0-only
//! Open project settings from the quick access toolbar.
use super::*;

#[test]
fn quick_access_projects_opens_the_active_project_without_closing_the_part() {
    let mut world = ApplicationWorld::new();
    let path = world.scratch_folder().join("toolbar-project.opj");
    std::fs::write(
        &path,
        "schemaVersion: 1\nname: Toolbar Project\nworkspace: work\n",
    )
    .unwrap();
    world.app().owner.load_project_for_test(&path);
    part_active(&mut world);
    assert!(on_row(&world, "Projects"));
    world.click(row_center(&world, "Projects"));
    world.expect_text("Project Settings");
    world.expect_text("Toolbar Project");
    world.click_text("Done");
    world.expect_no_text("Project Settings");
    part_stays(&mut world);
    world.close();
}
