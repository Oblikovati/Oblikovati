// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn only_the_active_project_is_marked() {
    let mut project = GuiProject {
        name: "Gearbox".into(),
        location: "/work/gearbox.opj".into(),
        active: true,
        settings: Vec::new(),
    };
    assert_eq!(
        project_row_text(&project),
        "(active) Gearbox  /work/gearbox.opj"
    );
    project.active = false;
    assert_eq!(project_row_text(&project), "Gearbox  /work/gearbox.opj");
}
#[test]
fn a_request_asks_for_one_frame_until_it_is_drawn() {
    let mut dialog = ProjectsDialog::default();
    assert!(!dialog.needs_frame());
    dialog.request();
    assert!(dialog.needs_frame());
}
