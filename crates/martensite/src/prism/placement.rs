// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkmath::gp::{Point3, Vector3};
pub(super) fn model_point(plane: SketchPlane, point: [f64; 2]) -> Result<Point3, String> {
    let [x, y, z] = plane.to_model(point).map_err(|error| error.to_string())?;
    Ok(Point3::new(x, y, z))
}
pub(super) fn model_displacement(plane: SketchPlane, local: [f64; 3]) -> Result<Vector3, String> {
    let recipe = plane.recipe();
    let normal = plane.normal();
    // A vector has no origin: subtracting two placed points loses small displacements at large origins.
    let components: [f64; 3] = std::array::from_fn(|axis| {
        (local[0] * recipe.x_axis[axis] + local[1] * recipe.y_axis[axis]) + local[2] * normal[axis]
    });
    if components.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "prism displacement {local:?} on plane {recipe:?}: expected finite model-space components, got {components:?}"
        ));
    }
    let [x, y, z] = components;
    Ok(Vector3::new(x, y, z))
}
