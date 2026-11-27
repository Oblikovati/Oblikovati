// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    types::{Variant, VariantValue},
    wire::{
        AddSheetArgs, SetModelReferenceArgs, SetPropertyArgs, SetSheetRevisionArgs,
        SetTitleBlockArgs, TitleBlockFieldsArgs,
    },
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::property::{DESIGN_TRACKING, DOCUMENT_SUMMARY, SUMMARY_INFORMATION};

#[test]
fn drawing_title_block_reads_live_model_properties() {
    let (mut owner, _) = owner();
    let part = super::super::part_scene::open_prism(&mut owner, false);
    create_drawing(&mut owner);
    let client = Client::new(owner.document_host());
    assert!(
        fields(&client, "")
            .iter()
            .all(|field| field.value.is_empty())
    );
    client
        .drawing()
        .set_model_reference(&SetModelReferenceArgs {
            full_document_name: std::env::temp_dir()
                .join("head-scene.opd")
                .to_string_lossy()
                .into_owned(),
        })
        .unwrap();
    for (set, name, value) in property_fixture() {
        property(&client, part, set, name, value);
    }
    let resolved = fields(&client, "Sheet:1");
    let expected = [
        ("Title", "Bracket", "Summary Information:Title"),
        (
            "Part Number",
            "42",
            "Design Tracking Properties:Part Number",
        ),
        (
            "Material",
            "1.25e+06",
            "Design Tracking Properties:Material",
        ),
        ("Drawn By", "", "Summary Information:Author"),
        (
            "Revision",
            "true",
            "Design Tracking Properties:Revision Number",
        ),
        ("Company", "", "Document Summary Information:Company"),
    ];
    assert_eq!(
        resolved
            .iter()
            .map(|f| (f.name.as_str(), f.value.as_str(), f.source.as_str()))
            .collect::<Vec<_>>(),
        expected
    );
    property(
        &client,
        part,
        DESIGN_TRACKING,
        "Part Number",
        VariantValue::String("PN-43".into()),
    );
    assert_eq!(fields(&client, "")[1].value, "PN-43");
    assert!(
        client
            .drawing()
            .title_block_fields(&TitleBlockFieldsArgs {
                sheet: "Missing".into()
            })
            .is_err()
    );
}

#[test]
fn drawing_sheet_revision_and_title_block_edits_target_named_or_active_sheet() {
    let (mut owner, _) = owner();
    create_drawing(&mut owner);
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
    assert_sheet_edits(&client);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    let shown = &state.drawing_documents[&drawing];
    assert_eq!(
        shown.sheets()[0].title_block_at,
        oblikovati_api::types::TOP_LEFT_TITLE_BLOCK
    );
    assert_eq!(shown.sheets()[shown.active_sheet()].revision, "B");
}

fn assert_sheet_edits(client: &Client<impl Caller>) {
    let corner = |location: &str| {
        client.drawing().set_title_block(&SetTitleBlockArgs {
            sheet: "Sheet:1".into(),
            location: location.into(),
        })
    };
    assert!(corner("middle").is_err());
    let moved = corner("topLeft").unwrap().sheet;
    assert_eq!(moved.title_block_location, "topLeft");
    assert!(!moved.active);
    let revision = |value: &str| {
        client.drawing().set_sheet_revision(&SetSheetRevisionArgs {
            revision: value.into(),
            ..Default::default()
        })
    };
    assert!(revision("bad\0revision").is_err());
    let revised = revision("B").unwrap().sheet;
    assert_eq!(
        (
            revised.name.as_str(),
            revised.active,
            revised.revision.as_str()
        ),
        ("Detail", true, "B")
    );
    client.transactions().undo().unwrap();
    let list = || {
        client
            .drawing()
            .list_sheets(&Default::default())
            .unwrap()
            .sheets
            .unwrap()
    };
    assert!(list()[1].revision.is_empty());
    assert_eq!(list()[0].title_block_location, "topLeft");
    client.transactions().redo().unwrap();
    assert_eq!(list()[1].revision, "B");
}

fn property_fixture() -> [(&'static str, &'static str, VariantValue); 6] {
    [
        (
            SUMMARY_INFORMATION,
            "Title",
            VariantValue::String("Bracket".into()),
        ),
        (DESIGN_TRACKING, "Part Number", VariantValue::Integer(42)),
        (DESIGN_TRACKING, "Material", VariantValue::Double(1250000.)),
        (
            DESIGN_TRACKING,
            "Revision Number",
            VariantValue::Boolean(true),
        ),
        (
            DOCUMENT_SUMMARY,
            "Company",
            VariantValue::Bytes(oblikovati_api::json::Bytes(Some(vec![1]))),
        ),
        (
            SUMMARY_INFORMATION,
            "Other",
            VariantValue::String("unbound".into()),
        ),
    ]
}

fn property(
    client: &Client<impl Caller>,
    document: u64,
    set: &str,
    name: &str,
    value: VariantValue,
) {
    client
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: set.into(),
            name: name.into(),
            value: Variant {
                value,
                unit: String::new(),
            },
        })
        .unwrap();
}

fn fields(client: &Client<impl Caller>, sheet: &str) -> Vec<oblikovati_api::wire::TitleBlockField> {
    let reply = client
        .drawing()
        .title_block_fields(&TitleBlockFieldsArgs {
            sheet: sheet.into(),
        })
        .unwrap();
    assert_eq!(reply.definition_name, "Default");
    reply.fields.unwrap()
}

fn create_drawing(owner: &mut TestOwner) {
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
}
