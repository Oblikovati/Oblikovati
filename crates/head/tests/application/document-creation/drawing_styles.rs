// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    wire::{SetStandardArgs, StandardStyleInfo},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_drafting_style_edits_are_undoable_and_persist() {
    let (mut owner, _) = owner();
    assert!(
        Client::new(owner.document_host())
            .drawing_styles()
            .list_standards(&Default::default())
            .is_err()
    );
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    verify_standard_history(&client);
    let path = "/memory/drafting-standard.odd";
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
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
    assert_ansi(&current(&client));
}

fn verify_standard_history(client: &Client<impl Caller>) {
    let styles = client.drawing_styles();
    let listed = styles.list_standards(&Default::default()).unwrap();
    assert_eq!(listed.standards.unwrap(), ["iso", "ansi"]);
    assert_eq!(listed.active, "iso");
    let iso = current(client);
    assert_eq!(
        (iso.dimension.unit.as_str(), iso.dimension.decimal_places),
        ("mm", 2)
    );
    assert_eq!(
        (iso.line.weight_mm, iso.dimension.line_weight_mm),
        (0.5, 0.25)
    );
    assert!(
        styles
            .set_standard(&SetStandardArgs {
                standard: "din".into()
            })
            .is_err()
    );
    assert_eq!(current(client), iso);
    let ansi = styles
        .set_standard(&SetStandardArgs {
            standard: "ansi".into(),
        })
        .unwrap()
        .style;
    assert_ansi(&ansi);
    assert_eq!(current(client), ansi);
    assert_eq!(
        styles.list_standards(&Default::default()).unwrap().active,
        "ansi"
    );
    client.transactions().undo().unwrap();
    assert_eq!(current(client), iso);
    client.transactions().redo().unwrap();
    assert_eq!(current(client), ansi);
}

fn current(client: &Client<impl Caller>) -> StandardStyleInfo {
    client
        .drawing_styles()
        .get_active_style(&Default::default())
        .unwrap()
        .style
}

fn assert_ansi(style: &StandardStyleInfo) {
    assert_eq!(style.standard, "ansi");
    assert_eq!(
        (
            style.dimension.unit.as_str(),
            style.dimension.decimal_places
        ),
        ("in", 3)
    );
    assert_eq!(
        (
            style.dimension.arrow_size_mm,
            style.dimension.text_height_mm
        ),
        (3., 3.)
    );
    assert_eq!(
        (style.line.weight_mm, style.dimension.line_weight_mm),
        (0.6, 0.3)
    );
    assert_eq!(style.text.name, "Note Text (ANSI)");
    assert_eq!(style.text.font_name, "Arial");
}
