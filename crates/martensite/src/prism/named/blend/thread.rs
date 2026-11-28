// SPDX-License-Identifier: GPL-2.0-only
//! Cylindrical/conical face analysis and compatibility validation for Part Modify Thread (#3918).
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorSurface,
    topods::{self, Shape},
};
use martensite_tkg3d::{
    adaptor3d::Adaptor3dSurface,
    top_abs::{Orientation, State},
};
use martensite_tkmath::{geom_abs::SurfaceType, gp::Point3};
use martensite_tktopalgo::brep_class3d::BRepClass3dSolidClassifier;
use oblikovati_model::feature::thread::ResolvedThread;

pub(crate) use super::thread_overlay::build_thread_overlay_triangles;

/// Analytic geometry of a cylindrical or conical face eligible for a Thread feature.
#[derive(Clone, Debug, PartialEq)]
pub struct ThreadableFaceInfo {
    /// Nominal diameter in working units (cm).
    pub diameter_cm: f64,
    /// Minimum diameter along the face in cm.
    pub min_diameter_cm: f64,
    /// Maximum diameter along the face in cm.
    pub max_diameter_cm: f64,
    /// Exact generator length (axial height for cylinder, slant length for cone) in cm.
    pub full_depth_cm: f64,
    /// True when the outward solid normal points toward the axis (bore/hole wall).
    pub internal: bool,
    /// True for conical/tapered faces, false for cylindrical faces.
    pub conical: bool,
}

/// Fast surface-kind check for thread eligibility without 3D point-in-solid classification.
pub(crate) fn is_threadable_surface(face: &Shape) -> bool {
    let Ok(native) = topods::face(face) else {
        return false;
    };
    let Ok(adaptor) = BRepAdaptorSurface::with_face(&native, false) else {
        return false;
    };
    matches!(
        adaptor.get_type(),
        Ok(SurfaceType::Cylinder | SurfaceType::Cone)
    )
}

/// Inspect a face shape and return its threadable analytic properties if cylindrical or conical.
pub(crate) fn inspect_threadable_face(
    body: Option<&Shape>,
    face: &Shape,
) -> Result<Option<ThreadableFaceInfo>, String> {
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, true).map_err(|cause| cause.to_string())?;
    let surface_type = adaptor.get_type().map_err(|cause| cause.to_string())?;
    match surface_type {
        SurfaceType::Cylinder => inspect_cylinder(body, &adaptor, face.orientation()).map(Some),
        SurfaceType::Cone => inspect_cone(body, &adaptor, face.orientation()).map(Some),
        _ => Ok(None),
    }
}

fn inspect_cylinder(
    body: Option<&Shape>,
    adaptor: &BRepAdaptorSurface,
    orientation: Orientation,
) -> Result<ThreadableFaceInfo, String> {
    let cyl = adaptor.cylinder().map_err(|cause| cause.to_string())?;
    let radius = cyl.radius();
    let (u_mid, v_min, v_max) = uv_span(adaptor)?;
    let p0 = point_coords(adaptor.value(u_mid, v_min).map_err(|e| e.to_string())?);
    let p1 = point_coords(adaptor.value(u_mid, v_max).map_err(|e| e.to_string())?);
    let full_depth_cm = distance3(p0, p1);
    let axis = cyl.axis();
    let axis_origin = point_coords(axis.location());
    let axis_dir = *axis.direction().coordinates();
    let internal = face_points_inward(
        body,
        adaptor,
        orientation,
        u_mid,
        0.5 * (v_min + v_max),
        axis_origin,
        axis_dir,
        radius,
    )?;
    let diameter_cm = 2.0 * radius;
    Ok(ThreadableFaceInfo {
        diameter_cm,
        min_diameter_cm: diameter_cm,
        max_diameter_cm: diameter_cm,
        full_depth_cm,
        internal,
        conical: false,
    })
}

fn inspect_cone(
    body: Option<&Shape>,
    adaptor: &BRepAdaptorSurface,
    orientation: Orientation,
) -> Result<ThreadableFaceInfo, String> {
    let cone = adaptor.cone().map_err(|cause| cause.to_string())?;
    let (u_mid, v_min, v_max) = uv_span(adaptor)?;
    let p0 = point_coords(adaptor.value(u_mid, v_min).map_err(|e| e.to_string())?);
    let p1 = point_coords(adaptor.value(u_mid, v_max).map_err(|e| e.to_string())?);
    let full_depth_cm = distance3(p0, p1);
    let axis = cone.axis();
    let axis_origin = point_coords(axis.location());
    let axis_dir = *axis.direction().coordinates();
    let r0 = radial_distance(p0, axis_origin, axis_dir);
    let r1 = radial_distance(p1, axis_origin, axis_dir);
    let r_min = r0.min(r1);
    let r_max = r0.max(r1);
    let r_mid = 0.5 * (r0 + r1);
    let internal = face_points_inward(
        body,
        adaptor,
        orientation,
        u_mid,
        0.5 * (v_min + v_max),
        axis_origin,
        axis_dir,
        r_mid,
    )?;
    Ok(ThreadableFaceInfo {
        diameter_cm: 2.0 * r_mid,
        min_diameter_cm: 2.0 * r_min,
        max_diameter_cm: 2.0 * r_max,
        full_depth_cm,
        internal,
        conical: true,
    })
}

/// Reject geometrically incompatible thread specs (such as straight JIS Pipe Threads with model
/// display on a conical face).
pub(crate) fn validate_thread_compatibility(
    info: &ThreadableFaceInfo,
    input: &ResolvedThread,
) -> Result<(), String> {
    if !info.full_depth_cm.is_finite() || info.full_depth_cm <= 1e-6 {
        return Err(format!(
            "thread face length {}: expected a finite positive face length",
            info.full_depth_cm
        ));
    }
    if info.conical
        && input.display_in_model
        && input
            .spec
            .thread_type
            .eq_ignore_ascii_case("JIS Pipe Threads")
    {
        return Err(format!(
            "Edit Thread Feature Failed: {} (size {}, class {}) cannot display on a conical face",
            input.spec.thread_type, input.spec.size, input.spec.class
        ));
    }
    Ok(())
}

fn uv_span(adaptor: &BRepAdaptorSurface) -> Result<(f64, f64, f64), String> {
    let u_min = adaptor.first_u_parameter().map_err(|e| e.to_string())?;
    let u_max = adaptor.last_u_parameter().map_err(|e| e.to_string())?;
    let v_min = adaptor.first_v_parameter().map_err(|e| e.to_string())?;
    let v_max = adaptor.last_v_parameter().map_err(|e| e.to_string())?;
    if !u_min.is_finite() || !u_max.is_finite() || !v_min.is_finite() || !v_max.is_finite() {
        return Err("thread face UV bounds: expected finite trimmed parameters".into());
    }
    Ok((0.5 * (u_min + u_max), v_min, v_max))
}

#[allow(clippy::too_many_arguments)]
fn face_points_inward(
    body: Option<&Shape>,
    adaptor: &BRepAdaptorSurface,
    orientation: Orientation,
    u: f64,
    v: f64,
    axis_origin: [f64; 3],
    axis_dir: [f64; 3],
    radius: f64,
) -> Result<bool, String> {
    let d1 = adaptor.d1(u, v).map_err(|e| e.to_string())?;
    let p = point_coords(d1.point);
    let rel = [
        p[0] - axis_origin[0],
        p[1] - axis_origin[1],
        p[2] - axis_origin[2],
    ];
    let proj = dot3(rel, axis_dir);
    let radial = [
        rel[0] - proj * axis_dir[0],
        rel[1] - proj * axis_dir[1],
        rel[2] - proj * axis_dir[2],
    ];
    if let Some(solid) = body {
        let r_hat = normalize3(radial);
        let eps = (radius * 0.02).clamp(1e-4, 1e-2);
        let p_out = Point3::new(
            p[0] + eps * r_hat[0],
            p[1] + eps * r_hat[1],
            p[2] + eps * r_hat[2],
        );
        let p_in = Point3::new(
            p[0] - eps * r_hat[0],
            p[1] - eps * r_hat[1],
            p[2] - eps * r_hat[2],
        );
        if let Ok(mut classifier) = BRepClass3dSolidClassifier::with_shape(solid)
            && classifier.perform(&p_out, 1e-6).is_ok()
        {
            let out_state = classifier.state();
            if classifier.perform(&p_in, 1e-6).is_ok() {
                let in_state = classifier.state();
                if out_state == State::In && in_state != State::In {
                    return Ok(true);
                }
                if in_state == State::In && out_state != State::In {
                    return Ok(false);
                }
            }
        }
    }
    let du = [d1.d1u.x(), d1.d1u.y(), d1.d1u.z()];
    let dv = [d1.d1v.x(), d1.d1v.y(), d1.d1v.z()];
    let mut n = cross3(du, dv);
    if orientation == Orientation::Reversed {
        n = n.map(|c| -c);
    }
    Ok(dot3(n, radial) < 0.0)
}

pub(super) fn radial_distance(p: [f64; 3], axis_origin: [f64; 3], axis_dir: [f64; 3]) -> f64 {
    let rel = [
        p[0] - axis_origin[0],
        p[1] - axis_origin[1],
        p[2] - axis_origin[2],
    ];
    let proj = dot3(rel, axis_dir);
    let radial = [
        rel[0] - proj * axis_dir[0],
        rel[1] - proj * axis_dir[1],
        rel[2] - proj * axis_dir[2],
    ];
    dot3(radial, radial).sqrt()
}

pub(super) fn point_coords(point: Point3) -> [f64; 3] {
    [point.x(), point.y(), point.z()]
}

fn distance3(a: [f64; 3], b: [f64; 3]) -> f64 {
    let d = [b[0] - a[0], b[1] - a[1], b[2] - a[2]];
    dot3(d, d).sqrt()
}

pub(super) fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

pub(super) fn cross3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

pub(super) fn normalize3(v: [f64; 3]) -> [f64; 3] {
    let len = dot3(v, v).sqrt();
    if len <= 1e-12 {
        [0.0, 0.0, 1.0]
    } else {
        [v[0] / len, v[1] / len, v[2] / len]
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
    use martensite_tkg3d::top_abs::ShapeEnum;
    use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeCone;
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
    use oblikovati_model::feature::thread::{ThreadHandedness, ThreadSpecification};

    #[test]
    fn cone_face_detected_as_external_conical_and_rejects_jis_pipe_display() {
        let mut maker = BRepPrimAPIMakeCone::new(0.8331, 0.98, 4.904).unwrap();
        let shape = maker.shape().unwrap().clone();
        let mut faces = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(&shape, ShapeEnum::Face, &mut faces).unwrap();
        let cone_face = (1..=faces.extent())
            .filter_map(|i| faces.find_key(i).ok().cloned())
            .find(is_threadable_surface)
            .expect("expected conical side face");
        let info = inspect_threadable_face(Some(&shape), &cone_face)
            .unwrap()
            .unwrap();
        assert!(info.conical && !info.internal);
        assert!((info.full_depth_cm - 4.906_199_7).abs() < 1e-3, "{info:?}");
        let ok_taper = ResolvedThread {
            body: Vec::new(),
            face: Vec::new(),
            full_depth: false,
            depth_cm: Some(1.0),
            offset_cm: 0.0,
            display_in_model: true,
            spec: ThreadSpecification {
                thread_type: "ISO Taper External".into(),
                size: "R 6".into(),
                designation: "R 6".into(),
                class: "6g".into(),
                handedness: ThreadHandedness::RightHand,
            },
        };
        assert!(validate_thread_compatibility(&info, &ok_taper).is_ok());
        assert!(
            !build_thread_overlay_triangles(&cone_face, &info, &ok_taper)
                .unwrap()
                .is_empty()
        );
        let bad_jis = ResolvedThread {
            spec: ThreadSpecification {
                thread_type: "JIS Pipe Threads".into(),
                size: "16.662".into(),
                designation: "G 3/8".into(),
                class: "B".into(),
                handedness: ThreadHandedness::RightHand,
            },
            ..ok_taper
        };
        assert!(validate_thread_compatibility(&info, &bad_jis).is_err());
        assert!(
            validate_thread_compatibility(
                &info,
                &ResolvedThread {
                    display_in_model: false,
                    ..bad_jis
                }
            )
            .is_ok()
        );
    }
}
