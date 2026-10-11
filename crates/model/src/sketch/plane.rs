// SPDX-License-Identifier: GPL-2.0-only
use super::SketchError;
mod orientation;
pub use orientation::SketchFrameAxis;
/// Versioned metric checks for authored sketch frame axes, separate from modelling tolerance.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPlaneOptions {
    pub version: u32,
    pub unit_deviation: f64,
    pub angular_deviation: f64,
}
impl Default for SketchPlaneOptions {
    fn default() -> Self {
        // tol:angular and tol:unit — Go math.DefaultTolerance at aa8297d1 (#3717).
        Self {
            version: 1,
            unit_deviation: 1e-9,
            angular_deviation: 1e-9,
        }
    }
}
/// Authored placement only; a derived normal is not a saved source of truth.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPlaneRecipe {
    pub origin: [f64; 3],
    pub x_axis: [f64; 3],
    pub y_axis: [f64; 3],
}
/// Validated right-handed sketch placement. Construct it before model-space queries.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchPlane {
    recipe: SketchPlaneRecipe,
    options: SketchPlaneOptions,
    normal: [f64; 3],
}
impl SketchPlane {
    /// Compare saved frame bits and validation policy; example: `old.same_recipe(candidate)`.
    pub fn same_recipe(self, other: Self) -> bool {
        use super::recipe_compare::same_components;
        same_components(self.recipe.origin, other.recipe.origin)
            && same_components(self.recipe.x_axis, other.recipe.x_axis)
            && same_components(self.recipe.y_axis, other.recipe.y_axis)
            && self.options.version == other.options.version
            && same_components(
                [self.options.unit_deviation, self.options.angular_deviation],
                [
                    other.options.unit_deviation,
                    other.options.angular_deviation,
                ],
            )
    }

    /// Validate saved axes without altering them; example: `SketchPlane::new([0.0; 3], [1.0,0.0,0.0], [0.0,1.0,0.0], Default::default())?`.
    pub fn new(
        origin: [f64; 3],
        x_axis: [f64; 3],
        y_axis: [f64; 3],
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        require_options(options)?;
        require_finite(&origin, "origin")?;
        require_unit(x_axis, options.unit_deviation)?;
        require_unit(y_axis, options.unit_deviation)?;
        require_perpendicular(x_axis, y_axis, options.angular_deviation)?;
        let normal = frame_normal(x_axis, y_axis)?;
        Ok(Self {
            recipe: SketchPlaneRecipe {
                origin,
                x_axis,
                y_axis,
            },
            options,
            normal,
        })
    }
    /// Restore saved placement using the same frame checks; example: `SketchPlane::from_recipe(recipe, Default::default())?`.
    pub fn from_recipe(
        recipe: SketchPlaneRecipe,
        options: SketchPlaneOptions,
    ) -> Result<Self, SketchError> {
        Self::new(recipe.origin, recipe.x_axis, recipe.y_axis, options)
    }
    /// Capture authored axes without the derived normal; example: `plane.recipe()`.
    pub fn recipe(self) -> SketchPlaneRecipe {
        self.recipe
    }
    /// Capture the frame validation policy for restore; example: `plane.options()`.
    pub fn options(self) -> SketchPlaneOptions {
        self.options
    }
    /// Inspect the derived right-handed unit normal; example: `plane.normal()`.
    pub fn normal(self) -> [f64; 3] {
        self.normal
    }
    /// Use the origin XY work plane; example: `let plane = SketchPlane::xy();`.
    pub fn xy() -> Self {
        basis_plane([1.0, 0.0, 0.0], [0.0, 1.0, 0.0], [0.0, 0.0, 1.0])
    }
    /// Use the origin XZ work plane; example: `let plane = SketchPlane::xz();`.
    pub fn xz() -> Self {
        basis_plane([1.0, 0.0, 0.0], [0.0, 0.0, 1.0], [0.0, -1.0, 0.0])
    }
    /// Use the origin YZ work plane; example: `let plane = SketchPlane::yz();`.
    pub fn yz() -> Self {
        basis_plane([0.0, 1.0, 0.0], [0.0, 0.0, 1.0], [1.0, 0.0, 0.0])
    }
    /// Map a sketch point into model working units; example: `plane.to_model([2.0,3.0])?`.
    pub fn to_model(self, point: [f64; 2]) -> Result<[f64; 3], SketchError> {
        require_finite(&point, "sketch point")?;
        let result = std::array::from_fn(|axis| {
            (self.recipe.origin[axis] + point[0] * self.recipe.x_axis[axis])
                + point[1] * self.recipe.y_axis[axis]
        });
        require_finite(&result, "model point result")?;
        Ok(result)
    }
    /// Orthogonally project a model point; example: `plane.to_sketch([2.0,3.0,7.0])?`.
    pub fn to_sketch(self, point: [f64; 3]) -> Result<[f64; 2], SketchError> {
        require_finite(&point, "model point")?;
        let offset = std::array::from_fn(|axis| point[axis] - self.recipe.origin[axis]);
        require_finite(&offset, "model displacement")?;
        let result = [
            dot_product(offset, self.recipe.x_axis),
            dot_product(offset, self.recipe.y_axis),
        ];
        require_finite(&result, "sketch point result")?;
        Ok(result)
    }
}
fn basis_plane(x_axis: [f64; 3], y_axis: [f64; 3], normal: [f64; 3]) -> SketchPlane {
    SketchPlane {
        recipe: SketchPlaneRecipe {
            origin: [0.0; 3],
            x_axis,
            y_axis,
        },
        options: SketchPlaneOptions::default(),
        normal,
    }
}
fn require_options(options: SketchPlaneOptions) -> Result<(), SketchError> {
    if options.version != 1
        || ![options.unit_deviation, options.angular_deviation]
            .into_iter()
            .all(|value| value.is_finite() && (0.0..1.0).contains(&value))
    {
        return Err(SketchError::Definition(format!(
            "sketch plane options {options:?}: expected version 1 with finite metric deviations in [0, 1)"
        )));
    }
    Ok(())
}
fn require_finite<const N: usize>(values: &[f64; N], field: &str) -> Result<(), SketchError> {
    if !values.iter().all(|value| value.is_finite()) {
        return Err(SketchError::Definition(format!(
            "sketch plane {field} {values:?}: expected {N} finite working-unit components"
        )));
    }
    Ok(())
}
fn require_unit(axis: [f64; 3], tolerance: f64) -> Result<(), SketchError> {
    require_finite(&axis, "axis")?;
    let length = vector_length(axis);
    if (length - 1.0).abs() > tolerance {
        return Err(SketchError::Definition(format!(
            "sketch plane axis {axis:?} length {length:?}: expected a unit axis with deviation at most {tolerance:?}"
        )));
    }
    Ok(())
}
fn require_perpendicular(
    x_axis: [f64; 3],
    y_axis: [f64; 3],
    tolerance: f64,
) -> Result<(), SketchError> {
    let x_length = vector_length(x_axis);
    let y_length = vector_length(y_axis);
    let dot = dot_product(
        x_axis.map(|value| value / x_length),
        y_axis.map(|value| value / y_length),
    );
    if dot.abs() > tolerance {
        return Err(SketchError::Definition(format!(
            "sketch plane axes {x_axis:?}, {y_axis:?} dot {dot:?}: expected perpendicular unit axes with deviation at most {tolerance:?}"
        )));
    }
    Ok(())
}
fn vector_length(vector: [f64; 3]) -> f64 {
    vector[0].hypot(vector[1]).hypot(vector[2])
}
fn dot_product(first: [f64; 3], second: [f64; 3]) -> f64 {
    (first[0] * second[0] + first[1] * second[1]) + first[2] * second[2]
}
fn cross_product(first: [f64; 3], second: [f64; 3]) -> [f64; 3] {
    [
        first[1] * second[2] - first[2] * second[1],
        first[2] * second[0] - first[0] * second[2],
        first[0] * second[1] - first[1] * second[0],
    ]
}

fn frame_normal(x_axis: [f64; 3], y_axis: [f64; 3]) -> Result<[f64; 3], SketchError> {
    let cross = cross_product(x_axis, y_axis);
    let length = vector_length(cross);
    if !length.is_finite() || length == 0.0 {
        return Err(SketchError::Definition(format!(
            "sketch plane axes {x_axis:?}, {y_axis:?} cross length {length:?}: expected a finite positive length for a defined normal"
        )));
    }
    Ok(cross.map(|value| value / length))
}
