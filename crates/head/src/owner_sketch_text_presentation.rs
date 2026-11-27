// SPDX-License-Identifier: GPL-2.0-only
//! Use the document's text field resolution for both preview and outline validation.
use super::*;
use std::borrow::Cow;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn gui_sketch_text_editor_fields(
        &self,
    ) -> Option<Arc<GuiSketchTextEditorFields>> {
        let SketchToolState::Text(tool) = self.sketch_tool.as_ref()? else {
            return None;
        };
        let placement = tool.placement.as_ref()?;
        placement
            .editor_fields
            .get_or_init(|| self.read_text_editor_fields(tool.edit.document, placement))
            .clone()
    }

    fn read_text_editor_fields(
        &self,
        document: u64,
        placement: &TextPlacementDraft,
    ) -> Option<Arc<GuiSketchTextEditorFields>> {
        let draft = &placement.fields;
        let fields = draft
            .formatting
            .linked_fields
            .as_ref()
            .filter(|list| !list.is_empty())?;
        let style = SketchTextStyle {
            content: draft.content.clone(),
            linked_fields: Some(fields.clone()),
            ..Default::default()
        };
        let snapshot = self.files.workspace().session(document).ok()?.snapshot();
        let projection = snapshot
            .project_sketch_text_editor(&style, placement.existing.as_ref())
            .ok()?;
        Some(Arc::new(GuiSketchTextEditorFields {
            authored: style.content,
            fields: style.linked_fields?,
            projection,
        }))
    }

    pub(in crate::owner) fn gui_sketch_text_field_sources(
        &self,
    ) -> Option<Arc<GuiSketchTextFieldSources>> {
        let SketchToolState::Text(tool) = self.sketch_tool.as_ref()? else {
            return None;
        };
        tool.placement.as_ref()?;
        let document = tool.edit.document;
        let workspace = self.files.workspace();
        let snapshot = workspace.session(document).ok()?.snapshot();
        let document_name = workspace.display_name(document).ok()?;
        Some(
            tool.field_sources
                .get_or_init(|| {
                    Arc::new(numeric_field_sources(document_name, &snapshot.parameters))
                })
                .clone(),
        )
    }

    pub(in crate::owner) fn sketch_text_preview(
        &self,
        tool: &SketchTextTool,
    ) -> Option<ViewportSketchPreview> {
        let placement = tool.placement.as_ref()?;
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return None;
        }
        let style = placement.style().ok()?;
        let display = match self
            .display_text_candidate(&style, placement, tool.edit.document)
            .ok()?
        {
            Cow::Borrowed(_) => style,
            Cow::Owned(display) => display,
        };
        Some(ViewportSketchPreview::Text {
            plane: tool.edit.plane,
            anchor: placement.anchor,
            style: Box::new(display),
            working_scale: tool.working_scale,
            replaces: placement
                .existing
                .as_ref()
                .map(|text| (tool.edit.sketch, text.id)),
        })
    }

    pub(super) fn display_text_candidate<'a>(
        &self,
        style: &'a SketchTextStyle,
        placement: &TextPlacementDraft,
        document: u64,
    ) -> Result<Cow<'a, SketchTextStyle>, String> {
        if style.linked_fields.as_ref().is_none_or(Vec::is_empty) {
            return Ok(Cow::Borrowed(style));
        }
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let file_name = crate::sketch_scene::sketch_document_file_name(&self.files, document);
        snapshot
            .resolve_sketch_text_candidate(style, placement.existing.as_ref(), &file_name)
            .map_err(|cause| cause.to_string())
    }
}

fn numeric_field_sources(
    document: &str,
    parameters: &oblikovati_model::parameter::ParameterCollection<u64>,
) -> GuiSketchTextFieldSources {
    use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
    let mut sources = GuiSketchTextFieldSources {
        document: document.into(),
        ..Default::default()
    };
    for source in parameters.ids().iter().filter_map(|&id| parameters.get(id)) {
        if source.current_value().is_some() && source.kind == MODEL_PARAM {
            sources.model_parameters.push(source.name.clone());
        }
        if source.current_value().is_some() && source.kind == USER_PARAM {
            sources.user_parameters.push(source.name.clone());
        }
    }
    sources
}
