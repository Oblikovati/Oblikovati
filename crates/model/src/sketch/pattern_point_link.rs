// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe, require_position};

/// A structural point relation owned by a sketch pattern occurrence.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPatternPointLink {
    pub source: SketchEntityId,
    pub member: SketchEntityId,
    pub offset: [f64; 2],
}
impl Sketch2D {
    /// Add a checked directed point link; example: `sketch.add_pattern_point_link(link)?;`.
    pub fn add_pattern_point_link(
        &mut self,
        link: SketchPatternPointLink,
    ) -> Result<(), SketchError> {
        self.point(link.source)?;
        self.point(link.member)?;
        require_position(link.offset)?;
        if self
            .pattern_point_links
            .iter()
            .any(|existing| existing.member == link.member)
        {
            return Err(link.invalid("a member with no existing pattern point link"));
        }
        let mut ancestor = link.source;
        loop {
            if ancestor == link.member {
                return Err(link.invalid("an acyclic point dependency"));
            }
            match self
                .pattern_point_links
                .iter()
                .find(|existing| existing.member == ancestor)
            {
                Some(existing) => ancestor = existing.source,
                None => break,
            }
        }
        self.pattern_point_links.push(link);
        Ok(())
    }
    /// Inspect structural point links; example: `sketch.pattern_point_links()`.
    pub fn pattern_point_links(&self) -> &[SketchPatternPointLink] {
        &self.pattern_point_links
    }
    pub(super) fn capture_pattern_point_links(&self, recipe: &mut SketchRecipe) {
        if self.pattern_point_links.is_empty() && self.pattern_radius_links.is_empty() {
            return;
        }
        self.capture_point_ownership_metadata(recipe);
        recipe.version = 14;
        recipe.pattern_point_links = Some(self.pattern_point_links.clone());
        if !self.pattern_radius_links.is_empty() {
            recipe.pattern_radius_links = Some(self.pattern_radius_links.clone());
        }
    }
    pub(super) fn restore_pattern_point_links(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        if recipe.version < 14 && recipe.pattern_radius_links.is_some() {
            return Err(SketchError::Definition(format!(
                "pattern radius links in version {}: expected version 14",
                recipe.version
            )));
        }
        match (recipe.version, &recipe.pattern_point_links) {
            (1..=13, None) | (16..=super::SKETCH_RECIPE_VERSION, None)
                if recipe.pattern_radius_links.is_none() =>
            {
                Ok(())
            }
            (14..=super::SKETCH_RECIPE_VERSION, Some(links))
                if !links.is_empty()
                    || recipe
                        .pattern_radius_links
                        .as_ref()
                        .is_some_and(|r| !r.is_empty()) =>
            {
                for link in links {
                    self.add_pattern_point_link(*link)?;
                }
                for link in recipe.pattern_radius_links.iter().flatten() {
                    self.add_pattern_radius_link(*link)?;
                }
                Ok(())
            }
            shape => Err(SketchError::Definition(format!(
                "pattern link recipe {shape:?}: expected no links before version 14 or at least one link in version 14"
            ))),
        }
    }
}
impl SketchPatternPointLink {
    fn invalid(self, expected: &str) -> SketchError {
        SketchError::Definition(format!("pattern point link {self:?}: expected {expected}"))
    }
}
