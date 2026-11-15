// SPDX-License-Identifier: GPL-2.0-only
//! The picture and label shown above a hovered tab, kept from each window's last frame.
use crate::GuiPreviewImage;
use imgui::Ui;
use std::collections::{BTreeMap, BTreeSet};

/// What a hovered tab shows: the Home workspace, or one window of a document.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) enum PreviewKey {
    Home,
    Window(u64),
}
/// The tab under the pointer in this frame.
#[derive(Clone, Debug, PartialEq)]
pub(crate) struct HoveredTab {
    pub key: PreviewKey,
    pub corner: [f32; 2],
    pub labels: Vec<String>,
}
const CELLS: [u32; 2] = [64, 36];
const CELL_SIZE: f32 = 3.;
const LABEL_ROW: f32 = 18.;
const CHROME: f32 = 16.;
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
}
impl TabPreviews {
    /// Keep what the workspace shows now; the area is in window pixels.
    pub(crate) fn note_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        let Some((key, [minimum, maximum])) = self.showing else {
            return;
        };
        if let Some(image) = cropped_picture(extent, rgba, minimum, maximum) {
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
        let current = open_windows(state);
        let revisions = identity_revisions(state);
        let collapsed = changed_window_identities(&self.revisions, &revisions);
        self.images.retain(|key, _| match key {
            PreviewKey::Home => true,
            PreviewKey::Window(id) => !collapsed.contains(id) && has_window(&current, *id),
        });
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
    /// Draw above the hovered tab; it takes no input, so moving away simply ends it.
    pub(crate) fn panel(&self, ui: &Ui, hovered: Option<&HoveredTab>) {
        let Some(tab) = hovered else {
            return;
        };
        let height = CELLS[1] as f32 * CELL_SIZE + tab.labels.len() as f32 * LABEL_ROW + CHROME;
        let position = [tab.corner[0], (tab.corner[1] - height - 4.).max(0.)];
        ui.window("##tab-preview")
            .position(position, imgui::Condition::Always)
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::ALWAYS_AUTO_RESIZE
                    | imgui::WindowFlags::NO_INPUTS
                    | imgui::WindowFlags::NO_SAVED_SETTINGS
                    | imgui::WindowFlags::NO_FOCUS_ON_APPEARING
                    | imgui::WindowFlags::NO_NAV,
            )
            .build(|| {
                for label in &tab.labels {
                    ui.text(label);
                }
                self.picture(ui, tab.key);
            });
    }
    fn picture(&self, ui: &Ui, key: PreviewKey) {
        let [x, y] = ui.cursor_screen_pos();
        let size = [CELLS[0] as f32 * CELL_SIZE, CELLS[1] as f32 * CELL_SIZE];
        let list = ui.get_window_draw_list();
        if let Some(image) = self.images.get(&key) {
            for (index, color) in image.rgb.iter().enumerate() {
                let cell = [
                    x + (index as u32 % CELLS[0]) as f32 * CELL_SIZE,
                    y + (index as u32 / CELLS[0]) as f32 * CELL_SIZE,
                ];
                let rgba = color.map(|channel| f32::from(channel) / 255.);
                list.add_rect(
                    cell,
                    [cell[0] + CELL_SIZE, cell[1] + CELL_SIZE],
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
