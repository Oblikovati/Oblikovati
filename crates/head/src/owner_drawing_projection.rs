// SPDX-License-Identifier: GPL-2.0-only
//! Drawing display state follows the current drawing and referenced part generations.
use super::*;
use oblikovati_gui::{GuiDrawingLine, GuiDrawingProjection};
use oblikovati_model::{drawing::DrawingDocument, feature::FeatureEvaluationStatus};
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
pub(super) struct DrawingProjectionAccess;
impl oblikovati_automation::DrawingViewProjectionAccess for DrawingProjectionAccess {
    fn counts(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        document: oblikovati_model::identity::DocumentGuid,
        view: &oblikovati_model::drawing::DrawingBaseViewDefinition,
    ) -> Result<[usize; 2], String> {
        let result = evaluate_part(history, document)?;
        let bodies: Vec<_> = result.ordered_bodies().collect();
        let projection =
            oblikovati_martensite::drawing_base_view_bodies(history, document, &bodies, view)?;
        Ok([
            projection.curves.visible.len(),
            projection.curves.hidden.len(),
        ])
    }
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
fn evaluate_part(
    history: &oblikovati_model::history::DocumentEditHistory,
    document: oblikovati_model::identity::DocumentGuid,
) -> Result<
    oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedPrismSnapshot,
        oblikovati_martensite::EvaluationFailure,
    >,
    String,
> {
    let mut evaluator = oblikovati_martensite::CertifiedNamedPrismEvaluator::with_policy(
        history,
        document,
        crate::part_scene::resolution_policy::desktop_policy(),
    );
    let result = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .map_err(|cause| cause.to_string())?;
    for health in result.health() {
        if matches!(
            health.status,
            FeatureEvaluationStatus::Failed(_)
                | FeatureEvaluationStatus::Blocked { .. }
                | FeatureEvaluationStatus::ConditionFailed(_)
        ) {
            return Err(format!(
                "drawing source feature {:?}: {:?}",
                health.feature, health.status
            ));
        }
    }
    Ok(result)
}
fn append_curves(
    output: &mut GuiDrawingProjection,
    view: &oblikovati_martensite::DrawingBaseProjection,
    style: oblikovati_api::types::DrawingViewStyle,
    history: &oblikovati_model::history::DocumentEditHistory,
    document: oblikovati_model::identity::DocumentGuid,
) -> Result<(), String> {
    use oblikovati_api::types::{
        HIDDEN_LINE_REMOVED_VIEW_STYLE, HIDDEN_LINE_VIEW_STYLE, WIREFRAME_VIEW_STYLE,
    };
    let hidden = match style {
        HIDDEN_LINE_REMOVED_VIEW_STYLE => false,
        HIDDEN_LINE_VIEW_STYLE | WIREFRAME_VIEW_STYLE => true,
        _ => {
            return Err(format!(
                "drawing view style {style:?}: expected wireframe or hidden-line style"
            ));
        }
    };
    for visible in [true, false] {
        if !visible && !hidden {
            continue;
        }
        let curves = if visible {
            &view.curves.visible
        } else {
            &view.curves.hidden
        };
        for index in 0..curves.len() {
            let points = view.sheet_polyline(visible, index, history, document, 0.05)?;
            output.lines.push(GuiDrawingLine {
                points,
                hidden: !visible && style != WIREFRAME_VIEW_STYLE,
                preview: false,
            });
        }
    }
    Ok(())
}
