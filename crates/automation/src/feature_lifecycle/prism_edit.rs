// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::extrusion::{DISTANCE_ROLE, SECOND_DISTANCE_ROLE};
use oblikovati_model::feature::{sketch_prism::SketchPrismDefinition, work_geometry::DatumLength};

/// Edit supported prism distances on a private candidate.
/// Example: `apply_feature_distance_edit(&mut candidate, &request)?` before preview.
pub fn apply_feature_distance_edit(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
) -> Result<(), Error> {
    apply_feature_extrusion_edit(snapshot, request, None)
}

/// Apply distance inputs and an optional Taper A together on a private candidate.
/// Example: `apply_feature_extrusion_edit(&mut candidate, &request, Some("5 deg"))?`.
pub fn apply_feature_extrusion_edit(
    snapshot: &mut DocumentSnapshot,
    request: &EditFeatureArgs,
    taper: Option<&str>,
) -> Result<(), Error> {
    let (feature, _) = locate(&snapshot.work_geometry, request.id)?;
    let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
        return Err(Error(format!(
            "features.edit id {}: expected a sketch prism feature",
            request.id
        )));
    };
    if definition.extent().is_some() {
        return Err(Error(format!(
            "features.edit id {}: plane-bounded extrusion has no authored distance",
            request.id
        )));
    }
    let edits = validated_edits(request, definition.second_depth().is_some())?;
    let taper = taper
        .map(|source| crate::move_body::angle(snapshot, source, "Extrude Taper A"))
        .transpose()?;
    if let Some(angle) = taper {
        definition
            .as_ref()
            .clone()
            .with_taper(Some(angle))
            .map_err(api_error)?;
    }
    let mut changes_parameters = false;
    for edit in edits {
        changes_parameters |= apply_distance(snapshot, FeatureId(request.id), edit)?;
    }
    if let Some(angle) = taper {
        let id = FeatureId(request.id);
        let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
        let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
            return Err(Error(format!("feature {id:?}: expected a sketch prism")));
        };
        let tapered = definition
            .as_ref()
            .clone()
            .with_taper(Some(angle))
            .map_err(api_error)?;
        if !definition.same_recipe(&tapered) {
            replace_prism(snapshot, id, tapered)?;
        }
    }
    if changes_parameters {
        crate::model_states::record_parameter_edits(snapshot)?;
    }
    Ok(())
}

fn validated_edits(request: &EditFeatureArgs, two_sided: bool) -> Result<&[ScalarEdit], Error> {
    require_empty_repicks(request)?;
    let edits = request.scalars.as_deref().unwrap_or_default();
    let last = i64::from(two_sided);
    let valid_slots = match edits {
        [_] => true,
        [a, b] => a.index != b.index,
        _ => false,
    };
    if !valid_slots
        || edits
            .iter()
            .any(|e| !(0..=last).contains(&e.index) || e.value.trim().is_empty())
    {
        return Err(Error(format!(
            "features.edit id {} scalars {edits:?}: expected distinct nonempty lengths at slots 0..={last}",
            request.id
        )));
    }
    Ok(edits)
}

fn apply_distance(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    edit: &ScalarEdit,
) -> Result<bool, Error> {
    let second = edit.index == 1;
    let role = if second {
        SECOND_DISTANCE_ROLE
    } else {
        DISTANCE_ROLE
    };
    let label = if second {
        "extrude Distance B"
    } else {
        "extrude distance"
    };
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    if let Some(binding) = feature
        .owned_scalars()
        .iter()
        .find(|binding| binding.role == role)
    {
        let distance = crate::extrusion::extrusion_length(snapshot, &edit.value, label)?;
        return edit_owned_distance(snapshot, binding.parameter, distance, &edit.value);
    }
    let distance = crate::extrusion::positive_length(snapshot, &edit.value, label)?;
    let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
        return Err(Error(format!("feature {id:?}: expected a sketch prism")));
    };
    let definition = if second {
        definition.as_ref().clone().with_second_depth(distance)
    } else {
        definition.with_normal_source(distance)
    }
    .map_err(api_error)?;
    replace_prism(snapshot, id, definition)?;
    Ok(false)
}

fn replace_prism(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SketchPrismDefinition,
) -> Result<(), Error> {
    let reads = definition.reads(snapshot).map_err(api_error)?;
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::SketchPrism(Box::new(definition)),
            reads,
        )
        .map(|_| ())
        .map_err(api_error)
}

fn edit_owned_distance(
    snapshot: &mut DocumentSnapshot,
    owned: u64,
    distance: DatumLength,
    expression: &str,
) -> Result<bool, Error> {
    if matches!(distance, DatumLength::Parameter(id) if id == owned) {
        return Ok(false);
    }
    snapshot
        .parameters
        .set_scalar_expression(owned, expression.trim(), Dimension::Length)
        .map_err(api_error)?;
    snapshot.parameters.model_value(owned).map_err(api_error)?;
    Ok(true)
}
