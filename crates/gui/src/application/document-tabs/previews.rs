// SPDX-License-Identifier: GPL-2.0-only
//! The picture and label shown above a hovered tab, kept from each window's last frame.
use super::preview_placement::preview_placement;
use crate::GuiPreviewImage;
use imgui::Ui;
use oblikovati_viewport::ViewportPreviewPixels;
use std::collections::{BTreeMap, BTreeSet};

#[path = "preview_texture.rs"]
mod preview_texture;
use preview_texture::PreviewTexture;
/// What a hovered tab shows: the Home workspace, or one window of a document.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum PreviewKey {
    Home,
    Window(u64),
}
/// Identify the workspace that the next frame will present.
/// Example: `front_key(&state, true)` selects Home.
pub(crate) fn front_key(state: &crate::GuiCommandState, home_active: bool) -> Option<PreviewKey> {
    if home_active {
        return Some(PreviewKey::Home);
    }
    state
        .document
        .map(|document| PreviewKey::Window(state.active_window.unwrap_or(document)))
}
/// The tab under the pointer in this frame.
#[derive(Clone, Debug, PartialEq)]
pub(crate) struct HoveredTab {
    pub key: PreviewKey,
    pub bounds: [[f32; 2]; 2],
    pub labels: Vec<String>,
}
const CELLS: [u32; 2] = [64, 36];
const CELL_SIZE: f32 = 3.;
const ASPECT: f32 = CELLS[0] as f32 / CELLS[1] as f32;
type PreviewRegion = (PreviewKey, [[f32; 2]; 2]);

#[derive(PartialEq)]
struct PreviewSample {
    generations: [u64; 4],
    region: PreviewRegion,
}

/// One small picture per window, updated from its presented frames.
#[derive(Default)]
pub(crate) struct TabPreviews {
    texture: PreviewTexture,
    /// The document in front, whose picture its next save stores.
    pub showing_document: Option<u64>,
    /// The picture its save should store, until the owner has been told.
    document_picture: Option<(u64, PreviewKey, GuiPreviewImage)>,
    /// The last one told: an unchanged picture must not wake the owner on each frame.
    told_picture: Option<(u64, GuiPreviewImage)>,
    /// What is in front, and where, as of the last drawn frame.
    pub showing: Option<PreviewRegion>,
    preview_visible: bool,
    deferred_preview: bool,
    // #3989: multiple draws can precede presentation; only a captured frame clears the guard.
    captured_front: Option<PreviewKey>,
    last_sample: Option<PreviewSample>,
}
impl TabPreviews {
    pub(crate) fn needs_frame_pixels(&self) -> bool {
        !self.preview_visible && self.showing.is_some()
    }
    pub(crate) fn needs_frame_pixels_at(&self, generations: Option<[u64; 4]>) -> bool {
        self.needs_frame_pixels()
            && generations
                .zip(self.showing)
                .is_none_or(|(generations, region)| {
                    self.last_sample.as_ref()
                        != Some(&PreviewSample {
                            generations,
                            region,
                        })
                })
    }

    pub(crate) fn note_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        self.note_frame_at(extent, rgba, None);
    }

    /// Keep what the workspace shows now; the area is in window pixels.
    pub(crate) fn note_frame_at(
        &mut self,
        extent: [u32; 2],
        rgba: &[u8],
        generations: Option<[u64; 4]>,
    ) {
        self.note_pixels_at(
            extent,
            ViewportPreviewPixels::Rgba(rgba.as_chunks::<4>().0),
            generations,
        );
    }

    pub(crate) fn note_pixels_at(
        &mut self,
        extent: [u32; 2],
        pixels: ViewportPreviewPixels<'_>,
        generations: Option<[u64; 4]>,
    ) {
        if !self.needs_frame_pixels() {
            return;
        }
        let Some(region @ (key, [minimum, maximum])) = self.showing else {
            return;
        };
        if let Some(image) = cropped_picture(extent, pixels, minimum, maximum) {
            self.captured_front = Some(key);
            self.last_sample = generations.map(|generations| PreviewSample {
                generations,
                region,
            });
            if let Some(document) = self.showing_document {
                self.document_picture = Some((document, key, image.clone()));
            }
            self.texture.note(key, extent, pixels, minimum, maximum);
        }
    }
    /// A changed picture of the front document, once; example: each frame, for the owner.
    pub(crate) fn take_changed_picture(&mut self) -> Option<(u64, GuiPreviewImage)> {
        let (document, _, image) = self.document_picture.take()?;
        let picture = (document, image);
        if self.told_picture.as_ref() == Some(&picture) {
            return None;
        }
        self.told_picture = Some(picture.clone());
        Some(picture)
    }
    /// Send a captured picture before a close consumes its source; example: `previews.enqueue_picture(&mut actions)`.
    pub(crate) fn enqueue_picture(&mut self, actions: &mut Vec<crate::GuiShellAction>) {
        let close = self.pending_close_position(actions);
        let Some((document, picture)) = self.take_changed_picture() else {
            return;
        };
        let action = crate::GuiShellAction::KeepDocumentPicture {
            document,
            picture: Some(picture),
        };
        actions.insert(close.unwrap_or(actions.len()), action);
    }
    fn pending_close_position(&self, actions: &[crate::GuiShellAction]) -> Option<usize> {
        let (document, key, _) = self.document_picture.as_ref()?;
        actions.iter().position(|action| match action {
            crate::GuiShellAction::CloseDocument(args) => args.id == *document,
            crate::GuiShellAction::CloseDocumentWindow(id) => *key == PreviewKey::Window(*id),
            _ => false,
        })
    }
    /// Drop closed views before sending pending pictures; example: `previews.synchronize(state)`.
    pub(crate) fn synchronize(&mut self, state: &crate::GuiCommandState) {
        self.preview_visible = false;
        self.deferred_preview = false;
        let current = open_windows(state);
        self.texture.retain(|key| match key {
            PreviewKey::Home => true,
            PreviewKey::Window(id) => has_window(&current, id),
        });
        if matches!(self.captured_front, Some(PreviewKey::Window(id)) if !has_window(&current, id))
        {
            self.captured_front = None;
            self.last_sample = None;
        }
        self.forget_closed_picture(&current);
        self.forget_closed_front(&current);
    }
    fn forget_closed_picture(&mut self, current: &BTreeMap<u64, BTreeSet<u64>>) {
        if self
            .document_picture
            .as_ref()
            .is_some_and(|(doc, key, _)| !picture_window_open(current, *doc, *key))
        {
            self.document_picture = None;
        }
        if self
            .told_picture
            .as_ref()
            .is_some_and(|(doc, _)| !current.contains_key(doc))
        {
            self.told_picture = None;
        }
    }
    fn forget_closed_front(&mut self, current: &BTreeMap<u64, BTreeSet<u64>>) {
        if let Some((key, _)) = self.showing
            && self
                .showing_document
                .is_some_and(|doc| !picture_window_open(current, doc, key))
        {
            self.showing = None;
            self.showing_document = None;
        }
    }
    /// Keep the preview in the viewport without covering the hovered tab.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        hovered: Option<&HoveredTab>,
        front: Option<PreviewKey>,
    ) {
        let Some(tab) = hovered else {
            return;
        };
        if self.needs_clean_front_capture(front) {
            self.deferred_preview = true;
            return;
        }
        let Some((placement, image_size)) = preview_dimensions(ui, tab) else {
            return;
        };
        self.preview_visible = true;
        crate::widget_abi::window_geometry(
            ui,
            placement.position,
            placement.size,
            imgui::Condition::Always,
        );
        ui.window("##tab-preview")
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::NO_INPUTS
                    | imgui::WindowFlags::NO_SAVED_SETTINGS
                    | imgui::WindowFlags::NO_FOCUS_ON_APPEARING
                    | imgui::WindowFlags::NO_SCROLLBAR
                    | imgui::WindowFlags::NO_NAV,
            )
            .build(|| {
                for label in &tab.labels {
                    ui.text_wrapped(label);
                }
                self.picture(ui, tab.key, image_size);
            });
    }
    fn needs_clean_front_capture(&self, front: Option<PreviewKey>) -> bool {
        front.is_some() && self.captured_front != front
    }
    /// Request the deferred preview frame after the clean capture.
    /// Example: `previews.needs_frame()` after drawing a changed front.
    pub(crate) fn needs_frame(&self) -> bool {
        self.deferred_preview
    }
    fn picture(&mut self, ui: &Ui, key: PreviewKey, size: [f32; 2]) {
        if self.texture.bind(key) {
            crate::widget_abi::image(ui, crate::GuiTextureKey::TAB_PREVIEW.raw(), size);
        } else {
            crate::widget_abi::dummy(ui, size);
        }
    }

    pub(crate) fn frame_resources(
        &mut self,
        base: &crate::GuiTextureResources,
    ) -> Result<crate::GuiTextureResources, String> {
        self.texture.resources(base)
    }
}
fn preview_dimensions(
    ui: &Ui,
    tab: &HoveredTab,
) -> Option<(super::preview_placement::PreviewPlacement, [f32; 2])> {
    let viewport = ui.io().display_size;
    let style = ui.clone_style();
    if viewport[0] < style.window_min_size[0] || viewport[1] < style.window_min_size[1] {
        return None;
    }
    let (width, chrome, image_width) = preview_metrics(ui, tab, viewport[0], &style)?;
    let desired = [
        width,
        (chrome + image_width / ASPECT).max(style.window_min_size[1]),
    ];
    let minimum = (chrome + CELLS[1] as f32).max(style.window_min_size[1]);
    let placement = preview_placement(viewport, tab.bounds, desired, minimum)?;
    let width = image_width.min((placement.size[1] - chrome) * ASPECT);
    Some((placement, [width, width / ASPECT]))
}

fn preview_metrics(
    ui: &Ui,
    tab: &HoveredTab,
    viewport_width: f32,
    style: &imgui::Style,
) -> Option<(f32, f32, f32)> {
    let width = preview_width(ui, &tab.labels, viewport_width, style);
    let inner_width = width - 2. * style.window_padding[0];
    // Keep at least one logical pixel for each source cell in the thumbnail.
    if inner_width < CELLS[0] as f32 {
        return None;
    }
    let label_height = wrapped_labels_height(ui, &tab.labels, inner_width, style.item_spacing[1]);
    Some((
        width,
        2. * style.window_padding[1] + label_height,
        (CELLS[0] as f32 * CELL_SIZE).min(inner_width),
    ))
}

fn preview_width(ui: &Ui, labels: &[String], viewport: f32, style: &imgui::Style) -> f32 {
    let widest = labels
        .iter()
        .map(|label| ui.calc_text_size(label)[0])
        .fold(0., f32::max);
    (widest.max(CELLS[0] as f32 * CELL_SIZE) + 2. * style.window_padding[0])
        .max(style.window_min_size[0])
        .min(viewport)
}

fn wrapped_labels_height(ui: &Ui, labels: &[String], width: f32, spacing: f32) -> f32 {
    labels
        .iter()
        .map(|label| ui.calc_text_size_with_opts(label, false, width)[1] + spacing)
        .sum()
}

fn open_windows(state: &crate::GuiCommandState) -> BTreeMap<u64, BTreeSet<u64>> {
    state
        .documents
        .iter()
        .map(|document| {
            let mut windows: BTreeSet<u64> = state
                .document_windows
                .iter()
                .filter(|window| window.document == document.id)
                .map(|window| window.id)
                .collect();
            if windows.is_empty() {
                windows.insert(document.id);
            }
            (document.id, windows)
        })
        .collect()
}
fn has_window(windows: &BTreeMap<u64, BTreeSet<u64>>, id: u64) -> bool {
    windows.values().any(|ids| ids.contains(&id))
}
fn picture_window_open(
    windows: &BTreeMap<u64, BTreeSet<u64>>,
    document: u64,
    key: PreviewKey,
) -> bool {
    match key {
        PreviewKey::Window(id) => windows.get(&document).is_some_and(|ids| ids.contains(&id)),
        PreviewKey::Home => false,
    }
}
/// Nearest cell centres inside the area: a tab preview shows a window's look, not its detail.
fn cropped_picture(
    extent: [u32; 2],
    pixels: ViewportPreviewPixels<'_>,
    minimum: [f32; 2],
    maximum: [f32; 2],
) -> Option<GuiPreviewImage> {
    let [width, height] = extent.map(|value| value as usize);
    let left = minimum[0].max(0.) as usize;
    let top = minimum[1].max(0.) as usize;
    let right = (maximum[0] as usize).min(width);
    let bottom = (maximum[1] as usize).min(height);
    let pixel_count = match pixels {
        ViewportPreviewPixels::Rgb(rgb) => rgb.len(),
        ViewportPreviewPixels::Rgba(rgba) => rgba.len(),
    };
    if right <= left || bottom <= top || pixel_count < width * height {
        return None;
    }
    let mut cells = Vec::with_capacity((CELLS[0] * CELLS[1]) as usize);
    for row in 0..CELLS[1] as usize {
        for column in 0..CELLS[0] as usize {
            let x = left + (2 * column + 1) * (right - left) / (2 * CELLS[0] as usize);
            let y = top + (2 * row + 1) * (bottom - top) / (2 * CELLS[1] as usize);
            let at = y * width + x;
            cells.push(match pixels {
                ViewportPreviewPixels::Rgb(rgb) => rgb[at],
                ViewportPreviewPixels::Rgba(rgba) => {
                    let [r, g, b, _] = rgba[at];
                    [r, g, b]
                }
            });
        }
    }
    Some(GuiPreviewImage {
        extent: CELLS,
        rgb: std::sync::Arc::new(cells),
    })
}

#[cfg(test)]
#[path = "../../../tests/application/document-tabs/previews.rs"]
mod tests;
