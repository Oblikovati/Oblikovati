// SPDX-License-Identifier: GPL-2.0-only
//! Read analytic geometry from one current named face.
use super::*;
use martensite_tkbrep::{brep_adaptor::BRepAdaptorSurface, topods};
use martensite_tkg3d::{adaptor3d::Adaptor3dSurface, top_abs::Orientation};
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
    /// Resolve a named planar face to an exact outward frame at the projected part origin.
    /// In-plane axes come from the kernel surface and can differ from a legacy face frame.
    /// Example: `body.face_plane(history, document, &key)?` for a dependent datum.
    pub fn face_plane(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        let role = self
            .bindings
            .bind(history, document, key, 1)?
            .ok_or_else(|| "face reference: expected a face construction role".to_string())?;
        let lineage = unique_face(&self.snapshot, role)?;
        planar_shape_frame(&self.snapshot.shape, lineage)
    }

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
        face_topology_from_shape(face)
    }
    /// Read an analytic cylindrical face radius in model units. Example: `body.face_cylinder_radius(history, guid, key)?`.
    pub fn face_cylinder_radius(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<Option<f64>, String> {
        let role = self
            .bindings
            .bind(history, document, key, 1)?
            .ok_or_else(|| "face reference: expected a face construction role".to_string())?;
        cylindrical_face_radius(unique_face(&self.snapshot, role)?)
    }
}
pub(super) fn face_topology_from_shape(face: &Shape) -> Result<PrismFaceTopology, String> {
    let point = super::super::provenance::representative(face, "named face")
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
pub(super) fn cylindrical_face_radius(face: &Shape) -> Result<Option<f64>, String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let surface =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    if surface.get_type().map_err(|cause| cause.to_string())? != SurfaceType::Cylinder {
        return Ok(None);
    }
    let cylinder = surface.cylinder().map_err(|cause| cause.to_string())?;
    Ok(Some(cylinder.radius()))
}
pub(super) fn planar_shape_frame(
    body: &Shape,
    lineage: &Shape,
) -> Result<oblikovati_model::sketch::SketchPlane, String> {
    let face = super::super::snapshot::unique_body_face(body, lineage)?;
    let native = topods::face(&face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, false).map_err(|cause| cause.to_string())?;
    let surface = adaptor.get_type().map_err(|cause| cause.to_string())?;
    if surface != SurfaceType::Plane {
        return Err(format!(
            "face surface {surface:?}: expected a planar surface"
        ));
    }
    let plane = adaptor.plane().map_err(|cause| cause.to_string())?;
    projected_plane(plane, face.orientation())
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

fn projected_plane(
    plane: martensite_tkmath::gp::Plane3,
    orientation: Orientation,
) -> Result<oblikovati_model::sketch::SketchPlane, String> {
    let frame = plane.position();
    let normal = *frame.direction().coordinates();
    let location = *frame.location().coordinates();
    let distance: f64 = normal.iter().zip(location).map(|(a, b)| a * b).sum();
    let origin = normal.map(|value| value * distance);
    let y_axis = *frame.y_direction().coordinates();
    let oriented_y = match orientation {
        Orientation::Forward => y_axis,
        Orientation::Reversed => y_axis.map(|value| -value),
        other => {
            return Err(format!(
                "face orientation {other:?}: expected Forward or Reversed"
            ));
        }
    };
    oblikovati_model::sketch::SketchPlane::from_directions(
        origin,
        *frame.x_direction().coordinates(),
        oriented_y,
        Default::default(),
    )
    .map_err(|cause| cause.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkmath::gp::{Direction3, Plane3, Point3};

    #[test]
    fn oriented_surface_frame_refuses_non_boundary_orientations() {
        let plane = Plane3::new(
            Point3::new(3., 4., 5.),
            Direction3::new(1., 0., 0.).unwrap(),
        )
        .unwrap();
        let forward = projected_plane(plane, Orientation::Forward).unwrap();
        let reversed = projected_plane(plane, Orientation::Reversed).unwrap();
        assert_eq!(forward.recipe().origin, [3., 0., 0.]);
        assert_eq!(forward.normal(), [1., 0., 0.]);
        assert_eq!(reversed.normal(), [-1., 0., 0.]);
        for orientation in [Orientation::Internal, Orientation::External] {
            assert!(projected_plane(plane, orientation).is_err());
        }
    }
}
