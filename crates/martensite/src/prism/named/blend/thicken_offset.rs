// SPDX-License-Identifier: GPL-2.0-only
//! Face thicken (solid prism/offset + Booleans) and offset surface shells (#3918).
use super::extrusion_boolean::BooleanFailure;
use super::thicken_offset_chain::{
    collect_shapes, ensure_positive_solid, planar_outward_normal, translate_shape,
    try_coaxial_rounded_cylinder_blend,
};
use super::*;
use martensite_tkbo::{bop_algo::Operation, brep_algo_api::BRepAlgoAPIBooleanOperation};
use martensite_tkbrep::topods::{Builder, Shape, Shell};
use martensite_tkernel::ncollection::List;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::Vector3;
use martensite_tkoffset::brep_offset_api::{
    BRepOffsetAPIMakeOffsetShape, BRepOffsetAPIMakeThickSolid,
};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakePrism;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::feature::{
    FeatureId,
    sketch_prism::PrismBoolean,
    thicken_offset::{ResolvedThickenOffset, ThickenBoolean, ThickenDirection, ThickenOutputMode},
};

impl NamedSolid {
    pub(in crate::prism::named) fn evaluate_thicken_offset_result(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        input: &ResolvedThickenOffset,
    ) -> Result<NamedSolid, BooleanFailure> {
        let resolution = self.body_resolution();
        let copied_faces = self
            .copied_selected_faces(history, document, &input.faces)
            .map_err(BooleanFailure::Refused)?;

        if input.output == ThickenOutputMode::Surface {
            let shell = build_offset_surface_shell(&copied_faces, input.distance, input.direction)
                .map_err(BooleanFailure::Refused)?;
            let topology = BlendNamedTopology::from_offset_shell(
                history, document, feature, shell, resolution,
            )
            .map_err(BooleanFailure::Refused)?;
            return Ok(NamedSolid::Derived(Box::new(topology)));
        }

        if input.automatic_blending && input.direction != ThickenDirection::Symmetric {
            let signed = match input.direction {
                ThickenDirection::Positive => input.distance,
                ThickenDirection::Negative => -input.distance,
                ThickenDirection::Symmetric => 0.0,
            };
            let body_shape = self
                .native_shape(history, document)
                .map_err(BooleanFailure::Refused)?;
            if let Ok(Some(blended_shape)) =
                try_coaxial_rounded_cylinder_blend(body_shape, &copied_faces, signed)
                && let Ok(solid) = ensure_positive_solid(blended_shape, "thicken rounded blend")
            {
                let primary_body = self
                    .body_reference_key(history, document)
                    .map_err(BooleanFailure::Refused)?
                    .to_vec();
                let deleted = self
                    .prior_non_body_keys(history, document, &primary_body)
                    .map_err(BooleanFailure::Refused)?;
                let topology = BlendNamedTopology::from_derived_preserving_body(
                    history,
                    document,
                    feature,
                    primary_body,
                    deleted,
                    solid,
                    resolution,
                )
                .map_err(BooleanFailure::Refused)?;
                return Ok(NamedSolid::Derived(Box::new(topology)));
            }
        }
        let embed_join = (input.automatic_blending || input.boolean == ThickenBoolean::Join)
            && input.direction == ThickenDirection::Positive;
        let tool_solids =
            build_thickened_solids(&copied_faces, input.distance, input.direction, embed_join)
                .map_err(BooleanFailure::Refused)?;

        if !input.automatic_blending && input.boolean == ThickenBoolean::NewSolid {
            let single = merge_tool_solids(&tool_solids).map_err(BooleanFailure::Refused)?;
            let named_tool =
                BlendNamedTopology::from_derived(history, document, feature, single, resolution)
                    .map_err(BooleanFailure::Refused)?;
            return Ok(NamedSolid::Derived(Box::new(named_tool)));
        }

        let op = if input.automatic_blending {
            PrismBoolean::Join
        } else {
            input
                .boolean
                .prism_boolean()
                .ok_or_else(|| BooleanFailure::Refused("thicken: expected solid Boolean".into()))?
        };
        let mut boolean_tools = Vec::with_capacity(tool_solids.len());
        for (idx, solid) in tool_solids.iter().enumerate() {
            let sub_feature = FeatureId(
                feature
                    .0
                    .saturating_mul(1000)
                    .saturating_add(idx as u64 + 1),
            );
            let named_tool = BlendNamedTopology::from_derived(
                history,
                document,
                sub_feature,
                solid.clone(),
                resolution,
            )
            .map_err(BooleanFailure::Refused)?;
            boolean_tools.push(
                named_tool
                    .sweep_boolean_tool(history, document)
                    .map_err(BooleanFailure::Refused)?,
            );
        }
        let candidate = match self.boolean_tools(
            history,
            document,
            boolean_tools,
            op,
            &format!("thicken {op}"),
        ) {
            Ok(candidate) => candidate,
            Err(failure) if input.automatic_blending && op == PrismBoolean::Join => {
                return self
                    .automatic_blending_join(history, document, feature, &tool_solids, resolution)
                    .map_err(|blend_err| {
                        BooleanFailure::Refused(format!("{failure} | blend: {blend_err}"))
                    });
            }
            Err(failure) => return Err(failure),
        };
        let named = candidate
            .name_topology(history, document, feature)
            .map_err(BooleanFailure::Refused)?;
        Ok(NamedSolid::Derived(Box::new(named)))
    }

    fn automatic_blending_join(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        tool_shapes: &[Shape],
        resolution: Option<PrismResolution>,
    ) -> Result<NamedSolid, String> {
        let body_shape = self.native_shape(history, document)?;
        let target_solid = ensure_positive_solid(
            crate::checked_shape::copy_shape(body_shape, "thicken blend target")
                .map_err(|cause| cause.to_string())?
                .shape()
                .map_err(|cause| cause.to_string())?
                .clone(),
            "thicken blend target",
        )?;
        let before =
            crate::checked_shape::analytic_properties(&target_solid, "thicken blend input")
                .map_err(|cause| cause.to_string())?;
        let tool_vols: Vec<f64> = tool_shapes
            .iter()
            .map(|s| {
                crate::checked_shape::analytic_properties(s, "tool")
                    .map(|p| p.volume)
                    .unwrap_or(-999.0)
            })
            .collect();
        let mut current = target_solid;
        for tool_shape in tool_shapes {
            let mut boolean = BRepAlgoAPIBooleanOperation::new();
            let mut arguments = List::default();
            arguments.append(current);
            let mut tools = List::default();
            tools.append(tool_shape.clone());
            boolean.set_arguments(&arguments);
            boolean.set_tools(&tools);
            boolean.set_operation(Operation::Fuse);
            boolean.build().map_err(|cause| cause.to_string())?;
            if !boolean.is_done() || boolean.has_errors() {
                return Err(format!(
                    "thicken automatic blending: Fuse failed (tool_vols={tool_vols:?})"
                ));
            }
            current = ensure_positive_solid(boolean.shape().clone(), "thicken blend result")?;
        }
        let after = crate::checked_shape::analytic_properties(&current, "thicken blend result")
            .map_err(|cause| cause.to_string())?;
        if !(after.volume.is_finite() && after.volume > before.volume) {
            return Err(format!(
                "thicken automatic blending volume {} <= {} (tool_vols={tool_vols:?})",
                after.volume, before.volume
            ));
        }
        let primary_body = self.body_reference_key(history, document)?.to_vec();
        let deleted = self.prior_non_body_keys(history, document, &primary_body)?;
        let topology = BlendNamedTopology::from_derived_preserving_body(
            history,
            document,
            feature,
            primary_body,
            deleted,
            current,
            resolution,
        )?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }

    fn prior_non_body_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        primary_body: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut keys = self.reference_keys(history, document)?;
        keys.extend(self.subentity_reference_keys(history, document)?);
        if let Self::Derived(body) = self {
            keys.extend(body.live_keys().iter().cloned());
            keys.extend(body.deleted_keys().iter().cloned());
        }
        keys.retain(|k| k.as_slice() != primary_body);
        keys.sort();
        keys.dedup();
        Ok(keys)
    }

    fn body_resolution(&self) -> Option<PrismResolution> {
        match self {
            Self::Prism(body) => body.certification.as_ref().map(|c| c.resolution),
            Self::Derived(body) => body.resolution(),
        }
    }

    fn copied_selected_faces(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_keys: &[Vec<u8>],
    ) -> Result<Vec<Shape>, String> {
        let body_shape = self.native_shape(history, document)?;
        let mut copier = crate::checked_shape::copy_shape(body_shape, "thicken/offset input")
            .map_err(|cause| cause.to_string())?;
        let mut out = Vec::with_capacity(face_keys.len());
        for key in face_keys {
            let source = self.native_face(history, document, key)?;
            let copies = copier.modified(source).map_err(|cause| cause.to_string())?;
            let Some(copied) = copies.first() else {
                return Err(format!(
                    "thicken/offset face {key:?}: expected one detached face copy"
                ));
            };
            out.push(copied.clone());
        }
        Ok(out)
    }
}

fn build_offset_surface_shell(
    faces: &[Shape],
    distance: f64,
    direction: ThickenDirection,
) -> Result<Shape, String> {
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shell_shape = shell.into_shape();

    if distance == 0.0 {
        for face in faces {
            builder
                .add(&mut shell_shape, face)
                .map_err(|cause| cause.to_string())?;
        }
        return Ok(shell_shape);
    }

    let offsets: Vec<f64> = match direction {
        ThickenDirection::Negative => vec![-distance],
        ThickenDirection::Positive => vec![distance],
        ThickenDirection::Symmetric => vec![-0.5 * distance, 0.5 * distance],
    };
    for signed_offset in offsets {
        for face in faces {
            let offset_shape = offset_single_face(face, signed_offset)?;
            for result_face in collect_shapes(&offset_shape, ShapeEnum::Face)? {
                builder
                    .add(&mut shell_shape, &result_face)
                    .map_err(|cause| cause.to_string())?;
            }
        }
    }
    Ok(shell_shape)
}

fn offset_single_face(face: &Shape, signed_offset: f64) -> Result<Shape, String> {
    if let Some(normal) = planar_outward_normal(face)? {
        let shift = [
            normal[0] * signed_offset,
            normal[1] * signed_offset,
            normal[2] * signed_offset,
        ];
        return translate_shape(face, shift);
    }
    let mut maker = BRepOffsetAPIMakeOffsetShape::new();
    maker
        .perform_by_simple(face, signed_offset)
        .map_err(|cause| format!("offset surface {signed_offset}: {cause}"))?;
    if !maker.is_done() {
        return Err(format!(
            "offset surface {signed_offset}: simple offset did not complete"
        ));
    }
    Ok(maker.shape().map_err(|cause| cause.to_string())?.clone())
}

fn build_thickened_solids(
    faces: &[Shape],
    distance: f64,
    direction: ThickenDirection,
    embed_for_join: bool,
) -> Result<Vec<Shape>, String> {
    let mut solids = Vec::with_capacity(faces.len());
    for face in faces {
        solids.push(build_single_face_thick_solid(
            face,
            distance,
            direction,
            embed_for_join,
        )?);
    }
    Ok(solids)
}

fn merge_tool_solids(solids: &[Shape]) -> Result<Shape, String> {
    let Some(first) = solids.first() else {
        return Err("thicken: expected at least one solid tool".into());
    };
    let mut current = first.clone();
    for next in &solids[1..] {
        let mut boolean = BRepAlgoAPIBooleanOperation::new();
        let mut args = List::default();
        args.append(current);
        let mut tools = List::default();
        tools.append(next.clone());
        boolean.set_arguments(&args);
        boolean.set_tools(&tools);
        boolean.set_operation(Operation::Fuse);
        boolean.build().map_err(|cause| cause.to_string())?;
        if !boolean.is_done() || boolean.has_errors() {
            return Err("thicken multi-face merge failed".into());
        }
        current = ensure_positive_solid(boolean.shape().clone(), "thicken merge")?;
    }
    Ok(current)
}

fn build_single_face_thick_solid(
    face: &Shape,
    distance: f64,
    direction: ThickenDirection,
    embed_for_join: bool,
) -> Result<Shape, String> {
    if let Some(normal) = planar_outward_normal(face)? {
        return build_planar_prism_solid(face, normal, distance, direction);
    }
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shell_shape = shell.into_shape();
    builder
        .add(&mut shell_shape, face)
        .map_err(|cause| cause.to_string())?;
    let (base_shell, signed_offset) = match direction {
        ThickenDirection::Positive if embed_for_join => {
            let embed = (distance * 0.2).min(0.02);
            let mut shift = BRepOffsetAPIMakeOffsetShape::new();
            shift
                .perform_by_simple(&shell_shape, -embed)
                .map_err(|cause| format!("thicken join embed shift: {cause}"))?;
            if shift.is_done() {
                (
                    shift.shape().map_err(|cause| cause.to_string())?.clone(),
                    distance + embed,
                )
            } else {
                (shell_shape, distance)
            }
        }
        ThickenDirection::Positive => (shell_shape, distance),
        ThickenDirection::Negative => (shell_shape, -distance),
        ThickenDirection::Symmetric => {
            let mut shift = BRepOffsetAPIMakeOffsetShape::new();
            shift
                .perform_by_simple(&shell_shape, -0.5 * distance)
                .map_err(|cause| format!("thicken symmetric shift: {cause}"))?;
            if !shift.is_done() {
                return Err("thicken symmetric shift did not complete".into());
            }
            (
                shift.shape().map_err(|cause| cause.to_string())?.clone(),
                distance,
            )
        }
    };
    let mut maker = BRepOffsetAPIMakeThickSolid::new();
    maker
        .make_thick_solid_by_simple(&base_shell, signed_offset)
        .map_err(|cause| format!("thicken solid {signed_offset}: {cause}"))?;
    if !maker.is_done() {
        return Err(format!(
            "thicken solid {signed_offset}: simple thick solid did not complete"
        ));
    }
    let result = maker.shape().map_err(|cause| cause.to_string())?.clone();
    ensure_positive_solid(result, "thicken solid")
}

fn build_planar_prism_solid(
    face: &Shape,
    normal: [f64; 3],
    distance: f64,
    direction: ThickenDirection,
) -> Result<Shape, String> {
    let (start_face, vec) = match direction {
        ThickenDirection::Positive => (
            face.clone(),
            Vector3::new(
                normal[0] * distance,
                normal[1] * distance,
                normal[2] * distance,
            ),
        ),
        ThickenDirection::Negative => (
            face.clone(),
            Vector3::new(
                -normal[0] * distance,
                -normal[1] * distance,
                -normal[2] * distance,
            ),
        ),
        ThickenDirection::Symmetric => {
            let shifted = translate_shape(
                face,
                [
                    -0.5 * distance * normal[0],
                    -0.5 * distance * normal[1],
                    -0.5 * distance * normal[2],
                ],
            )?;
            (
                shifted,
                Vector3::new(
                    normal[0] * distance,
                    normal[1] * distance,
                    normal[2] * distance,
                ),
            )
        }
    };
    let mut prism = BRepPrimAPIMakePrism::new(&start_face, vec, true, true)
        .map_err(|cause| cause.to_string())?;
    let shape = prism.shape().map_err(|cause| cause.to_string())?.clone();
    ensure_positive_solid(shape, "thicken planar prism")
}
