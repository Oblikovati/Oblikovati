// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    wire::{
        AddFeatureArgs,
        featureargs::{
            KIND_MIRROR, KIND_PATTERN_CIRCULAR, Mirror, PatternCircular, PatternPlacement,
            PatternRectangular,
        },
    },
};
use oblikovati_model::{
    feature::{
        FeatureId,
        circular_pattern::{
            CircularPatternDefinition, CircularPatternRecipe, CircularPatternSpacing,
        },
        mirror::{MirrorDefinition, MirrorRecipe},
        rectangular_pattern::{RectangularPatternDefinition, RectangularPatternRecipe},
    },
    history::DocumentSnapshot,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::collections::BTreeSet;

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let owner = replay
        .ok_or_else(|| Error("extrusion pattern: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("extrusion pattern: expected a certified part resolution".into()))?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let value = request.args.0.unwrap_or_else(|| serde_json::json!({}));
    let (name_prefix, definition) = match request.kind.as_str() {
        KIND_MIRROR => (
            "Mirror",
            Pattern::Mirror(prepare_mirror(
                snapshot,
                serde_json::from_value(value).map_err(api_error)?,
            )?),
        ),
        KIND_PATTERN_CIRCULAR => (
            "CircularPattern",
            Pattern::Circular(prepare_circular(
                snapshot,
                serde_json::from_value(value).map_err(api_error)?,
            )?),
        ),
        _ => (
            "RectangularPattern",
            Pattern::Rectangular(prepare_rectangular(
                snapshot,
                serde_json::from_value(value).map_err(api_error)?,
            )?),
        ),
    };
    let name = crate::fillet::next_name(snapshot, name_prefix)?;
    owner.ensure_part(files, document).map_err(Error)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "extrusion pattern",
        |candidate, ids| {
            match definition {
                Pattern::Mirror(definition) => {
                    candidate.create_mirrored_features(&name, definition, ids)
                }
                Pattern::Circular(definition) => {
                    candidate.create_circular_feature_pattern(&name, definition, ids)
                }
                Pattern::Rectangular(definition) => {
                    candidate.create_rectangular_feature_pattern(&name, definition, ids)
                }
            }
            .map_err(api_error)
        },
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

enum Pattern {
    Mirror(MirrorDefinition),
    Circular(CircularPatternDefinition),
    Rectangular(RectangularPatternDefinition),
}

fn prepare_mirror(snapshot: &DocumentSnapshot, input: Mirror) -> Result<MirrorDefinition, Error> {
    if !matches!(input.mode.trim(), "" | "features") {
        return Err(Error(format!(
            "mirror mode {:?}: expected features",
            input.mode
        )));
    }
    if input.remove_original || !matches!(input.operation.trim(), "" | "newBody") {
        return Err(Error(format!(
            "feature mirror removeOriginal={}, operation {:?}: expected feature replay with the source retained",
            input.remove_original, input.operation
        )));
    }
    MirrorDefinition::new(MirrorRecipe {
        sources: source_ids(snapshot, input.source_features)?,
        origin: input
            .origin
            .map(|value| vector(value, "origin"))
            .transpose()?
            .unwrap_or([0.; 3]),
        normal: vector(input.normal.unwrap_or_default(), "normal")?,
    })
    .map_err(Error)
}

fn prepare_rectangular(
    snapshot: &DocumentSnapshot,
    input: PatternRectangular,
) -> Result<RectangularPatternDefinition, Error> {
    let options = &input.pattern_placement;
    validate_placement(options, &[&input.count_x_expr, &input.count_y_expr])?;
    let fitted = match options.spacing_type.as_str() {
        "" | "spacing" => false,
        "fitted" => true,
        value => {
            return Err(Error(format!(
                "rectangular pattern spacingType {value:?}: expected spacing or fitted"
            )));
        }
    };
    let counts = [count(input.count_x, 2)?, count(input.count_y, 1)?];
    RectangularPatternDefinition::new(RectangularPatternRecipe {
        sources: source_ids(snapshot, input.source_features)?,
        counts,
        steps: [
            step(input.step_x, "stepX", counts[0])?,
            step(input.step_y, "stepY", counts[1])?,
        ],
        mid_plane: [input.mid_plane_x, input.mid_plane_y],
        fitted,
        suppressed: suppression(options)?,
    })
    .map_err(Error)
}

fn prepare_circular(
    snapshot: &DocumentSnapshot,
    input: PatternCircular,
) -> Result<CircularPatternDefinition, Error> {
    let options = &input.pattern_placement;
    validate_placement(options, &[&input.count_expr])?;
    let spacing = match options.spacing_type.as_str() {
        "" | "spacing" => CircularPatternSpacing::Sweep,
        "between" => CircularPatternSpacing::Between,
        "fitted" => CircularPatternSpacing::Fitted,
        value => {
            return Err(Error(format!(
                "circular pattern spacingType {value:?}: expected spacing, between or fitted"
            )));
        }
    };
    CircularPatternDefinition::new(CircularPatternRecipe {
        sources: source_ids(snapshot, input.source_features)?,
        count: count(input.count, 4)?,
        angle: crate::move_body::angle(snapshot, &input.angle, "circular pattern angle")?,
        origin: input
            .axis_point
            .map(|value| vector(value, "axisPoint"))
            .transpose()?
            .unwrap_or([0.; 3]),
        direction: input
            .axis_dir
            .map(|value| vector(value, "axisDir"))
            .transpose()?
            .unwrap_or([0., 0., 1.]),
        mid_plane: input.mid_plane,
        spacing,
        suppressed: suppression(options)?,
    })
    .map_err(Error)
}

fn source_ids(
    snapshot: &DocumentSnapshot,
    names: Option<Vec<String>>,
) -> Result<Vec<FeatureId>, Error> {
    names
        .unwrap_or_default()
        .iter()
        .map(|name| {
            snapshot
                .work_geometry
                .features()
                .iter()
                .find(|feature| feature.name() == name)
                .map(|feature| feature.id())
                .ok_or_else(|| {
                    Error(format!(
                        "pattern source {name:?}: expected an existing feature name"
                    ))
                })
        })
        .collect()
}

fn suppression(options: &PatternPlacement) -> Result<BTreeSet<u64>, Error> {
    options
        .suppressed_elements
        .as_deref()
        .unwrap_or_default()
        .iter()
        .copied()
        .map(|index| u64::try_from(index).map_err(api_error))
        .collect()
}

fn count(value: i64, default: u32) -> Result<u32, Error> {
    if value == 0 {
        return Ok(default);
    }
    u32::try_from(value).map_err(|_| {
        Error(format!(
            "pattern count {value}: expected a positive occurrence count"
        ))
    })
}

fn step(value: Option<Vec<f64>>, label: &str, count: u32) -> Result<[f64; 3], Error> {
    match value {
        Some(components) => vector(components, label),
        None if count == 1 => Ok([0.; 3]),
        None => Err(Error(format!(
            "rectangular pattern {label}: expected three components for count {count}"
        ))),
    }
}

fn vector(components: Vec<f64>, label: &str) -> Result<[f64; 3], Error> {
    components.try_into().map_err(|components: Vec<f64>| {
        Error(format!(
            "pattern {label} {components:?}: expected three components"
        ))
    })
}

fn validate_placement(options: &PatternPlacement, count_expressions: &[&str]) -> Result<(), Error> {
    if count_expressions.iter().any(|value| !value.is_empty()) || options.boundary.is_some() {
        return Err(Error(
            "extrusion pattern: expected numeric counts and unclipped placement".into(),
        ));
    }
    for (label, value, supported) in [
        ("computeType", options.compute_type.as_str(), "identical"),
        ("orientation", options.orientation.as_str(), "identical"),
        (
            "positioningMethod",
            options.positioning_method.as_str(),
            "incremental",
        ),
    ] {
        if !value.is_empty() && value != supported {
            return Err(Error(format!(
                "extrusion pattern {label} {value:?}: expected {supported:?}"
            )));
        }
    }
    Ok(())
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
