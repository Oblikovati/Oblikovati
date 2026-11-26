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
    let session = owner.files.workspace().session(part).unwrap();
    let published = session
        .owned_part_slot(Some(oblikovati_drawing::modelling_policy()))
        .unwrap();
    let initial = published.current(session).unwrap() as *const _;
    owner.gui_command_state().unwrap();
    let session = owner.files.workspace().session(part).unwrap();
    assert!(std::ptr::eq(
        session
            .owned_part_slot(Some(oblikovati_drawing::modelling_policy()))
            .unwrap()
            .current(session)
            .unwrap(),
        initial
    ));
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
    let session = owner.files.workspace().session(part).unwrap();
    assert!(
        session
            .owned_part_slot(Some(oblikovati_drawing::modelling_policy()))
            .unwrap()
            .current(session)
            .is_ok()
    );
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

#[test]
fn drawing_model_reference_api_uses_history_and_rejects_other_documents() {
    use oblikovati_api::wire::SetModelReferenceArgs;
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    let (mut owner, _) = owner();
    let request = SetModelReferenceArgs {
        full_document_name: "/parts/bracket.opd".into(),
    };
    assert!(
        Client::new(owner.document_host())
            .drawing()
            .set_model_reference(&request)
            .is_err()
    );
    owner.create_part_document().unwrap();
    assert!(
        Client::new(owner.document_host())
            .drawing()
            .set_model_reference(&request)
            .is_err()
    );
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let reply = Client::new(owner.document_host())
        .drawing()
        .set_model_reference(&request)
        .unwrap();
    assert_eq!(reply.model_reference, request.full_document_name);
    let reference = |owner: &TestOwner| {
        owner
            .documents()
            .workspace()
            .session(drawing)
            .unwrap()
            .snapshot()
            .drawing
            .as_ref()
            .unwrap()
            .model_reference()
            .to_owned()
    };
    assert_eq!(reference(&owner), request.full_document_name);
    owner.files.commands(drawing).unwrap().undo().unwrap();
    assert_eq!(reference(&owner), "");
    owner.files.commands(drawing).unwrap().redo().unwrap();
    assert_eq!(reference(&owner), request.full_document_name);
    let path = "/memory/model-reference.odd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let stored = owner
        .documents()
        .workspace()
        .session(reopened)
        .unwrap()
        .snapshot()
        .drawing
        .as_ref()
        .unwrap();
    assert_eq!(stored.model_reference(), request.full_document_name);
}

#[test]
fn drawing_base_view_api_validates_current_geometry_and_replays() {
    use oblikovati_api::wire::{AddBaseViewArgs, SetModelReferenceArgs};
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
    let (mut owner, _) = owner();
    let part = super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let args = AddBaseViewArgs {
        center_xmm: 100.,
        center_ymm: 80.,
        ..Default::default()
    };
    assert!(
        Client::new(owner.document_host())
            .drawing_views()
            .add_base(&args)
            .is_err()
    );
    let reference = std::env::temp_dir()
        .join("head-scene.opd")
        .to_string_lossy()
        .into_owned();
    Client::new(owner.document_host())
        .drawing()
        .set_model_reference(&SetModelReferenceArgs {
            full_document_name: reference,
        })
        .unwrap();
    let reply = Client::new(owner.document_host())
        .drawing_views()
        .add_base(&args)
        .unwrap();
    assert_eq!(
        (
            reply.view.name.as_str(),
            reply.view.orientation.as_str(),
            reply.view.scale
        ),
        ("VIEW:1", "front", 1.)
    );
    assert!(reply.view.visible_count > 0);
    assert_eq!(reply.view.label, "VIEW:1  SCALE 1:1");
    let state = owner.gui_command_state().unwrap();
    assert!(!state.drawing_projections[&drawing].lines.is_empty());
    let duplicate = AddBaseViewArgs {
        name: "VIEW:1".into(),
        ..args.clone()
    };
    assert!(
        Client::new(owner.document_host())
            .drawing_views()
            .add_base(&duplicate)
            .is_err()
    );
    owner.files.commands(drawing).unwrap().undo().unwrap();
    assert!(
        owner.gui_command_state().unwrap().drawing_documents[&drawing].sheets()[0]
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
    owner
        .files
        .commands(part)
        .unwrap()
        .run("Break extrusion", |snapshot, _| {
            snapshot.parameters.set_expression(3, "0 cm")
        })
        .unwrap();
    assert!(
        Client::new(owner.document_host())
            .drawing_views()
            .add_base(&args)
            .is_err()
    );
    assert_eq!(
        owner.gui_command_state().unwrap().drawing_documents[&drawing].sheets()[0]
            .views
            .len(),
        1
    );
    owner.files.commands(part).unwrap().undo().unwrap();
    let path = "/memory/base-view.odd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let state = owner.gui_command_state().unwrap();
    assert_eq!(
        state.drawing_documents[&reopened].sheets()[0].views[0].name,
        "VIEW:1"
    );
    assert!(!state.drawing_projections[&reopened].lines.is_empty());
    drawing_view_collection::verify(&Client::new(owner.document_host()), reply.view);
}

#[path = "owner/drawing_view.rs"]
mod drawing_view;

#[test]
fn builtin_part_recipe_material_matches_the_created_document() {
    use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateMaterial};
    let (mut owner, _) = owner();
    let template = GuiDocumentTemplate::default();
    let state = owner.gui_command_state().unwrap();
    let entry = state
        .template_materials
        .iter()
        .find(|row| row.template == template)
        .unwrap();
    assert_eq!(entry.material, GuiTemplateMaterial::Unassigned);
    owner.create_document_template(template).unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .browser
            .unwrap()
            .part_material,
        None
    );
}

struct UnavailableTemplateCatalog;
impl oblikovati_session::files::WorkspaceAssetCatalogSource for UnavailableTemplateCatalog {
    fn load_catalog(&self) -> Result<oblikovati_model::assets::BuiltinAssetCatalog, String> {
        Err("fixture catalog unavailable".into())
    }
}
#[test]
fn unassigned_template_metadata_does_not_require_a_catalog() {
    let files: FileWorkspace<DesktopObservations, DesktopObservations, MemoryDocumentFiles> =
        FileWorkspace::new_with_asset_catalog_source(
            DesktopObservations::default(),
            MemoryDocumentFiles::default(),
            Default::default(),
            Box::new(UnavailableTemplateCatalog),
        );
    let rows = document_creation::gui_template_materials(&files);
    assert_eq!(rows.len(), 4);
    assert!(
        rows.iter()
            .all(|row| row.material == oblikovati_gui::GuiTemplateMaterial::Unassigned)
    );
}

#[path = "drawing_replay.rs"]
mod drawing_replay;

#[path = "drawing_view_collection.rs"]
mod drawing_view_collection;

#[path = "drawing_sheet_collection.rs"]
mod drawing_sheet_collection;

#[path = "drawing_title_block.rs"]
mod drawing_title_block;

#[path = "drawing_border.rs"]
mod drawing_border;

#[path = "drawing_sheet_format.rs"]
mod drawing_sheet_format;

#[path = "drawing_styles.rs"]
mod drawing_styles;

#[path = "drawing_view_label.rs"]
mod drawing_view_label;
#[path = "drawing_view_rotation.rs"]
mod drawing_view_rotation;
