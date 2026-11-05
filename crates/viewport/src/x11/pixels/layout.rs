// SPDX-License-Identifier: GPL-2.0-only
use super::*;
// X11 image byte order and scanline padding come from server setup, independent of request byte order.
pub(super) struct NativeImageLayout {
    pixel_bytes: usize,
    row_bytes: usize,
    order: ImageOrder,
    masks: [u64; 3],
    width: u16,
}
impl NativeImageLayout {
    pub(super) fn from_setup(
        setup: &Setup,
        image: &GetImageReply,
        width: u16,
    ) -> Result<Self, String> {
        let format = matching_pixmap_format(setup, image.depth)?;
        let visual = find_visual(setup, image.visual, image.depth)?;
        let (pixel_bytes, row_bytes) = storage_dimensions(format, width)?;
        let masks = native_image_masks(visual, format.depth)?;
        require_image_order(setup.image_byte_order)?;
        Ok(Self {
            pixel_bytes,
            row_bytes,
            order: setup.image_byte_order,
            masks,
            width,
        })
    }
    pub(super) fn rgba(&self, bytes: &[u8], extent: [u16; 2]) -> Result<Vec<u8>, String> {
        if extent[0] != self.width || extent[1] == 0 {
            return Err(format!(
                "native capture extent {extent:?}: expected width {} and positive height",
                self.width
            ));
        }
        let (length, padded) = expected_image_bytes(self.row_bytes, extent[1])?;
        if bytes.len() != padded {
            return Err(format!(
                "native capture bytes {}, extent {extent:?}, stride {}: expected {} padded bytes",
                bytes.len(),
                self.row_bytes,
                padded
            ));
        }
        let mut rgba = super::super::rgba_buffer(i32::from(extent[0]), i32::from(extent[1]))?;
        for row in bytes[..length].chunks_exact(self.row_bytes) {
            self.append_row(&mut rgba, row, extent[0]);
        }
        Ok(rgba)
    }
    fn append_row(&self, rgba: &mut Vec<u8>, row: &[u8], width: u16) {
        let pixels = &row[..usize::from(width) * self.pixel_bytes];
        for pixel in pixels.chunks_exact(self.pixel_bytes) {
            let value = pixel_value(pixel, self.order);
            rgba.extend(
                self.masks
                    .map(|mask| super::super::color_channel(value, mask)),
            );
            rgba.push(255);
        }
    }
}
fn matching_pixmap_format(setup: &Setup, depth: u8) -> Result<&Format, String> {
    setup
        .pixmap_formats
        .iter()
        .find(|format| format.depth == depth)
        .ok_or_else(|| format!("native capture depth {depth}: expected a server pixmap format"))
}
fn find_visual(setup: &Setup, id: Visualid, depth: u8) -> Result<&Visualtype, String> {
    let visual = setup
        .roots
        .iter()
        .flat_map(|screen| &screen.allowed_depths)
        .filter(|description| description.depth == depth)
        .flat_map(|depth| &depth.visuals)
        .find(|visual| visual.visual_id == id)
        .ok_or_else(|| {
            format!("native capture visual {id}, depth {depth}: expected a matching server visual")
        })?;
    if visual.class != VisualClass::TRUE_COLOR {
        return Err(format!(
            "native capture visual {id}, class {:?}: expected TrueColor RGB channels",
            visual.class
        ));
    }
    Ok(visual)
}
fn storage_dimensions(format: &Format, width: u16) -> Result<(usize, usize), String> {
    if width == 0
        || !matches!(format.bits_per_pixel, 8 | 16 | 24 | 32)
        || !matches!(format.scanline_pad, 8 | 16 | 32)
        || format.depth > format.bits_per_pixel
    {
        return Err(format!(
            "native capture width {width}, format {format:?}: expected positive width, byte-addressable pixels and 8/16/32-bit scanline padding"
        ));
    }
    let pixel_bytes = usize::from(format.bits_per_pixel / 8);
    let padding = usize::from(format.scanline_pad / 8);
    let row_bytes = (usize::from(width) * pixel_bytes).div_ceil(padding) * padding;
    Ok((pixel_bytes, row_bytes))
}
fn native_image_masks(visual: &Visualtype, bits: u8) -> Result<[u64; 3], String> {
    let masks = [visual.red_mask, visual.green_mask, visual.blue_mask].map(u64::from);
    super::super::validate_masks(masks)?;
    let maximum = (1_u64 << bits) - 1;
    if masks.iter().any(|mask| *mask > maximum) {
        return Err(format!(
            "native capture masks {masks:?}: expected channels within {bits}-bit pixels"
        ));
    }
    Ok(masks)
}
fn expected_image_bytes(row_bytes: usize, height: u16) -> Result<(usize, usize), String> {
    let length = row_bytes.checked_mul(usize::from(height)).ok_or_else(|| {
        format!(
            "native capture stride {row_bytes}, height {height}: expected representable image size"
        )
    })?;
    let padded = length.div_ceil(4).checked_mul(4).ok_or_else(|| {
        format!("native capture length {length}: expected representable padded image size")
    })?;
    Ok((length, padded))
}
fn require_image_order(order: ImageOrder) -> Result<(), String> {
    if order != ImageOrder::LSB_FIRST && order != ImageOrder::MSB_FIRST {
        return Err(format!(
            "native capture image order {order:?}: expected LSBFirst or MSBFirst"
        ));
    }
    Ok(())
}
fn pixel_value(bytes: &[u8], order: ImageOrder) -> u64 {
    if order == ImageOrder::MSB_FIRST {
        return bytes
            .iter()
            .fold(0, |value, byte| (value << 8) | u64::from(*byte));
    }
    bytes.iter().enumerate().fold(0, |value, (index, byte)| {
        value | (u64::from(*byte) << (index * 8))
    })
}
#[cfg(test)]
#[path = "../../../tests/native_pixel_layout.rs"]
mod tests;
