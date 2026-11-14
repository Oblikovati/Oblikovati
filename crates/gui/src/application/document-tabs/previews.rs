// SPDX-License-Identifier: GPL-2.0-only
//! The picture and label shown above a hovered tab, kept from each window's last frame.
use crate::GuiPreviewImage;
use imgui::Ui;
use std::collections::BTreeMap;

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
    document_picture: Option<(u64, GuiPreviewImage)>,
    /// The last one told: an unchanged picture must not wake the owner on each frame.
    told_picture: Option<(u64, GuiPreviewImage)>,
    /// What is in front, and where, as of the last drawn frame.
    pub showing: Option<(PreviewKey, [[f32; 2]; 2])>,
}
impl TabPreviews {
    /// Keep what the workspace shows now; the area is in window pixels.
    pub(crate) fn note_frame(&mut self, extent: [u32; 2], rgba: &[u8]) {
        let Some((key, [minimum, maximum])) = self.showing else {
            return;
        };
        if let Some(image) = cropped_picture(extent, rgba, minimum, maximum) {
            if let Some(document) = self.showing_document {
                self.document_picture = Some((document, image.clone()));
            }
            self.images.insert(key, image);
        }
    }
    /// A changed picture of the front document, once; example: each frame, for the owner.
    pub(crate) fn take_changed_picture(&mut self) -> Option<(u64, GuiPreviewImage)> {
        let picture = self.document_picture.take()?;
        if self.told_picture.as_ref() == Some(&picture) {
            return None;
        }
        self.told_picture = Some(picture.clone());
        Some(picture)
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
