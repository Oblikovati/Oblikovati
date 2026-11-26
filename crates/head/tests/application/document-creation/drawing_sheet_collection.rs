// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::client::Caller;
use oblikovati_api::wire::{
    AddBaseViewArgs, AddSheetArgs, RemoveSheetArgs, SetActiveSheetArgs, SetModelReferenceArgs,
    SheetInfo,
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_sheet_api_edits_the_document_and_switches_the_displayed_projection() {
    let (mut owner, _) = owner();
    assert!(
        Client::new(owner.document_host())
            .drawing()
            .list_sheets(&Default::default())
            .is_err()
    );
    assert!(
        Client::new(owner.document_host())
            .drawing()
            .add_sheet(&Default::default())
            .is_err()
    );
    super::super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let reference = std::env::temp_dir()
        .join("head-scene.opd")
        .to_string_lossy()
        .into_owned();
    let client = Client::new(owner.document_host());
    client
        .drawing()
        .set_model_reference(&SetModelReferenceArgs {
            full_document_name: reference,
        })
        .unwrap();
    client
        .drawing_views()
        .add_base(&AddBaseViewArgs {
            center_xmm: 100.,
            center_ymm: 80.,
            ..Default::default()
        })
        .unwrap();
    assert_formats(&client);
    drop(client);

    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&drawing].lines.is_empty());
    let client = Client::new(owner.document_host());
    activate(&client, "Sheet:1");
    assert_eq!(
        client
            .drawing_views()
            .list(&Default::default())
            .unwrap()
            .views
            .unwrap()
            .len(),
        1
    );
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(!state.drawing_projections[&drawing].lines.is_empty());
    let client = Client::new(owner.document_host());
    assert_removal_and_history(&client);
    assert_round_trip(&client, drawing);
}

fn assert_formats(client: &Client<impl Caller>) {
    let first = sheets(client);
    assert_eq!(first.len(), 1);
    assert_eq!(
        (first[0].width_mm, first[0].height_mm, first[0].active),
        (420., 297., true)
    );
    assert!(first[0].has_border && first[0].has_title_block);
    for args in [
        AddSheetArgs {
            size: "b5".into(),
            ..Default::default()
        },
        AddSheetArgs {
            size: "a4".into(),
            orientation: "wide".into(),
            ..Default::default()
        },
        AddSheetArgs {
            size: "custom".into(),
            width_mm: 250.,
            ..Default::default()
        },
        AddSheetArgs {
            size: "a4".into(),
            name: "Sheet:1".into(),
            ..Default::default()
        },
    ] {
        assert!(client.drawing().add_sheet(&args).is_err());
    }
    assert_eq!(sheets(client), first);
    let second = client
        .drawing()
        .add_sheet(&AddSheetArgs {
            size: "a4".into(),
            ..Default::default()
        })
        .unwrap()
        .sheet;
    assert_eq!(
        (
            second.name.as_str(),
            second.orientation.as_str(),
            second.width_mm,
            second.height_mm
        ),
        ("Sheet:2", "portrait", 210., 297.)
    );
    let third = client
        .drawing()
        .add_sheet(&AddSheetArgs {
            name: "Custom".into(),
            orientation: "landscape".into(),
            width_mm: 180.,
            height_mm: 250.,
            ..Default::default()
        })
        .unwrap()
        .sheet;
    assert_eq!(
        (
            third.size.as_str(),
            third.width_mm,
            third.height_mm,
            third.active
        ),
        ("custom", 180., 250., true)
    );
    assert_eq!(
        sheets(client)
            .iter()
            .map(|sheet| sheet.active)
            .collect::<Vec<_>>(),
        [false, false, true]
    );
}

fn assert_removal_and_history(client: &Client<impl Caller>) {
    assert!(
        client
            .drawing()
            .set_active_sheet(&SetActiveSheetArgs {
                name: "Missing".into()
            })
            .is_err()
    );
    client.transactions().undo().unwrap();
    assert!(sheets(client)[2].active);
    client.transactions().redo().unwrap();
    assert!(sheets(client)[0].active);
    let previous = sheets(client);
    let survivors = client
        .drawing()
        .remove_sheet(&RemoveSheetArgs {
            name: "Sheet:1".into(),
        })
        .unwrap()
        .sheets
        .unwrap();
    assert_eq!(
        survivors
            .iter()
            .map(|sheet| sheet.name.as_str())
            .collect::<Vec<_>>(),
        ["Sheet:2", "Custom"]
    );
    assert!(survivors[0].active);
    assert!(
        client
            .drawing_views()
            .list(&Default::default())
            .unwrap()
            .views
            .is_none()
    );
    client.transactions().undo().unwrap();
    assert_eq!(sheets(client), previous);
    assert!(
        client
            .drawing_views()
            .list(&Default::default())
            .unwrap()
            .views
            .is_some()
    );
    client.transactions().redo().unwrap();
    assert_eq!(sheets(client), survivors);
}

fn assert_round_trip(client: &Client<impl Caller>, drawing: u64) {
    client
        .drawing()
        .remove_sheet(&RemoveSheetArgs {
            name: "Custom".into(),
        })
        .unwrap();
    assert!(
        client
            .drawing()
            .remove_sheet(&RemoveSheetArgs {
                name: "Sheet:2".into()
            })
            .unwrap_err()
            .0
            .contains("at least one sheet")
    );
    let expected = sheets(client);
    let path = "/memory/edited-sheets.odd";
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(sheets(client), expected);
}

fn sheets(client: &Client<impl Caller>) -> Vec<SheetInfo> {
    client
        .drawing()
        .list_sheets(&Default::default())
        .unwrap()
        .sheets
        .unwrap()
}

fn activate(client: &Client<impl Caller>, name: &str) {
    let response = client
        .drawing()
        .set_active_sheet(&SetActiveSheetArgs { name: name.into() })
        .unwrap();
    assert!(response.sheet.active);
    assert_eq!(response.sheet.name, name);
}
