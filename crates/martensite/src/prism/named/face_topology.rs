// SPDX-License-Identifier: GPL-2.0-only
//! Read analytic geometry from one current named face.
use super::*;
use martensite_tkbrep::{brep_adaptor::BRepAdaptorSurface, topods};
use martensite_tkg3d::adaptor3d::Adaptor3dSurface;
use martensite_tkmath::geom_abs::SurfaceType;

/// Surface classifications represented by the public topology contract.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum PrismSurfaceKind {
    Plane,
    Cylinder,
    Cone,
    Sphere,
    Torus,
    Spline,
}
/// Exact face reference data, independent from display tessellation.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PrismFaceTopology {
    pub point: [f64; 3],
    pub surface: PrismSurfaceKind,
}
impl NamedPrismSnapshot {
    /// Resolve a face key to its analytic range-box center and surface type.
    /// Example: `body.face_topology(history, document, &key)?`.
    pub fn face_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismFaceTopology, String> {
        let role = self
            .bindings
            .bind(history, document, key, 1)?
            .ok_or_else(|| "face reference: expected a face construction role".to_string())?;
        let face = unique_face(&self.snapshot, role)?;
        let point = super::super::provenance::representative(face, &format!("face {role:?}"))
            .map_err(|cause| cause.to_string())?;
        let native = topods::face(face).map_err(|cause| cause.to_string())?;
        let adaptor =
            BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
        let surface = adaptor.get_type().map_err(|cause| cause.to_string())?;
        Ok(PrismFaceTopology {
            point,
            surface: surface_kind(surface)?,
        })
    }
}
fn surface_kind(surface: SurfaceType) -> Result<PrismSurfaceKind, String> {
    Ok(match surface {
        SurfaceType::Plane => PrismSurfaceKind::Plane,
        SurfaceType::Cylinder => PrismSurfaceKind::Cylinder,
        SurfaceType::Cone => PrismSurfaceKind::Cone,
        SurfaceType::Sphere => PrismSurfaceKind::Sphere,
        SurfaceType::Torus => PrismSurfaceKind::Torus,
        SurfaceType::BezierSurface | SurfaceType::BSplineSurface => PrismSurfaceKind::Spline,
        other => {
            return Err(format!(
                "face surface {other:?}: expected an enumerated analytic or spline surface"
            ));
        }
    })
}
