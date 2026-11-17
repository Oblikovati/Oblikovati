// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn only_the_active_project_is_marked() {
    let mut project = GuiProject {
        name: "Gearbox".into(),
        location: "/work/gearbox.opj".into(),
        active: true,
        settings: Vec::new(),
        groups: Vec::new(),
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

#[test]
fn parent_close_notifies_only_its_project_tab_once() {
    let mut registry = crate::dockable_lifecycle::DockRegistry::default();
    let mut dialog = ProjectsDialog::default();
    dialog
        .register_dockable(&mut registry, "template-projects")
        .unwrap();
    dialog.open.set_open(true);
    registry.reconcile(|_| {}).unwrap();
    dialog.close();
    let mut titles = Vec::new();
    assert!(
        registry
            .reconcile(|title| titles.push(title.to_string_lossy().into_owned()))
            .unwrap()
    );
    assert_eq!(titles, ["Projects###template-projects"]);
    assert!(!registry.reconcile(|_| {}).unwrap());
}
