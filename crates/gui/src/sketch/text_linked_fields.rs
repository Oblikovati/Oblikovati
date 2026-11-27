// SPDX-License-Identifier: GPL-2.0-only
//! Insert one authored placeholder for each live document field.
use super::text_field_projection::inferred_byte_edit;
use super::{GuiSketchTextDraft, GuiSketchTextFieldSources, character_ranges, text_fraction};
use imgui::Ui;
use oblikovati_api::types::{
    SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER,
    SketchTextField,
};
use oblikovati_model::sketch::SKETCH_TEXT_FILE_NAME_LABEL as FILENAME;
use std::ops::Range;

const DECIMALS: [&str; 9] = [
    "0",
    "0.0",
    "0.00",
    "0.000",
    "0.0000",
    "0.00000",
    "0.000000",
    "0.0000000",
    "0.00000000",
];

#[derive(Clone, Copy, PartialEq, Eq)]
enum SourceKind {
    Model,
    User,
    Finish,
    Standard,
}
impl SourceKind {
    const CHOICES: [Self; 4] = [Self::Model, Self::User, Self::Finish, Self::Standard];

    fn caption(self) -> &'static str {
        match self {
            Self::Model => "Parameters - Model",
            Self::User => "Parameters - User",
            Self::Finish => "Parameters - Finish",
            Self::Standard => "Standard iProperties",
        }
    }

    fn names(self, sources: &GuiSketchTextFieldSources) -> impl Iterator<Item = &str> + Clone {
        let parameters: &[String] = match self {
            Self::Model => &sources.model_parameters,
            Self::User => &sources.user_parameters,
            _ => &[],
        };
        parameters
            .iter()
            .map(String::as_str)
            .chain((self == Self::Standard).then_some(FILENAME))
    }

    fn shows_precision(self) -> bool {
        matches!(self, Self::Model | Self::User)
    }
}

pub(super) struct LinkedFieldControls {
    kind: SourceKind,
    field_name: Option<String>,
    precision: usize,
}

impl Default for LinkedFieldControls {
    fn default() -> Self {
        Self {
            kind: SourceKind::Model,
            field_name: None,
            precision: 3,
        }
    }
}

impl LinkedFieldControls {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        sources: &GuiSketchTextFieldSources,
    ) -> Option<SketchTextField> {
        self.draw_source(ui, sources);
        let names = self.kind.names(sources);
        self.draw_field_selector(ui, names.clone());
        ui.same_line();
        self.draw_precision(ui);
        ui.same_line();
        let field = self.resolve_selection(names);
        let _disabled = ui.begin_disabled(field.is_none());
        ui.button("Insert").then_some(field).flatten()
    }

    fn draw_source(&mut self, ui: &Ui, sources: &GuiSketchTextFieldSources) {
        ui.set_next_item_width(180.);
        if let Some(_list) = ui.begin_combo("Type", self.kind.caption()) {
            for kind in SourceKind::CHOICES {
                if ui
                    .selectable_config(kind.caption())
                    .selected(kind == self.kind)
                    .build()
                {
                    self.kind = kind;
                    self.field_name = None;
                }
            }
        }
        ui.same_line();
        ui.text("Source");
        if self.kind != SourceKind::Finish {
            ui.same_line();
            ui.text(&sources.document);
        }
    }

    fn draw_field_selector<'a>(&mut self, ui: &Ui, names: impl Iterator<Item = &'a str> + Clone) {
        let _disabled = ui.begin_disabled(names.clone().next().is_none());
        ui.set_next_item_width(128.);
        let caption = self.field_name.as_deref().unwrap_or("Select field");
        if let Some(_choices) = ui.begin_combo("Field", caption) {
            for name in names {
                if ui
                    .selectable_config(name)
                    .selected(Some(name) == self.field_name.as_deref())
                    .build()
                {
                    self.field_name = Some(name.into());
                }
            }
        }
    }

    fn draw_precision(&mut self, ui: &Ui) {
        let available = self.kind.shows_precision();
        let _disabled = ui.begin_disabled(!available);
        ui.set_next_item_width(94.);
        let caption = if available {
            DECIMALS[self.precision]
        } else {
            ""
        };
        if let Some(_choices) = ui.begin_combo("Precision", caption) {
            for (digits, value) in DECIMALS.iter().enumerate() {
                if ui
                    .selectable_config(value)
                    .selected(digits == self.precision)
                    .build()
                {
                    self.precision = digits;
                }
            }
        }
    }

    fn resolve_selection<'a>(
        &self,
        mut names: impl Iterator<Item = &'a str>,
    ) -> Option<SketchTextField> {
        let parameter = self
            .field_name
            .as_deref()
            .filter(|name| names.any(|choice| choice == *name))?;
        if self.kind == SourceKind::Standard {
            return Some(SketchTextField {
                kind: SKETCH_TEXT_FIELD_FILE_NAME,
                ..Default::default()
            });
        }
        Some(SketchTextField {
            kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
            parameter: parameter.into(),
            precision: Some(Box::new(self.precision as u8)),
            ..Default::default()
        })
    }
}

pub(super) fn insert_field(
    draft: &mut GuiSketchTextDraft,
    selection: Range<usize>,
    mut field: SketchTextField,
) -> Option<Range<usize>> {
    let start = u32::try_from(draft.content.get(..selection.start)?.chars().count()).ok()?;
    let count = u32::try_from(draft.content.get(selection.clone())?.chars().count()).ok()?;
    let old = start..start.checked_add(count)?;
    let new = start..start.checked_add(1)?;
    let end = selection.start + SKETCH_TEXT_FIELD_PLACEHOLDER.len();
    let before = draft.content.clone();
    draft
        .content
        .replace_range(selection, SKETCH_TEXT_FIELD_PLACEHOLDER);
    keep_fields_outside(draft, old, new);
    keep_other_formatting(draft, &before);
    let fields = draft.formatting.linked_fields.get_or_insert_with(Vec::new);
    field.at = start;
    let index = fields.partition_point(|existing| existing.at < field.at);
    fields.insert(index, field);
    Some(end..end)
}

fn keep_other_formatting(draft: &mut GuiSketchTextDraft, before: &str) {
    if before == draft.content {
        return;
    }
    let runs = draft.formatting.runs.as_deref().unwrap_or_default();
    let retained = character_ranges::retain_unchanged_runs(runs, before, &draft.content);
    draft.formatting.runs = (!retained.is_empty()).then_some(retained);
    text_fraction::retain_unchanged(draft, before);
}

#[cfg(test)]
#[path = "text_linked_fields_tests.rs"]
mod tests;

pub(super) fn retain_unchanged(draft: &mut GuiSketchTextDraft, before: &str) {
    if draft.formatting.linked_fields.is_some()
        && let Some((old, new)) = character_ranges::replacement_bounds(before, &draft.content)
    {
        keep_fields_outside(draft, old, new);
    }
}

pub(super) fn retain_after_input(
    draft: &mut GuiSketchTextDraft,
    before: &str,
    previous: Range<usize>,
    caret: usize,
) {
    if draft.formatting.linked_fields.is_none() {
        return;
    }
    if let Some((old, new)) = edited_scalar_ranges(before, &draft.content, previous, caret) {
        keep_fields_outside(draft, old, new);
    } else {
        retain_unchanged(draft, before);
    }
}

fn edited_scalar_ranges(
    before: &str,
    after: &str,
    previous: Range<usize>,
    caret: usize,
) -> Option<(Range<u32>, Range<u32>)> {
    let (old, new) = inferred_byte_edit(before, after, previous, caret)?;
    let first = u32::try_from(before.get(..old.start)?.chars().count()).ok()?;
    let last = u32::try_from(before.get(..old.end)?.chars().count()).ok()?;
    let next = u32::try_from(after.get(..new.end)?.chars().count()).ok()?;
    Some((first..last, first..next))
}

fn keep_fields_outside(draft: &mut GuiSketchTextDraft, old: Range<u32>, new: Range<u32>) {
    let Some(fields) = draft.formatting.linked_fields.as_mut() else {
        return;
    };
    fields.retain_mut(|field| {
        if field.at < old.start {
            return true;
        }
        if field.at < old.end {
            return false;
        }
        field.at = new.end + (field.at - old.end);
        true
    });
    if fields.is_empty() {
        draft.formatting.linked_fields = None;
    }
}
