// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::{ncollection::Vec4, quantity::ColorRgba, standard::Handle};
use martensite_tkservice::{
    aspect::AspectTypeOfMarker,
    graphic3d::*,
    image::{ImageFormat, ImagePixMap},
};

/// How a sketch point reads: a center point is a cross, a regular point a dot.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum SketchPointGlyph {
    Dot,
    Cross,
}
impl SketchPointGlyph {
    pub(super) fn from_center_role(center: bool) -> Self {
        if center { Self::Cross } else { Self::Dot }
    }
}
/// A dot's sprite, in pixels: the kernel's own ball reads as a ring at this size.
const DOT_EXTENT: usize = 7;
const DOT_RADIUS: f64 = 3.2;
const CROSS_SCALE: f32 = 3.;
pub(super) fn theme_marker(
    rgba: ColorRgba,
    glyph: SketchPointGlyph,
) -> Result<Graphic3dAspectMarker3d, String> {
    let marker = Graphic3dAspectMarker3d::new(AspectTypeOfMarker::Userdefined, *rgba.rgb(), 1.)
        .map_err(|cause| cause.to_string())?;
    let sprite = match glyph {
        SketchPointGlyph::Dot => dot_glyph(rgba)?,
        SketchPointGlyph::Cross => colored_glyph(rgba)?,
    };
    marker.set_marker_image(sprite);
    Ok(marker)
}
fn dot_glyph(rgba: ColorRgba) -> Result<Handle<Graphic3dMarkerImage>, String> {
    let image = ImagePixMap::default();
    if !image.init_zero(ImageFormat::Rgbaf, DOT_EXTENT, DOT_EXTENT, 0, 0) {
        return Err(format!(
            "dot extent {DOT_EXTENT}: expected allocated RGBA pixels"
        ));
    }
    let center = (DOT_EXTENT - 1) as f64 / 2.;
    for y in 0..DOT_EXTENT {
        for x in 0..DOT_EXTENT {
            if (x as f64 - center).hypot(y as f64 - center) <= DOT_RADIUS {
                image.set_pixel_color_rgba(x as i32, y as i32, &rgba, false);
            }
        }
    }
    Graphic3dMarkerImage::from_images(Handle::new(image).into_upcast(), Handle::default())
        .map(Handle::new)
        .map_err(|cause| cause.to_string())
}
fn colored_glyph(rgba: ColorRgba) -> Result<Handle<Graphic3dMarkerImage>, String> {
    // Native predefined AlphaSprite markers ignore material opacity. Author an RGBA sprite
    // from the kernel's own mask; the kernel still owns its glyph and rasterization.
    let glyph = Graphic3dMarkerImage::standard_marker(
        AspectTypeOfMarker::Plus,
        CROSS_SCALE,
        Vec4::new(1., 1., 1., 1.),
    )
    .map_err(|cause| cause.to_string())?;
    let mask = glyph.get_image();
    let image = tinted_mask(&mask, rgba)?;
    Graphic3dMarkerImage::from_images(Handle::new(image).into_upcast(), Handle::default())
        .map(Handle::new)
        .map_err(|cause| cause.to_string())
}
fn tinted_mask(mask: &ImagePixMap, rgba: ColorRgba) -> Result<ImagePixMap, String> {
    let image = ImagePixMap::default();
    if !image.init_zero(ImageFormat::Rgbaf, mask.width(), mask.height(), 0, 0) {
        return Err(format!(
            "marker extent {:?}: expected allocated RGBA pixels",
            (mask.width(), mask.height())
        ));
    }
    tint_pixels(mask, &image, rgba)?;
    Ok(image)
}
fn tint_pixels(mask: &ImagePixMap, image: &ImagePixMap, rgba: ColorRgba) -> Result<(), String> {
    for y in 0..mask.height() as i32 {
        for x in 0..mask.width() as i32 {
            let coverage = mask
                .pixel_color(x, y, false)
                .map_err(|cause| cause.to_string())?
                .alpha();
            let pixel = ColorRgba::from_rgb(*rgba.rgb(), coverage * rgba.alpha());
            image.set_pixel_color_rgba(x, y, &pixel, false);
        }
    }
    Ok(())
}
