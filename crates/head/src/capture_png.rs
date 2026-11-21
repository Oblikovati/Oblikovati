// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::ViewportCapture;
use std::{io::Write, path::Path};

/// File publication boundary; image encoding completes before the path becomes visible.
pub trait WindowCaptureFiles {
    /// Publish a completed frame atomically; example: `files.write_capture(path, &capture)?`.
    fn write_capture(&mut self, path: &Path, capture: &ViewportCapture) -> Result<(), String>;
}
/// Native PNG encoding and atomic replacement in the requested output directory.
pub struct NativeWindowCaptureFiles;
impl WindowCaptureFiles for NativeWindowCaptureFiles {
    fn write_capture(&mut self, path: &Path, capture: &ViewportCapture) -> Result<(), String> {
        let bytes = encode_window_png(capture)?;
        publish_png_bytes(path, &bytes).map_err(|cause| {
            format!("window capture path {path:?}: expected an atomic PNG publication: {cause}")
        })
    }
}
fn publish_png_bytes(path: &Path, bytes: &[u8]) -> Result<(), std::io::Error> {
    let parent = path
        .parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or(Path::new("."));
    let mut temporary = tempfile::NamedTempFile::new_in(parent)?;
    temporary.write_all(bytes)?;
    temporary.flush()?;
    temporary.persist(path).map_err(|cause| cause.error)?;
    Ok(())
}
fn encode_window_png(capture: &ViewportCapture) -> Result<Vec<u8>, String> {
    validate_capture_bytes(capture)?;
    let mut bytes = Vec::new();
    let mut encoder = png::Encoder::new(&mut bytes, capture.extent[0], capture.extent[1]);
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    encoder.set_source_srgb(png::SrgbRenderingIntent::Perceptual);
    let mut writer = encoder.write_header().map_err(png_encoding_error)?;
    writer
        .write_image_data(&capture.rgba)
        .map_err(png_encoding_error)?;
    writer.finish().map_err(png_encoding_error)?;
    Ok(bytes)
}
fn validate_capture_bytes(capture: &ViewportCapture) -> Result<(), String> {
    let [width, height] = capture.extent;
    let expected = usize::try_from(width)
        .ok()
        .and_then(|width| {
            usize::try_from(height)
                .ok()
                .and_then(|height| width.checked_mul(height))
        })
        .and_then(|pixels| pixels.checked_mul(4));
    if width == 0
        || height == 0
        || width > i32::MAX as u32
        || height > i32::MAX as u32
        || expected != Some(capture.rgba.len())
    {
        return Err(format!(
            "window capture extent {:?}, {} RGBA bytes: expected a positive signed-32-bit extent and exactly width * height * 4 top-down bytes",
            capture.extent,
            capture.rgba.len()
        ));
    }
    Ok(())
}
fn png_encoding_error(cause: png::EncodingError) -> String {
    format!("window PNG encoding {cause}: expected a complete sRGB RGBA image")
}

#[cfg(test)]
#[path = "../tests/capture_png.rs"]
mod tests;
