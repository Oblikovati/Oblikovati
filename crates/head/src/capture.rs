// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::ViewportCapture;
use std::{fs::File, path::Path};

/// Write one complete RGBA capture as PNG; example: `write_rgba_png(Path::new("cube.png"), &capture)?`.
pub(crate) fn write_rgba_png(path: &Path, capture: &ViewportCapture) -> Result<(), String> {
    let file = File::create(path).map_err(|cause| capture_error(path, cause))?;
    let mut encoder = png::Encoder::new(file, capture.extent[0], capture.extent[1]);
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    let mut writer = encoder
        .write_header()
        .map_err(|cause| capture_error(path, cause))?;
    writer
        .write_image_data(&capture.rgba)
        .map_err(|cause| capture_error(path, cause))
}

fn capture_error(path: &Path, cause: impl std::fmt::Display) -> String {
    format!("window capture path {path:?}: expected a writable PNG destination: {cause}")
}
