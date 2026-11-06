// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::{ncollection::Vec4, quantity::ColorRgba, standard::Handle};
use martensite_tkservice::{
    aspect::AspectTypeOfMarker,
    graphic3d::*,
    image::{ImageFormat, ImagePixMap},
};

pub(super) fn theme_marker(rgba: ColorRgba) -> Result<Graphic3dAspectMarker3d, String> {
    let marker = Graphic3dAspectMarker3d::new(AspectTypeOfMarker::Userdefined, *rgba.rgb(), 1.)
        .map_err(|cause| cause.to_string())?;
    marker.set_marker_image(colored_glyph(rgba)?);
    Ok(marker)
}
fn colored_glyph(rgba: ColorRgba) -> Result<Handle<Graphic3dMarkerImage>, String> {
    // Native predefined AlphaSprite markers ignore material opacity. Author an RGBA sprite
    // from the kernel's own OPlus mask; the kernel still owns its glyph and rasterization.
    let glyph = Graphic3dMarkerImage::standard_marker(
        AspectTypeOfMarker::OPlus,
        1.,
        Vec4::new(1., 1., 1., 1.),
    )
    .map_err(|cause| cause.to_string())?;
    let mask = glyph.get_image();
    let image = tinted_mask(&mask, rgba)?;
    Graphic3dMarkerImage::from_images(Handle::new(image), Handle::default())
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
