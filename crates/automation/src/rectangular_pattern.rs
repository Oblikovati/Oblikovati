// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::PatternRectangular},
};
use oblikovati_model::{
    feature::rectangular_pattern::{RectangularPatternDefinition, RectangularPatternRecipe},
    history::DocumentSnapshot,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input: PatternRectangular =
        serde_json::from_value(request.args.0.unwrap_or_else(|| serde_json::json!({})))
            .map_err(api_error)?;
    let document = crate::part_owner::active_part(files)?;
    let owner = replay.ok_or_else(|| {
        Error("rectangular pattern: expected a certified part replay owner".into())
    })?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("rectangular pattern: expected a certified part resolution".into()))?;
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
    let name = crate::fillet::next_name(snapshot, "RectangularPattern")?;
    let definition = prepare(snapshot, input)?;
    owner.ensure_part(files, document).map_err(Error)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "rectangular pattern",
        |candidate, ids| {
            candidate
                .create_rectangular_feature_pattern(&name, definition, ids)
                .map_err(api_error)
        },
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn prepare(
    snapshot: &DocumentSnapshot,
    input: PatternRectangular,
) -> Result<RectangularPatternDefinition, Error> {
    let fitted = pattern_spacing(&input)?;
    let sources = input
        .source_features
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
                        "rectangular pattern source {name:?}: expected an existing feature name"
                    ))
                })
        })
        .collect::<Result<_, _>>()?;
    let counts = [count(input.count_x, 2)?, count(input.count_y, 1)?];
    let suppressed = input
        .pattern_placement
        .suppressed_elements
        .unwrap_or_default()
        .into_iter()
        .map(|index| u64::try_from(index).map_err(api_error))
        .collect::<Result<_, _>>()?;
    RectangularPatternDefinition::new(RectangularPatternRecipe {
        sources,
        counts,
        steps: [
            step(input.step_x, "stepX", counts[0])?,
            step(input.step_y, "stepY", counts[1])?,
        ],
        mid_plane: [input.mid_plane_x, input.mid_plane_y],
        fitted,
        suppressed,
    })
    .map_err(Error)
}

fn count(value: i64, default: u32) -> Result<u32, Error> {
    if value == 0 {
        return Ok(default);
    }
    u32::try_from(value).map_err(|_| {
        Error(format!(
            "rectangular pattern count {value}: expected a positive grid count"
        ))
    })
}

fn step(value: Option<Vec<f64>>, label: &str, count: u32) -> Result<[f64; 3], Error> {
    match value {
        Some(vector) => vector.try_into().map_err(|vector: Vec<f64>| {
            Error(format!(
                "rectangular pattern {label} {vector:?}: expected three components in centimetres"
            ))
        }),
        None if count == 1 => Ok([0.; 3]),
        None => Err(Error(format!(
            "rectangular pattern {label}: expected three components for count {count}"
        ))),
    }
}

fn pattern_spacing(input: &PatternRectangular) -> Result<bool, Error> {
    let options = &input.pattern_placement;
    if !input.count_x_expr.is_empty()
        || !input.count_y_expr.is_empty()
        || options.boundary.is_some()
    {
        return Err(Error(
            "rectangular pattern: expected numeric counts and an unclipped grid".into(),
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
                "rectangular pattern {label} {value:?}: expected {supported:?}"
            )));
        }
    }
    match options.spacing_type.as_str() {
        "" | "spacing" => Ok(false),
        "fitted" => Ok(true),
        value => Err(Error(format!(
            "rectangular pattern spacingType {value:?}: expected spacing or fitted"
        ))),
    }
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
