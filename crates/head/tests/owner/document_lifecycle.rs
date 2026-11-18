// SPDX-License-Identifier: GPL-2.0-only
//! Native owner document and basic sketch lifecycle tests.
use super::*;

#[test]
fn gui_new_part_creates_an_active_unsaved_document() {
    let (mut owner, _) = owner();
    assert!(owner.create_part_document().unwrap());
    let workspace = owner.documents().workspace();
    let first = workspace.active_document().unwrap();
    assert_eq!(workspace.documents(), &[first]);
    assert!(!owner.close_allowed().unwrap());
    assert!(owner.create_part_document().unwrap());
    assert_eq!(owner.documents().workspace().documents().len(), 2);
    assert_ne!(owner.documents().workspace().active_document(), Some(first));
}
#[test]
fn gui_sketch_xy_enters_and_finishes_the_owned_edit() {
    use oblikovati_gui::GuiAuthoringContext::*;
    let (mut owner, _) = owner();
    assert!(owner.edit_sketch_from_gui(true).is_err());
    gui_commands::assert_authoring_context(&mut owner, NoDocument);
    owner.create_part_document().unwrap();
    gui_commands::assert_authoring_context(&mut owner, Part);
    assert!(owner.edit_sketch_from_gui(true).unwrap());
    gui_commands::assert_authoring_context(&mut owner, Sketch);
    let edit = owner.active_sketch_edit().unwrap();
    assert_eq!(edit.plane, oblikovati_model::sketch::SketchPlane::xy());
    assert!(owner.edit_sketch_from_gui(true).is_err());
    assert!(owner.edit_sketch_from_gui(false).unwrap());
    assert!(owner.active_sketch_edit().is_none());
    gui_commands::assert_authoring_context(&mut owner, Part);
    assert!(owner.edit_sketch_from_gui(false).is_err());
}
#[test]
fn gui_rectangle_requires_an_edit_and_uses_owned_placement() {
    let (mut owner, _) = owner();
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .is_err()
    );
    owner.create_part_document().unwrap();
    owner.edit_sketch_from_gui(true).unwrap();
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .unwrap()
    );
    assert!(owner.placement_active());
    owner.edit_sketch_from_gui(false).unwrap();
    assert!(!owner.placement_active());
    assert!(
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Rectangle)
            .is_err()
    );
}
#[test]
fn typed_workers_create_save_reopen_and_close_through_the_document_owner() {
    let (mut owner, awake) = owner();
    assert!(owner.close_allowed().unwrap());
    let id = create(&mut owner, &awake);
    assert!(!owner.close_allowed().unwrap());
    assert_eq!(owner.refused_close(), [id]);
    invoke(&mut owner, &awake, move |client| {
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/memory/part.opd".into(),
            })
            .unwrap()
    });
    assert!(owner.close_allowed().unwrap());
    verify_transaction_close_guard(&mut owner, &awake);
    verify_close_and_reopen(&mut owner, &awake, id);
}
fn verify_close_and_reopen(owner: &mut TestOwner, awake: &Receiver<()>, id: u64) {
    invoke(owner, awake, move |client| {
        client
            .documents()
            .close(&CloseDocumentArgs { id, force: false })
            .unwrap()
    });
    assert!(owner.documents().workspace().documents().is_empty());
    verify_reopened_owner(owner, awake);
}
fn verify_reopened_owner(owner: &mut TestOwner, awake: &Receiver<()>) {
    let reopened = invoke(owner, awake, |client| {
        client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/memory/part.opd".into(),
                ..Default::default()
            })
            .unwrap()
    });
    assert!(
        owner
            .documents()
            .workspace()
            .documents()
            .contains(&reopened.id)
    );
    assert!(owner.close_allowed().unwrap());
}
fn verify_transaction_close_guard(owner: &mut TestOwner, awake: &Receiver<()>) {
    invoke(owner, awake, |client| {
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "edit".into(),
            })
            .unwrap()
    });
    assert!(!owner.close_allowed().unwrap());
    invoke(owner, awake, |client| {
        client.transactions().abort().unwrap()
    });
    assert!(owner.close_allowed().unwrap());
}

struct FixedStartupPreference(Option<oblikovati_model::feature::work_plane::WorkPlaneOrigin>);
impl oblikovati_model::application_options::PartPreferenceAccess for FixedStartupPreference {
    fn part_options(&self) -> oblikovati_model::application_options::PartApplicationOptions {
        oblikovati_model::application_options::PartApplicationOptions {
            startup_sketch_plane: self.0,
            ..Default::default()
        }
    }
    fn set_part_options(
        &mut self,
        options: oblikovati_model::application_options::PartApplicationOptions,
    ) -> Result<(), String> {
        self.0 = options.startup_sketch_plane;
        Ok(())
    }
}
#[test]
fn new_part_and_metric_template_enter_the_selected_startup_sketch() {
    use oblikovati_model::{
        document_sketch::SketchPlacement, feature::work_plane::WorkPlaneOrigin,
    };
    for plane in [
        WorkPlaneOrigin::XY,
        WorkPlaneOrigin::YZ,
        WorkPlaneOrigin::XZ,
    ] {
        for template in [false, true] {
            let (owner, _) = owner();
            let mut owner =
                owner.with_part_preferences(Box::new(FixedStartupPreference(Some(plane))));
            if template {
                owner
                    .create_document_template(oblikovati_gui::GuiDocumentTemplate {
                        kind: oblikovati_gui::GuiTemplateKind::Part,
                        units: oblikovati_gui::GuiTemplateUnits::Metric,
                    })
                    .unwrap();
            } else {
                owner.create_part_document().unwrap();
            }
            let edit = owner
                .active_sketch_edit()
                .expect("expected active startup sketch");
            let session = owner
                .documents()
                .workspace()
                .session(edit.document)
                .unwrap();
            let sketches = session.snapshot().sketches.sketches();
            assert_eq!(sketches.len(), 1);
            assert_eq!(sketches[0].name(), "Sketch1");
            assert_eq!(sketches[0].placement(), SketchPlacement::OriginPlane(plane));
            assert_eq!(sketches[0].id(), edit.sketch);
        }
    }
}
#[test]
fn refused_startup_sketch_preserves_the_previous_part() {
    use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
    let (mut owner, _) = super::tool_veto::veto_owner("Create sketch");
    owner.create_part_document().unwrap();
    let previous = owner.documents().workspace().active_document().unwrap();
    let mut owner =
        owner.with_part_preferences(Box::new(FixedStartupPreference(Some(WorkPlaneOrigin::XY))));
    let error = owner.create_part_document().unwrap_err();
    assert!(error.contains("expected successful creation"), "{error}");
    assert_eq!(owner.documents().workspace().documents(), &[previous]);
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(previous)
    );
}
