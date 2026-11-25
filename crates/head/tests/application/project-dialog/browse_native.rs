// SPDX-License-Identifier: GPL-2.0-only
//! Browse from the real Projects window and verify the desktop owner's registrations.
use super::super::diagnostics;
use super::{
    ApplicationWorld, PROJECT_NAME, choose_button, expect_projects_dialog, holds_text,
    open_projects_dialog,
};

const PICKER: &str = "Browse for Project";
const VALID: &str = "Second.opj";
const INVALID: &str = "Damaged.opj";

#[test]
fn browse_registers_a_project_after_validation() {
    let mut world = ApplicationWorld::new();
    let directory = world.scratch_folder();
    std::fs::write(directory.join(INVALID), "schemaVersion: 99\n").unwrap();
    std::fs::write(
        directory.join(VALID),
        "schemaVersion: 1\nname: Design Office\nworkspace: other-work\n",
    )
    .unwrap();
    std::fs::write(directory.join("Notes.txt"), "Not a project").unwrap();
    open_projects_dialog(&mut world);
    browse(&mut world);
    click_in_picker(&mut world, INVALID);
    click_in_picker(&mut world, "Open");
    assert!(holds_text(&world, "expected 1"));
    assert!(diagnostics::window_geometry(PICKER).is_some());
    assert_eq!(world.state().projects.len(), 1);

    click_in_picker(&mut world, VALID);
    click_in_picker(&mut world, "Cancel");
    world.expect_no_text(PICKER);
    assert_eq!(world.state().projects.len(), 1);
    browse(&mut world);
    open_valid_row(&mut world);
    assert_registered(&mut world, &directory.join(VALID));
    browse(&mut world);
    world.type_text(directory.join(VALID).to_str().unwrap());
    submit_open(&mut world);
    assert_registered(&mut world, &directory.join(VALID));
    world.close();
}

fn browse(world: &mut ApplicationWorld) {
    choose_button(world, "Browse");
    world.run_in_window(PICKER, "ProjectFiles(*.opj)");
    world.expect_no_text("Notes.txt");
}

fn click_in_picker(world: &mut ApplicationWorld, text: &str) {
    let run = if matches!(text, "Open" | "Cancel") {
        let list = diagnostics::window_draw_list(PICKER).unwrap();
        world
            .frame
            .runs
            .iter()
            .find(|run| run.list == list && run.text == text)
            .unwrap_or_else(|| {
                panic!(
                    "expected {text:?} in the project picker: {:?}",
                    world.text()
                )
            })
    } else {
        world.run_in_window(PICKER, text)
    };
    let point = run.center();
    world.click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn open_valid_row(world: &mut ApplicationWorld) {
    click_in_picker(world, VALID);
    submit_open(world);
}

fn submit_open(world: &mut ApplicationWorld) {
    click_in_picker(world, "Open");
    assert!(
        diagnostics::window_geometry(PICKER).is_none(),
        "picker still visible after result {:?}: {}",
        world.state().project_file_result,
        world.text()
    );
    expect_projects_dialog(world);
}

fn assert_registered(world: &mut ApplicationWorld, path: &std::path::Path) {
    let projects = world.state().projects;
    assert_eq!(projects.len(), 2);
    assert!(
        projects
            .iter()
            .any(|row| row.name == PROJECT_NAME && row.active)
    );
    assert!(projects.iter().any(|row| row.name == "Design Office"
        && !row.active
        && row.location == path.display().to_string()));
    assert!(holds_text(world, "other-work"));
}
