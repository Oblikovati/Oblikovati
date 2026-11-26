// SPDX-License-Identifier: GPL-2.0-only
//! Validate detached solid Booleans and carry input lineage into the result.
use super::*;
use martensite_tkbo::{bop_algo::Operation, brep_algo_api::BRepAlgoAPIBooleanOperation};
use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
use martensite_tkernel::ncollection::List;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::brep_gprop::BRepGProp;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

pub(super) struct BooleanTool<'a> {
    pub shape: &'a Shape,
    pub sources: Vec<(Vec<u8>, Shape)>,
    pub resolution: Option<PrismResolution>,
}

pub(super) fn build_boolean_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    tool: BooleanTool<'_>,
    kind: PrismBoolean,
    operation: &str,
) -> Result<BlendCandidate, String> {
    let mut copier = crate::checked_shape::copy_shape(input.body, &format!("{operation} input"))
        .map_err(|error| error.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let target = copier.shape().map_err(|error| error.to_string())?;
    let boolean = performed_boolean(target, tool.shape, kind)
        .map_err(|error| format!("{operation}: expected completed Boolean operation: {error}"))?;
    let (shape, achieved_tolerance) = checked_boolean(
        target,
        &boolean,
        input.resolution,
        tool.resolution.map(|policy| (tool.shape, policy)),
        kind,
        operation,
    )?;
    let mut records = boolean_history(&boolean, copied, false)?;
    records.extend(boolean_history(&boolean, tool.sources, true)?);
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: Some(achieved_tolerance),
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        document,
        scope: history.evaluation_scope(),
    })
}

fn performed_boolean(
    target: &Shape,
    tool: &Shape,
    kind: PrismBoolean,
) -> Result<BRepAlgoAPIBooleanOperation, martensite_tkernel::standard::Failure> {
    let mut boolean = BRepAlgoAPIBooleanOperation::new();
    let mut arguments = List::default();
    arguments.append(target.clone());
    let mut tools = List::default();
    tools.append(tool.clone());
    boolean.set_arguments(&arguments);
    boolean.set_tools(&tools);
    boolean.set_operation(match kind {
        PrismBoolean::Cut => Operation::Cut,
        PrismBoolean::Join => Operation::Fuse,
    });
    boolean.build()?;
    Ok(boolean)
}

fn checked_boolean(
    target: &Shape,
    boolean: &BRepAlgoAPIBooleanOperation,
    resolution: Option<PrismResolution>,
    tool: Option<(&Shape, PrismResolution)>,
    kind: PrismBoolean,
    operation: &str,
) -> Result<(Shape, PrismAchievedTolerance), String> {
    if !boolean.is_done() || boolean.has_errors() || boolean.has_warnings() {
        return Err(format!(
            "{operation}: expected a completed Boolean without errors or warnings (done={}, errors={}, warnings={})",
            boolean.is_done(),
            boolean.has_errors(),
            boolean.has_warnings()
        ));
    }
    let result = single_solid(boolean.shape(), operation)?;
    crate::checked_shape::require_geometry(&result, &format!("{operation} result"))
        .map_err(|error| error.to_string())?;
    validate_result_faces(&result, operation)?;
    let achieved = match tool {
        Some((tool, policy)) => tolerances::check_two_operand_boolean(
            target, tool, &result, resolution, policy, operation,
        )?,
        None => tolerances::check(target, &result, resolution, operation)?,
    };
    let before = crate::checked_shape::analytic_properties(target, &format!("{operation} input"))
        .map_err(|error| error.to_string())?;
    let after = crate::checked_shape::analytic_properties(&result, &format!("{operation} result"))
        .map_err(|error| error.to_string())?;
    let changed = match kind {
        PrismBoolean::Cut => after.volume > 0. && after.volume < before.volume,
        PrismBoolean::Join => after.volume > before.volume,
    };
    if !changed {
        return Err(format!(
            "{operation} volume {} from {}: expected a nonempty solid with material changed by {kind}",
            after.volume, before.volume
        ));
    }
    Ok((result, achieved))
}

fn validate_result_faces(result: &Shape, operation: &str) -> Result<(), String> {
    let faces = entities(result, ShapeEnum::Face)?;
    if faces.is_empty() {
        return Err(format!(
            "{operation} result faces []: expected nonempty solid boundaries"
        ));
    }
    for face in faces {
        let mut properties = GProps::new();
        BRepGProp::surface_properties(&face, &mut properties, false, false)
            .map_err(|error| format!("{operation} result face: expected analytic area: {error}"))?;
        let area = properties.mass();
        let loops = entities(&face, ShapeEnum::Wire)?.len();
        if !area.is_finite() || area <= 0. || loops == 0 {
            return Err(format!(
                "{operation} result face area {area}, loops {loops}: expected positive finite area and closed loops"
            ));
        }
    }
    Ok(())
}

fn single_solid(result: &Shape, operation: &str) -> Result<Shape, String> {
    let solids = entities(result, ShapeEnum::Solid)?;
    if solids.len() != 1 {
        return Err(format!(
            "{operation} result has {} solids: expected exactly one",
            solids.len()
        ));
    }
    let solid = solids[0].clone();
    for kind in [ShapeEnum::Face, ShapeEnum::Edge, ShapeEnum::Vertex] {
        let all = entities(result, kind)?;
        let retained = entities(&solid, kind)?;
        if all.len() != retained.len()
            || all
                .iter()
                .any(|item| !retained.iter().any(|other| item.is_same(other)))
        {
            return Err(format!(
                "{operation} result {kind:?}: expected all result entities in the single solid"
            ));
        }
    }
    Ok(solid)
}
pub(super) fn entities(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut map).map_err(|error| error.to_string())?;
    (1..=map.extent())
        .map(|index| {
            map.find_key(index)
                .cloned()
                .map_err(|error| error.to_string())
        })
        .collect()
}

fn boolean_history(
    boolean: &BRepAlgoAPIBooleanOperation,
    sources: Vec<(Vec<u8>, Shape)>,
    transient: bool,
) -> Result<Vec<BlendHistoryRecord>, String> {
    sources
        .into_iter()
        .map(|(name, source)| {
            Ok(BlendHistoryRecord {
                generated: boolean
                    .generated(&source)
                    .map_err(|error| error.to_string())?,
                modified: boolean
                    .modified(&source)
                    .map_err(|error| error.to_string())?,
                deleted: boolean
                    .is_deleted(&source)
                    .map_err(|error| error.to_string())?,
                source,
                name,
                transient,
            })
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkmath::gp::Point3;
    use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;

    #[test]
    fn square_prism_cut_proves_geometry_and_reports_tolerance_growth() {
        let target = BRepPrimAPIMakeBox::from_sizes(2., 2., 2.)
            .unwrap()
            .shape()
            .unwrap()
            .clone();
        let tool =
            BRepPrimAPIMakeBox::from_points(Point3::new(0.5, 0.5, -1.), Point3::new(1.5, 1.5, 3.))
                .unwrap()
                .shape()
                .unwrap()
                .clone();
        let resolution = PrismResolution {
            version: 1,
            weld: 1e-6,
            sew: 1e-7,
            angular: 1e-9,
            display_chordal: 1e-3,
            display_angular: 0.1,
            maximum_tolerance: 1e-5,
        };
        let boolean = performed_boolean(&target, &tool, PrismBoolean::Cut).unwrap();
        let result = single_solid(boolean.shape(), "square prism").unwrap();
        crate::checked_shape::require_geometry(&result, "cut result").unwrap();
        let decline = checked_boolean(
            &target,
            &boolean,
            Some(resolution),
            None,
            PrismBoolean::Cut,
            "square prism",
        )
        .unwrap_err();
        assert!(
            decline.contains("square prism vertex tolerance"),
            "{decline}"
        );
        let (_, tolerance) = checked_boolean(
            &target,
            &boolean,
            Some(resolution),
            Some((&tool, resolution)),
            PrismBoolean::Cut,
            "square prism",
        )
        .unwrap();
        assert!(tolerance.vertex <= resolution.maximum_tolerance);
        let tight_tool = PrismResolution {
            maximum_tolerance: 1e-7,
            ..resolution
        };
        assert!(
            checked_boolean(
                &target,
                &boolean,
                Some(resolution),
                Some((&tool, tight_tool)),
                PrismBoolean::Cut,
                "square prism",
            )
            .is_err()
        );
        let properties = crate::checked_shape::analytic_properties(&result, "cut result").unwrap();
        assert!((properties.volume - 6.).abs() < 1e-10);
        assert_eq!(
            entities(&result, ShapeEnum::Face)
                .unwrap()
                .iter()
                .filter(|face| entities(face, ShapeEnum::Wire).unwrap().len() == 2)
                .count(),
            2
        );
        let original = crate::checked_shape::analytic_properties(&target, "target").unwrap();
        assert!((original.volume - 8.).abs() < 1e-10);
    }
}
