// SPDX-License-Identifier: GPL-2.0-only
use crate::{ViewportScene, ViewportTriangle};
use oblikovati_martensite::PrismPresentation;
impl ViewportScene {
    /// Convert owned kernel mesh coordinates to display precision; construction roles are not pick keys.
    /// Example: `ViewportScene::from_prism(&mesh, [0., 0., 1.], [0., 1., 0., 1.])?`.
    pub fn from_prism(
        presentation: &PrismPresentation,
        background: [f64; 3],
        color: [f32; 4],
    ) -> Result<Self, String> {
        let mut triangles = Vec::new();
        for face in &presentation.faces {
            for positions in &face.triangles {
                triangles.push(display_triangle(*positions, color)?);
            }
        }
        let scene = Self {
            background,
            triangles,
        };
        scene.validate()?;
        Ok(scene)
    }
}
fn display_triangle(positions: [[f64; 3]; 3], color: [f32; 4]) -> Result<ViewportTriangle, String> {
    let converted = positions.map(|point| point.map(|value| value as f32));
    if converted.iter().flatten().any(|value| !value.is_finite()) {
        return Err(format!(
            "prism triangle {positions:?}: expected coordinates representable as finite display floats"
        ));
    }
    Ok(ViewportTriangle {
        positions: converted,
        color,
    })
}
#[cfg(test)]
#[path = "../tests/presentation.rs"]
mod tests;
