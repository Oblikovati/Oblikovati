// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    wire::{AddSheetUsingFormatArgs, DefineSheetFormatArgs, SheetInfo},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_formats_survive_redefinition_history_and_reopen() {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    let original = plate_format();
    verify_format_revisions(&client, original.clone());
    let reopened = reopen(&client, document);
    let saved = stamp(&client, "", &original.name);
    assert_eq!(saved.name, "Sheet:4");
    assert_custom_shape(&saved);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    let drawing = &state.drawing_documents[&reopened];
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    assert_eq!(
        sheet.border_h_label,
        oblikovati_api::types::NUMERIC_BORDER_LABEL
    );
    assert_eq!(sheet.border_v_label, oblikovati_api::types::NO_BORDER_LABEL);
}

fn verify_format_revisions(client: &Client<impl Caller>, original: DefineSheetFormatArgs) {
    let invalid = DefineSheetFormatArgs {
        name: String::new(),
        ..original.clone()
    };
    assert!(client.drawing().define_sheet_format(&invalid).is_err());
    let defined = client.drawing().define_sheet_format(&original).unwrap();
    assert_eq!(defined.sheets.unwrap().len(), 1);
    client.transactions().undo().unwrap();
    let request = AddSheetUsingFormatArgs {
        format: original.name.clone(),
        ..Default::default()
    };
    assert!(client.drawing().add_sheet_using_format(&request).is_err());
    client.transactions().redo().unwrap();
    let earlier = stamp(client, "Plate 1", &original.name);
    assert_eq!((earlier.width_mm, earlier.height_mm), (594., 420.));
    assert_eq!((earlier.border_h_zones, earlier.border_v_zones), (6, 4));
    assert_eq!(earlier.title_block_location, "bottomLeft");
    redefine_format(client, original, earlier);
}

fn redefine_format(
    client: &Client<impl Caller>,
    original: DefineSheetFormatArgs,
    earlier: SheetInfo,
) {
    let updated = DefineSheetFormatArgs {
        size: "custom".into(),
        width_mm: 200.,
        height_mm: 120.,
        h_zones: 3,
        v_zones: 2,
        v_label_mode: "none".into(),
        title_block_location: "topRight".into(),
        ..original
    };
    client.drawing().define_sheet_format(&updated).unwrap();
    let newer = stamp(client, "Plate 2", &updated.name);
    assert_custom_shape(&newer);
    assert!(
        client
            .drawing()
            .add_sheet_using_format(&AddSheetUsingFormatArgs {
                name: earlier.name.clone(),
                format: updated.name,
            })
            .is_err()
    );
    let sheets = client
        .drawing()
        .list_sheets(&Default::default())
        .unwrap()
        .sheets
        .unwrap();
    assert_eq!(sheets.len(), 3);
    assert_eq!(
        sheets[1],
        SheetInfo {
            active: false,
            ..earlier
        }
    );
    assert_eq!(sheets[2], newer);
}

fn plate_format() -> DefineSheetFormatArgs {
    DefineSheetFormatArgs {
        name: "Plate".into(),
        size: "a2".into(),
        orientation: "landscape".into(),
        h_zones: 6,
        v_zones: 4,
        h_label_mode: "numeric".into(),
        title_block_location: "bottomLeft".into(),
        ..Default::default()
    }
}

fn assert_custom_shape(sheet: &SheetInfo) {
    assert_eq!((sheet.width_mm, sheet.height_mm), (200., 120.));
    assert_eq!((sheet.border_h_zones, sheet.border_v_zones), (3, 2));
    assert!(sheet.active && sheet.has_border && sheet.has_title_block);
    assert_eq!(sheet.title_block_location, "topRight");
}

fn stamp(client: &Client<impl Caller>, name: &str, format: &str) -> SheetInfo {
    client
        .drawing()
        .add_sheet_using_format(&AddSheetUsingFormatArgs {
            name: name.into(),
            format: format.into(),
        })
        .unwrap()
        .sheet
}

fn reopen(client: &Client<impl Caller>, drawing: u64) -> u64 {
    let path = document_fixture_path!("/memory/reusable-format.odd");
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
