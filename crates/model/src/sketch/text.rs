// SPDX-License-Identifier: GPL-2.0-only
//! Retain authored sketch text independently of its derived glyph outlines.
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe};
use oblikovati_api::types::{
    Color, SketchTextLineSpacing, SketchTextRun, SketchTextStyle, TEXT_ALIGN_BASELINE,
    TEXT_ALIGN_CENTER, TEXT_ALIGN_LEFT, TEXT_ALIGN_LOWER, TEXT_ALIGN_MIDDLE, TEXT_ALIGN_RIGHT,
    TEXT_ALIGN_UPPER,
};

const TEXT_RECIPE_VERSION: u32 = 22;

/// Read a uniform width in percent, for example a label with no runs returns `Some(100.)`.
pub fn uniform_sketch_text_width(content: &str, runs: &[SketchTextRun]) -> Option<f64> {
    uniform_text_property(content, runs, 100., |run| {
        *run.width_percent.as_deref().unwrap_or(&100.)
    })
}

/// Read an overall color, for example `Some(None)` means all characters inherit their color.
/// Mixed inherited and explicit colors, or different explicit colors, return `None`.
pub fn uniform_sketch_text_color<'a>(
    content: &str,
    runs: &'a [SketchTextRun],
) -> Option<Option<&'a Color>> {
    uniform_text_property(content, runs, None, |run| run.color.as_deref())
}

fn uniform_text_property<'a, T: Copy + PartialEq>(
    content: &str,
    runs: &'a [SketchTextRun],
    base: T,
    value: impl Fn(&'a SketchTextRun) -> T,
) -> Option<T> {
    let mut prior_end = 0;
    let mut result = None;
    for run in runs {
        let current = value(run);
        let expected = *result.get_or_insert(if run.start == 0 { current } else { base });
        if current != expected || (run.start != prior_end && expected != base) {
            return None;
        }
        prior_end = run.end;
    }
    let uniform = result.unwrap_or(base);
    (uniform == base || usize::try_from(prior_end).ok() == Some(content.chars().count()))
        .then_some(uniform)
}

/// Convert percent to the font scale, for example `sketch_text_width_factor(50.) == Some(0.5)`.
pub fn sketch_text_width_factor(percent: f64) -> Option<f32> {
    let factor = (percent / 100.) as f32;
    (factor.is_finite() && factor > 0.).then_some(factor)
}

#[cfg(test)]
mod width_tests {
    use super::*;

    #[test]
    fn width_requires_the_same_scale_for_all_unicode_scalars() {
        let run = SketchTextRun {
            start: 0,
            end: 1,
            width_percent: Some(Box::new(50.)),
            ..Default::default()
        };
        assert_eq!(
            uniform_sketch_text_width("界", std::slice::from_ref(&run)),
            Some(50.)
        );
        assert_eq!(uniform_sketch_text_width("界B", &[run]), None);
        assert_eq!(uniform_sketch_text_width("界", &[]), Some(100.));
        assert_eq!(sketch_text_width_factor(50.), Some(0.5));
        assert_eq!(sketch_text_width_factor(f64::MIN_POSITIVE), None);
    }

    #[test]
    fn color_distinguishes_inherited_explicit_and_mixed_runs() {
        let run = SketchTextRun {
            start: 0,
            end: 1,
            color: Some(Box::new(Color {
                b: 255,
                opacity: 1.,
                source: oblikovati_api::types::OVERRIDE_COLOR_SOURCE,
                ..Default::default()
            })),
            ..Default::default()
        };
        let runs = std::slice::from_ref(&run);
        assert_eq!(uniform_sketch_text_color("界", &[]), Some(None));
        assert_eq!(
            uniform_sketch_text_color("界", runs),
            Some(run.color.as_deref())
        );
        assert_eq!(uniform_sketch_text_color("界B", runs), None);
    }
}

/// A text entity and its defining sketch point. The contract style uses centimetres and radians.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchText2D {
    pub id: SketchEntityId,
    pub anchor: SketchEntityId,
    pub style: SketchTextStyle,
}

impl Sketch2D {
    /// Inspect authored text in insertion order, for example `sketch.texts()`.
    pub fn texts(&self) -> &[SketchText2D] {
        &self.texts
    }

    /// Read text by stable identity, for example `sketch.text(id)?`.
    pub fn text(&self, id: SketchEntityId) -> Result<&SketchText2D, SketchError> {
        self.texts.iter().find(|text| text.id == id).ok_or_else(|| {
            SketchError::Definition(format!(
                "sketch text {id:?}: expected an existing text entity"
            ))
        })
    }

    /// Add an authored text entity after its anchor, for example `sketch.add_text(text)?`.
    pub fn add_text(&mut self, mut text: SketchText2D) -> Result<(), SketchError> {
        self.require_new_entity(text.id)?;
        self.point(text.anchor)?;
        normalize_text_style(&mut text.style)?;
        self.entity_order.push(text.id);
        self.texts.push(text);
        Ok(())
    }

    /// Change content and style without changing identity, for example `sketch.set_text_style(id, style)?`.
    pub fn set_text_style(
        &mut self,
        id: SketchEntityId,
        mut style: SketchTextStyle,
    ) -> Result<(), SketchError> {
        let text = self
            .texts
            .iter_mut()
            .find(|text| text.id == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch text {id:?}: expected an existing text entity"
                ))
            })?;
        normalize_text_style(&mut style)?;
        text.style = style;
        Ok(())
    }

    pub(super) fn capture_text_recipe(&self, recipe: &mut SketchRecipe) {
        if !self.texts.is_empty() {
            self.capture_point_ownership_metadata(recipe);
            recipe.texts = Some(self.texts.clone());
            recipe.version = TEXT_RECIPE_VERSION;
        }
    }

    pub(super) fn restore_text_recipe(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        match (recipe.version, recipe.texts.as_deref()) {
            (1..TEXT_RECIPE_VERSION, None) => Ok(()),
            (TEXT_RECIPE_VERSION, Some(texts)) if !texts.is_empty() => texts
                .iter()
                .try_for_each(|text| self.add_text(text.clone())),
            (version, texts) => Err(SketchError::Definition(format!(
                "text recipe {texts:?} in version {version}: expected a nonempty text list in version {TEXT_RECIPE_VERSION} and no list in earlier versions"
            ))),
        }
    }
}

fn normalize_text_style(style: &mut SketchTextStyle) -> Result<(), SketchError> {
    require_text_metrics(style)?;
    if style.font_size == 0. {
        style.font_size = 0.;
    }
    if style.rotation == 0. {
        style.rotation = 0.;
    }
    if style.h_align.0.is_empty() {
        style.h_align = TEXT_ALIGN_LEFT;
    }
    if style.v_align.0.is_empty() {
        style.v_align = TEXT_ALIGN_BASELINE;
    }
    require_text_alignment(style)?;
    SketchTextLineSpacing::validate(style.line_spacing.as_deref())
        .map_err(|error| SketchError::Definition(error.to_string()))?;
    SketchTextRun::validate(
        Some(&style.content),
        style.formatting_runs.as_deref().unwrap_or_default(),
    )
    .map_err(|error| SketchError::Definition(error.to_string()))
}

fn require_text_metrics(style: &SketchTextStyle) -> Result<(), SketchError> {
    let SketchTextStyle {
        height,
        font_size,
        rotation,
        ..
    } = *style;
    if !height.is_finite()
        || height <= 0.
        || !font_size.is_finite()
        || font_size < 0.
        || !rotation.is_finite()
    {
        return Err(SketchError::Definition(format!(
            "sketch text metrics {height:?}, {font_size:?}, {rotation:?}: expected positive height, nonnegative font size and finite rotation"
        )));
    }
    Ok(())
}

fn require_text_alignment(style: &SketchTextStyle) -> Result<(), SketchError> {
    if ![TEXT_ALIGN_LEFT, TEXT_ALIGN_CENTER, TEXT_ALIGN_RIGHT].contains(&style.h_align)
        || ![
            TEXT_ALIGN_BASELINE,
            TEXT_ALIGN_LOWER,
            TEXT_ALIGN_MIDDLE,
            TEXT_ALIGN_UPPER,
        ]
        .contains(&style.v_align)
    {
        return Err(SketchError::Definition(format!(
            "sketch text alignment {:?}, {:?}: expected left/center/right and baseline/lower/middle/upper",
            style.h_align, style.v_align
        )));
    }
    Ok(())
}
