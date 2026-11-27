// SPDX-License-Identifier: GPL-2.0-only
//! Create or copy an appearance from the editor and select it after the Part reports success.
use super::{
    AppearanceCopy, AppearanceEditor, AppearanceTemplate, EditorExit, GuiAppearanceChange,
    GuiAppearanceEdit, available_identity,
};
use crate::{GuiAssetChoice, GuiShellAction};
use imgui::Ui;

#[derive(Clone, Copy, PartialEq, Eq)]
pub(super) enum EditorCreation {
    Template(AppearanceTemplate),
    Copy(AppearanceCopy),
}

pub(super) fn template_choice(ui: &Ui) -> Option<AppearanceTemplate> {
    [
        ("New Generic Appearance", AppearanceTemplate::Generic),
        ("Ceramic", AppearanceTemplate::Ceramic),
        ("Concrete", AppearanceTemplate::Concrete),
        ("Metal", AppearanceTemplate::Metal),
    ]
    .into_iter()
    .find_map(|(label, template)| ui.menu_item(label).then_some(template))
}

pub(super) fn menu(ui: &Ui, can_create: bool) -> EditorExit {
    ui.same_line();
    let _disabled = ui.begin_disabled(!can_create);
    if ui.button("Create appearance") {
        ui.open_popup("##editor-create-appearance");
    }
    let Some(_popup) = ui.begin_popup("##editor-create-appearance") else {
        return EditorExit::KeepOpen;
    };
    if let Some(template) = template_choice(ui) {
        return EditorExit::Create(EditorCreation::Template(template));
    }
    ui.separator();
    if ui.menu_item("Duplicate") {
        EditorExit::Create(EditorCreation::Copy(AppearanceCopy::SameType))
    } else if ui.menu_item("Duplicate as Generic") {
        EditorExit::Create(EditorCreation::Copy(AppearanceCopy::Generic))
    } else {
        EditorExit::KeepOpen
    }
}

impl AppearanceEditor {
    pub(super) fn queue_creation(
        &mut self,
        asset: &GuiAssetChoice,
        assets: &[GuiAssetChoice],
        document: u64,
        action: EditorCreation,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (id, change) = creation_action(asset, assets, action);
        self.pending_creation = Some(id);
        actions.push(GuiShellAction::EditDocumentAppearance(GuiAppearanceEdit {
            document,
            change,
        }));
    }

    pub(super) fn accept_creation(
        &mut self,
        document: u64,
        source: String,
        assets: &[GuiAssetChoice],
        selection: &mut Option<String>,
    ) -> String {
        let Some(id) = self.pending_creation.take().filter(|id| {
            assets.iter().any(|asset| {
                asset.id == *id
                    && asset
                        .appearance_preview
                        .is_some_and(|finish| finish.document_owned)
            })
        }) else {
            return source;
        };
        self.target = Some((document, id.clone()));
        self.draft = None;
        self.texture_open = false;
        self.image_chooser = Default::default();
        *selection = Some(id.clone());
        id
    }
}

fn creation_action(
    asset: &GuiAssetChoice,
    assets: &[GuiAssetChoice],
    action: EditorCreation,
) -> (String, GuiAppearanceChange) {
    match action {
        EditorCreation::Template(template) => {
            let (id, name) = available_identity(assets, template);
            (
                id.clone(),
                GuiAppearanceChange::Create { id, name, template },
            )
        }
        EditorCreation::Copy(kind) => {
            let (id, name) = duplicate_identity(asset, assets);
            (
                id.clone(),
                GuiAppearanceChange::Duplicate {
                    source: asset.id.clone(),
                    id,
                    name,
                    kind,
                },
            )
        }
    }
}

fn duplicate_identity(source: &GuiAssetChoice, assets: &[GuiAssetChoice]) -> (String, String) {
    let base_name = source
        .name
        .rsplit_once('(')
        .filter(|(_, suffix)| {
            suffix
                .strip_suffix(')')
                .is_some_and(|n| n.parse::<u32>().is_ok())
        })
        .map_or(source.name.as_str(), |(base, _)| base);
    let id_base = source
        .id
        .rsplit_once("-copy-")
        .filter(|(_, suffix)| suffix.parse::<u32>().is_ok())
        .map_or(source.id.as_str(), |(base, _)| base);
    let mut index = 1;
    loop {
        let id = format!("{id_base}-copy-{index}");
        let name = format!("{base_name}({index})");
        if !assets
            .iter()
            .any(|asset| asset.id == id || asset.name == name)
        {
            return (id, name);
        }
        index += 1;
    }
}
