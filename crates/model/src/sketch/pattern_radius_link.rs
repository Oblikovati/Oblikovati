// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError};

/// Equal-radius relation between a seed circle and one pattern member.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPatternRadiusLink {
    pub source: SketchEntityId,
    pub member: SketchEntityId,
}
impl Sketch2D {
    /// Add a checked circle-radius link; example: `sketch.add_pattern_radius_link(link)?;`.
    pub fn add_pattern_radius_link(
        &mut self,
        link: SketchPatternRadiusLink,
    ) -> Result<(), SketchError> {
        self.circle(link.source)?;
        self.circle(link.member)?;
        if self
            .pattern_radius_links
            .iter()
            .any(|existing| existing.member == link.member)
        {
            return Err(link.invalid("a member without an existing radius link"));
        }
        let mut ancestor = link.source;
        loop {
            if ancestor == link.member {
                return Err(link.invalid("an acyclic circle dependency"));
            }
            match self
                .pattern_radius_links
                .iter()
                .find(|existing| existing.member == ancestor)
            {
                Some(existing) => ancestor = existing.source,
                None => break,
            }
        }
        self.pattern_radius_links.push(link);
        Ok(())
    }
    /// Inspect structural circle-radius links; example: `sketch.pattern_radius_links()`.
    pub fn pattern_radius_links(&self) -> &[SketchPatternRadiusLink] {
        &self.pattern_radius_links
    }
}
impl SketchPatternRadiusLink {
    fn invalid(self, expected: &str) -> SketchError {
        SketchError::Definition(format!("pattern radius link {self:?}: expected {expected}"))
    }
}
