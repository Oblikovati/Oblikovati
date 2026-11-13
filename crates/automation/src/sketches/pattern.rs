// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::RecipeReservations,
    quantity::Dimension,
    sketch::{SketchEntityId, SketchRectangularPatternDefinition},
};
use std::collections::BTreeSet;

struct RectangularPatternInput {
    seed: Vec<SketchEntityId>,
    counts: [u32; 2],
    directions: [[f64; 2]; 2],
    spacing: [String; 2],
}
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
            |snapshot, reservations| create(snapshot, reservations, sketch, input),
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
fn prepare(request: AddSketchPatternArgs) -> Result<RectangularPatternInput, Error> {
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
    Ok(RectangularPatternInput {
        seed,
        counts: [count(request.count1)?, count(request.count2)?],
        directions: [
            direction(request.dir1, [1., 0.])?,
            direction(request.dir2, [0., 1.])?,
        ],
        spacing: [request.spacing1, request.spacing2],
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
fn create(
    snapshot: &mut DocumentSnapshot,
    reservations: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    input: RectangularPatternInput,
) -> Result<oblikovati_model::sketch::SketchRectangularPattern, Error> {
    let spacing = [
        insert_spacing(snapshot, reservations, sketch, &input.spacing[0])?,
        insert_spacing(snapshot, reservations, sketch, &input.spacing[1])?,
    ];
    let definition = SketchRectangularPatternDefinition {
        seed: input.seed,
        counts: input.counts,
        directions: input.directions,
        spacing,
        fitted: false,
        suppressed: BTreeSet::new(),
    };
    snapshot
        .create_rectangular_pattern(sketch, definition, reservations)
        .map_err(api_error)
}
fn insert_spacing(
    snapshot: &mut DocumentSnapshot,
    reservations: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    source: &str,
) -> Result<u64, Error> {
    let name = reservations
        .allocate_dimension_name(sketch, &snapshot.parameters)
        .map_err(api_error)?;
    let id = reservations.allocate().map_err(api_error)?;
    snapshot
        .parameters
        .insert_scalar_expression(id, &name, source, Dimension::Length)
        .map_err(api_error)?;
    Ok(id)
}
