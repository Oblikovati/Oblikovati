// SPDX-License-Identifier: GPL-2.0-only
//! Drawing display state follows the current drawing and referenced part generations.
use super::*;
use oblikovati_drawing::evaluate_part;
use oblikovati_gui::{GuiDrawingLine, GuiDrawingProjection};
use oblikovati_model::drawing::DrawingDocument;
use std::collections::BTreeMap;

type ProjectionKey = (u64, u64, u64);
#[derive(Default)]
pub(super) struct DrawingProjectionCache(BTreeMap<u64, (ProjectionKey, GuiDrawingProjection)>);
pub(super) struct DrawingViewPreview {
    drawing: u64,
    drawing_generation: u64,
    source: u64,
    source_generation: u64,
    focus: oblikovati_session::workspace::WorkspaceFocusIdentity,
    projection: GuiDrawingProjection,
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
    pub(super) fn gui_drawing_projections(
        &self,
    ) -> Result<BTreeMap<u64, GuiDrawingProjection>, String> {
        let mut outputs = BTreeMap::new();
        let workspace = self.files.workspace();
        for &id in workspace.documents() {
            let session = workspace.session(id).map_err(|cause| cause.to_string())?;
            let Some(drawing) = &session.snapshot().drawing else {
                continue;
            };
            let output = self.drawing_projection(id, session.generation(), drawing);
            outputs.insert(id, output);
        }
        if let Some(preview) = &self.drawing_view_preview
            && workspace.active_document() == Some(preview.drawing)
            && workspace.retains_focus(&preview.focus)
            && workspace
                .session(preview.drawing)
                .is_ok_and(|session| session.generation() == preview.drawing_generation)
            && workspace
                .session(preview.source)
                .is_ok_and(|session| session.generation() == preview.source_generation)
            && let Some(output) = outputs.get_mut(&preview.drawing)
        {
            output.lines.extend(preview.projection.lines.clone());
        }
        self.drawing_projections
            .borrow_mut()
            .0
            .retain(|id, _| outputs.contains_key(id));
        Ok(outputs)
    }
    pub(super) fn set_drawing_view_preview(
        &mut self,
        request: Option<&oblikovati_gui::GuiBaseViewRequest>,
    ) -> Result<bool, String> {
        let changed = self.drawing_view_preview.take().is_some();
        let Some(request) = request else {
            return Ok(changed);
        };
        self.drawing_source_reference(request)?;
        let workspace = self.files.workspace();
        let drawing = workspace
            .session(request.drawing)
            .map_err(|cause| cause.to_string())?;
        let source = workspace
            .session(request.source)
            .map_err(|cause| cause.to_string())?;
        let content = drawing
            .snapshot()
            .drawing
            .as_ref()
            .ok_or("drawing preview: expected drawing content")?;
        let args = oblikovati_api::wire::AddBaseViewArgs {
            orientation: request.orientation.clone(),
            scale: request.scale,
            center_xmm: request.center[0],
            center_ymm: request.center[1],
            ..Default::default()
        };
        let view = oblikovati_automation::prepare_base_view(content, &args)
            .map_err(|cause| cause.to_string())?;
        let document = workspace
            .identity(request.source)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let history = source.history();
        let evaluated = evaluate_part(history, document)?;
        let bodies: Vec<_> = evaluated.ordered_bodies().collect();
        let projected =
            oblikovati_martensite::drawing_base_view_bodies(history, document, &bodies, &view)?;
        let mut projection = GuiDrawingProjection::default();
        append_curves(&mut projection, &projected, view.style, history, document)?;
        for line in &mut projection.lines {
            line.preview = true;
        }
        self.drawing_view_preview = Some(DrawingViewPreview {
            drawing: request.drawing,
            drawing_generation: drawing.generation(),
            source: request.source,
            source_generation: source.generation(),
            focus: workspace.focus_identity(),
            projection,
        });
        Ok(true)
    }
    fn drawing_projection(
        &self,
        drawing_id: u64,
        generation: u64,
        drawing: &DrawingDocument,
    ) -> GuiDrawingProjection {
        match self.files.deferred_drawing_views(drawing_id) {
            Ok(Some(views)) => {
                return saved_sheet_projection(
                    views,
                    &drawing.sheets()[drawing.active_sheet()].name,
                );
            }
            Ok(None) => {}
            Err(cause) => {
                return GuiDrawingProjection {
                    lines: Vec::new(),
                    error: Some(cause.to_string()),
                };
            }
        }
        if drawing.sheets()[drawing.active_sheet()].views.is_empty() {
            return GuiDrawingProjection::default();
        }
        let result = self.resolve_drawing_projection(drawing_id, generation, drawing);
        result.unwrap_or_else(|error| GuiDrawingProjection {
            lines: Vec::new(),
            error: Some(error),
        })
    }
    fn resolve_drawing_projection(
        &self,
        drawing_id: u64,
        generation: u64,
        drawing: &DrawingDocument,
    ) -> Result<GuiDrawingProjection, String> {
        let name = drawing.model_reference();
        let source = match self.files.drawing_reference(drawing_id, name) {
            Some(source) => Some(source),
            None => self
                .files
                .resolve_parameter_source(name)
                .map_err(|cause| cause.to_string())?,
        }
        .ok_or_else(|| {
            self.files
                .drawing_reference_error(drawing_id, name)
                .map(str::to_owned)
                .unwrap_or_else(|| {
                    format!("drawing model {name:?}: expected an open part document")
                })
        })?;
        if self
            .files
            .document_type(source)
            .map_err(|cause| cause.to_string())?
            != oblikovati_api::types::DOCUMENT_PART
        {
            return Err(format!(
                "drawing model {name:?}: expected a part; assembly drawing views are unavailable"
            ));
        }
        let session = self
            .files
            .workspace()
            .session(source)
            .map_err(|cause| cause.to_string())?;
        let key = (generation, source, session.generation());
        if let Some((_, cached)) = self
            .drawing_projections
            .borrow()
            .0
            .get(&drawing_id)
            .filter(|(old, _)| *old == key)
        {
            return Ok(cached.clone());
        }
        let output = self
            .evaluate_drawing_projection(source, drawing)
            .unwrap_or_else(|error| GuiDrawingProjection {
                lines: Vec::new(),
                error: Some(error),
            });
        self.drawing_projections
            .borrow_mut()
            .0
            .insert(drawing_id, (key, output.clone()));
        Ok(output)
    }
    fn evaluate_drawing_projection(
        &self,
        source: u64,
        drawing: &DrawingDocument,
    ) -> Result<GuiDrawingProjection, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(source)
            .map_err(|cause| cause.to_string())?;
        let document = workspace
            .identity(source)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let history = session.history();
        let result = evaluate_part(history, document)?;
        let bodies: Vec<_> = result.ordered_bodies().collect();
        let mut output = GuiDrawingProjection::default();
        for definition in &drawing.sheets()[drawing.active_sheet()].views {
            let projection = oblikovati_martensite::drawing_base_view_bodies(
                history, document, &bodies, definition,
            )?;
            append_curves(
                &mut output,
                &projection,
                definition.style,
                history,
                document,
            )?;
        }
        Ok(output)
    }
}
fn saved_sheet_projection(
    views: &oblikovati_persistence::package::SavedDrawingViews,
    sheet: &str,
) -> GuiDrawingProjection {
    let lines = views
        .views
        .iter()
        .filter(|view| view.sheet == sheet)
        .flat_map(|view| view.lines.iter())
        .map(|line| GuiDrawingLine {
            points: line.points.clone(),
            hidden: line.hidden,
            preview: false,
        })
        .collect();
    GuiDrawingProjection { lines, error: None }
}

#[cfg(test)]
mod saved_sheet_tests {
    use super::*;
    use oblikovati_persistence::package::{SavedDrawingLine, SavedDrawingView, SavedDrawingViews};

    #[test]
    fn deferred_projection_uses_only_the_active_sheet_and_preserves_line_style() {
        let line = SavedDrawingLine {
            points: vec![[1., 2.], [3., 4.]],
            hidden: true,
        };
        let views = SavedDrawingViews {
            version: SavedDrawingViews::VERSION,
            drawing_recipe: DrawingDocument::new().recipe().clone(),
            views: vec![
                SavedDrawingView {
                    sheet: "Other".into(),
                    view: "Side".into(),
                    lines: vec![line.clone()],
                },
                SavedDrawingView {
                    sheet: "Sheet".into(),
                    view: "Front".into(),
                    lines: vec![line],
                },
            ],
        };
        let output = saved_sheet_projection(&views, "Sheet");
        assert_eq!(output.lines.len(), 1);
        assert_eq!(output.lines[0].points, vec![[1., 2.], [3., 4.]]);
        assert!(output.lines[0].hidden);
        assert!(!output.lines[0].preview);
        assert_eq!(output.error, None);
        assert!(saved_sheet_projection(&views, "Missing").lines.is_empty());
    }
}
fn append_curves(
    output: &mut GuiDrawingProjection,
    view: &oblikovati_martensite::DrawingBaseProjection,
    style: oblikovati_api::types::DrawingViewStyle,
    history: &oblikovati_model::history::DocumentEditHistory,
    document: oblikovati_model::identity::DocumentGuid,
) -> Result<(), String> {
    let lines = oblikovati_drawing::sample_projection(view, style, history, document)?;
    output
        .lines
        .extend(lines.into_iter().map(|line| GuiDrawingLine {
            points: line.points,
            hidden: line.hidden,
            preview: false,
        }));
    Ok(())
}
