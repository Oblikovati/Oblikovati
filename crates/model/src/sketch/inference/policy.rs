// SPDX-License-Identifier: GPL-2.0-only
use super::SketchError;
use crate::units::UnitsOfMeasure;
/// Interaction distances are proposal thresholds, never modelling or topological tolerances.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchInferencePolicy {
    pub angle_tolerance_radians: f64,
    pub snap_distance_working: f64,
}
impl SketchInferencePolicy {
    /// Keep physical snap distance stable across document units (#2619, #3774).
    /// Example: `let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());`.
    pub fn for_units(units: &UnitsOfMeasure) -> Self {
        Self {
            // tol:interaction — reproduce Go's 3-degree proposal threshold at the reference scale.
            angle_tolerance_radians: 3.0 * std::f64::consts::PI / 180.0,
            // tol:interaction — Go's cm-scale snap reach; converting removes its scale-dependent defect.
            snap_distance_working: 0.001 / units.working_scale(),
        }
    }
    pub(crate) fn require_valid(self) -> Result<(), SketchError> {
        let angle = self.angle_tolerance_radians;
        let distance = self.snap_distance_working;
        if !angle.is_finite() || angle <= 0.0 || angle >= std::f64::consts::FRAC_PI_4 {
            return Err(SketchError::Definition(format!(
                "inference angle {angle:?}: expected finite radians in (0, pi/4) with disjoint direction families"
            )));
        }
        if !distance.is_finite() || distance <= 0.0 {
            return Err(SketchError::Definition(format!(
                "inference snap distance {distance:?}: expected positive finite working units"
            )));
        }
        Ok(())
    }
}
