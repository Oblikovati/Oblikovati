// SPDX-License-Identifier: GPL-2.0-only
//! Select and embed a sampled height image from the editor's shared file browser.
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field, file_type_field, full_path,
};
use crate::application::file_system::NativeFolders;
use image::{ImageFormat, ImageReader, Limits, imageops::FilterType};
use imgui::Ui;
use oblikovati_model::assets::{AppearanceHeightMap, AppearanceMetadata};
use std::io::{Cursor, Read};
use std::{fs::File, path::Path, rc::Rc};

const TITLE: &str = "Choose Image File";

#[derive(Clone, Copy)]
pub(super) enum HeightMapTarget {
    FinishBumps,
    Relief,
    ConcreteStamp,
    ConcreteWeathering,
    MetalRelief,
    MetalCutouts,
}

impl HeightMapTarget {
    fn assign(self, metadata: &mut AppearanceMetadata, map: AppearanceHeightMap) {
        let destination = match self {
            Self::FinishBumps => metadata
                .ceramic
                .as_mut()
                .map(|value| &mut value.custom_bump),
            Self::Relief => metadata
                .ceramic
                .as_mut()
                .map(|value| &mut value.relief_image),
            Self::ConcreteStamp => metadata
                .concrete
                .as_mut()
                .map(|value| &mut value.stamped_image),
            Self::ConcreteWeathering => metadata
                .concrete
                .as_mut()
                .map(|value| &mut value.weathering_image),
            Self::MetalRelief => metadata.metal.as_mut().map(|value| &mut value.relief_image),
            Self::MetalCutouts => metadata.metal.as_mut().map(|value| &mut value.cutout_image),
        };
        if let Some(destination) = destination {
            *destination = Some(map);
        }
    }
}

#[derive(Default)]
pub(super) struct ImageChooser {
    request: Option<HeightMapTarget>,
    draft: Option<ImageSelection>,
}

struct ImageSelection {
    target: HeightMapTarget,
    browser: FolderBrowser,
    file_name: String,
    escape: DialogEscape,
    error: Option<String>,
}

impl ImageChooser {
    pub(super) fn request(&mut self, target: HeightMapTarget) {
        self.request = Some(target);
    }

    pub(super) fn draw(&mut self, ui: &Ui, metadata: &mut AppearanceMetadata) {
        if let Some(target) = self.request.take() {
            self.draft = Some(ImageSelection::new(target));
            ui.open_popup(TITLE);
        }
        let Some(draft) = self.draft.as_mut() else {
            return;
        };
        center_dialog(ui, [800., 405.]);
        let Some(_popup) = ui.begin_modal_popup(TITLE) else {
            self.draft = None;
            return;
        };
        match draft.controls(ui) {
            Selection::Pending => return,
            Selection::Cancelled => (),
            Selection::Chosen(map) => draft.target.assign(metadata, map),
        }
        ui.close_current_popup();
        self.draft = None;
    }
}

enum Selection {
    Pending,
    Cancelled,
    Chosen(AppearanceHeightMap),
}

impl ImageSelection {
    fn new(target: HeightMapTarget) -> Self {
        let mut browser = FolderBrowser::new(Rc::new(NativeFolders));
        browser.refresh();
        Self {
            target,
            browser,
            file_name: String::new(),
            escape: DialogEscape::default(),
            error: None,
        }
    }

    fn controls(&mut self, ui: &Ui) -> Selection {
        let types = FileTypeList::appearance_images();
        if let Some(name) = self.browser.controls(ui, "Look in", types.filter(0)) {
            self.file_name = name;
            self.error = None;
        }
        let entered = file_name_field(ui, &mut self.file_name);
        file_type_field(ui, &types, &mut 0);
        if let Some(error) = &self.error {
            ui.text_wrapped(error);
        }
        match dialog_buttons(
            ui,
            "Open",
            !self.file_name.trim().is_empty(),
            entered,
            &mut self.escape,
        ) {
            Some(DialogEnd::Accepted) => self.accept(),
            Some(DialogEnd::Cancelled) => Selection::Cancelled,
            None => Selection::Pending,
        }
    }

    fn accept(&mut self) -> Selection {
        let path = full_path(&self.browser.folder, self.file_name.trim());
        match read_height_map(Path::new(&path)) {
            Ok(map) => Selection::Chosen(map),
            Err(error) => {
                self.error = Some(format!("Could not open image. {error}"));
                Selection::Pending
            }
        }
    }
}

fn read_height_map(path: &Path) -> Result<AppearanceHeightMap, String> {
    let name = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| "Choose an image with a valid file name.".to_owned())?;
    let decode_error = |error: image::ImageError| format!("{name}: {error}");
    let bytes = image_bytes(path).map_err(|error| format!("{name}: {error}"))?;
    let format = image::guess_format(&bytes).map_err(decode_error)?;
    if !matches!(
        format,
        ImageFormat::Png | ImageFormat::Jpeg | ImageFormat::Bmp
    ) {
        return Err(format!("{name}: Choose a PNG, JPEG, or BMP image."));
    }
    let mut reader = ImageReader::with_format(Cursor::new(bytes), format);
    let mut limits = Limits::default();
    limits.max_image_width = Some(4096);
    limits.max_image_height = Some(4096);
    limits.max_alloc = Some(128 * 1024 * 1024);
    reader.limits(limits);
    let pixels = reader.decode().map_err(decode_error)?.into_luma8();
    let side = AppearanceHeightMap::SIDE as u32;
    let samples = image::imageops::resize(&pixels, side, side, FilterType::Triangle);
    AppearanceHeightMap::new(name.into(), samples.into_raw())
}

fn image_bytes(path: &Path) -> std::io::Result<Vec<u8>> {
    const MAX_BYTES: u64 = 32 * 1024 * 1024;
    let mut bytes = Vec::new();
    let info = path.metadata()?;
    if !info.is_file() || info.len() > MAX_BYTES {
        return Err(std::io::Error::other(
            "Choose an image file of 32 MB or less.",
        ));
    }
    File::open(path)?
        .take(MAX_BYTES + 1)
        .read_to_end(&mut bytes)?;
    if bytes.len() as u64 > MAX_BYTES {
        return Err(std::io::Error::other(
            "Choose an image file of 32 MB or less.",
        ));
    }
    Ok(bytes)
}
