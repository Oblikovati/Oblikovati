// SPDX-License-Identifier: GPL-2.0-only
//! Exact bounds for selected, named prism entities.
use super::*;
use martensite_tkmath::bnd::BoundingBox3;
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;

impl NamedPrismSnapshot {
    /// Find the analytic bounds center of a current named body.
    /// Example: `body.body_bounds_center(history, document, key)?`.
    pub fn body_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        self.bindings.bind(history, document, key, 4)?;
        shape_bounds_center(&self.snapshot.shape)
    }

    /// Find the analytic bounds center of a current named face.
    /// Example: `body.face_bounds_center(history, document, key)?`.
    pub fn face_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let role = self
            .bindings
            .bind(history, document, key, 1)?
            .ok_or_else(|| format!("face key {key:?}: expected a face role"))?;
        shape_bounds_center(unique_face(&self.snapshot, role)?)
    }
}

fn shape_bounds_center(shape: &Shape) -> Result<[f64; 3], String> {
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bounds, false, false)
        .map_err(|cause| format!("selection bounds: expected analytic B-rep bounds: {cause}"))?;
    let center = bounds
        .center()
        .ok_or("selection bounds: expected nonempty finite bounds")?;
    let center = [center.x(), center.y(), center.z()];
    if center.iter().any(|coordinate| !coordinate.is_finite()) {
        return Err(format!(
            "selection bounds center {center:?}: expected finite coordinates"
        ));
    }
    Ok(center)
}
