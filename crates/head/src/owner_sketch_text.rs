// SPDX-License-Identifier: GPL-2.0-only
//! Format Text owns a transient draft until the existing document command accepts it.
use super::{sketch_picking::ordinary_click, *};
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::types::{SketchTextStyle, TEXT_ALIGN_BASELINE, TEXT_ALIGN_LEFT};
use oblikovati_gui::{
    GuiSketchTextAction, GuiSketchTextDraft, GuiSketchTextFormatting, GuiSketchTextPresetId,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    identity::DocumentGuid,
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchText2D},
};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportSketchPreview};
use std::sync::{Arc, OnceLock};

#[path = "owner_sketch_text_activation.rs"]
mod activation;
#[path = "owner_sketch_text_commit.rs"]
mod commit;

/// Session links use the document recipe identity so reopening a file keeps the same link.
pub(super) type TextPresetLinks = std::collections::BTreeMap<TextPresetKey, GuiSketchTextPresetId>;
type TextPresetKey = (DocumentGuid, u64, SketchEntityId);

pub(crate) struct SketchTextTool {
    pub(crate) edit: HeadSketchEdit,
    pub(crate) generation: u64,
    working_scale: f64,
    placement: Option<TextPlacementDraft>,
    font_families: OnceLock<Arc<[String]>>,
}

struct TextPlacementDraft {
    anchor: [f64; 2],
    fields: GuiSketchTextDraft,
    existing: Option<SketchText2D>,
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Activate Text for the selected planar sketch, for example from the Part ribbon.
    pub(super) fn start_sketch_text(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.select_placement_sketch(document, sketch)?;
        self.retire_sketch_tool()?;
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let working_scale = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| e.to_string())?
            .snapshot()
            .parameters
            .units()
            .working_scale();
        self.sketch_tool = Some(SketchToolState::Text(Box::new(SketchTextTool {
            edit,
            generation,
            working_scale,
            placement: None,
            font_families: OnceLock::new(),
        })));
        self.sketch_text_prompt();
        Ok(())
    }

    pub(super) fn process_sketch_text(
        &mut self,
        mut tool: Box<SketchTextTool>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if matches!(
            event,
            ViewportEvent::Input(ViewportInput::Key {
                key: ViewportKey::Cancel,
                pressed: true
            })
        ) {
            if tool
                .placement
                .take()
                .is_some_and(|placement| placement.existing.is_none())
            {
                self.sketch_tool = Some(SketchToolState::Text(tool));
                self.sketch_text_prompt();
            } else {
                self.command_notice = Some(DesktopCommandNotice::Ready);
            }
            return true;
        }
        let result = pick_text_anchor(&mut tool, event, window);
        self.sketch_tool = Some(SketchToolState::Text(tool));
        result.unwrap_or_else(|cause| {
            self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
            true
        })
    }

    pub(super) fn gui_sketch_text(&self) -> Option<GuiSketchTextDraft> {
        let Some(SketchToolState::Text(tool)) = &self.sketch_tool else {
            return None;
        };
        tool.placement
            .as_ref()
            .map(|placement| placement.fields.clone())
    }

    pub(super) fn gui_sketch_text_existing(&self) -> bool {
        matches!(&self.sketch_tool, Some(SketchToolState::Text(tool))
            if tool.placement.as_ref().is_some_and(|draft| draft.existing.is_some()))
    }

    pub(super) fn gui_sketch_text_preset(&self) -> Option<GuiSketchTextPresetId> {
        let Some(SketchToolState::Text(tool)) = &self.sketch_tool else {
            return None;
        };
        let entity = tool.placement.as_ref()?.existing.as_ref()?.id;
        let document = self.files.workspace().identity(tool.edit.document).ok()?;
        self.text_preset_links
            .get(&(document.internal_name, tool.edit.sketch.0, entity))
            .copied()
    }

    pub(super) fn gui_sketch_text_font_families(&self) -> Option<Arc<[String]>> {
        let Some(SketchToolState::Text(tool)) = &self.sketch_tool else {
            return None;
        };
        tool.placement.as_ref()?;
        Some(
            tool.font_families
                .get_or_init(|| {
                    oblikovati_martensite::planar_curve::sketch::available_sketch_font_families()
                        .into()
                })
                .clone(),
        )
    }

    pub(super) fn sketch_text_preview(
        &self,
        tool: &SketchTextTool,
    ) -> Option<ViewportSketchPreview> {
        let placement = tool.placement.as_ref()?;
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return None;
        }
        let style = placement.style().ok()?;
        Some(ViewportSketchPreview::Text {
            plane: tool.edit.plane,
            anchor: placement.anchor,
            style: Box::new(style),
            working_scale: tool.working_scale,
            replaces: placement
                .existing
                .as_ref()
                .map(|text| (tool.edit.sketch, text.id)),
        })
    }

    pub(super) fn sketch_text_command(
        &mut self,
        action: GuiSketchTextAction,
    ) -> Result<bool, String> {
        self.check_sketch_tool_context();
        if let GuiSketchTextAction::Edit { token, entity } = action {
            return self.start_text_edit(token, entity);
        }
        let Some(SketchToolState::Text(tool)) = &mut self.sketch_tool else {
            return Ok(false);
        };
        let Some(placement) = tool.placement.as_mut() else {
            return Ok(false);
        };
        match action {
            GuiSketchTextAction::Preview(fields) => placement.fields = fields,
            GuiSketchTextAction::Cancel => return self.cancel_text_editor(),
            GuiSketchTextAction::Accept { draft, preset } => {
                return self.accept_sketch_text(draft, preset);
            }
            GuiSketchTextAction::Edit { .. } => unreachable!("handled before reading the draft"),
        }
        Ok(true)
    }

    fn cancel_text_editor(&mut self) -> Result<bool, String> {
        let Some(SketchToolState::Text(mut tool)) = self.sketch_tool.take() else {
            return Ok(false);
        };
        if tool
            .placement
            .take()
            .is_some_and(|draft| draft.existing.is_none())
        {
            self.sketch_tool = Some(SketchToolState::Text(tool));
            self.sketch_text_prompt();
        } else {
            self.command_notice = Some(DesktopCommandNotice::Ready);
        }
        Ok(true)
    }

    fn start_text_edit(&mut self, token: u64, entity: SketchEntityId) -> Result<bool, String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        let target_matches = self.sketch_memory.context.as_ref().is_some_and(|target| {
            target.token == token
                && target.text == Some(entity)
                && target.generation == generation
                && target.document == edit.document
                && target.sketch == edit.sketch
        });
        if !target_matches || self.sketch_tool.is_some() {
            return Err(format!(
                "text context {token}, entity {entity:?}: expected the current glyph in a Ready sketch"
            ));
        }
        self.open_text_draft(edit, generation, entity)
    }

    fn open_text_draft(
        &mut self,
        edit: HeadSketchEdit,
        generation: u64,
        entity: SketchEntityId,
    ) -> Result<bool, String> {
        self.sketch_tool = Some(SketchToolState::Text(Box::new(
            self.existing_text_tool(edit, generation, entity)?,
        )));
        self.sketch_memory.context = None;
        Ok(true)
    }

    fn existing_text_tool(
        &self,
        edit: HeadSketchEdit,
        generation: u64,
        entity: SketchEntityId,
    ) -> Result<SketchTextTool, String> {
        let snapshot = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?
            .snapshot();
        let geometry = snapshot
            .sketches
            .get(edit.sketch)
            .map_err(|e| e.to_string())?
            .geometry();
        let text = geometry.text(entity).map_err(|e| e.to_string())?.clone();
        let anchor = geometry
            .point(text.anchor)
            .map_err(|e| e.to_string())?
            .position;
        let fields = fields_from_style(&text.style);
        Ok(SketchTextTool {
            edit,
            generation,
            working_scale: snapshot.parameters.units().working_scale(),
            font_families: OnceLock::new(),
            placement: Some(TextPlacementDraft {
                anchor,
                fields,
                existing: Some(text),
            }),
        })
    }

    fn accept_sketch_text(
        &mut self,
        fields: GuiSketchTextDraft,
        preset: Option<GuiSketchTextPresetId>,
    ) -> Result<bool, String> {
        let Some(SketchToolState::Text(mut tool)) = self.sketch_tool.take() else {
            return Ok(false);
        };
        let editing = tool
            .placement
            .as_ref()
            .is_some_and(|draft| draft.existing.is_some());
        let result = self.commit_text_candidate(&mut tool, fields, preset);
        if editing && result.is_ok() {
            self.command_notice = Some(DesktopCommandNotice::Ready);
        } else {
            self.sketch_tool = Some(SketchToolState::Text(tool));
            if result.is_ok() {
                self.sketch_text_prompt();
            }
        }
        result
    }

    fn sketch_text_prompt(&mut self) {
        self.placement_notice("Select text insertion point".into());
    }
}

fn pick_text_anchor(
    tool: &mut SketchTextTool,
    event: ViewportEvent,
    window: &dyn HeadWindow,
) -> Result<bool, String> {
    let Some((pixel, _)) =
        ordinary_click(event).filter(|_| tool.placement.is_none() && !window.navigation_active())
    else {
        return Ok(false);
    };
    let Some(anchor) = window.sketch_point(pixel, tool.edit.plane)? else {
        return Ok(false);
    };
    tool.placement = Some(TextPlacementDraft {
        anchor,
        fields: GuiSketchTextDraft::default(),
        existing: None,
    });
    Ok(true)
}

fn text_style(fields: &GuiSketchTextDraft) -> Result<SketchTextStyle, String> {
    let height = finite_text_number(&fields.height_mm, "height in millimetres")? / 10.;
    let rotation =
        finite_text_number(&fields.rotation_degrees, "rotation in degrees")?.to_radians();
    if height <= 0. || !rotation.is_finite() {
        return Err(format!(
            "text height {height:?} and rotation {rotation:?}: expected positive height and finite rotation"
        ));
    }
    Ok(SketchTextStyle {
        content: fields.content.clone(),
        family: fields.family.trim().into(),
        height,
        rotation,
        h_align: fields.h_align.clone(),
        v_align: fields.v_align.clone(),
        line_spacing: fields.formatting.line_spacing.clone(),
        formatting_runs: fields.formatting.runs.clone(),
        stacked_fractions: fields.formatting.stacked_fractions.clone(),
        ..Default::default()
    })
}

fn finite_text_number(value: &str, label: &str) -> Result<f64, String> {
    value
        .trim()
        .parse::<f64>()
        .ok()
        .filter(|value| value.is_finite())
        .ok_or_else(|| format!("text {label} {value:?}: expected a finite decimal value"))
}

fn verify_text_outline(
    style: &SketchTextStyle,
    anchor: [f64; 2],
    working_scale: f64,
) -> Result<(), String> {
    use oblikovati_martensite::planar_curve::sketch::{
        SketchSamplingOptionsV1, sample_sketch_text,
    };
    // The viewport can refine these outlines at its own pixel scale after acceptance.
    let options = SketchSamplingOptionsV1 {
        working_scale,
        deflection: style.height / working_scale / 64.,
        maximum_curve_points: 4096,
    };
    sample_sketch_text(style, anchor, options).map(|_| ())
}

impl TextPlacementDraft {
    fn style(&self) -> Result<SketchTextStyle, String> {
        let draft = text_style(&self.fields)?;
        let Some(text) = &self.existing else {
            return Ok(draft);
        };
        let initial = fields_from_style(&text.style);
        let mut style = text.style.clone();
        style.content = draft.content;
        style.family = draft.family;
        style.h_align = draft.h_align;
        style.v_align = draft.v_align;
        style.line_spacing = draft.line_spacing;
        style.formatting_runs = draft.formatting_runs;
        style.stacked_fractions = draft.stacked_fractions;
        if self.fields.height_mm != initial.height_mm {
            style.height = draft.height;
        }
        if self.fields.rotation_degrees != initial.rotation_degrees {
            style.rotation = draft.rotation;
        }
        Ok(style)
    }
}

fn text_height_field(height_cm: f64) -> String {
    let mut value = (height_cm * 10.).to_string();
    match value.split_once('.').map_or(0, |(_, digits)| digits.len()) {
        0 => value.push_str(".00"),
        1 => value.push('0'),
        _ => {}
    }
    value
}

fn fields_from_style(style: &SketchTextStyle) -> GuiSketchTextDraft {
    GuiSketchTextDraft {
        content: style.content.clone(),
        family: style.family.clone(),
        height_mm: text_height_field(style.height),
        rotation_degrees: style.rotation.to_degrees().to_string(),
        h_align: if style.h_align.0.is_empty() {
            TEXT_ALIGN_LEFT
        } else {
            style.h_align.clone()
        },
        v_align: if style.v_align.0.is_empty() {
            TEXT_ALIGN_BASELINE
        } else {
            style.v_align.clone()
        },
        formatting: Box::new(GuiSketchTextFormatting {
            line_spacing: style.line_spacing.clone(),
            runs: style.formatting_runs.clone(),
            stacked_fractions: style.stacked_fractions.clone(),
        }),
    }
}

pub(super) fn pick_existing_text(
    geometry: &Sketch2D,
    plane: SketchPlane,
    pixel: [i32; 2],
    scale: f64,
    window: &dyn HeadWindow,
) -> Result<Option<SketchEntityId>, String> {
    if geometry.texts().is_empty() {
        return Ok(None);
    }
    let Some(cursor) = window.sketch_point(pixel, plane)? else {
        return Ok(None);
    };
    let mut candidates = Vec::new();
    for text in geometry.texts() {
        if let Some(distance) =
            text_hit_distance(geometry, text, cursor, scale, plane, pixel, window)?
        {
            candidates.push((distance, text.id));
        }
    }
    candidates.sort_by(|first, next| first.0.total_cmp(&next.0).then(first.1.cmp(&next.1)));
    Ok(candidates.first().map(|(_, id)| *id))
}

fn text_hit_distance(
    geometry: &Sketch2D,
    text: &SketchText2D,
    cursor: [f64; 2],
    scale: f64,
    plane: SketchPlane,
    pixel: [i32; 2],
    window: &dyn HeadWindow,
) -> Result<Option<f64>, String> {
    use oblikovati_martensite::planar_curve::sketch::pick_sketch_text;
    let anchor = geometry
        .point(text.anchor)
        .map_err(|cause| cause.to_string())?
        .position;
    // Unavailable fonts already have an entity-specific display decline.
    let Ok(Some(point)) = pick_sketch_text(&text.style, anchor, scale, cursor) else {
        return Ok(None);
    };
    let projected = window.sketch_pixel(plane, point)?;
    let distance = super::sketch_picking::screen_distance(projected, pixel);
    Ok((distance <= 6.).then_some(distance))
}

#[cfg(test)]
mod tests {
    use super::text_height_field;

    #[test]
    fn reopened_text_height_shows_two_decimals_without_rounding_custom_sizes() {
        assert_eq!(text_height_field(0.61), "6.10");
        assert_eq!(text_height_field(0.6), "6.00");
        assert_eq!(text_height_field(0.0125), "0.125");
    }
}
