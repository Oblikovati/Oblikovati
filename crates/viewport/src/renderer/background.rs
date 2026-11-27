// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportBackground;
use martensite_tkernel::{
    quantity::{Color, ColorSpace},
    standard::Handle,
};
use martensite_tkservice::aspect::AspectGradientFillMethod;
use martensite_tkv3d::v3d::V3dViewSource;

pub(super) fn apply_background(
    view: &Handle<dyn V3dViewSource>,
    background: ViewportBackground,
) -> Result<(), String> {
    let native = view.view_base();
    match background {
        ViewportBackground::Solid(rgb) => {
            let color = srgb_color(rgb)?;
            native.set_bg_gradient_style(AspectGradientFillMethod::None, false);
            native.set_background_color(color);
        }
        ViewportBackground::VerticalGradient { top, bottom } => {
            let (top, bottom) = (srgb_color(top)?, srgb_color(bottom)?);
            native.set_bg_gradient_colors(top, bottom, AspectGradientFillMethod::Vertical, false);
        }
    }
    Ok(())
}
fn srgb_color(rgb: [u8; 3]) -> Result<Color, String> {
    // Martensite owns colour-space conversion; scene geometry remains linear RGB.
    let [r, g, b] = rgb.map(|channel| f64::from(channel) / 255.);
    Color::new(r, g, b, ColorSpace::Srgb).map_err(|cause| cause.to_string())
}

pub(super) fn apply_scene_background(
    view: &Handle<dyn V3dViewSource>,
    application: Option<ViewportBackground>,
    fallback: [f64; 3],
) -> Result<(), String> {
    if let Some(background) = application {
        return apply_background(view, background);
    }
    let [r, g, b] = fallback;
    let color = Color::new(r, g, b, ColorSpace::Rgb).map_err(|cause| cause.to_string())?;
    view.set_background_color(color);
    Ok(())
}
