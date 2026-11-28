// SPDX-License-Identifier: GPL-2.0-only
//! Authored User Coordinate System (UCS) datum feature (#3918).
use crate::{
    feature::{FeatureReadFootprint, work_axis::WorkAxisLine},
    sketch::{SketchPlane, SketchPlaneOptions},
};

/// Default browser prefix for newly created User Coordinate Systems.
pub const DEFAULT_UCS_PREFIX: &str = "UCS";

/// The seven standard child work-feature names listed under every UCS node.
pub const UCS_CHILD_NAMES: [&str; 7] = [
    "YZ Plane",
    "XZ Plane",
    "XY Plane",
    "X Axis",
    "Y Axis",
    "Z Axis",
    "Center Point",
];

const BASIS_EPSILON: f64 = 1e-12;

fn default_visible() -> bool {
    true
}

/// Serialized recipe for a User Coordinate System datum feature.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct UcsRecipe {
    pub origin: [f64; 3],
    pub x_dir: [f64; 3],
    pub y_dir: [f64; 3],
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub vertex_ref: Option<Vec<u8>>,
    #[serde(default = "default_visible")]
    pub visible: bool,
}

/// Evaluated orthonormal coordinate frame for a User Coordinate System.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ResolvedUcs {
    pub origin: [f64; 3],
    pub x_dir: [f64; 3],
    pub y_dir: [f64; 3],
    pub z_dir: [f64; 3],
}

impl ResolvedUcs {
    /// Local YZ plane (normal along +X); example: `ucs.yz_plane()?`.
    pub fn yz_plane(&self) -> Result<SketchPlane, String> {
        SketchPlane::from_axis_directions(
            self.origin,
            self.y_dir,
            self.z_dir,
            SketchPlaneOptions::default(),
        )
        .map_err(|cause| cause.to_string())
    }

    /// Local XZ plane (normal along +Y via Z x X); example: `ucs.xz_plane()?`.
    pub fn xz_plane(&self) -> Result<SketchPlane, String> {
        SketchPlane::from_axis_directions(
            self.origin,
            self.z_dir,
            self.x_dir,
            SketchPlaneOptions::default(),
        )
        .map_err(|cause| cause.to_string())
    }

    /// Local XY plane (normal along +Z); example: `ucs.xy_plane()?`.
    pub fn xy_plane(&self) -> Result<SketchPlane, String> {
        SketchPlane::from_axis_directions(
            self.origin,
            self.x_dir,
            self.y_dir,
            SketchPlaneOptions::default(),
        )
        .map_err(|cause| cause.to_string())
    }

    /// Local X work axis through origin; example: `ucs.x_axis()?`.
    pub fn x_axis(&self) -> Result<WorkAxisLine, String> {
        WorkAxisLine::new(self.origin, self.x_dir).map_err(|cause| cause.to_string())
    }

    /// Local Y work axis through origin; example: `ucs.y_axis()?`.
    pub fn y_axis(&self) -> Result<WorkAxisLine, String> {
        WorkAxisLine::new(self.origin, self.y_dir).map_err(|cause| cause.to_string())
    }

    /// Local Z work axis through origin; example: `ucs.z_axis()?`.
    pub fn z_axis(&self) -> Result<WorkAxisLine, String> {
        WorkAxisLine::new(self.origin, self.z_dir).map_err(|cause| cause.to_string())
    }

    /// Local center point at the UCS origin; example: `ucs.center_point()`.
    pub fn center_point(&self) -> [f64; 3] {
        self.origin
    }
}

/// Validated authored User Coordinate System definition with orthonormal right-handed basis.
#[derive(Clone, Debug, PartialEq)]
pub struct UcsDefinition {
    origin: [f64; 3],
    x_dir: [f64; 3],
    y_dir: [f64; 3],
    z_dir: [f64; 3],
    vertex_ref: Option<Vec<u8>>,
    visible: bool,
}

impl UcsDefinition {
    /// Validate and orthonormalize a right-handed UCS definition.
    /// Example: `UcsDefinition::new([0.; 3], [1., 0., 0.], [0., 1., 0.], None, true)?`.
    pub fn new(
        origin: [f64; 3],
        x_dir: [f64; 3],
        y_dir: [f64; 3],
        vertex_ref: Option<Vec<u8>>,
        visible: bool,
    ) -> Result<Self, String> {
        require_finite("origin", origin)?;
        require_finite("x_dir", x_dir)?;
        require_finite("y_dir", y_dir)?;
        let x = normalize("x_dir", x_dir)?;
        let proj = dot(y_dir, x);
        let y_ortho = [
            y_dir[0] - proj * x[0],
            y_dir[1] - proj * x[1],
            y_dir[2] - proj * x[2],
        ];
        let y = normalize("y_dir (orthogonal to x_dir)", y_ortho)?;
        let z = normalize("z_dir", cross(x, y))?;
        Ok(Self {
            origin,
            x_dir: x,
            y_dir: y,
            z_dir: z,
            vertex_ref,
            visible,
        })
    }

    /// Restore a validated UCS definition from its serialized recipe.
    /// Example: `UcsDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: UcsRecipe) -> Result<Self, String> {
        Self::new(
            recipe.origin,
            recipe.x_dir,
            recipe.y_dir,
            recipe.vertex_ref,
            recipe.visible,
        )
    }

    /// Capture the serialized recipe without evaluated caches; example: `def.recipe()`.
    pub fn recipe(&self) -> UcsRecipe {
        UcsRecipe {
            origin: self.origin,
            x_dir: self.x_dir,
            y_dir: self.y_dir,
            vertex_ref: self.vertex_ref.clone(),
            visible: self.visible,
        }
    }

    /// Compare authored fields using exact IEEE bit equality; example: `a.same_recipe(&b)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        same_vec3(self.origin, other.origin)
            && same_vec3(self.x_dir, other.x_dir)
            && same_vec3(self.y_dir, other.y_dir)
            && same_vec3(self.z_dir, other.z_dir)
            && self.vertex_ref == other.vertex_ref
            && self.visible == other.visible
    }

    /// Attribute read footprint for program ordering; example: `def.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(Vec::new(), false)
    }

    /// Evaluated orthonormal coordinate frame; example: `def.resolved()`.
    pub fn resolved(&self) -> ResolvedUcs {
        ResolvedUcs {
            origin: self.origin,
            x_dir: self.x_dir,
            y_dir: self.y_dir,
            z_dir: self.z_dir,
        }
    }

    /// Origin point in model coordinates; example: `def.origin()`.
    pub fn origin(&self) -> [f64; 3] {
        self.origin
    }

    /// Unit X axis direction; example: `def.x_dir()`.
    pub fn x_dir(&self) -> [f64; 3] {
        self.x_dir
    }

    /// Unit Y axis direction; example: `def.y_dir()`.
    pub fn y_dir(&self) -> [f64; 3] {
        self.y_dir
    }

    /// Unit Z axis direction (right-handed cross of X and Y); example: `def.z_dir()`.
    pub fn z_dir(&self) -> [f64; 3] {
        self.z_dir
    }

    /// Optional topology vertex key selected for origin placement; example: `def.vertex_ref()`.
    pub fn vertex_ref(&self) -> Option<&[u8]> {
        self.vertex_ref.as_deref()
    }

    /// Current authored visibility flag; example: `def.is_visible()`.
    pub fn is_visible(&self) -> bool {
        self.visible
    }

    /// Return a copy with updated visibility; example: `def.with_visible(false)`.
    pub fn with_visible(&self, visible: bool) -> Self {
        let mut copy = self.clone();
        copy.visible = visible;
        copy
    }
}

fn require_finite(label: &str, v: [f64; 3]) -> Result<(), String> {
    if v.iter().all(|c| c.is_finite()) {
        return Ok(());
    }
    Err(format!("UCS {label} {v:?}: expected finite coordinates"))
}

fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn normalize(label: &str, v: [f64; 3]) -> Result<[f64; 3], String> {
    let len = dot(v, v).sqrt();
    if !len.is_finite() || len <= BASIS_EPSILON {
        return Err(format!(
            "UCS {label} {v:?}: expected a nonzero non-collinear direction vector"
        ));
    }
    Ok([v[0] / len, v[1] / len, v[2] / len])
}

fn same_vec3(a: [f64; 3], b: [f64; 3]) -> bool {
    a.into_iter()
        .zip(b)
        .all(|(x, y)| x.to_bits() == y.to_bits())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn orthonormalizes_right_handed_basis_and_derives_children() {
        let def = UcsDefinition::new(
            [1.0, 2.0, 3.0],
            [2.0, 0.0, 0.0],
            [1.0, 3.0, 0.0],
            Some(vec![1, 2, 3]),
            true,
        )
        .unwrap();
        assert_eq!(def.origin(), [1.0, 2.0, 3.0]);
        assert_eq!(def.x_dir(), [1.0, 0.0, 0.0]);
        assert_eq!(def.y_dir(), [0.0, 1.0, 0.0]);
        assert_eq!(def.z_dir(), [0.0, 0.0, 1.0]);
        assert_eq!(def.vertex_ref(), Some(&[1, 2, 3][..]));
        assert!(def.is_visible());

        let resolved = def.resolved();
        assert_eq!(resolved.center_point(), [1.0, 2.0, 3.0]);
        assert!(resolved.xy_plane().is_ok());
        assert!(resolved.xz_plane().is_ok());
        assert!(resolved.yz_plane().is_ok());
        assert!(resolved.x_axis().is_ok());
        assert!(resolved.y_axis().is_ok());
        assert!(resolved.z_axis().is_ok());
    }

    #[test]
    fn rejects_degenerate_or_collinear_axes() {
        assert!(UcsDefinition::new([0.; 3], [0.; 3], [0., 1., 0.], None, true).is_err());
        assert!(UcsDefinition::new([0.; 3], [1., 0., 0.], [2., 0., 0.], None, true).is_err());
        assert!(
            UcsDefinition::new([f64::NAN, 0., 0.], [1., 0., 0.], [0., 1., 0.], None, true).is_err()
        );
    }

    #[test]
    fn recipe_round_trips_with_visibility_and_vertex_ref() {
        let def = UcsDefinition::new(
            [4.0, -1.0, 2.5],
            [0.0, 1.0, 0.0],
            [0.0, 0.0, 1.0],
            Some(vec![9, 8, 7]),
            false,
        )
        .unwrap();
        let json = serde_json::to_string(&def.recipe()).unwrap();
        let restored = UcsDefinition::from_recipe(serde_json::from_str(&json).unwrap()).unwrap();
        assert!(def.same_recipe(&restored));
        assert!(!restored.is_visible());
        assert_eq!(restored.z_dir(), [1.0, 0.0, 0.0]);
    }
}
