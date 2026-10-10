// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchEntityId, SketchError};
use std::collections::BTreeSet;
impl Sketch2D {
    pub(super) fn shared_line_contact(
        &self,
        line: SketchEntityId,
        curve: SketchEntityId,
    ) -> Result<Option<bool>, SketchError> {
        let line_points = self.line(line)?.endpoints;
        let frame = self.circular_frame(curve)?;
        let Some(endpoints) = frame.endpoints else {
            return Ok(None);
        };
        Ok(unique_contact(&line_points, &endpoints)?.map(|touch| touch == line_points[1]))
    }
    pub(super) fn shared_circular_contact(
        &self,
        curves: [SketchEntityId; 2],
    ) -> Result<Option<SketchEntityId>, SketchError> {
        let first = self.circular_frame(curves[0])?.endpoints;
        let second = self.circular_frame(curves[1])?.endpoints;
        let (Some(first), Some(second)) = (first, second) else {
            return Ok(None);
        };
        unique_contact(&first, &second)
    }
    pub(super) fn validate_contact_ownership(
        &self,
        relation: &PlanarConstraint,
    ) -> Result<(), SketchError> {
        match relation {
            PlanarConstraint::LineTangent { line, curve, .. } => {
                self.shared_line_contact(*line, *curve)?;
            }
            PlanarConstraint::CircularTangent { curves, .. } => {
                self.shared_circular_contact(*curves)?;
            }
            _ => {}
        }
        Ok(())
    }
}
fn unique_contact(
    first: &[SketchEntityId; 2],
    second: &[SketchEntityId; 2],
) -> Result<Option<SketchEntityId>, SketchError> {
    let shared = first
        .iter()
        .copied()
        .filter(|point| second.contains(point))
        .collect::<BTreeSet<_>>();
    if shared.len() > 1 {
        return Err(SketchError::Definition(format!(
            "tangent shared points {shared:?}: expected at most one defining contact identity"
        )));
    }
    Ok(shared.into_iter().next())
}
