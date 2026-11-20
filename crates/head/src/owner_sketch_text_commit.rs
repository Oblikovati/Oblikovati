// SPDX-License-Identifier: GPL-2.0-only
//! Write the text recipe before updating the editor's session-only preset association.
use super::*;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn commit_text_candidate(
        &mut self,
        tool: &mut SketchTextTool,
        fields: GuiSketchTextDraft,
        preset: Option<GuiSketchTextPresetId>,
    ) -> Result<bool, String> {
        let Some(placement) = tool.placement.as_mut() else {
            return Ok(false);
        };
        let creating = placement.existing.is_none();
        let committed =
            self.commit_text_and_link(placement, tool.edit, tool.working_scale, fields, preset)?;
        if creating && committed {
            tool.generation = self.capture_sketch_tool_context()?.1;
        }
        tool.placement = None;
        Ok(true)
    }

    fn commit_text_and_link(
        &mut self,
        placement: &mut TextPlacementDraft,
        edit: HeadSketchEdit,
        working_scale: f64,
        fields: GuiSketchTextDraft,
        preset: Option<GuiSketchTextPresetId>,
    ) -> Result<bool, String> {
        placement.fields = fields;
        if placement.existing.is_none() && placement.fields.content.trim().is_empty() {
            return Ok(false);
        }
        let document = self
            .files
            .workspace()
            .identity(edit.document)
            .map_err(|e| e.to_string())?
            .internal_name;
        let style = placement.style()?;
        verify_text_outline(&style, placement.anchor, working_scale)?;
        let entity = self.write_sketch_text(placement, edit, style)?;
        self.link_text_preset((document, edit.sketch.0, entity), preset);
        Ok(true)
    }

    fn link_text_preset(&mut self, key: TextPresetKey, preset: Option<GuiSketchTextPresetId>) {
        if let Some(preset) = preset {
            self.text_preset_links.insert(key, preset);
        } else {
            self.text_preset_links.remove(&key);
        }
    }

    fn write_sketch_text(
        &mut self,
        placement: &TextPlacementDraft,
        edit: HeadSketchEdit,
        style: SketchTextStyle,
    ) -> Result<SketchEntityId, String> {
        if let Some(existing) = &placement.existing {
            self.change_sketch_text(edit, existing.id, style)?;
            return Ok(existing.id);
        }
        self.commit_sketch_text(edit, placement.anchor, style)
    }

    fn change_sketch_text(
        &mut self,
        edit: HeadSketchEdit,
        entity: SketchEntityId,
        style: SketchTextStyle,
    ) -> Result<(), String> {
        self.files
            .commands(edit.document)
            .map_err(|e| e.to_string())?
            .run("Edit sketch text", |snapshot, _| {
                snapshot.edit_sketch_text(edit.sketch, entity, style)
            })
            .map_err(|e| e.to_string())
    }

    fn commit_sketch_text(
        &mut self,
        edit: HeadSketchEdit,
        anchor: [f64; 2],
        style: SketchTextStyle,
    ) -> Result<SketchEntityId, String> {
        self.files
            .commands(edit.document)
            .map_err(|e| e.to_string())?
            .run("Create sketch text", |snapshot, ids| {
                snapshot.create_sketch_text(edit.sketch, anchor, style, ids)
            })
            .map_err(|e| e.to_string())
    }
}
