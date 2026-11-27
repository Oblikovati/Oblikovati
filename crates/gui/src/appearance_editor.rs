// SPDX-License-Identifier: GPL-2.0-only
//! Edit a document appearance draft and apply it through the owning Part.
use crate::application::model_browser::color_dialog::{self, ColorAnswer, ColorDialog};
use crate::dockable_lifecycle::{DockRegistry, DockableState};
use crate::dockable_window::DockableWindow;
use crate::{GuiAssetChoice, GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_model::assets::AppearanceMetadata;

mod ceramic;
mod concrete;
mod creation;
mod image_chooser;
mod metal;
mod optional_effects;
mod preview_options;
mod texture;
mod transparency;
pub(crate) use texture::CheckerSampler;

/// One document appearance edit; example: `GuiAppearanceChange::Create { id, name, template }`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiAppearanceEdit {
    pub document: u64,
    pub change: GuiAppearanceChange,
}

/// Create, duplicate, or update a document appearance.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiAppearanceChange {
    Create {
        id: String,
        name: String,
        template: AppearanceTemplate,
    },
    Duplicate {
        source: String,
        id: String,
        name: String,
        kind: AppearanceCopy,
    },
    Apply {
        id: String,
        name: String,
        color: Option<[u8; 3]>,
        roughness: Option<u8>,
        highlights: Option<u8>,
        metadata: Option<Box<AppearanceMetadata>>,
    },
}

/// Keep the source property set or convert its surface to Generic controls.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AppearanceCopy {
    SameType,
    Generic,
}

/// Select a document appearance's initial property set when creating it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AppearanceTemplate {
    Generic,
    Ceramic,
    Concrete,
    Metal,
}

impl AppearanceTemplate {
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Generic => "Generic",
            Self::Ceramic => "Ceramic",
            Self::Concrete => "Concrete",
            Self::Metal => "Metal",
        }
    }

    pub(crate) fn from_metadata(metadata: &AppearanceMetadata) -> Self {
        if metadata.metal.is_some() {
            Self::Metal
        } else if metadata.concrete.is_some() {
            Self::Concrete
        } else if metadata.ceramic.is_some() {
            Self::Ceramic
        } else {
            Self::Generic
        }
    }

    fn id_suffix(self) -> &'static str {
        match self {
            Self::Generic => "generic",
            Self::Ceramic => "ceramic",
            Self::Concrete => "concrete",
            Self::Metal => "metal",
        }
    }
}

#[derive(Default)]
pub(crate) struct AppearanceEditor {
    dock: DockableState,
    target: Option<(u64, String)>,
    draft: Option<AppearanceDraft>,
    color_dialog: ColorDialog,
    image_chooser: image_chooser::ImageChooser,
    preview_options: preview_options::PreviewOptions,
    confirm_close: bool,
    texture_open: bool,
    texture_window: texture::TextureWindow,
    close_after_apply: bool,
    pending_creation: Option<String>,
}
impl DockableWindow for AppearanceEditor {}

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
            metadata: (self.metadata != original.metadata).then(|| Box::new(self.metadata.clone())),
        }
    }
}

impl AppearanceEditor {
    pub(crate) fn register_dockables(&self, registry: &mut DockRegistry) -> Result<(), String> {
        registry.register::<Self>("###appearance-editor", &self.dock)?;
        self.texture_window.register_dockable(registry)
    }

    pub(crate) fn sync_docking(&self, state: &GuiCommandState, home_in_front: bool) {
        let visible = !home_in_front
            && self.target.as_ref().is_some_and(|(document, _)| {
                state.document == Some(*document)
                    && state.authoring == crate::GuiAuthoringContext::Part
            });
        self.dock.set_open(visible);
        self.texture_window.set_open(visible && self.texture_open);
    }
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
        ui.separator();
        let _disabled = ui.begin_disabled(state.transaction_open || self.target.is_some());
        let avail = ui.content_region_avail()[0];
        let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([6., 4.]));
        let inline = avail >= ui.calc_text_size("Create appearanceEdit appearance")[0] + 30.;
        let half = if inline { (avail - 6.) * 0.5 } else { avail };
        if crate::theme::secondary_button(ui, "Create appearance", [half, 0.]) {
            ui.open_popup("##new-appearance");
        }
        if inline {
            ui.same_line_with_spacing(0., 6.);
        }
        let editable = selected.as_deref().and_then(|id| {
            browser
                .appearances
                .iter()
                .find(|a| a.id == id && a.appearance_preview.is_some_and(|f| f.document_owned))
        });
        let read_only = ui.begin_disabled(editable.is_none());
        if crate::theme::secondary_button(ui, "Edit appearance", [half, 0.])
            && let Some(asset) = editable
        {
            self.target = Some((document, asset.id.clone()));
        }
        drop(read_only);
        let Some(_popup) = ui.begin_popup("##new-appearance") else {
            return;
        };
        let Some(template) = creation::template_choice(ui) else {
            return;
        };
        let (id, name) = available_identity(&browser.appearances, template);
        *selected = Some(id.clone());
        self.target = Some((document, id.clone()));
        actions.push(GuiShellAction::EditDocumentAppearance(GuiAppearanceEdit {
            document,
            change: GuiAppearanceChange::Create { id, name, template },
        }));
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        selected: &mut Option<String>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some((document, id)) = self.target.clone().filter(|(document, _)| {
            state.document == Some(*document) && state.authoring == crate::GuiAuthoringContext::Part
        }) else {
            return;
        };
        let Some(browser) = state.browser.as_ref() else {
            self.close();
            return;
        };
        let id = self.accept_creation(document, id, &browser.appearances, selected);
        let Some(asset) = browser
            .appearances
            .iter()
            .find(|asset| asset.id == id && asset.document)
        else {
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
        if let Some(action) = self.show_editor(
            ui,
            asset,
            &original,
            document,
            state.transaction_open,
            actions,
        ) {
            self.queue_creation(asset, &browser.appearances, document, action, actions);
        }
    }

    fn show_editor(
        &mut self,
        ui: &Ui,
        asset: &GuiAssetChoice,
        original: &AppearanceDraft,
        document: u64,
        read_only: bool,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<creation::EditorCreation> {
        let draft = self.draft.get_or_insert_with(|| original.clone());
        let title = format!("Appearance Editor — {}###appearance-editor", original.name);
        let mut open = true;
        let mut exit = EditorExit::KeepOpen;
        crate::widget_abi::window_geometry(
            ui,
            [ui.io().display_size[0] * 0.34, 173.],
            [405., 520.],
            Condition::FirstUseEver,
        );
        Self::dockable_window(ui, &title, &mut open).build(|| {
            self.preview_options.draw(ui, &preview_asset(asset, draft));
            ui.input_text("Name", &mut draft.name).build();
            let height = (ui.content_region_avail()[1] - ui.frame_height_with_spacing()).max(1.);
            ui.child_window("##appearance-properties")
                .size([0., height])
                .build(|| {
                    draw_properties(
                        ui,
                        draft,
                        &mut self.color_dialog,
                        &mut self.texture_open,
                        &mut self.image_chooser,
                    )
                });
            self.image_chooser.draw(ui, &mut draft.metadata);
            let changed = draft != original;
            let editable = changed && !draft.name.trim().is_empty() && !read_only;
            exit = editor_buttons(ui, editable);
            if exit == EditorExit::KeepOpen {
                exit = creation::menu(ui, !changed && !read_only);
            }
            if self.confirm_close {
                exit = confirm_exit(ui, editable);
            }
        });
        self.texture_window
            .draw(ui, &mut draft.metadata.checker, &mut self.texture_open);
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
            EditorExit::Create(action) => return Some(action),
            EditorExit::Discard => self.close(),
            EditorExit::CancelClose => self.confirm_close = false,
            EditorExit::KeepOpen => {}
        }
        None
    }

    fn close(&mut self) {
        self.dock.set_open(false);
        self.texture_window.set_open(false);
        self.target = None;
        self.draft = None;
        self.confirm_close = false;
        self.texture_open = false;
        self.image_chooser = Default::default();
        self.preview_options = Default::default();
        self.close_after_apply = false;
        self.pending_creation = None;
    }
}

fn available_identity(assets: &[GuiAssetChoice], template: AppearanceTemplate) -> (String, String) {
    let base_id = format!("default-{}", template.id_suffix());
    let base_name = format!("Default {}", template.label());
    let mut index = 1;
    loop {
        let (id, name) = if index == 1 {
            (base_id.clone(), base_name.clone())
        } else {
            (format!("{base_id}-{index}"), format!("{base_name} {index}"))
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

fn preview_asset(asset: &GuiAssetChoice, draft: &AppearanceDraft) -> GuiAssetChoice {
    let mut preview = asset.clone();
    preview.swatch = Some(draft.color);
    preview.appearance_metadata = Some(draft.metadata.clone());
    if let Some(finish) = &mut preview.appearance_preview {
        finish.roughness = draft.roughness;
        finish.highlights = draft.highlights;
        if let Some(metal) = &draft.metadata.metal {
            finish.anisotropy = metal.finish.anisotropy();
        }
    }
    preview
}

fn draw_properties(
    ui: &Ui,
    draft: &mut AppearanceDraft,
    color_dialog: &mut ColorDialog,
    texture_open: &mut bool,
    image_chooser: &mut image_chooser::ImageChooser,
) {
    let template = AppearanceTemplate::from_metadata(&draft.metadata);
    draw_information(ui, draft, template);
    match template {
        AppearanceTemplate::Ceramic => {
            return ceramic::draw(ui, draft, color_dialog, image_chooser);
        }
        AppearanceTemplate::Concrete => {
            return concrete::draw(ui, draft, color_dialog, image_chooser);
        }
        AppearanceTemplate::Metal => return metal::draw(ui, draft, image_chooser),
        AppearanceTemplate::Generic => (),
    }
    if let Some(_generic) = ui.tree_node_config("Generic").default_open(true).push() {
        base_color(ui, &mut draft.color, color_dialog);
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

fn draw_information(ui: &Ui, draft: &mut AppearanceDraft, template: AppearanceTemplate) {
    let Some(_information) = ui.tree_node_config("Information").push() else {
        return;
    };
    ui.input_text("Name##information", &mut draft.name).build();
    ui.input_text_multiline("Description", &mut draft.metadata.description, [0., 48.])
        .build();
    ui.input_text("Keywords", &mut draft.metadata.keywords)
        .build();
    ui.text(format!("Type: {}", template.label()));
    ui.text("Source: Document");
}

fn base_color(ui: &Ui, color: &mut [u8; 3], dialog: &mut ColorDialog) {
    let [r, g, b] = *color;
    if ui.button(format!("Color  {r}, {g}, {b}###editor-color")) {
        dialog.begin_with_color(Some(*color));
        ui.open_popup(color_dialog::TITLE);
    }
    if let Some(ColorAnswer::Chosen(chosen)) = dialog.panel(ui) {
        *color = chosen;
    }
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
    Create(creation::EditorCreation),
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
