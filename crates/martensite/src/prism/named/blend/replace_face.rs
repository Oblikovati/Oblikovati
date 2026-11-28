// SPDX-License-Identifier: GPL-2.0-only
//! Detached B-rep face replacement with planar extension/trimming and general surface splitting (#3918).
use super::replace_face_geom::{
    box_center, box_diagonal, boxes_overlap_in_tangent_plane, build_large_plane_face,
    combined_bounds, dot, face_plane_point_normal, projected_extent, shape_bounds,
    signed_plane_distances, solid_center_of_mass, sub,
};
use super::thicken_offset_chain::{
    collect_shapes, ensure_positive_solid, planar_outward_normal, translate_shape,
};
use super::*;
use martensite_tkbo::{
    bop_algo::Operation,
    brep_algo_api::{BRepAlgoAPIBooleanOperation, BRepAlgoAPISplitter},
};
use martensite_tkbrep::topods::Shape;
use martensite_tkernel::ncollection::List;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::Vector3;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeShape, brep_check::BRepCheckAnalyzer,
};
use oblikovati_model::feature::{FeatureId, replace_face::ResolvedReplaceFace};
use std::collections::BTreeSet;

impl NamedSolid {
    /// Replace one or more faces on this solid with replacement surface/solid faces (#3918).
    pub(in crate::prism::named) fn evaluate_replace_face_result(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        new_bodies: &[&NamedSolid],
        input: &ResolvedReplaceFace,
    ) -> Result<NamedSolid, String> {
        let resolution = match self {
            Self::Prism(body) => body.certification.as_ref().map(|c| c.resolution),
            Self::Derived(body) => body.resolution(),
        };
        let existing_keys = expand_existing_faces(
            self,
            history,
            document,
            &input.existing_faces,
            input.automatic_face_chain,
        )?;
        let (body_copy, existing_shapes) =
            copy_body_and_faces(self, history, document, &existing_keys)?;
        let new_shapes = copy_replacement_faces(new_bodies, history, document, &input.new_faces)?;

        let result_shape = replace_faces_brep(&body_copy, &existing_shapes, &new_shapes)?;
        let valid = BRepCheckAnalyzer::new(&result_shape, true, false, false)
            .and_then(|analyzer| analyzer.is_valid())
            .map_err(|cause| {
                format!("Create Replace Face feature failed: BRep check error ({cause})")
            })?;
        if !valid {
            return Err(
                "Create Replace Face feature failed: resulting solid failed BRep validation".into(),
            );
        }

        let primary_body = self.body_reference_key(history, document)?.to_vec();
        let deleted = collect_prior_non_body_keys(self, history, document, &primary_body)?;
        let topology = BlendNamedTopology::from_tagged_preserving_body(
            history,
            document,
            feature,
            "replace-face",
            primary_body,
            deleted,
            result_shape,
            resolution,
        )?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
}

fn expand_existing_faces(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    seeds: &[Vec<u8>],
    automatic_face_chain: bool,
) -> Result<Vec<Vec<u8>>, String> {
    if !automatic_face_chain {
        return Ok(seeds.to_vec());
    }
    let mut seen = BTreeSet::new();
    let mut expanded = Vec::new();
    for seed in seeds {
        for key in body.tangent_face_chain(history, document, seed)? {
            if seen.insert(key.clone()) {
                expanded.push(key);
            }
        }
    }
    Ok(expanded)
}

fn copy_body_and_faces(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    face_keys: &[Vec<u8>],
) -> Result<(Shape, Vec<Shape>), String> {
    let body_shape = body.native_shape(history, document)?;
    let mut copier = crate::checked_shape::copy_shape(body_shape, "replace face target")
        .map_err(|cause| cause.to_string())?;
    let copied_body = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let mut copied_faces = Vec::with_capacity(face_keys.len());
    for key in face_keys {
        let source = body.native_face(history, document, key)?;
        let copies = copier.modified(source).map_err(|cause| cause.to_string())?;
        let Some(copied) = copies.first() else {
            return Err(format!(
                "Create Replace Face feature failed: missing detached copy for face {key:?}"
            ));
        };
        copied_faces.push(copied.clone());
    }
    Ok((copied_body, copied_faces))
}

fn copy_replacement_faces(
    new_bodies: &[&NamedSolid],
    history: &DocumentEditHistory,
    document: DocumentGuid,
    new_face_keys: &[Vec<u8>],
) -> Result<Vec<Shape>, String> {
    if new_bodies.len() != new_face_keys.len() || new_bodies.is_empty() {
        return Err(
            "Create Replace Face feature failed: expected at least one replacement face".into(),
        );
    }
    let mut out = Vec::with_capacity(new_face_keys.len());
    for (owner, key) in new_bodies.iter().zip(new_face_keys) {
        let source = owner.native_face(history, document, key)?;
        let mut copier = crate::checked_shape::copy_shape(source, "replace face new surface")
            .map_err(|cause| cause.to_string())?;
        let copied = copier.shape().map_err(|cause| cause.to_string())?.clone();
        out.push(copied);
    }
    Ok(out)
}

fn collect_prior_non_body_keys(
    body: &NamedSolid,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    primary_body: &[u8],
) -> Result<Vec<Vec<u8>>, String> {
    let mut keys = body.reference_keys(history, document)?;
    keys.extend(body.subentity_reference_keys(history, document)?);
    if let NamedSolid::Derived(derived) = body {
        keys.extend(derived.live_keys().iter().cloned());
        keys.extend(derived.deleted_keys().iter().cloned());
    }
    keys.retain(|k| k.as_slice() != primary_body);
    keys.sort();
    keys.dedup();
    Ok(keys)
}

fn replace_faces_brep(
    body: &Shape,
    existing_faces: &[Shape],
    new_faces: &[Shape],
) -> Result<Shape, String> {
    if collect_shapes(body, ShapeEnum::Solid)?.is_empty() {
        return Err("Create Replace Face feature failed: target body must be a solid".into());
    }
    let all_faces = collect_shapes(body, ShapeEnum::Face)?;
    if existing_faces.len() >= all_faces.len() {
        return Err(
            "Create Replace Face feature failed: cannot replace every face of the target solid"
                .into(),
        );
    }
    if !has_adjacent_remaining_face(&all_faces, existing_faces)? {
        return Err(
            "Create Replace Face feature failed: selected faces have no adjacent faces to extend or trim"
                .into(),
        );
    }
    if let Some((n_old, p_new, n_new)) = common_planar_replacement(existing_faces, new_faces)? {
        return replace_planar_faces(body, existing_faces, new_faces, n_old, p_new, n_new);
    }
    replace_general_faces_by_split(body, existing_faces, new_faces)
}

fn has_adjacent_remaining_face(all_faces: &[Shape], existing: &[Shape]) -> Result<bool, String> {
    let mut existing_edges = Vec::new();
    for face in existing {
        existing_edges.extend(collect_shapes(face, ShapeEnum::Edge)?);
    }
    for candidate in all_faces {
        if existing.iter().any(|face| face.is_same(candidate)) {
            continue;
        }
        let edges = collect_shapes(candidate, ShapeEnum::Edge)?;
        if edges
            .iter()
            .any(|edge| existing_edges.iter().any(|shared| edge.is_same(shared)))
        {
            return Ok(true);
        }
    }
    Ok(false)
}

type PlanarReplacement = ([f64; 3], [f64; 3], [f64; 3]);

fn common_planar_replacement(
    existing_faces: &[Shape],
    new_faces: &[Shape],
) -> Result<Option<PlanarReplacement>, String> {
    let Some(first_existing) = existing_faces.first() else {
        return Ok(None);
    };
    let Some(n_old) = planar_outward_normal(first_existing)? else {
        return Ok(None);
    };
    for face in &existing_faces[1..] {
        let Some(normal) = planar_outward_normal(face)? else {
            return Ok(None);
        };
        if dot(n_old, normal) < 0.999 {
            return Ok(None);
        }
    }
    let Some(first_new) = new_faces.first() else {
        return Ok(None);
    };
    let Some((p_new, raw_n_new)) = face_plane_point_normal(first_new)? else {
        return Ok(None);
    };
    let n_new = if dot(raw_n_new, n_old) < 0.0 {
        [-raw_n_new[0], -raw_n_new[1], -raw_n_new[2]]
    } else {
        raw_n_new
    };
    for face in &new_faces[1..] {
        let Some((pt, n)) = face_plane_point_normal(face)? else {
            return Ok(None);
        };
        if dot(n, n_new).abs() < 0.999 || dot(sub(pt, p_new), n_new).abs() > 1.0e-4 {
            return Ok(None);
        }
    }
    Ok(Some((n_old, p_new, n_new)))
}

fn replace_planar_faces(
    body: &Shape,
    existing_faces: &[Shape],
    new_faces: &[Shape],
    n_old: [f64; 3],
    p_new: [f64; 3],
    n_new: [f64; 3],
) -> Result<Shape, String> {
    let alignment = dot(n_old, n_new);
    if alignment < 0.05 {
        return Err(
            "Create Replace Face feature failed: replacement plane is nearly perpendicular to the existing face"
                .into(),
        );
    }
    let existing_box = combined_bounds(existing_faces)?;
    let new_box = combined_bounds(new_faces)?;
    if !boxes_overlap_in_tangent_plane(existing_box, new_box, n_old) {
        return Err(
            "Create Replace Face feature failed: replacement surface does not overlap the selected face"
                .into(),
        );
    }
    let (t_min, t_max) = signed_plane_distances(existing_box, p_new, n_new, alignment);
    if t_min.abs() < 1.0e-7 && t_max.abs() < 1.0e-7 {
        return Err(
            "Create Replace Face feature failed: replacement surface is coplanar with the selected face"
                .into(),
        );
    }
    let body_box = shape_bounds(body)?;
    let body_span_n = projected_extent(body_box, n_old);
    if alignment > 0.9999 && t_min > 1.0e-7 {
        let distance = 0.5 * (t_min + t_max);
        return fuse_outward_prisms(body, existing_faces, n_old, distance, body_span_n);
    }
    let working = if t_max > 1.0e-7 {
        let diag = box_diagonal(body_box);
        let ext = t_max + (diag * 0.25).max(0.25);
        fuse_outward_prisms(body, existing_faces, n_old, ext, body_span_n)?
    } else {
        body.clone()
    };
    let span = box_diagonal(body_box) * 2.5 + t_max.abs() + 1.0;
    let cutter = build_large_plane_face(p_new, n_new, span)?;
    split_and_keep_interior(&working, &cutter, p_new, n_new)
}

fn fuse_outward_prisms(
    body: &Shape,
    existing_faces: &[Shape],
    normal: [f64; 3],
    distance: f64,
    body_span_n: f64,
) -> Result<Shape, String> {
    let embed = (distance * 0.25)
        .min(0.01)
        .min((body_span_n * 0.15).max(1.0e-4));
    let mut current = ensure_positive_solid(body.clone(), "replace face target")?;
    for face in existing_faces {
        let shifted = translate_shape(
            face,
            [-normal[0] * embed, -normal[1] * embed, -normal[2] * embed],
        )?;
        let vec = Vector3::new(
            normal[0] * (distance + embed),
            normal[1] * (distance + embed),
            normal[2] * (distance + embed),
        );
        let mut prism = BRepPrimAPIMakePrism::new(&shifted, vec, true, true)
            .map_err(|cause| format!("Create Replace Face feature failed: {cause}"))?;
        let prism_shape = prism
            .shape()
            .map_err(|cause| format!("Create Replace Face feature failed: {cause}"))?
            .clone();
        let prism_solid = ensure_positive_solid(prism_shape, "replace face extension")?;
        let mut boolean = BRepAlgoAPIBooleanOperation::new();
        let mut args = List::default();
        args.append(current);
        let mut tools = List::default();
        tools.append(prism_solid);
        boolean.set_arguments(&args);
        boolean.set_tools(&tools);
        boolean.set_operation(Operation::Fuse);
        boolean
            .build()
            .map_err(|cause| format!("Create Replace Face feature failed: {cause}"))?;
        if !boolean.is_done() || boolean.has_errors() {
            return Err("Create Replace Face feature failed: solid extension Fuse failed".into());
        }
        current = ensure_positive_solid(boolean.shape().clone(), "replace face fused solid")?;
    }
    Ok(current)
}

fn split_and_keep_interior(
    solid: &Shape,
    cutter: &Shape,
    p_new: [f64; 3],
    n_new: [f64; 3],
) -> Result<Shape, String> {
    let mut splitter = BRepAlgoAPISplitter::new();
    let mut args = List::default();
    args.append(solid.clone());
    let mut tools = List::default();
    tools.append(cutter.clone());
    splitter.set_arguments(&args);
    splitter.set_tools(&tools);
    splitter
        .build()
        .map_err(|cause| format!("Create Replace Face feature failed: {cause}"))?;
    if !splitter.is_done() || splitter.has_errors() {
        return Err(
            "Create Replace Face feature failed: replacement plane did not split the solid".into(),
        );
    }
    let pieces = collect_shapes(splitter.shape(), ShapeEnum::Solid)?;
    if pieces.len() < 2 {
        return Err(
            "Create Replace Face feature failed: replacement plane does not intersect the target solid"
                .into(),
        );
    }
    for piece in pieces {
        let Ok(candidate) = ensure_positive_solid(piece, "replace face trimmed solid") else {
            continue;
        };
        let com = solid_center_of_mass(&candidate)?;
        if dot(sub(com, p_new), n_new) < -1.0e-7 {
            return Ok(candidate);
        }
    }
    Err(
        "Create Replace Face feature failed: replacement surface eliminates the target solid"
            .into(),
    )
}

fn replace_general_faces_by_split(
    body: &Shape,
    existing_faces: &[Shape],
    new_faces: &[Shape],
) -> Result<Shape, String> {
    let mut splitter = BRepAlgoAPISplitter::new();
    let mut args = List::default();
    args.append(body.clone());
    let mut tools = List::default();
    for face in new_faces {
        tools.append(face.clone());
    }
    splitter.set_arguments(&args);
    splitter.set_tools(&tools);
    splitter
        .build()
        .map_err(|cause| format!("Create Replace Face feature failed: {cause}"))?;
    if !splitter.is_done() || splitter.has_errors() {
        return Err(
            "Create Replace Face feature failed: replacement surface cannot trim the target solid"
                .into(),
        );
    }
    let pieces = collect_shapes(splitter.shape(), ShapeEnum::Solid)?;
    if pieces.len() < 2 {
        return Err(
            "Create Replace Face feature failed: replacement surface does not intersect the target solid"
                .into(),
        );
    }
    let existing_center = box_center(combined_bounds(existing_faces)?);
    let mut best: Option<(f64, Shape)> = None;
    for piece in pieces {
        let Ok(candidate) = ensure_positive_solid(piece, "replace face candidate") else {
            continue;
        };
        let com = solid_center_of_mass(&candidate)?;
        let dist2 = dot(sub(com, existing_center), sub(com, existing_center));
        if best.as_ref().is_none_or(|(best_d, _)| dist2 > *best_d) {
            best = Some((dist2, candidate));
        }
    }
    best.map(|(_, solid)| solid).ok_or_else(|| {
        "Create Replace Face feature failed: no valid solid remained after replacement".into()
    })
}
