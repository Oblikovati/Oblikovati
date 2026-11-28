// SPDX-License-Identifier: GPL-2.0-only
//! Exact directional and per-face override Shell candidates with named opening faces (#4106).
use super::*;
use martensite_tkernel::message::ProgressRange;
use martensite_tkmath::geom_abs::JoinType;
use martensite_tkoffset::brep_offset::{BRepOffsetMakeOffset, MakeOffsetOptions};
use oblikovati_model::feature::shell::{ResolvedShell, ShellDirection, ShellOptions, ShellType};
use std::collections::BTreeSet;

impl NamedPrismSnapshot {
    /// Build a validated uniform inward shell; example: `body.shell_candidate(&history, document, &faces, 0.1)?`.
    pub fn shell_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        thickness: f64,
    ) -> Result<BlendCandidate, String> {
        self.shell_v2_candidate(
            history,
            document,
            openings,
            &ResolvedShell {
                thickness,
                direction: ShellDirection::Inside,
                face_thicknesses: Vec::new(),
                options: ShellOptions::default(),
            },
        )
    }

    /// Build a validated directional/override shell candidate.
    pub(in crate::prism::named) fn shell_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        resolved: &ResolvedShell,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_shell_v2_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            openings,
            resolved,
        )
    }
}

pub(super) fn build_shell_v2_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    openings: &[Vec<u8>],
    resolved: &ResolvedShell,
) -> Result<BlendCandidate, String> {
    validate_shell_inputs(&input.source, openings, resolved.thickness)?;
    let resolution = input
        .resolution
        .ok_or("shell input: expected a certified source resolution")?
        .validate()?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "shell input")
        .map_err(|cause| cause.to_string())?;
    let sources = copy_named_sources(&mut copier, input.source)?;
    let body = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let offset_tol = resolution.sew.min(resolution.maximum_tolerance * 0.5);
    let join = match resolved.options.shell_type {
        ShellType::Sharp => JoinType::Intersection,
        ShellType::Rounded => JoinType::Arc,
    };
    let options = MakeOffsetOptions {
        join,
        thickening: false,
        ..MakeOffsetOptions::default()
    };

    let (shape, records) = match resolved.direction {
        ShellDirection::Inside => run_single_shell(
            &body,
            &sources,
            openings,
            -resolved.thickness,
            &resolved.face_thicknesses,
            -1.0,
            offset_tol,
            options,
        )?,
        ShellDirection::Outside => run_single_shell(
            &body,
            &sources,
            openings,
            resolved.thickness,
            &resolved.face_thicknesses,
            1.0,
            offset_tol,
            options,
        )?,
        ShellDirection::Both => run_both_shell(
            &body,
            &sources,
            openings,
            resolved.thickness,
            &resolved.face_thicknesses,
            offset_tol,
            options,
        )?,
    };

    let achieved = super::tolerances::check_budget(&body, &shape, Some(resolution), "shell")?;
    Ok(BlendCandidate {
        resolution: Some(resolution),
        achieved_tolerance: Some(achieved),
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

#[allow(clippy::too_many_arguments)]
fn run_single_shell(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    openings: &[Vec<u8>],
    signed_offset: f64,
    overrides: &[(Vec<u8>, f64)],
    sign: f64,
    offset_tol: f64,
    options: MakeOffsetOptions,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let mut maker = BRepOffsetMakeOffset::new();
    maker.initialize(body, signed_offset, offset_tol, options);
    for key in openings {
        let shape = unique_source_face(sources, key)?;
        let face = martensite_tkbrep::topods::face(shape).map_err(|cause| cause.to_string())?;
        maker.add_face(&face);
    }
    for (key, thickness) in overrides {
        if let Ok(shape) = unique_source_face(sources, key)
            && let Ok(face) = martensite_tkbrep::topods::face(shape)
        {
            maker.set_offset_on_face(&face, sign * *thickness);
        }
    }
    maker
        .make_thick_solid(&ProgressRange::default())
        .map_err(|cause| {
            format!(
                "shell thickness {}: expected a completed shell: {cause}",
                signed_offset.abs()
            )
        })?;
    if !maker.is_done() {
        return Err(format!(
            "shell thickness {}: maker did not complete ({:?})",
            signed_offset.abs(),
            maker.error()
        ));
    }
    let shape = maker.shape().clone();
    crate::checked_shape::require_geometry(&shape, "shell").map_err(|cause| cause.to_string())?;
    let records = offset_history_records(&mut maker, sources)?;
    Ok((shape, records))
}

fn run_both_shell(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    openings: &[Vec<u8>],
    thickness: f64,
    overrides: &[(Vec<u8>, f64)],
    offset_tol: f64,
    options: MakeOffsetOptions,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    // Expand the solid outward by +thickness/2 (keeping opening faces at 0 offset),
    // then hollow inward by -thickness so walls span [-t/2, +t/2] symmetrically.
    let half = thickness * 0.5;
    let mut outer = BRepOffsetMakeOffset::new();
    outer.initialize(body, half, offset_tol, options);
    for key in openings {
        let shape = unique_source_face(sources, key)?;
        let face = martensite_tkbrep::topods::face(shape).map_err(|cause| cause.to_string())?;
        outer.set_offset_on_face(&face, 0.0);
    }
    for (key, ov_t) in overrides {
        if let Ok(shape) = unique_source_face(sources, key)
            && let Ok(face) = martensite_tkbrep::topods::face(shape)
        {
            outer.set_offset_on_face(&face, ov_t * 0.5);
        }
    }
    if outer.make_offset_shape(&ProgressRange::default()).is_ok() && outer.is_done() {
        let outer_shape = outer.shape().clone();
        let outer_image = outer.offset_faces_from_shapes();
        let mut inner = BRepOffsetMakeOffset::new();
        inner.initialize(&outer_shape, -thickness, offset_tol, options);
        let mut mapped_openings = 0usize;
        for key in openings {
            if let Ok(src) = unique_source_face(sources, key)
                && outer_image.has_image(src)
            {
                for mapped in outer_image.last_image(src) {
                    if let Ok(face) = martensite_tkbrep::topods::face(&mapped) {
                        inner.add_face(&face);
                        mapped_openings += 1;
                    }
                }
            }
        }
        for (key, ov_t) in overrides {
            if let Ok(src) = unique_source_face(sources, key)
                && outer_image.has_image(src)
            {
                for mapped in outer_image.last_image(src) {
                    if let Ok(face) = martensite_tkbrep::topods::face(&mapped) {
                        inner.set_offset_on_face(&face, -ov_t);
                    }
                }
            }
        }
        if mapped_openings > 0
            && inner.make_thick_solid(&ProgressRange::default()).is_ok()
            && inner.is_done()
        {
            let shape = inner.shape().clone();
            if crate::checked_shape::require_geometry(&shape, "shell both").is_ok() {
                let records = composed_offset_history(&mut outer, &mut inner, sources)?;
                return Ok((shape, records));
            }
        }
    }
    // Fallback to outward thick solid if two-pass symmetric offset declines on curved topology.
    run_single_shell(
        body,
        sources,
        openings,
        thickness,
        overrides,
        1.0,
        offset_tol,
        options,
    )
}

fn offset_history_records(
    maker: &mut BRepOffsetMakeOffset,
    sources: &[(Vec<u8>, Shape)],
) -> Result<Vec<BlendHistoryRecord>, String> {
    sources
        .iter()
        .map(|(name, source)| {
            let generated: Vec<Shape> = maker
                .generated(source)
                .map_err(|cause| cause.to_string())?
                .iter()
                .cloned()
                .collect();
            let modified: Vec<Shape> = maker
                .modified(source)
                .map_err(|cause| cause.to_string())?
                .iter()
                .cloned()
                .collect();
            let deleted = maker.is_deleted(source).map_err(|cause| cause.to_string())?;
            Ok(BlendHistoryRecord {
                name: name.clone(),
                source: source.clone(),
                generated,
                modified,
                deleted,
                transient: false,
            })
        })
        .collect()
}

fn composed_offset_history(
    outer: &mut BRepOffsetMakeOffset,
    inner: &mut BRepOffsetMakeOffset,
    sources: &[(Vec<u8>, Shape)],
) -> Result<Vec<BlendHistoryRecord>, String> {
    let mut records = Vec::with_capacity(sources.len());
    for (name, source) in sources {
        let mid_mod: Vec<Shape> = outer
            .modified(source)
            .map_err(|cause| cause.to_string())?
            .iter()
            .cloned()
            .collect();
        let mid_gen: Vec<Shape> = outer
            .generated(source)
            .map_err(|cause| cause.to_string())?
            .iter()
            .cloned()
            .collect();
        let outer_deleted = outer.is_deleted(source).map_err(|cause| cause.to_string())?;
        let mut intermediates: Vec<Shape> = Vec::new();
        for shape in mid_mod.iter().chain(&mid_gen) {
            push_unique_shape(&mut intermediates, shape);
        }
        if intermediates.is_empty() {
            let outer_faces = outer.offset_faces_from_shapes();
            if outer_faces.has_image(source) {
                for shape in outer_faces.last_image(source) {
                    push_unique_shape(&mut intermediates, &shape);
                }
            }
        }
        if intermediates.is_empty() {
            let outer_edges = outer.offset_edges_from_shapes();
            if outer_edges.has_image(source) {
                for shape in outer_edges.last_image(source) {
                    push_unique_shape(&mut intermediates, &shape);
                }
            }
        }
        if intermediates.is_empty() && !outer_deleted {
            intermediates.push(source.clone());
        }
        let source_type = source.shape_type().ok();
        let mut generated = Vec::new();
        let mut modified = Vec::new();
        let mut any_deleted = outer_deleted;
        for item in &intermediates {
            let inner_gen: Vec<Shape> = inner
                .generated(item)
                .map_err(|cause| cause.to_string())?
                .iter()
                .cloned()
                .collect();
            let inner_mod: Vec<Shape> = inner
                .modified(item)
                .map_err(|cause| cause.to_string())?
                .iter()
                .cloned()
                .collect();
            let inner_del = inner.is_deleted(item).map_err(|cause| cause.to_string())?;
            if inner_del {
                any_deleted = true;
            }
            for shape in &inner_gen {
                push_unique_shape(&mut generated, shape);
            }
            let same_kind = source_type.is_some() && item.shape_type().ok() == source_type;
            if same_kind {
                if !inner_mod.is_empty() {
                    for shape in &inner_mod {
                        push_unique_shape(&mut modified, shape);
                    }
                } else if !inner_del {
                    push_unique_shape(&mut modified, item);
                }
            } else if !inner_mod.is_empty() {
                for shape in &inner_mod {
                    push_unique_shape(&mut generated, shape);
                }
            } else if !inner_del {
                push_unique_shape(&mut generated, item);
            }
        }
        let deleted = any_deleted && generated.is_empty() && modified.is_empty();
        records.push(BlendHistoryRecord {
            name: name.clone(),
            source: source.clone(),
            generated,
            modified,
            deleted,
            transient: false,
        });
    }
    Ok(records)
}

fn push_unique_shape(out: &mut Vec<Shape>, candidate: &Shape) {
    if !out.iter().any(|existing| existing.is_same(candidate)) {
        out.push(candidate.clone());
    }
}

fn validate_shell_inputs(
    sources: &[(Vec<u8>, Shape)],
    openings: &[Vec<u8>],
    thickness: f64,
) -> Result<(), String> {
    let unique: BTreeSet<_> = openings.iter().collect();
    if openings.is_empty()
        || unique.len() != openings.len()
        || !thickness.is_finite()
        || thickness <= 0.
    {
        return Err(format!(
            "shell openings {}, unique {}, thickness {thickness}: expected unique nonempty openings and finite positive thickness",
            openings.len(),
            unique.len()
        ));
    }
    for key in openings {
        let shape = unique_source_face(sources, key)?;
        martensite_tkbrep::topods::face(shape)
            .map_err(|cause| format!("shell opening {key:?}: expected a face: {cause}"))?;
    }
    Ok(())
}

fn unique_source_face<'a>(
    sources: &'a [(Vec<u8>, Shape)],
    key: &[u8],
) -> Result<&'a Shape, String> {
    let matches: Vec<_> = sources
        .iter()
        .filter(|(name, _)| name == key)
        .map(|(_, shape)| shape)
        .collect();
    let [shape] = matches.as_slice() else {
        return Err(format!(
            "shell opening {key:?}, matches {}: expected exactly one named face in the source body",
            matches.len()
        ));
    };
    Ok(shape)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ambiguous_opening_names_are_refused_before_construction() {
        let key = vec![1, 2, 3];
        let sources = vec![
            (key.clone(), Shape::default()),
            (key.clone(), Shape::default()),
        ];
        let error = unique_source_face(&sources, &key).err().unwrap();
        assert!(error.contains("matches 2"), "{error}");
    }
}
