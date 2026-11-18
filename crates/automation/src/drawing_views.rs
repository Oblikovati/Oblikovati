// SPDX-License-Identifier: GPL-2.0-only
//! The canonical drawing view call validates geometry before committing the recipe.
use oblikovati_api::{
    Error,
    types::*,
    wire::{self, AddBaseViewArgs, DrawingViewInfo, ViewResult},
};
use oblikovati_model::{
    drawing::DrawingBaseViewDefinition, history::DocumentEditHistory, identity::DocumentGuid,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Project a current part before a drawing view is adopted.
pub trait DrawingViewProjectionAccess {
    /// Give the certified policy for drawing geometry; example: `backend.resolution_policy()`.
    fn resolution_policy(&self) -> oblikovati_martensite::PrismResolutionPolicy;
    /// Check whether retained geometry uses this projection's policy; example: `backend.accepts_policy(policy)`.
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool;
    /// Project a generation-checked part; example: `backend.counts_current(history, guid, view, part)?`.
    fn counts_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &oblikovati_model::feature::part::PartFeatureEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<[usize; 2], String>;
    /// Sample one view from a current part; example: `backend.saved_lines_current(history, guid, view, part)?`.
    fn saved_lines_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        view: &DrawingBaseViewDefinition,
        part: &oblikovati_model::feature::part::PartFeatureEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<Vec<oblikovati_persistence::package::SavedDrawingLine>, String>;
}

/// Project a current document-owned part under the drawing policy.
/// Example: `with_current_drawing_part(files, backend, owner, id, project)?`.
pub fn with_current_drawing_part<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    T,
>(
    files: &mut FileWorkspace<O, W, F>,
    backend: &dyn DrawingViewProjectionAccess,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    source: u64,
    project: impl FnOnce(
        &DocumentEditHistory,
        DocumentGuid,
        &oblikovati_model::feature::part::PartFeatureEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<T, String>,
) -> Result<T, String> {
    let policy = backend.resolution_policy();
    if !backend.accepts_policy(Some(policy)) {
        return Err(format!(
            "drawing policy {policy:?}: expected a policy accepted by the projection service"
        ));
    }
    let mut fallback = crate::PartReplayService::with_policy(policy)?;
    let owner: &mut dyn crate::PartUpdateAccess<O, W, F> = match replay {
        Some(owner) if backend.accepts_policy(owner.resolution_policy()) => owner,
        _ => &mut fallback,
    };
    owner.ensure_part(files, source)?;
    let workspace = files.workspace();
    let session = workspace
        .session(source)
        .map_err(|cause| cause.to_string())?;
    let guid = workspace
        .identity(source)
        .map_err(|cause| cause.to_string())?
        .internal_name;
    project(session.history(), guid, owner.current_part(files, source)?)
}
/// Drawing calls served by the document owner.
pub const DRAWING_COMMAND_METHODS: [&str; 2] = [
    wire::METHOD_DRAWING_SET_MODEL_REFERENCE,
    wire::METHOD_DRAWING_VIEWS_ADD_BASE,
];

pub(super) fn add_base<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddBaseViewArgs,
) -> Result<Vec<u8>, Error> {
    let drawing_id = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("drawing view: expected an active drawing".into()))?;
    if files.document_type(drawing_id).map_err(api_error)? != DOCUMENT_DRAWING {
        return Err(Error(format!(
            "drawing view document {drawing_id}: expected a drawing"
        )));
    }
    let drawing = files
        .workspace()
        .session(drawing_id)
        .map_err(api_error)?
        .snapshot()
        .drawing
        .as_ref()
        .ok_or_else(|| {
            Error(format!(
                "drawing document {drawing_id}: expected drawing content"
            ))
        })?;
    let definition = prepare_base_view(drawing, &request)?;
    let source = files
        .resolve_parameter_source(drawing.model_reference())
        .map_err(api_error)?
        .ok_or_else(|| {
            Error(format!(
                "drawing model {:?}: expected an open part",
                drawing.model_reference()
            ))
        })?;
    if files.document_type(source).map_err(api_error)? != DOCUMENT_PART {
        return Err(Error(format!("drawing source {source}: expected a part")));
    }
    let backend = backend.ok_or_else(|| {
        Error("drawing view: expected an injected exact projection service".into())
    })?;
    let counts =
        with_current_drawing_part(files, backend, replay, source, |history, guid, part| {
            backend.counts_current(history, guid, &definition, part)
        })
        .map_err(Error)?;
    let response = view_result(&definition, &request, counts)?;
    files
        .commands(drawing_id)
        .map_err(api_error)?
        .run_projected(
            "Add View",
            |snapshot, _| {
                let drawing = snapshot.drawing.as_mut().ok_or_else(|| {
                    Error(format!(
                        "drawing document {drawing_id}: expected drawing content"
                    ))
                })?;
                drawing
                    .add_base_view_definition(drawing.active_sheet(), definition)
                    .map_err(api_error)
            },
            |_, ()| crate::encode(&response),
        )
        .map_err(api_error)
}
/// Validate a base-view draft without changing a document; example: `prepare_base_view(&drawing, &args)?`.
pub fn prepare_base_view(
    drawing: &oblikovati_model::drawing::DrawingDocument,
    request: &AddBaseViewArgs,
) -> Result<DrawingBaseViewDefinition, Error> {
    let orientation = match request.orientation.as_str() {
        "" | "front" => BASE_VIEW_FRONT,
        "back" => BASE_VIEW_BACK,
        "top" => BASE_VIEW_TOP,
        "bottom" => BASE_VIEW_BOTTOM,
        "left" => BASE_VIEW_LEFT,
        "right" => BASE_VIEW_RIGHT,
        "iso" => BASE_VIEW_ISO,
        value => {
            return Err(Error(format!(
                "view orientation {value:?}: expected front, back, top, bottom, left, right, or iso"
            )));
        }
    };
    let style = match request.style.as_str() {
        "" | "hiddenLine" => HIDDEN_LINE_VIEW_STYLE,
        "hiddenLineRemoved" => HIDDEN_LINE_REMOVED_VIEW_STYLE,
        "wireframe" => WIREFRAME_VIEW_STYLE,
        "shaded" => SHADED_VIEW_STYLE,
        "shadedHiddenLine" => SHADED_HIDDEN_LINE_VIEW_STYLE,
        "fromBase" => FROM_BASE_VIEW_STYLE,
        value => {
            return Err(Error(format!(
                "view style {value:?}: expected a canonical drawing view style"
            )));
        }
    };
    if !request.scale.is_finite() {
        return Err(Error(format!(
            "view scale {}: expected a finite scale",
            request.scale
        )));
    }
    let name = view_name(drawing, &request.name);
    let mut view = DrawingBaseViewDefinition::new(
        &name,
        orientation,
        if request.scale <= 0. {
            1.
        } else {
            request.scale
        },
        [request.center_xmm, request.center_ymm],
    )
    .map_err(api_error)?;
    view.style = style;
    Ok(view)
}
fn view_name(drawing: &oblikovati_model::drawing::DrawingDocument, requested: &str) -> String {
    if !requested.is_empty() {
        return requested.to_string();
    }
    let views = &drawing.sheets()[drawing.active_sheet()].views;
    let mut index = views.len() + 1;
    while views
        .iter()
        .any(|view| view.name == format!("VIEW:{index}"))
    {
        index += 1;
    }
    format!("VIEW:{index}")
}
fn view_result(
    view: &DrawingBaseViewDefinition,
    request: &AddBaseViewArgs,
    counts: [usize; 2],
) -> Result<ViewResult, Error> {
    let count = |value| {
        i64::try_from(value).map_err(|_| {
            Error(format!(
                "drawing curve count {value}: expected a signed 64-bit count"
            ))
        })
    };
    Ok(ViewResult {
        view: DrawingViewInfo {
            name: view.name.clone(),
            type_: "base".into(),
            orientation: if request.orientation.is_empty() {
                "front".into()
            } else {
                request.orientation.clone()
            },
            style: if request.style.is_empty() {
                "hiddenLine".into()
            } else {
                request.style.clone()
            },
            scale: view.scale,
            center_xmm: view.center_x_mm,
            center_ymm: view.center_y_mm,
            visible_count: count(counts[0])?,
            hidden_count: count(counts[1])?,
            display_tangent_edges: true,
            show_label: true,
            show_name: true,
            show_scale: true,
            label: format!("{}  SCALE {}", view.name, scale_note(view.scale)),
            justification: "centered".into(),
            ..Default::default()
        },
    })
}
fn scale_note(scale: f64) -> String {
    if scale >= 1. {
        format!("{scale}:1")
    } else {
        format!("1:{}", 1. / scale)
    }
}
fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn base_view_inputs_follow_contract_defaults_and_reject_invalid_spellings() {
        let drawing = oblikovati_model::drawing::DrawingDocument::new();
        for (spelling, orientation) in [
            ("front", BASE_VIEW_FRONT),
            ("back", BASE_VIEW_BACK),
            ("top", BASE_VIEW_TOP),
            ("bottom", BASE_VIEW_BOTTOM),
            ("left", BASE_VIEW_LEFT),
            ("right", BASE_VIEW_RIGHT),
            ("iso", BASE_VIEW_ISO),
        ] {
            let input = AddBaseViewArgs {
                orientation: spelling.into(),
                scale: -1.,
                ..Default::default()
            };
            let view = prepare_base_view(&drawing, &input).unwrap();
            assert_eq!(
                (view.orientation, view.scale, view.name.as_str()),
                (orientation, 1., "VIEW:1")
            );
        }
        for input in [
            AddBaseViewArgs {
                orientation: "unknown".into(),
                ..Default::default()
            },
            AddBaseViewArgs {
                style: "unknown".into(),
                ..Default::default()
            },
            AddBaseViewArgs {
                scale: f64::NAN,
                ..Default::default()
            },
            AddBaseViewArgs {
                center_xmm: f64::INFINITY,
                ..Default::default()
            },
        ] {
            assert!(prepare_base_view(&drawing, &input).is_err());
        }
    }
}
