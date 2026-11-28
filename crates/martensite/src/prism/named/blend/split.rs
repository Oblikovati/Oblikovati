// SPDX-License-Identifier: GPL-2.0-only
//! Detached B-rep face and solid splitting using Martensite `BRepAlgoAPISplitter` (#3918).
use super::{
    split_tool::{BuiltSplitTool, build_section_edges, build_split_tool},
    *,
};
use martensite_tkbo::brep_algo_api::BRepAlgoAPISplitter;
use martensite_tkbrep::{topexp, topods, toptools::IndexedMapOfShape};
use martensite_tkernel::ncollection::List;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeSolid, brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp,
};
use oblikovati_model::{
    feature::split::{ResolvedSplit, ResolvedSplitTarget, SplitKeepSide, SplitMode},
    identity::DerivedReferenceName,
};

impl NamedPrismSnapshot {
    /// Construct detached split candidate(s) from a named prism solid (#3918).
    pub(in crate::prism::named) fn split_candidates(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        input: &ResolvedSplit,
    ) -> Result<Vec<BlendCandidate>, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_split_candidates(
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
            feature,
            input,
        )
    }
}

pub(super) fn build_split_candidates(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
    resolved: &ResolvedSplit,
) -> Result<Vec<BlendCandidate>, String> {
    let BlendInputTopology {
        resolution,
        body,
        source,
        source_bodies,
        body_parent,
        primary_body,
        inherited_deleted,
        inherited_splits,
    } = input;
    let mut copier =
        crate::checked_shape::copy_shape(body, "split input").map_err(|cause| cause.to_string())?;
    let sources = copy_named_sources(&mut copier, source)?;
    let copied_body = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let built_tool = build_split_tool(&copied_body, &resolved.tool)?;
    let meta = BlendInputMeta {
        resolution,
        source_bodies,
        body_parent,
        primary_body,
        inherited_deleted,
        inherited_splits,
        document,
        feature,
    };
    match resolved.mode {
        SplitMode::Faces => build_face_split_candidate(
            &copied_body,
            sources,
            meta,
            history,
            &built_tool,
            &resolved.target,
        )
        .map(|candidate| vec![candidate]),
        SplitMode::Solid => build_solid_split_candidates(
            &copied_body,
            sources,
            meta,
            history,
            &built_tool,
            resolved.keep_side,
        ),
    }
}

struct BlendInputMeta {
    resolution: Option<PrismResolution>,
    source_bodies: Vec<Vec<u8>>,
    body_parent: Vec<u8>,
    primary_body: Vec<u8>,
    inherited_deleted: Vec<Vec<u8>>,
    inherited_splits: SplitTopologyNames,
    document: DocumentGuid,
    feature: FeatureId,
}

fn build_face_split_candidate(
    copied_body: &Shape,
    sources: Vec<(Vec<u8>, Shape)>,
    input: BlendInputMeta,
    history: &DocumentEditHistory,
    built_tool: &BuiltSplitTool,
    target: &ResolvedSplitTarget,
) -> Result<BlendCandidate, String> {
    let shells = collect_entities(copied_body, ShapeEnum::Shell)?;
    if shells.is_empty() {
        return Err("split faces: expected at least one shell in target solid".into());
    }
    let tool_shapes: Vec<Shape> = match target {
        ResolvedSplitTarget::AllFaces => vec![built_tool.surface.clone()],
        ResolvedSplitTarget::Faces(face_keys) => {
            let mut selected = Vec::with_capacity(face_keys.len());
            for key in face_keys {
                let face = named_face_source(&sources, key)?;
                selected.push(face.clone());
            }
            build_section_edges(&selected, &built_tool.surface)?
        }
        ResolvedSplitTarget::Solid => {
            return Err("split faces: expected face targets, got solid target".into());
        }
    };
    let mut splitter = BRepAlgoAPISplitter::new();
    let mut arguments = List::default();
    for shell in &shells {
        arguments.append(shell.clone());
    }
    let mut tools = List::default();
    for tool in &tool_shapes {
        tools.append(tool.clone());
    }
    splitter.set_arguments(&arguments);
    splitter.set_tools(&tools);
    splitter
        .build()
        .map_err(|cause| format!("split faces build: {cause}"))?;
    if !splitter.is_done() || splitter.has_errors() {
        return Err("split faces: BRepAlgoAPISplitter did not complete without errors".into());
    }
    let split_shells = collect_entities(splitter.shape(), ShapeEnum::Shell)?;
    if split_shells.is_empty() {
        return Err("split faces: expected split shell output".into());
    }
    let solid = solid_from_shells(&split_shells)?;
    validate_solid(&solid, "split faces")?;

    let before = crate::checked_shape::analytic_properties(copied_body, "split faces input")
        .map_err(|cause| cause.to_string())?;
    let after = crate::checked_shape::analytic_properties(&solid, "split faces result")
        .map_err(|cause| cause.to_string())?;
    let vol_tol = 1e-4 * before.volume.abs().max(1.0);
    if (after.volume - before.volume).abs() > vol_tol {
        return Err(format!(
            "split faces volume {} from {}: expected unchanged solid volume",
            after.volume, before.volume
        ));
    }
    let before_edges = collect_entities(copied_body, ShapeEnum::Edge)?.len();
    let after_edges = collect_entities(&solid, ShapeEnum::Edge)?.len();
    let before_faces = collect_entities(copied_body, ShapeEnum::Face)?.len();
    let after_faces = collect_entities(&solid, ShapeEnum::Face)?.len();
    if after_faces <= before_faces && after_edges <= before_edges {
        return Err(format!(
            "split faces: tool did not split any selected face (faces {before_faces}->{after_faces}, edges {before_edges}->{after_edges})"
        ));
    }

    let records = splitter_history(&splitter, &sources, &solid, None)?;
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: None,
        shape: solid,
        history: records,
        source_bodies: input.source_bodies.clone(),
        body_parent: input.body_parent.clone(),
        primary_body: input.primary_body.clone(),
        inherited_deleted: input.inherited_deleted.clone(),
        inherited_splits: input.inherited_splits.clone(),
        document: input.document,
        scope: history.evaluation_scope(),
    })
}

fn build_solid_split_candidates(
    copied_body: &Shape,
    sources: Vec<(Vec<u8>, Shape)>,
    input: BlendInputMeta,
    history: &DocumentEditHistory,
    built_tool: &BuiltSplitTool,
    keep_side: SplitKeepSide,
) -> Result<Vec<BlendCandidate>, String> {
    let mut splitter = BRepAlgoAPISplitter::new();
    let mut arguments = List::default();
    arguments.append(copied_body.clone());
    let mut tools = List::default();
    tools.append(built_tool.surface.clone());
    splitter.set_arguments(&arguments);
    splitter.set_tools(&tools);
    splitter
        .build()
        .map_err(|cause| format!("split solid build: {cause}"))?;
    if !splitter.is_done() || splitter.has_errors() {
        return Err("split solid: BRepAlgoAPISplitter did not complete without errors".into());
    }
    let solids = collect_entities(splitter.shape(), ShapeEnum::Solid)?;
    if solids.len() != 2 {
        return Err(format!(
            "split solid produced {} solids: expected splitting plane to divide target into 2 solids",
            solids.len()
        ));
    }
    let before = crate::checked_shape::analytic_properties(copied_body, "split solid input")
        .map_err(|cause| cause.to_string())?;
    let mut classified = Vec::with_capacity(2);
    let mut total_vol = 0.0;
    for solid in solids {
        validate_solid(&solid, "split solid")?;
        let props = crate::checked_shape::analytic_properties(&solid, "split solid half")
            .map_err(|cause| cause.to_string())?;
        if !props.volume.is_finite() || props.volume <= 0.0 {
            return Err(format!(
                "split solid half volume {}: expected positive volume",
                props.volume
            ));
        }
        total_vol += props.volume;
        let signed =
            signed_center_distance(&solid, built_tool.plane_origin, built_tool.plane_normal)?;
        classified.push((signed, solid));
    }
    let vol_tol = 1e-4 * before.volume.abs().max(1.0);
    if (total_vol - before.volume).abs() > vol_tol {
        return Err(format!(
            "split solid combined volume {total_vol} from {}: expected volume conservation",
            before.volume
        ));
    }
    // Sort descending by signed distance: index 0 is SideA (+normal), index 1 is SideB (-normal).
    classified.sort_by(|a, b| b.0.total_cmp(&a.0));
    let [(_, side_a), (_, side_b)] = <[_; 2]>::try_from(classified)
        .map_err(|_| "split solid: expected 2 classified halves".to_string())?;
    let kept: Vec<Shape> = match keep_side {
        SplitKeepSide::Both => vec![side_a, side_b],
        SplitKeepSide::SideA => vec![side_a],
        SplitKeepSide::SideB => vec![side_b],
    };
    let tool_name = DerivedReferenceName::new(
        input.document,
        input.feature.0,
        1,
        "split-tool",
        std::slice::from_ref(&input.body_parent),
    )?
    .encode()?;
    let mut candidates = Vec::with_capacity(kept.len());
    for solid in kept {
        let records = splitter_history(
            &splitter,
            &sources,
            &solid,
            Some((&tool_name, &built_tool.surface)),
        )?;
        candidates.push(BlendCandidate {
            resolution: input.resolution,
            achieved_tolerance: None,
            shape: solid,
            history: records,
            source_bodies: input.source_bodies.clone(),
            body_parent: input.body_parent.clone(),
            primary_body: input.primary_body.clone(),
            inherited_deleted: input.inherited_deleted.clone(),
            inherited_splits: input.inherited_splits.clone(),
            document: input.document,
            scope: history.evaluation_scope(),
        });
    }
    Ok(candidates)
}

fn solid_from_shells(shells: &[Shape]) -> Result<Shape, String> {
    let first = topods::shell(&shells[0]).map_err(|cause| cause.to_string())?;
    let mut maker =
        BRepBuilderAPIMakeSolid::from_shell(&first).map_err(|cause| cause.to_string())?;
    for shell in &shells[1..] {
        let s = topods::shell(shell).map_err(|cause| cause.to_string())?;
        maker.add(&s).map_err(|cause| cause.to_string())?;
    }
    maker
        .solid()
        .map(Shape::from)
        .map_err(|cause| format!("split solid assembly: {cause}"))
}

fn validate_solid(solid: &Shape, label: &str) -> Result<(), String> {
    crate::checked_shape::require_geometry(solid, label).map_err(|cause| cause.to_string())?;
    let valid = BRepCheckAnalyzer::new(solid, true, false, false)
        .and_then(|analyzer| analyzer.is_valid())
        .map_err(|cause| format!("{label} BRepCheck failed: {cause}"))?;
    if !valid {
        return Err(format!(
            "{label}: resulting solid failed BRepCheck validation"
        ));
    }
    Ok(())
}

fn signed_center_distance(
    solid: &Shape,
    origin: [f64; 3],
    normal: [f64; 3],
) -> Result<f64, String> {
    let mut props = GProps::new();
    BRepGProp::volume_properties(solid, &mut props, false, false, false)
        .map_err(|cause| format!("split solid center of mass: {cause}"))?;
    let com = props.centre_of_mass();
    Ok((com.x() - origin[0]) * normal[0]
        + (com.y() - origin[1]) * normal[1]
        + (com.z() - origin[2]) * normal[2])
}

fn splitter_history(
    splitter: &BRepAlgoAPISplitter,
    sources: &[(Vec<u8>, Shape)],
    result_solid: &Shape,
    tool_cap: Option<(&[u8], &Shape)>,
) -> Result<Vec<BlendHistoryRecord>, String> {
    let live = collect_all_subshapes(result_solid)?;
    let in_solid = |candidate: &Shape| live.iter().any(|s| s.is_same(candidate));
    let mut records = Vec::with_capacity(sources.len() + usize::from(tool_cap.is_some()));
    let mut claimed_faces: Vec<Shape> = Vec::new();
    for (name, source) in sources {
        let modified: Vec<Shape> = splitter
            .modified(source)
            .map_err(|cause| cause.to_string())?
            .into_iter()
            .filter(&in_solid)
            .collect();
        let generated: Vec<Shape> = splitter
            .generated(source)
            .map_err(|cause| cause.to_string())?
            .into_iter()
            .filter(&in_solid)
            .collect();
        let present = in_solid(source) || !modified.is_empty() || !generated.is_empty();
        if present {
            if in_solid(source) && matches!(source.shape_type(), Ok(ShapeEnum::Face)) {
                claimed_faces.push(source.clone());
            }
            for s in modified.iter().chain(&generated) {
                if matches!(s.shape_type(), Ok(ShapeEnum::Face)) {
                    claimed_faces.push(s.clone());
                }
            }
        }
        let deleted = !present
            || splitter
                .is_deleted(source)
                .map_err(|cause| cause.to_string())?;
        records.push(BlendHistoryRecord {
            name: name.clone(),
            source: source.clone(),
            generated,
            modified,
            deleted,
            transient: false,
        });
    }
    if let Some((tool_name, tool_shape)) = tool_cap {
        let all_faces = collect_entities(result_solid, ShapeEnum::Face)?;
        let cap_faces: Vec<Shape> = all_faces
            .into_iter()
            .filter(|face| !claimed_faces.iter().any(|claimed| claimed.is_same(face)))
            .collect();
        records.push(BlendHistoryRecord {
            name: tool_name.to_vec(),
            source: tool_shape.clone(),
            generated: cap_faces,
            modified: Vec::new(),
            deleted: false,
            transient: true,
        });
    }
    Ok(records)
}

fn named_face_source<'a>(sources: &'a [(Vec<u8>, Shape)], key: &[u8]) -> Result<&'a Shape, String> {
    let matches: Vec<_> = sources
        .iter()
        .filter(|(name, _)| name == key)
        .map(|(_, shape)| shape)
        .collect();
    let [shape] = matches.as_slice() else {
        return Err(format!(
            "split face {key:?}, matches {}: expected one named face in source body",
            matches.len()
        ));
    };
    topods::face(shape).map_err(|cause| format!("split face {key:?}: expected a face: {cause}"))?;
    Ok(shape)
}

fn collect_entities(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut map).map_err(|cause| cause.to_string())?;
    (1..=map.extent())
        .map(|index| {
            map.find_key(index)
                .cloned()
                .map_err(|cause| cause.to_string())
        })
        .collect()
}

fn collect_all_subshapes(shape: &Shape) -> Result<Vec<Shape>, String> {
    let mut all = Vec::new();
    for kind in [ShapeEnum::Face, ShapeEnum::Edge, ShapeEnum::Vertex] {
        all.extend(collect_entities(shape, kind)?);
    }
    Ok(all)
}
