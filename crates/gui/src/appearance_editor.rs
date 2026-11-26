// SPDX-License-Identifier: GPL-2.0-only
//! Edit a document appearance draft and apply it through the owning Part.
use crate::application::model_browser::color_dialog::{self, ColorAnswer, ColorDialog};
use crate::{GuiAssetChoice, GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui, WindowFlags};
use oblikovati_model::assets::AppearanceMetadata;

mod optional_effects;
mod texture;
mod transparency;
pub(crate) use texture::CheckerSampler;

/// One document appearance edit; example: `GuiAppearanceChange::Create { id, name }`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiAppearanceEdit {
    pub document: u64,
    pub change: GuiAppearanceChange,
}

/// Create a document appearance or apply edits to an existing appearance.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiAppearanceChange {
    Create {
        id: String,
        name: String,
    },
    Apply {
        id: String,
        name: String,
        color: Option<[u8; 3]>,
        roughness: Option<u8>,
        highlights: Option<u8>,
        metadata: Option<AppearanceMetadata>,
    },
}

#[derive(Default)]
pub(crate) struct AppearanceEditor {
    target: Option<(u64, String)>,
    draft: Option<AppearanceDraft>,
    color_dialog: ColorDialog,
    confirm_close: bool,
    texture_open: bool,
    close_after_apply: bool,
}

#[derive(Clone, Debug, PartialEq, Eq)]
struct AppearanceDraft {
    name: String,
    color: [u8; 3],
    roughness: u8,
    highlights: u8,
    metadata: AppearanceMetadata,
}

impl AppearanceDraft {
    fn read(asset: &GuiAssetChoice) -> Option<Self> {
        let finish = asset.appearance_preview?;
        Some(Self {
            name: asset.name.clone(),
            color: asset.swatch?,
            roughness: finish.roughness,
            highlights: finish.highlights,
            metadata: asset.appearance_metadata.clone().unwrap_or_default(),
        })
    }

    fn edit(&self, id: &str, original: &Self) -> GuiAppearanceChange {
        GuiAppearanceChange::Apply {
            id: id.into(),
            name: self.name.trim().into(),
            color: (self.color != original.color).then_some(self.color),
            roughness: (self.roughness != original.roughness).then_some(self.roughness),
            highlights: (self.highlights != original.highlights).then_some(self.highlights),
            metadata: (self.metadata != original.metadata).then(|| self.metadata.clone()),
        }
    }
}

impl AppearanceEditor {
    pub(crate) fn create_menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        selected: &mut Option<String>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (Some(document), Some(browser)) = (state.document, &state.browser) else {
            return;
        };
        let _disabled = ui.begin_disabled(state.transaction_open || self.target.is_some());
        if ui.button("Create appearance") {
            ui.open_popup("##new-appearance");
        }
        ui.same_line();
        let editable = selected.as_deref().and_then(|id| {
            browser.appearances.iter().find(|asset| {
                asset.id == id
                    && asset
                        .appearance_preview
                        .is_some_and(|finish| finish.document_owned)
            })
        });
        let read_only = ui.begin_disabled(editable.is_none());
        if ui.button("Edit appearance")
            && let Some(asset) = editable
        {
            self.target = Some((document, asset.id.clone()));
        }
        drop(read_only);
        if let Some(_popup) = ui.begin_popup("##new-appearance")
            && ui.menu_item("New Generic Appearance")
        {
            let (id, name) = available_identity(&browser.appearances);
            *selected = Some(id.clone());
            self.target = Some((document, id.clone()));
            actions.push(GuiShellAction::EditDocumentAppearance(GuiAppearanceEdit {
                document,
                change: GuiAppearanceChange::Create { id, name },
            }));
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some((document, id)) = self.target.clone().filter(|(document, _)| {
            state.document == Some(*document) && state.authoring == crate::GuiAuthoringContext::Part
        }) else {
            return;
        };
        let Some(asset) = state.browser.as_ref().and_then(|browser| {
            browser
                .appearances
                .iter()
                .find(|asset| asset.id == id && asset.document)
        }) else {
            self.close();
            return;
        };
        let Some(original) = AppearanceDraft::read(asset) else {
            return;
        };
        if self.close_after_apply && self.draft.as_ref() == Some(&original) {
            self.close();
            return;
        }
        self.window(
            ui,
            asset,
            &original,
            document,
            state.transaction_open,
            actions,
        );
    }

    fn window(
        &mut self,
        ui: &Ui,
        asset: &GuiAssetChoice,
        original: &AppearanceDraft,
        document: u64,
        read_only: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let draft = self.draft.get_or_insert_with(|| original.clone());
        let title = format!("Appearance Editor — {}###appearance-editor", original.name);
        let mut open = true;
        let mut exit = EditorExit::KeepOpen;
        ui.window(&title)
            .size([405., 410.], Condition::FirstUseEver)
            .position(
                [ui.io().display_size[0] * 0.34, 173.],
                Condition::FirstUseEver,
            )
            .flags(WindowFlags::NO_DOCKING)
            .opened(&mut open)
            .build(|| {
                draw_preview(ui, asset, draft);
                ui.input_text("Name", &mut draft.name).build();
                let height =
                    (ui.content_region_avail()[1] - ui.frame_height_with_spacing()).max(1.);
                ui.child_window("##appearance-properties")
                    .size([0., height])
                    .build(|| {
                        draw_properties(ui, draft, &mut self.color_dialog, &mut self.texture_open)
                    });
                let changed = draft != original;
                let editable = changed && !draft.name.trim().is_empty() && !read_only;
                exit = editor_buttons(ui, editable);
                if self.confirm_close {
                    exit = confirm_exit(ui, editable);
                }
            });
        texture::window(ui, &mut draft.metadata.checker, &mut self.texture_open);
        if !open {
            if draft == original {
                exit = EditorExit::Discard;
            } else {
                self.confirm_close = true;
            }
        }
        match exit {
            EditorExit::Apply | EditorExit::SaveAndClose => {
                draft.name = draft.name.trim().into();
                actions.push(GuiShellAction::EditDocumentAppearance(GuiAppearanceEdit {
                    document,
                    change: draft.edit(&asset.id, original),
                }));
                self.close_after_apply = exit == EditorExit::SaveAndClose;
                self.confirm_close = false;
            }
            EditorExit::Discard => self.close(),
            EditorExit::CancelClose => self.confirm_close = false,
            EditorExit::KeepOpen => {}
        }
    }

    fn close(&mut self) {
        self.target = None;
        self.draft = None;
        self.confirm_close = false;
        self.texture_open = false;
        self.close_after_apply = false;
    }
}

fn available_identity(assets: &[GuiAssetChoice]) -> (String, String) {
    let mut index = 1;
    loop {
        let (id, name) = if index == 1 {
            ("default-generic".into(), "Default Generic".into())
        } else {
            (
                format!("default-generic-{index}"),
                format!("Default Generic {index}"),
            )
        };
        if !assets
            .iter()
            .any(|asset| asset.id == id || asset.name == name)
        {
            return (id, name);
        }
        index += 1;
    }
}

fn draw_preview(ui: &Ui, asset: &GuiAssetChoice, draft: &AppearanceDraft) {
    let mut preview = asset.clone();
    preview.swatch = Some(draft.color);
    let preview_metadata = preview
        .appearance_metadata
        .get_or_insert_with(Default::default);
    preview_metadata.reflectivity = draft.metadata.reflectivity;
    preview_metadata.transparency = draft.metadata.transparency;
    preview_metadata.checker = draft.metadata.checker;
    preview_metadata
        .generic_effects
        .clone_from(&draft.metadata.generic_effects);
    if let Some(finish) = &mut preview.appearance_preview {
        finish.roughness = draft.roughness;
        finish.highlights = draft.highlights;
    }
    let [left, top] = ui.cursor_screen_pos();
    let width = ui.content_region_avail()[0];
    ui.dummy([width, 116.]);
    crate::asset_browser_view::paint_appearance_preview(
        &ui.get_window_draw_list(),
        [left + width / 2., top + 55.],
        49.,
        &preview,
    );
}

fn draw_properties(
    ui: &Ui,
    draft: &mut AppearanceDraft,
    color_dialog: &mut ColorDialog,
    texture_open: &mut bool,
) {
    if let Some(_information) = ui.tree_node_config("Information").push() {
        ui.input_text("Name##information", &mut draft.name).build();
        ui.input_text_multiline("Description", &mut draft.metadata.description, [0., 48.])
            .build();
        ui.input_text("Keywords", &mut draft.metadata.keywords)
            .build();
        ui.text("Type: Generic");
        ui.text("Source: Document");
    }
    if let Some(_generic) = ui.tree_node_config("Generic").default_open(true).push() {
        let [r, g, b] = draft.color;
        if ui.button(format!("Color  {r}, {g}, {b}###editor-color")) {
            color_dialog.begin_with_color(Some(draft.color));
            ui.open_popup(color_dialog::TITLE);
        }
        if let Some(ColorAnswer::Chosen(color)) = color_dialog.panel(ui) {
            draft.color = color;
        }
        texture::slot(ui, &mut draft.metadata.checker, texture_open);
        let mut gloss = 255 - draft.roughness;
        ui.slider("Glossiness", 0, 255, &mut gloss);
        draft.roughness = 255 - gloss;
        ui.slider("Highlights", 0, 255, &mut draft.highlights);
    }
    draw_reflectivity(ui, &mut draft.metadata);
    transparency::draw(ui, &mut draft.metadata);
    optional_effects::draw(ui, &mut draft.metadata);
}

fn draw_reflectivity(ui: &Ui, metadata: &mut AppearanceMetadata) {
    let _id = ui.push_id("reflectivity");
    let original = metadata.reflectivity.unwrap_or_default();
    let mut effect = original;
    let expanded = ui.tree_node_config("Reflectivity").push();
    ui.same_line();
    ui.checkbox("Enabled", &mut effect.enabled);
    if expanded.is_some() {
        let _disabled = ui.begin_disabled(!effect.enabled);
        ui.slider_config("Direct", 0, 100)
            .display_format("%d%%")
            .build(&mut effect.direct);
        ui.slider_config("Oblique", 0, 100)
            .display_format("%d%%")
            .build(&mut effect.oblique);
    }
    if effect != original {
        metadata.reflectivity = (effect != Default::default()).then_some(effect);
    }
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum EditorExit {
    KeepOpen,
    Apply,
    SaveAndClose,
    Discard,
    CancelClose,
}

fn editor_buttons(ui: &Ui, editable: bool) -> EditorExit {
    let _disabled = ui.begin_disabled(!editable);
    if ui.button("OK") {
        return EditorExit::SaveAndClose;
    }
    ui.same_line();
    if ui.button("Apply") {
        return EditorExit::Apply;
    }
    EditorExit::KeepOpen
}

fn confirm_exit(ui: &Ui, editable: bool) -> EditorExit {
    ui.open_popup("Save Edits");
    let Some(_dialog) = ui.begin_modal_popup("Save Edits") else {
        return EditorExit::KeepOpen;
    };
    ui.text("Save changes to this appearance?");
    let disabled = ui.begin_disabled(!editable);
    let yes = ui.button("Yes");
    drop(disabled);
    ui.same_line();
    let no = ui.button("No");
    ui.same_line();
    let cancel = ui.button("Cancel");
    let choice = match (yes, no, cancel) {
        (true, _, _) => EditorExit::SaveAndClose,
        (_, true, _) => EditorExit::Discard,
        (_, _, true) => EditorExit::CancelClose,
        _ => return EditorExit::KeepOpen,
    };
    ui.close_current_popup();
    choice
}
