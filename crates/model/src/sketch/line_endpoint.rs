// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityEditError, SketchEntityId, SketchError, require_position};
use crate::identity::RecipeIdError;

/// Select an authored line end without using its point's position as identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchLineEnd {
    First,
    Last,
}
impl SketchLineEnd {
    fn index(self) -> usize {
        match self {
            Self::First => 0,
            Self::Last => 1,
        }
    }
}
/// Identity changes from an endpoint seed edit; the previous point remains owned.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchLineEndpointReport {
    pub line: SketchEntityId,
    pub previous: SketchEntityId,
    pub created: SketchEntityId,
}
impl Sketch2D {
    /// Detach one end onto a fresh defining point, preserving shared points and constraints.
    /// This edits seed geometry; it does not certify a trim or extension intersection.
    /// Example: `sketch.replace_line_endpoint(line, SketchLineEnd::Last, [5.0, 0.0], &mut reserve)?;`.
    pub fn replace_line_endpoint(
        &mut self,
        line: SketchEntityId,
        end: SketchLineEnd,
        position: [f64; 2],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchLineEndpointReport, SketchEntityEditError> {
        let previous = self
            .validate_endpoint_replacement(line, end, position)
            .map_err(SketchEntityEditError::Sketch)?;
        let created = self.reserve_edit_identities(1, reserve)?[0];
        let mut candidate = self.clone();
        candidate
            .adopt_endpoint_replacement(line, end, created, position)
            .map_err(SketchEntityEditError::Sketch)?;
        *self = candidate;
        Ok(SketchLineEndpointReport {
            line,
            previous,
            created,
        })
    }
    fn validate_endpoint_replacement(
        &self,
        line: SketchEntityId,
        end: SketchLineEnd,
        position: [f64; 2],
    ) -> Result<SketchEntityId, SketchError> {
        let endpoints = self.line(line)?.endpoints;
        require_position(position)?;
        for endpoint in endpoints {
            if self.point(endpoint)?.position == position {
                return Err(SketchError::Definition(format!(
                    "line {line:?} replacement {position:?}: expected a position distinct from endpoint {endpoint:?}"
                )));
            }
        }
        Ok(endpoints[end.index()])
    }
    fn adopt_endpoint_replacement(
        &mut self,
        line: SketchEntityId,
        end: SketchLineEnd,
        created: SketchEntityId,
        position: [f64; 2],
    ) -> Result<(), SketchError> {
        self.add_defining_point(created, position)?;
        self.rewire_line_endpoint(line, end, created)
    }
    pub(super) fn rewire_line_endpoint(
        &mut self,
        line: SketchEntityId,
        end: SketchLineEnd,
        point: SketchEntityId,
    ) -> Result<(), SketchError> {
        self.point(point)?;
        let target = self
            .lines
            .iter_mut()
            .find(|candidate| candidate.id == line)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "endpoint replacement line {line:?}: expected an owned line"
                ))
            })?;
        target.endpoints[end.index()] = point;
        Ok(())
    }
}
