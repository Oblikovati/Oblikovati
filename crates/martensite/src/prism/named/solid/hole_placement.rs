// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Hole placement on the exact named face before constructing geometry (#4093).
use super::*;
use martensite_tkbrep::topods;
use martensite_tkg3d::top_abs::State;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::{brep_class::BRepClassFaceClassifier, brep_gprop::BRepGProp};

impl NamedSolid {
    /// Resolve a face point and inward axis; example: `body.hole_axis(&history, document, key, None, resolution)?`.
    pub fn hole_axis(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
        position: Option<[f64; 3]>,
        resolution: PrismResolution,
    ) -> Result<[[f64; 3]; 2], String> {
        let resolution = resolution.validate()?;
        let body = self.native_shape(history, document)?;
        let face = self.native_face(history, document, face_key)?;
        let frame = super::super::face_topology::planar_shape_frame(body, face)?;
        let center = match position {
            Some(point) => point,
            None => surface_centroid(face)?,
        };
        require_on_face(face, center, &frame, resolution.sew)?;
        Ok([center, frame.normal().map(|component| -component)])
    }
}

fn surface_centroid(face: &Shape) -> Result<[f64; 3], String> {
    let mut properties = GProps::new();
    BRepGProp::surface_properties(face, &mut properties, false, false).map_err(|cause| {
        format!("hole placement face: expected analytic area properties: {cause}")
    })?;
    if !properties.mass().is_finite() || properties.mass() <= 0. {
        return Err(format!(
            "hole placement face area {}: expected finite positive area",
            properties.mass()
        ));
    }
    let center = properties.centre_of_mass();
    Ok([center.x(), center.y(), center.z()])
}

fn require_on_face(
    face: &Shape,
    point: [f64; 3],
    frame: &oblikovati_model::sketch::SketchPlane,
    tolerance: f64,
) -> Result<(), String> {
    if point.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "hole position {point:?}: expected finite coordinates"
        ));
    }
    let origin = frame.recipe().origin;
    let distance: f64 = point
        .into_iter()
        .zip(origin)
        .zip(frame.normal())
        .map(|((value, base), normal)| (value - base) * normal)
        .sum();
    if distance.abs() > tolerance {
        return Err(format!(
            "hole position {point:?}, plane distance {distance}: expected a point on the face within {tolerance}"
        ));
    }
    let face =
        topods::face(face).map_err(|cause| format!("hole face: expected a valid face: {cause}"))?;
    let point3 = Point3::new(point[0], point[1], point[2]);
    let classifier =
        BRepClassFaceClassifier::with_face_point3d(&face, &point3, tolerance, false, tolerance)
            .map_err(|cause| {
                format!("hole position {point:?}: expected face classification: {cause}")
            })?;
    if classifier.state() != State::In {
        return Err(format!(
            "hole position {point:?}, classification {:?}: expected an interior point",
            classifier.state()
        ));
    }
    Ok(())
}
