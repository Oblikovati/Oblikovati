// SPDX-License-Identifier: GPL-2.0-only
//! Drawing view history and document units.
use super::*;

#[test]
fn gui_base_view_is_one_undo_step_and_rolls_back_on_failure() {
    use oblikovati_gui::{
        GuiBaseViewRequest, GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits,
    };
    let (mut owner, _) = owner();
    let part = super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let mut request = GuiBaseViewRequest {
        drawing,
        source: part,
        orientation: "invalid".into(),
        scale: 1.,
        center: [100., 80.],
    };
    assert!(owner.base_view_preview_from_gui(Some(&request)).is_err());
    assert!(owner.base_view_from_gui(&request).is_err());
    let current = owner.gui_command_state().unwrap();
    assert!(
        current.drawing_documents[&drawing]
            .model_reference()
            .is_empty()
    );
    assert!(
        current.drawing_documents[&drawing].sheets()[0]
            .views
            .is_empty()
    );
    request.orientation = "front".into();
    assert!(owner.base_view_preview_from_gui(Some(&request)).unwrap());
    let preview = owner.gui_command_state().unwrap();
    assert!(!preview.drawing_projections[&drawing].lines.is_empty());
    assert!(
        preview.drawing_projections[&drawing]
            .lines
            .iter()
            .all(|line| line.preview)
    );
    assert!(
        preview.drawing_documents[&drawing].sheets()[0]
            .views
            .is_empty()
    );
    owner
        .files
        .commands(part)
        .unwrap()
        .run("Change depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    assert!(
        owner.gui_command_state().unwrap().drawing_projections[&drawing]
            .lines
            .is_empty()
    );
    assert!(owner.base_view_preview_from_gui(Some(&request)).unwrap());
    owner.files.activate(part).unwrap();
    owner.files.activate(drawing).unwrap();
    assert!(
        owner.gui_command_state().unwrap().drawing_projections[&drawing]
            .lines
            .is_empty()
    );
    assert!(owner.base_view_preview_from_gui(Some(&request)).unwrap());
    assert!(owner.base_view_preview_from_gui(None).unwrap());
    assert!(
        owner.gui_command_state().unwrap().drawing_projections[&drawing]
            .lines
            .is_empty()
    );
    assert!(owner.base_view_from_gui(&request).unwrap());
    let current = owner.gui_command_state().unwrap();
    assert_eq!(
        current.drawing_documents[&drawing].sheets()[0].views.len(),
        1
    );
    assert!(!current.drawing_projections[&drawing].lines.is_empty());
    owner.files.commands(drawing).unwrap().undo().unwrap();
    let current = owner.gui_command_state().unwrap();
    assert!(
        current.drawing_documents[&drawing]
            .model_reference()
            .is_empty()
    );
    assert!(
        current.drawing_documents[&drawing].sheets()[0]
            .views
            .is_empty()
    );
    owner.files.commands(drawing).unwrap().redo().unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().drawing_documents[&drawing].sheets()[0]
            .views
            .len(),
        1
    );
}

#[test]
fn gui_document_units_use_public_history_and_reject_stale_document() {
    use oblikovati_api::wire::SetDocumentUnitsArgs;
    let (mut owner, _) = owner();
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let original = owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap()
        .length_unit;
    let request = SetDocumentUnitsArgs {
        length_unit: Some(Box::new("in".into())),
        ..Default::default()
    };
    assert!(
        owner
            .set_document_units_from_gui(document + 1, &request)
            .is_err()
    );
    assert!(
        owner
            .set_document_units_from_gui(document, &request)
            .unwrap()
    );
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        "in"
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        original
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    let path = "/memory/units.opd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        "in"
    );
}
