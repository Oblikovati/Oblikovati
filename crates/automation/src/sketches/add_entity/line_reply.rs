// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    types::{
        ConstraintInferenceKind, INFER_HORIZONTAL, INFER_PARALLEL, INFER_PERPENDICULAR,
        INFER_VERTICAL, SKETCH_ENTITY_LINE,
    },
    wire::{AppliedConstraintInference, AppliedPointInference},
};
use oblikovati_model::history::{SketchAppliedLineDirection, SketchLineCreationReport};
use oblikovati_session::SketchLineProjectionInput;
pub(super) fn encode_created(result: SketchLineProjectionInput) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    let direction = report
        .direction
        .map(|applied| direction_record(report.primitive.entity.0, applied))
        .transpose()?;
    encode(&AddSketchEntityResult {
        entity_id: report.primitive.entity.0,
        kind: SKETCH_ENTITY_LINE.0.to_string(),
        point_i_ds: Some(report.primitive.points.iter().map(|id| id.0).collect()),
        inferred_constraints: direction.map(|record| vec![record]),
        inferred_points: Some(point_records(&report)),
        ..Default::default()
    })
}
fn direction_record(
    line: u64,
    applied: SketchAppliedLineDirection,
) -> Result<AppliedConstraintInference, Error> {
    let index = crate::sketches::indices::geometric_wire_index(applied.geometric_index)?;
    let mut entities = vec![line];
    entities.extend(applied.inference.target.map(|target| target.0));
    Ok(AppliedConstraintInference {
        kind: direction_kind(applied.inference.kind)?.into(),
        constraint_index: index,
        entities: Some(entities),
    })
}
fn point_records(report: &SketchLineCreationReport) -> Vec<AppliedPointInference> {
    report
        .inference
        .points
        .iter()
        .map(|snap| AppliedPointInference {
            kind: "onPoint".into(),
            point_id: report.primitive.points[snap.endpoint].0,
            entities: Some(vec![snap.target.0]),
        })
        .collect()
}

// The Rust API freezes numeric enum values; the wire DTO requires Go String spellings (#3775).
fn direction_kind(kind: ConstraintInferenceKind) -> Result<&'static str, Error> {
    match kind {
        INFER_HORIZONTAL => Ok("horizontal"),
        INFER_VERTICAL => Ok("vertical"),
        INFER_PARALLEL => Ok("parallel"),
        INFER_PERPENDICULAR => Ok("perpendicular"),
        _ => Err(Error(format!(
            "line direction kind {kind:?}: expected horizontal, vertical, parallel, or perpendicular"
        ))),
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::sketch::{SketchConstraintId, SketchDirectionInference};
    fn applied(index: usize, kind: ConstraintInferenceKind) -> SketchAppliedLineDirection {
        SketchAppliedLineDirection {
            inference: SketchDirectionInference {
                kind,
                target: None,
                score: 100,
            },
            constraint: SketchConstraintId(1),
            geometric_index: index,
        }
    }
    #[test]
    fn unknown_direction_is_a_named_decline_instead_of_a_numeric_wire_kind() {
        let cause = direction_record(1, applied(0, ConstraintInferenceKind(999))).unwrap_err();
        assert!(cause.to_string().contains("999"));
        assert!(cause.to_string().contains("expected horizontal"));
    }
    #[test]
    fn largest_local_index_is_checked_against_the_wire_range() {
        let index = usize::MAX;
        let actual = direction_record(1, applied(index, INFER_HORIZONTAL));
        let Ok(expected) = i64::try_from(index) else {
            let cause = actual.unwrap_err().to_string();
            assert!(cause.contains(&index.to_string()));
            assert!(cause.contains("expected a nonnegative i64"));
            return;
        };
        assert_eq!(actual.unwrap().constraint_index, expected);
    }
}
