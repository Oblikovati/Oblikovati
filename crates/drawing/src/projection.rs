// SPDX-License-Identifier: GPL-2.0-only
//! Projection over the current analytic part, with samples used only for presentation.
use oblikovati_api::types::{
    DrawingViewStyle, HIDDEN_LINE_REMOVED_VIEW_STYLE, HIDDEN_LINE_VIEW_STYLE, WIREFRAME_VIEW_STYLE,
};
use oblikovati_automation::DrawingViewProjectionAccess;
use oblikovati_martensite::{DrawingBaseProjection, EvaluationFailure, NamedSolid};
use oblikovati_model::{
    drawing::DrawingBaseViewDefinition,
    feature::{FeatureEvaluationStatus, part::PartFeatureEvaluation},
    history::DocumentEditHistory,
    identity::DocumentGuid,
};
use oblikovati_persistence::package::SavedDrawingLine;

/// The shared Martensite projection backend; example: `let backend = DrawingProjectionService`.
pub struct DrawingProjectionService;
impl DrawingViewProjectionAccess for DrawingProjectionService {
    fn resolution_policy(&self) -> oblikovati_martensite::PrismResolutionPolicy {
        crate::modelling_policy()
    }
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool {
        policy == Some(crate::modelling_policy())
    }
    fn counts_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    ) -> Result<[usize; 2], String> {
        let projection = project_evaluated_part(history, document, view, part)?;
        Ok([
            projection.curves.visible.len(),
            projection.curves.hidden.len(),
        ])
    }
    fn saved_lines_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    ) -> Result<Vec<SavedDrawingLine>, String> {
        let projection = project_evaluated_part(history, document, view, part)?;
        sample_projection(&projection, view.style, history, document)
    }
}
/// Project one current, healthy part; example: `project_evaluated_part(history, guid, view, part)?`.
pub fn project_evaluated_part(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    view: &DrawingBaseViewDefinition,
    part: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
) -> Result<DrawingBaseProjection, String> {
    require_healthy(part)?;
    let bodies: Vec<_> = part.ordered_bodies().collect();
    oblikovati_martensite::drawing_base_view_solids(history, document, &bodies, view)
}
fn require_healthy(
    result: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
) -> Result<(), String> {
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
    Ok(())
}
/// Sample one projected view for display; example: `sample_projection(&view, style, history, guid)?`.
pub fn sample_projection(
    view: &DrawingBaseProjection,
    style: DrawingViewStyle,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<Vec<SavedDrawingLine>, String> {
    let include_hidden = match style {
        HIDDEN_LINE_REMOVED_VIEW_STYLE => false,
        HIDDEN_LINE_VIEW_STYLE | WIREFRAME_VIEW_STYLE => true,
        _ => {
            return Err(format!(
                "drawing view style {style:?}: expected wireframe or hidden-line style"
            ));
        }
    };
    let mut lines = Vec::new();
    for visible in [true, false] {
        if !visible && !include_hidden {
            continue;
        }
        let curves = if visible {
            &view.curves.visible
        } else {
            &view.curves.hidden
        };
        for index in 0..curves.len() {
            lines.push(SavedDrawingLine {
                points: view.sheet_polyline(
                    visible,
                    index,
                    history,
                    document,
                    crate::DISPLAY_MESH.chord_error,
                )?,
                hidden: !visible && style != WIREFRAME_VIEW_STYLE,
            });
        }
    }
    Ok(lines)
}
