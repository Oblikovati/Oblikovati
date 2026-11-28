// SPDX-License-Identifier: GPL-2.0-only
//! Cosmetic helical/annular thread display overlay generation for cylindrical and conical faces (#3918).
use super::thread::{ThreadableFaceInfo, cross3, dot3, normalize3, point_coords, radial_distance};
use crate::DisplayTriangle;
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorSurface,
    topods::{self, Shape},
};
use martensite_tkg3d::adaptor3d::Adaptor3dSurface;
use oblikovati_model::feature::thread::{ResolvedThread, ThreadHandedness, ThreadSpecification};
use std::f64::consts::TAU;

/// Build cosmetic thread indicator bands on a cylindrical or conical face spanning
/// `[offset_cm, offset_cm + depth_cm]`. Returns empty when display is disabled or offset exceeds
/// the face length.
pub(crate) fn build_thread_overlay_triangles(
    face: &Shape,
    info: &ThreadableFaceInfo,
    input: &ResolvedThread,
) -> Result<Vec<DisplayTriangle>, String> {
    if !input.display_in_model {
        return Ok(Vec::new());
    }
    let full = info.full_depth_cm;
    let start_cm = input.offset_cm;
    if start_cm >= full - 1e-6 {
        return Ok(Vec::new());
    }
    let requested_depth = if input.full_depth {
        full
    } else {
        input.depth_cm.unwrap_or(full)
    };
    let end_cm = (start_cm + requested_depth).min(full);
    if end_cm <= start_cm + 1e-6 {
        return Ok(Vec::new());
    }
    let native = topods::face(face).map_err(|cause| cause.to_string())?;
    let adaptor =
        BRepAdaptorSurface::with_face(&native, true).map_err(|cause| cause.to_string())?;
    let u_min = adaptor.first_u_parameter().map_err(|e| e.to_string())?;
    let u_max = adaptor.last_u_parameter().map_err(|e| e.to_string())?;
    let v_min = adaptor.first_v_parameter().map_err(|e| e.to_string())?;
    let v_max = adaptor.last_v_parameter().map_err(|e| e.to_string())?;
    let (v_start_end, v_far_end) =
        thread_start_v_ends(&adaptor, info, 0.5 * (u_min + u_max), v_min, v_max)?;

    let pitch_cm = pitch_cm_from_spec(&input.spec);
    let span_cm = end_cm - start_cm;
    let raw_count = (span_cm / pitch_cm).floor().max(1.0) as usize;
    let band_count = raw_count.clamp(1, 64);
    let effective_pitch = if raw_count > 64 {
        span_cm / 64.0
    } else {
        pitch_cm
    };
    let band_half_w = (effective_pitch * 0.14).min(span_cm * 0.2);
    let helix_sign = match input.spec.handedness {
        ThreadHandedness::RightHand => 1.0,
        ThreadHandedness::LeftHand => -1.0,
    };
    let segments: usize = 24;
    let lift_cm = 0.003;
    let (axis_origin, axis_dir) = surface_axis(&adaptor, info)?;
    let v_span = v_far_end - v_start_end;

    let mut triangles = Vec::with_capacity(band_count * segments * 2);
    for band_idx in 0..band_count {
        let base_s = start_cm + (band_idx as f64 + 0.5) * effective_pitch;
        if base_s >= end_cm - 1e-6 {
            break;
        }
        for seg in 0..segments {
            let t0 = seg as f64 / segments as f64;
            let t1 = (seg + 1) as f64 / segments as f64;
            let u0 = u_min + t0 * (u_max - u_min);
            let u1 = u_min + t1 * (u_max - u_min);
            let helix0 = helix_sign * 0.20 * effective_pitch * ((u0 - u_min) / TAU - 0.5);
            let helix1 = helix_sign * 0.20 * effective_pitch * ((u1 - u_min) / TAU - 0.5);
            let s0_lo = (base_s + helix0 - band_half_w).clamp(start_cm, end_cm);
            let s0_hi = (base_s + helix0 + band_half_w).clamp(start_cm, end_cm);
            let s1_lo = (base_s + helix1 - band_half_w).clamp(start_cm, end_cm);
            let s1_hi = (base_s + helix1 + band_half_w).clamp(start_cm, end_cm);
            let v0_lo = v_start_end + (s0_lo / full) * v_span;
            let v0_hi = v_start_end + (s0_hi / full) * v_span;
            let v1_lo = v_start_end + (s1_lo / full) * v_span;
            let v1_hi = v_start_end + (s1_hi / full) * v_span;

            let (p00, n00) = lifted_point(
                &adaptor,
                info.internal,
                axis_origin,
                axis_dir,
                u0,
                v0_lo,
                lift_cm,
            )?;
            let (p01, n01) = lifted_point(
                &adaptor,
                info.internal,
                axis_origin,
                axis_dir,
                u0,
                v0_hi,
                lift_cm,
            )?;
            let (p10, n10) = lifted_point(
                &adaptor,
                info.internal,
                axis_origin,
                axis_dir,
                u1,
                v1_lo,
                lift_cm,
            )?;
            let (p11, n11) = lifted_point(
                &adaptor,
                info.internal,
                axis_origin,
                axis_dir,
                u1,
                v1_hi,
                lift_cm,
            )?;

            triangles.push(DisplayTriangle {
                positions: [p00, p10, p11],
                normals: Some([n00, n10, n11]),
            });
            triangles.push(DisplayTriangle {
                positions: [p00, p11, p01],
                normals: Some([n00, n11, n01]),
            });
        }
    }
    Ok(triangles)
}

fn thread_start_v_ends(
    adaptor: &BRepAdaptorSurface,
    info: &ThreadableFaceInfo,
    u_mid: f64,
    v_min: f64,
    v_max: f64,
) -> Result<(f64, f64), String> {
    if !info.conical {
        return Ok((v_min, v_max));
    }
    let cone = adaptor.cone().map_err(|cause| cause.to_string())?;
    let axis = cone.axis();
    let origin = point_coords(axis.location());
    let dir = *axis.direction().coordinates();
    let p0 = point_coords(adaptor.value(u_mid, v_min).map_err(|e| e.to_string())?);
    let p1 = point_coords(adaptor.value(u_mid, v_max).map_err(|e| e.to_string())?);
    if radial_distance(p0, origin, dir) <= radial_distance(p1, origin, dir) {
        Ok((v_min, v_max))
    } else {
        Ok((v_max, v_min))
    }
}

/// Parse thread pitch in working units (cm) from a specification designation.
pub(crate) fn pitch_cm_from_spec(spec: &ThreadSpecification) -> f64 {
    if let Some((_, rhs)) = spec.designation.split_once('x')
        && let Ok(mm) = rhs.trim().parse::<f64>()
        && mm.is_finite()
        && mm > 0.0
    {
        return mm * 0.1;
    }
    if spec.thread_type.eq_ignore_ascii_case("ISO Taper External") {
        return 0.2309;
    }
    if spec.thread_type.eq_ignore_ascii_case("JIS Pipe Threads") {
        return 0.1337;
    }
    0.125
}

fn surface_axis(
    adaptor: &BRepAdaptorSurface,
    info: &ThreadableFaceInfo,
) -> Result<([f64; 3], [f64; 3]), String> {
    let axis = if info.conical {
        adaptor.cone().map_err(|e| e.to_string())?.axis()
    } else {
        adaptor.cylinder().map_err(|e| e.to_string())?.axis()
    };
    Ok((
        point_coords(axis.location()),
        *axis.direction().coordinates(),
    ))
}

fn lifted_point(
    adaptor: &BRepAdaptorSurface,
    internal: bool,
    axis_origin: [f64; 3],
    axis_dir: [f64; 3],
    u: f64,
    v: f64,
    lift: f64,
) -> Result<([f64; 3], [f64; 3]), String> {
    let d1 = adaptor.d1(u, v).map_err(|e| e.to_string())?;
    let p = point_coords(d1.point);
    let du = [d1.d1u.x(), d1.d1u.y(), d1.d1u.z()];
    let dv = [d1.d1v.x(), d1.d1v.y(), d1.d1v.z()];
    let mut n = normalize3(cross3(du, dv));
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
    let points_inward = dot3(n, radial) < 0.0;
    if points_inward != internal {
        n = n.map(|c| -c);
    }
    let lifted = [p[0] + n[0] * lift, p[1] + n[1] * lift, p[2] + n[2] * lift];
    Ok((lifted, n))
}
