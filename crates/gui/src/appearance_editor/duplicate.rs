// SPDX-License-Identifier: GPL-2.0-only
//! Queue independent appearance copies and select them after the Part reports success.
use super::{AppearanceCopy, AppearanceEditor, EditorExit, GuiAppearanceChange, GuiAppearanceEdit};
use crate::{GuiAssetChoice, GuiShellAction};
use imgui::Ui;

pub(super) fn menu(ui: &Ui, can_duplicate: bool) -> EditorExit {
    ui.same_line();
    let _disabled = ui.begin_disabled(!can_duplicate);
    if ui.button("Create appearance") {
        ui.open_popup("##editor-create-appearance");
    }
    let Some(_popup) = ui.begin_popup("##editor-create-appearance") else {
        return EditorExit::KeepOpen;
    };
    if ui.menu_item("Duplicate") {
        EditorExit::Duplicate(AppearanceCopy::SameType)
    } else if ui.menu_item("Duplicate as Generic") {
        EditorExit::Duplicate(AppearanceCopy::Generic)
    } else {
        EditorExit::KeepOpen
    }
}

impl AppearanceEditor {
    pub(super) fn queue_duplicate(
        &mut self,
        asset: &GuiAssetChoice,
        assets: &[GuiAssetChoice],
        document: u64,
        kind: AppearanceCopy,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (id, name) = duplicate_identity(asset, assets);
        self.pending_duplicate = Some(id.clone());
        actions.push(GuiShellAction::EditDocumentAppearance(GuiAppearanceEdit {
            document,
            change: GuiAppearanceChange::Duplicate {
                source: asset.id.clone(),
                id,
                name,
                kind,
            },
        }));
    }

    pub(super) fn accept_duplicate(
        &mut self,
        document: u64,
        source: String,
        assets: &[GuiAssetChoice],
        selection: &mut Option<String>,
    ) -> String {
        let Some(id) = self.pending_duplicate.take().filter(|id| {
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
