// SPDX-License-Identifier: GPL-2.0-only
//! Format Text owns a transient draft until the existing document command accepts it.
use super::{sketch_picking::ordinary_click, *};
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::types::SketchTextStyle;
use oblikovati_gui::{GuiSketchTextAction, GuiSketchTextDraft};
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportSketchPreview};

pub(crate) struct SketchTextTool {
    pub(crate) edit: HeadSketchEdit,
    pub(crate) generation: u64,
    working_scale: f64,
    placement: Option<TextPlacementDraft>,
}

struct TextPlacementDraft {
    anchor: [f64; 2],
    fields: GuiSketchTextDraft,
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
            if tool.placement.take().is_some() {
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

    pub(super) fn sketch_text_preview(
        &self,
        tool: &SketchTextTool,
    ) -> Option<ViewportSketchPreview> {
        let placement = tool.placement.as_ref()?;
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return None;
        }
        let style = text_style(&placement.fields).ok()?;
        Some(ViewportSketchPreview::Text {
            plane: tool.edit.plane,
            anchor: placement.anchor,
            style: Box::new(style),
            working_scale: tool.working_scale,
        })
    }

    pub(super) fn sketch_text_command(
        &mut self,
        action: GuiSketchTextAction,
    ) -> Result<bool, String> {
        self.check_sketch_tool_context();
        let Some(SketchToolState::Text(tool)) = &mut self.sketch_tool else {
            return Ok(false);
        };
        let Some(placement) = tool.placement.as_mut() else {
            return Ok(false);
        };
        match action {
            GuiSketchTextAction::Preview(fields) => placement.fields = fields,
            GuiSketchTextAction::Cancel => {
                tool.placement = None;
                self.sketch_text_prompt();
            }
            GuiSketchTextAction::Accept(fields) => return self.accept_sketch_text(fields),
        }
        Ok(true)
    }

    fn accept_sketch_text(&mut self, fields: GuiSketchTextDraft) -> Result<bool, String> {
        let Some(SketchToolState::Text(mut tool)) = self.sketch_tool.take() else {
            return Ok(false);
        };
        let result = self.commit_text_candidate(&mut tool, fields);
        self.sketch_tool = Some(SketchToolState::Text(tool));
        if result.is_ok() {
            self.sketch_text_prompt();
        }
        result
    }

    fn commit_text_candidate(
        &mut self,
        tool: &mut SketchTextTool,
        fields: GuiSketchTextDraft,
    ) -> Result<bool, String> {
        let Some(placement) = tool.placement.as_mut() else {
            return Ok(false);
        };
        placement.fields = fields;
        if !placement.fields.content.trim().is_empty() {
            let style = text_style(&placement.fields)?;
            verify_text_outline(&style, placement.anchor, tool.working_scale)?;
            self.commit_sketch_text(tool.edit, placement.anchor, style)?;
            tool.generation = self.capture_sketch_tool_context()?.1;
        }
        tool.placement = None;
        Ok(true)
    }

    fn commit_sketch_text(
        &mut self,
        edit: HeadSketchEdit,
        anchor: [f64; 2],
        style: SketchTextStyle,
    ) -> Result<(), String> {
        self.files
            .commands(edit.document)
            .map_err(|e| e.to_string())?
            .run("Create sketch text", |snapshot, ids| {
                snapshot.create_sketch_text(edit.sketch, anchor, style, ids)
            })
            .map(|_| ())
            .map_err(|e| e.to_string())
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
