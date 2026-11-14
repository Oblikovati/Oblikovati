// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::{
    quantity::{Color, NamedColor},
    standard::Handle,
};
use martensite_tkmath::gp::Direction3;
use martensite_tkservice::graphic3d::Graphic3dCLight;
use martensite_tkv3d::v3d::{V3dAmbientLight, V3dDirectionalLight, V3dViewer};

pub(super) fn configure(viewer: &V3dViewer) -> Result<(), String> {
    let white = Color::named(NamedColor::WHITE);
    let direction = Direction3::new(-0.4, -0.6, -0.8).map_err(|error| error.to_string())?;
    // Match the direction of the Go application's default headlight.
    let directional = V3dDirectionalLight::with_direction(&direction, white, true)
        .map_err(|error| error.to_string())?;
    let ambient = V3dAmbientLight::new(white);
    ambient
        .set_intensity(0.18)
        .map_err(|error| error.to_string())?;
    for light in [
        Handle::new(directional).cast::<Graphic3dCLight>(),
        Handle::new(ambient).cast(),
    ] {
        viewer.add_light(&light);
        viewer
            .set_light_on_one(&light)
            .map_err(|error| error.to_string())?;
    }
    Ok(())
}
