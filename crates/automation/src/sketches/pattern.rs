// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::SketchRectangularPatternRequest, sketch::SketchEntityId};
use std::collections::BTreeSet;

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddSketchPatternArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let input = prepare(request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create rectangular sketch pattern",
            |snapshot, reservations| {
                snapshot
                    .create_pattern_from_expressions(sketch, input, reservations)
                    .map_err(api_error)
            },
            |_, pattern| {
                encode(&AddSketchPatternResult {
                    created: Some(
                        pattern
                            .members
                            .into_iter()
                            .flat_map(|member| member.entities.into_iter().map(|id| id.0))
                            .collect(),
                    ),
                })
            },
        )
        .map_err(api_error)
}
fn prepare(request: AddSketchPatternArgs) -> Result<SketchRectangularPatternRequest, Error> {
    if request.kind != "rectangular" {
        return Err(Error(format!(
            "sketch.addPattern kind {:?}: expected rectangular; circular patterns are not available yet",
            request.kind
        )));
    }
    let seed: Vec<_> = request
        .entities
        .unwrap_or_default()
        .into_iter()
        .map(SketchEntityId)
        .collect();
    if seed.is_empty() {
        return Err(Error(
            "sketch.addPattern entities []: expected at least one seed entity".into(),
        ));
    }
    Ok(SketchRectangularPatternRequest {
        seed,
        counts: [count(request.count1)?, count(request.count2)?],
        directions: [
            direction(request.dir1, [1., 0.])?,
            direction(request.dir2, [0., 1.])?,
        ],
        spacing: [request.spacing1, request.spacing2],
        fitted: false,
        suppressed: BTreeSet::new(),
    })
}
fn count(value: i64) -> Result<u32, Error> {
    u32::try_from(value)
        .ok()
        .filter(|count| *count > 0)
        .ok_or_else(|| {
            Error(format!(
                "sketch.addPattern count {value}: expected a positive 32-bit integer"
            ))
        })
}
fn direction(input: Option<Vec<f64>>, fallback: [f64; 2]) -> Result<[f64; 2], Error> {
    let values = input.unwrap_or_default();
    let vector = if values.len() == 2 {
        [values[0], values[1]]
    } else {
        fallback
    };
    let scale = vector[0].abs().max(vector[1].abs());
    if !scale.is_finite() || scale == 0.0 {
        return Err(Error(format!(
            "sketch.addPattern direction {vector:?}: expected a finite nonzero vector"
        )));
    }
    let scaled = vector.map(|value| value / scale);
    let length = scaled[0].hypot(scaled[1]);
    Ok(scaled.map(|value| value / length))
}
