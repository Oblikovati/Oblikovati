// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    types::{ALPHABETICAL_BORDER_LABEL, NO_BORDER_LABEL, NUMERIC_BORDER_LABEL},
    wire::{AddDefaultBorderArgs, AddSheetArgs, SheetInfo},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_border_api_edits_named_sheet_and_restores_modes_after_reopen() {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    client
        .drawing()
        .add_sheet(&AddSheetArgs {
            size: "a4".into(),
            name: "Detail".into(),
            ..Default::default()
        })
        .unwrap();
    verify_sheet_history(&client);
    let reopened = reopen(&client, drawing);
    assert_eq!(zones(&sheets(&client)[0]), [4, 3]);
    assert_eq!(zones(&sheets(&client)[1]), [2, 2]);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    let sheets = state.drawing_documents[&reopened].sheets();
    assert_eq!(sheets[0].border_h_label, NUMERIC_BORDER_LABEL);
    assert_eq!(sheets[0].border_v_label, ALPHABETICAL_BORDER_LABEL);
    assert_eq!(sheets[1].border_h_label, NO_BORDER_LABEL);
}

fn verify_sheet_history(client: &Client<impl Caller>) {
    let initial = sheets(client);
    let mut draft = AddDefaultBorderArgs {
        sheet: "Sheet:1".into(),
        h_zones: 4,
        v_zones: 3,
        h_label_mode: "roman".into(),
        ..Default::default()
    };
    assert!(client.drawing().add_default_border(&draft).is_err());
    draft.h_label_mode = "numeric".into();
    draft.v_zones = 0;
    assert!(client.drawing().add_default_border(&draft).is_err());
    assert_eq!(sheets(client), initial);
    draft.v_zones = 3;
    let named = client.drawing().add_default_border(&draft).unwrap().sheet;
    assert!(!named.active && named.has_border);
    assert_eq!(zones(&named), [4, 3]);
    client.transactions().undo().unwrap();
    assert_eq!(sheets(client), initial);
    client.transactions().redo().unwrap();
    assert_eq!(sheets(client)[0], named);
    set_active_border(client);
}

fn set_active_border(client: &Client<impl Caller>) {
    let active = client
        .drawing()
        .add_default_border(&AddDefaultBorderArgs {
            h_zones: 2,
            v_zones: 2,
            h_label_mode: "none".into(),
            ..Default::default()
        })
        .unwrap()
        .sheet;
    assert!(active.active && active.name == "Detail");
    assert_eq!(zones(&active), [2, 2]);
}

fn reopen(client: &Client<impl Caller>, drawing: u64) -> u64 {
    let path = "/memory/zoned-drawing.odd";
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
        .unwrap()
        .id
}

fn sheets(client: &Client<impl Caller>) -> Vec<SheetInfo> {
    client
        .drawing()
        .list_sheets(&Default::default())
        .unwrap()
        .sheets
        .unwrap()
}

fn zones(sheet: &SheetInfo) -> [i64; 2] {
    [sheet.border_h_zones, sheet.border_v_zones]
}
