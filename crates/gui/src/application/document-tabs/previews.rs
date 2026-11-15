// SPDX-License-Identifier: GPL-2.0-only
//! The picture and label shown above a hovered tab, kept from each window's last frame.
use super::preview_placement::preview_placement;
use crate::GuiPreviewImage;
use imgui::Ui;
use std::collections::{BTreeMap, BTreeSet};

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
/// One small picture per window, replaced each time that window is presented.
#[derive(Default)]
pub(crate) struct TabPreviews {
    images: BTreeMap<PreviewKey, GuiPreviewImage>,
    /// The document in front, whose picture its next save stores.
    pub showing_document: Option<u64>,
    /// The picture its save should store, until the owner has been told.
    document_picture: Option<(u64, PreviewKey, GuiPreviewImage)>,
    /// The last one told: an unchanged picture must not wake the owner on each frame.
    told_picture: Option<(u64, GuiPreviewImage)>,
    /// What is in front, and where, as of the last drawn frame.
    pub showing: Option<(PreviewKey, [[f32; 2]; 2])>,
    revisions: BTreeMap<u64, u64>,
    preview_visible: bool,
    deferred_preview: bool,
    // #3989: multiple draws can precede presentation; only a captured frame clears the guard.
    captured_front: Option<PreviewKey>,
}
impl TabPreviews {
    /// Keep what the workspace shows now; the area is in window pixels.
    pub(crate) fn note_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        // A preview below the strip is inside the workspace being captured.
        if self.preview_visible {
            return;
        }
        let Some((key, [minimum, maximum])) = self.showing else {
            return;
        };
        if let Some(image) = cropped_picture(extent, rgba, minimum, maximum) {
            self.captured_front = Some(key);
            if let Some(document) = self.showing_document {
                self.document_picture = Some((document, key, image.clone()));
            }
            self.images.insert(key, image);
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
        let revisions = identity_revisions(state);
        let collapsed = changed_window_identities(&self.revisions, &revisions);
        self.images.retain(|key, _| match key {
            PreviewKey::Home => true,
            PreviewKey::Window(id) => !collapsed.contains(id) && has_window(&current, *id),
        });
        if matches!(self.captured_front, Some(PreviewKey::Window(id)) if collapsed.contains(&id) || !has_window(&current, id))
        {
            self.captured_front = None;
        }
        self.forget_closed_picture(&current, &collapsed);
        self.forget_closed_front(&current, &collapsed);
        self.revisions = revisions;
    }
    fn forget_closed_picture(
        &mut self,
        current: &BTreeMap<u64, BTreeSet<u64>>,
        collapsed: &BTreeSet<u64>,
    ) {
        if self
            .document_picture
            .as_ref()
            .is_some_and(|(doc, key, _)| !picture_window_open(current, collapsed, *doc, *key))
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
    fn forget_closed_front(
        &mut self,
        current: &BTreeMap<u64, BTreeSet<u64>>,
        collapsed: &BTreeSet<u64>,
    ) {
        if let Some((key, _)) = self.showing
            && self
                .showing_document
                .is_some_and(|doc| !picture_window_open(current, collapsed, doc, key))
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
        ui.window("##tab-preview")
            .position(placement.position, imgui::Condition::Always)
            .size(placement.size, imgui::Condition::Always)
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
    fn picture(&self, ui: &Ui, key: PreviewKey, size: [f32; 2]) {
        let [x, y] = ui.cursor_screen_pos();
        let cell_size = [size[0] / CELLS[0] as f32, size[1] / CELLS[1] as f32];
        let list = ui.get_window_draw_list();
        if let Some(image) = self.images.get(&key) {
            for (index, color) in image.rgb.iter().enumerate() {
                let cell = [
                    x + (index as u32 % CELLS[0]) as f32 * cell_size[0],
                    y + (index as u32 / CELLS[0]) as f32 * cell_size[1],
                ];
                let rgba = color.map(|channel| f32::from(channel) / 255.);
                list.add_rect(
                    cell,
                    [cell[0] + cell_size[0], cell[1] + cell_size[1]],
                    [rgba[0], rgba[1], rgba[2], 1.],
                )
                .filled(true)
                .build();
            }
        }
        drop(list);
        ui.dummy(size);
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
fn identity_revisions(state: &crate::GuiCommandState) -> BTreeMap<u64, u64> {
    state
        .documents
        .iter()
        .map(|document| {
            (
                document.id,
                state
                    .document_window_identity_revisions
                    .get(&document.id)
                    .copied()
                    .unwrap_or(0),
            )
        })
        .collect()
}
fn changed_window_identities(
    before: &BTreeMap<u64, u64>,
    after: &BTreeMap<u64, u64>,
) -> BTreeSet<u64> {
    before
        .iter()
        .filter_map(|(document, revision)| {
            after
                .get(document)
                .filter(|current| *current != revision)
                .map(|_| *document)
        })
        .collect()
}
fn has_window(windows: &BTreeMap<u64, BTreeSet<u64>>, id: u64) -> bool {
    windows.values().any(|ids| ids.contains(&id))
}
fn picture_window_open(
    windows: &BTreeMap<u64, BTreeSet<u64>>,
    collapsed: &BTreeSet<u64>,
    document: u64,
    key: PreviewKey,
) -> bool {
    match key {
        PreviewKey::Window(id) => {
            !collapsed.contains(&id) && windows.get(&document).is_some_and(|ids| ids.contains(&id))
        }
        PreviewKey::Home => false,
    }
}
/// Nearest cell centres inside the area: a tab preview shows a window's look, not its detail.
fn cropped_picture(
    extent: [u32; 2],
    rgba: &[u8],
    minimum: [f32; 2],
    maximum: [f32; 2],
) -> Option<GuiPreviewImage> {
    let [width, height] = extent.map(|value| value as usize);
    let left = minimum[0].max(0.) as usize;
    let top = minimum[1].max(0.) as usize;
    let right = (maximum[0] as usize).min(width);
    let bottom = (maximum[1] as usize).min(height);
    if right <= left || bottom <= top || rgba.len() < width * height * 4 {
        return None;
    }
    let mut cells = Vec::with_capacity((CELLS[0] * CELLS[1]) as usize);
    for row in 0..CELLS[1] as usize {
        for column in 0..CELLS[0] as usize {
            let x = left + (2 * column + 1) * (right - left) / (2 * CELLS[0] as usize);
            let y = top + (2 * row + 1) * (bottom - top) / (2 * CELLS[1] as usize);
            let at = (y * width + x) * 4;
            cells.push([rgba[at], rgba[at + 1], rgba[at + 2]]);
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
