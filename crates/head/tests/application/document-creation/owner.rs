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

#[test]
fn gui_snapshot_keeps_current_sheets_for_inactive_drawings() {
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    use oblikovati_model::drawing::DrawingSheetRecipe;
    let (mut owner, _) = owner();
    let template = GuiDocumentTemplate {
        kind: GuiTemplateKind::Drawing,
        units: GuiTemplateUnits::Metric,
    };
    owner.create_document_template(template).unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    owner
        .files
        .commands(drawing)
        .unwrap()
        .run("Add sheet", |snapshot, _| {
            let mut sheet = DrawingSheetRecipe::a3_landscape("Portrait");
            sheet.orientation = oblikovati_api::types::SHEET_PORTRAIT;
            sheet.border = false;
            snapshot.drawing.as_mut().unwrap().add_sheet(sheet)
        })
        .unwrap();
    owner.create_part_document().unwrap();
    let state = owner.gui_command_state().unwrap();
    let content = &state.drawing_documents[&drawing];
    assert_eq!(content.active_sheet(), 1);
    assert_eq!(content.sheets()[1].dimensions_mm().unwrap(), [297., 420.]);
    assert!(!content.sheets()[1].border);
    owner.files.commands(drawing).unwrap().undo().unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.drawing_documents[&drawing].active_sheet(), 0);
}

#[test]
fn drawing_projection_tracks_open_part_edits_and_missing_sources() {
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    use oblikovati_model::drawing::DrawingBaseViewDefinition;
    let (mut owner, _) = owner();
    let part = super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let path = std::env::temp_dir()
        .join("head-scene.opd")
        .to_string_lossy()
        .into_owned();
    owner
        .files
        .commands(drawing)
        .unwrap()
        .run("Add view", |snapshot, _| {
            let drawing = snapshot.drawing.as_mut().unwrap();
            drawing.set_model_reference(&path)?;
            drawing.add_base_view_definition(
                0,
                DrawingBaseViewDefinition::new(
                    "Front",
                    oblikovati_api::types::BASE_VIEW_FRONT,
                    1.,
                    [100., 80.],
                )?,
            )
        })
        .unwrap();
    let before = owner.gui_command_state().unwrap().drawing_projections[&drawing].clone();
    assert!(before.error.is_none(), "{:?}", before.error);
    assert!(!before.lines.is_empty());
    owner
        .files
        .commands(part)
        .unwrap()
        .run("Change depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let after = owner.gui_command_state().unwrap().drawing_projections[&drawing].clone();
    assert!(after.error.is_none(), "{:?}", after.error);
    assert_ne!(before.lines, after.lines);
    owner.files.commands(part).unwrap().undo().unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().drawing_projections[&drawing].lines,
        before.lines
    );
    owner
        .files
        .commands(drawing)
        .unwrap()
        .run("Break reference", |snapshot, _| {
            snapshot
                .drawing
                .as_mut()
                .unwrap()
                .set_model_reference("missing.opd")
        })
        .unwrap();
    let missing = &owner.gui_command_state().unwrap().drawing_projections[&drawing];
    assert!(missing.lines.is_empty());
    assert!(missing.error.as_ref().unwrap().contains("missing.opd"));
}
