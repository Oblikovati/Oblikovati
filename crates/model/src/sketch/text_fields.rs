// SPDX-License-Identifier: GPL-2.0-only
//! Expand field values for display without changing the authored sketch text.
use super::SketchError;
use oblikovati_api::types::{SketchTextField, SketchTextFraction, SketchTextRun, SketchTextStyle};
use std::borrow::Cow;
use std::ops::Range;

/// The authored marker and display value use independent UTF-8 byte ranges.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchTextFieldSpan {
    pub authored: Range<usize>,
    pub display: Range<usize>,
}

/// Text and field bounds for an editor displaying expanded linked values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchTextFieldProjection {
    pub content: String,
    pub fields: Vec<SketchTextFieldSpan>,
}

/// Resolve linked values and move style boundaries to their display offsets.
/// For example, a field whose resolver returns `"10.000"` styles all six characters
/// with the style of its one authored placeholder. Literal text can stay borrowed.
pub fn resolve_sketch_text_display(
    authored: &SketchTextStyle,
    mut value_of: impl FnMut(&SketchTextField) -> Result<String, SketchError>,
) -> Result<Cow<'_, SketchTextStyle>, SketchError> {
    if authored.linked_fields.as_ref().is_none_or(Vec::is_empty) {
        return Ok(Cow::Borrowed(authored));
    }
    let expansion = expand_style(authored, &mut value_of)?;
    Ok(Cow::Owned(expansion.rebase_style(authored)?))
}

pub(super) fn project_sketch_text_fields(
    authored: &SketchTextStyle,
    mut value_of: impl FnMut(&SketchTextField) -> Result<String, SketchError>,
) -> Result<SketchTextFieldProjection, SketchError> {
    let expansion = expand_style(authored, &mut value_of)?;
    Ok(SketchTextFieldProjection {
        content: expansion.content,
        fields: expansion
            .offsets
            .into_iter()
            .map(|offset| offset.span)
            .collect(),
    })
}

fn expand_style(
    authored: &SketchTextStyle,
    value_of: &mut impl FnMut(&SketchTextField) -> Result<String, SketchError>,
) -> Result<FieldExpansion, SketchError> {
    let fields = authored.linked_fields.as_deref().unwrap_or_default();
    let fractions = authored.stacked_fractions.as_deref().unwrap_or_default();
    SketchTextFraction::validate(Some(&authored.content), fractions)
        .and_then(|()| SketchTextField::validate(Some(&authored.content), fields, fractions))
        .and_then(|()| {
            SketchTextRun::validate(
                Some(&authored.content),
                authored.formatting_runs.as_deref().unwrap_or_default(),
            )
        })
        .map_err(|error| SketchError::Definition(error.to_string()))?;
    expand_source(&authored.content, fields, value_of)
}

struct FieldExpansion {
    content: String,
    offsets: Vec<FieldOffset>,
}

struct FieldOffset {
    source_scalar: u32,
    // A field extends only the boundaries strictly after its placeholder.
    cumulative_shift: u32,
    span: SketchTextFieldSpan,
}

fn expand_source(
    source: &str,
    fields: &[SketchTextField],
    value_of: &mut impl FnMut(&SketchTextField) -> Result<String, SketchError>,
) -> Result<FieldExpansion, SketchError> {
    let mut expansion = FieldExpansion {
        content: String::with_capacity(source.len()),
        offsets: Vec::with_capacity(fields.len()),
    };
    let mut next = fields.iter().peekable();
    for (scalar, (byte, ch)) in source.char_indices().enumerate() {
        if next.peek().is_some_and(|field| field.at as usize == scalar) {
            let field = next.next().unwrap();
            expansion.push_field(field.at, byte..byte + ch.len_utf8(), &value_of(field)?)?;
        } else {
            expansion.content.push(ch);
        }
    }
    Ok(expansion)
}

impl FieldExpansion {
    fn push_field(
        &mut self,
        position: u32,
        authored: Range<usize>,
        value: &str,
    ) -> Result<(), SketchError> {
        if value.is_empty() || value.contains('\0') {
            return Err(SketchError::Definition(format!(
                "text field at {position} resolves to {value:?}: expected nonempty text without NUL"
            )));
        }
        let prior_shift = self
            .offsets
            .last()
            .map_or(0, |offset| offset.cumulative_shift);
        let length = field_offset(value.chars().count())?;
        let shift = prior_shift
            .checked_add(length - 1)
            .ok_or_else(|| expansion_error(u64::from(prior_shift) + u64::from(length) - 1))?;
        let display_start = self.content.len();
        self.content.push_str(value);
        self.offsets.push(FieldOffset {
            source_scalar: position,
            cumulative_shift: shift,
            span: SketchTextFieldSpan {
                authored,
                display: display_start..self.content.len(),
            },
        });
        Ok(())
    }

    fn rebase_style(self, authored: &SketchTextStyle) -> Result<SketchTextStyle, SketchError> {
        field_offset(self.content.chars().count())?;
        let mut display = authored.clone();
        for run in display.formatting_runs.iter_mut().flatten() {
            (run.start, run.end) = self.display_range(run.start, run.end)?;
        }
        for fraction in display.stacked_fractions.iter_mut().flatten() {
            (fraction.start, fraction.end) = self.display_range(fraction.start, fraction.end)?;
        }
        display.content = self.content;
        display.linked_fields = None;
        Ok(display)
    }

    fn display_range(&self, start: u32, end: u32) -> Result<(u32, u32), SketchError> {
        Ok((self.display_boundary(start)?, self.display_boundary(end)?))
    }

    fn display_boundary(&self, source: u32) -> Result<u32, SketchError> {
        let preceding = self
            .offsets
            .partition_point(|offset| offset.source_scalar < source);
        let shift = preceding
            .checked_sub(1)
            .map_or(0, |index| self.offsets[index].cumulative_shift);
        let expanded = u64::from(source) + u64::from(shift);
        u32::try_from(expanded).map_err(|_| expansion_error(expanded))
    }
}

fn field_offset(count: usize) -> Result<u32, SketchError> {
    u32::try_from(count).map_err(|_| expansion_error(count as u64))
}

fn expansion_error(count: u64) -> SketchError {
    SketchError::Definition(format!(
        "expanded text offset {count}: expected a 32-bit Unicode scalar offset"
    ))
}

#[cfg(test)]
#[path = "text_fields_tests.rs"]
mod tests;
