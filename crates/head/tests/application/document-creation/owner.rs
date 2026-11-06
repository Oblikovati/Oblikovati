// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn selected_part_templates_create_real_documents_with_the_selected_units() {
    let (mut owner, _) = owner();
    for units in [
        oblikovati_gui::GuiTemplateUnits::Metric,
        oblikovati_gui::GuiTemplateUnits::Inch,
    ] {
        let template = oblikovati_gui::GuiDocumentTemplate {
            kind: oblikovati_gui::GuiTemplateKind::Part,
            units,
        };
        assert!(owner.create_document_template(template).unwrap());
        let client = Client::new(owner.document_host());
        let units = client.documents().get_units().unwrap();
        assert_eq!(units.length_unit, template.length_unit());
        let state = owner.gui_command_state().unwrap();
        assert_eq!(
            state.document_type,
            Some(oblikovati_api::types::DOCUMENT_PART)
        );
        assert!(state.browser.as_ref().unwrap().has_origin);
        assert!(state.browser.as_ref().unwrap().features.is_empty());
    }
}
#[test]
fn each_template_kind_creates_its_own_document_type_and_units() {
    use oblikovati_api::types::*;
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    let (mut owner, _) = owner();
    for (kind, document_type, name) in [
        (GuiTemplateKind::SheetMetal, DOCUMENT_PART, "Part1"),
        (GuiTemplateKind::Assembly, DOCUMENT_ASSEMBLY, "Assembly1"),
        (GuiTemplateKind::Drawing, DOCUMENT_DRAWING, "Drawing1"),
        (
            GuiTemplateKind::Presentation,
            DOCUMENT_PRESENTATION,
            "Presentation1",
        ),
    ] {
        let template = GuiDocumentTemplate {
            kind,
            units: GuiTemplateUnits::Metric,
        };
        assert!(owner.create_document_template(template).unwrap());
        let state = owner.gui_command_state().unwrap();
        assert_eq!(state.document_type, Some(document_type));
        let active = state.documents.iter().find(|row| row.active).unwrap();
        assert_eq!(active.name, name);
        assert_eq!(
            active.sub_type.is_empty(),
            kind != GuiTemplateKind::SheetMetal
        );
        let units = Client::new(owner.document_host())
            .documents()
            .get_units()
            .unwrap();
        assert_eq!(units.length_unit, "mm");
    }
}
#[test]
fn an_open_transaction_refuses_a_template_and_creates_nothing() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    invoke(&mut owner, &awake, |client| {
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "edit".into(),
            })
            .unwrap()
    });
    let refusal = owner
        .create_document_template(oblikovati_gui::GuiDocumentTemplate::default())
        .unwrap_err();
    assert!(refusal.contains("transaction"), "refusal {refusal:?}");
    assert_eq!(owner.documents().workspace().documents().len(), 1);
}
