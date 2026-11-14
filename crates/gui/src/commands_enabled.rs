// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl GuiShellAction {
    /// Preserve Go empty-stack enablement; example: `GuiShellAction::Undo.enabled(&state)`.
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        match self {
            Self::PreviewSketchPattern(None) => true,
            Self::OpenSketchPattern { sketch, pattern } => {
                state.sketch_pattern_available
                    && state.sketch_at_ready
                    && !state.transaction_open
                    && state.active_sketch == Some(*sketch)
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser.rectangular_patterns.iter().any(|row| {
                            row.sketch == *sketch
                                && row.pattern.id == *pattern
                                && row.spacing.is_ok()
                                && row
                                    .pattern
                                    .definition
                                    .counts
                                    .iter()
                                    .all(|count| i32::try_from(*count).is_ok())
                        })
                    })
            }
            Self::PreviewSketchPattern(Some(_))
            | Self::CreateSketchPattern(_)
            | Self::PreviewSketchPatternEdit { .. }
            | Self::EditSketchPattern { .. } => {
                state.sketch_pattern_available && state.sketch_at_ready && !state.transaction_open
            }

            Self::SketchMove(crate::GuiSketchMoveAction::Start) => {
                state.sketch_at_ready && !state.transaction_open
            }
            Self::SketchMove(crate::GuiSketchMoveAction::Done) => state.sketch_move.is_some(),
            Self::SketchMove(
                crate::GuiSketchMoveAction::SetOptimizeSingle(_)
                | crate::GuiSketchMoveAction::SetCopy(_),
            ) => state
                .sketch_move
                .is_some_and(|status| status.stage == crate::GuiSketchMoveStage::Geometry),
            Self::SketchMove(crate::GuiSketchMoveAction::ChooseBase) => {
                state.sketch_move.is_some_and(|status| {
                    status.stage == crate::GuiSketchMoveStage::Geometry
                        && status.selection_count > 0
                }) && !state.transaction_open
            }
            Self::PlaceSketchBlock(name) => {
                state.authoring == GuiAuthoringContext::Sketch
                    && !state.transaction_open
                    && state.sketch_block_names.contains(name)
            }
            Self::SelectBrowserBlock {
                document,
                sketch,
                instance,
                ..
            } => {
                state.document == Some(*document)
                    && state.authoring == GuiAuthoringContext::Sketch
                    && state.sketch_at_ready
                    && !state.transaction_open
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser
                            .block_instances
                            .iter()
                            .any(|row| row.sketch == *sketch && row.id == *instance)
                    })
            }
            Self::SelectBlockGeometry(false) => true,
            Self::SelectBlockGeometry(true) => state.sketch_at_ready && !state.transaction_open,
            Self::CreateSketchBlock(name) => {
                state.sketch_at_ready
                    && !state.transaction_open
                    && !name.trim().is_empty()
                    && !name.contains('\0')
                    && !state.sketch_block_names.contains(name)
                    && !state.sketch_selection.entities.is_empty()
            }
            Self::SetDocumentUnits { document, .. } => {
                state.document == Some(*document)
                    && state.document_units.is_some()
                    && !state.transaction_open
            }
            Self::PreviewDrawingBaseView(None) => true,
            Self::PlaceComponent { assembly, request } => {
                state.document == Some(*assembly)
                    && state.document_type == Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
                    && !state.transaction_open
                    && request.document != *assembly
                    && state.documents.iter().any(|row| {
                        row.id == request.document
                            && matches!(row.type_.as_str(), "part" | "assembly")
                    })
                    && state
                        .document_storage
                        .get(&request.document)
                        .is_some_and(|file| file.path.is_some())
                    && !request.name.trim().is_empty()
            }
            Self::PlaceComponentFile {
                assembly,
                path,
                name,
            } => {
                state.document == Some(*assembly)
                    && state.document_type == Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
                    && !state.transaction_open
                    && !path.trim().is_empty()
                    && !name.trim().is_empty()
            }
            Self::PreviewComponent(None) => true,
            Self::PreviewComponentFile { assembly, path } => {
                state.document == Some(*assembly)
                    && state.document_type == Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
                    && !path.is_empty()
            }
            Self::PreviewComponent(Some((assembly, source))) => {
                state.document == Some(*assembly)
                    && state.document_type == Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
                    && *source != *assembly
                    && state.documents.iter().any(|row| {
                        row.id == *source && matches!(row.type_.as_str(), "part" | "assembly")
                    })
            }
            Self::PreviewDrawingBaseView(Some(request)) => {
                Self::CreateDrawingBaseView(request.clone()).enabled(state)
            }
            Self::CreateDrawingBaseView(request) => {
                state.document == Some(request.drawing)
                    && !state.drawing_updates_deferred
                    && state.document_type == Some(oblikovati_api::types::DOCUMENT_DRAWING)
                    && !state.transaction_open
                    && state
                        .documents
                        .iter()
                        .any(|row| row.id == request.source && row.type_ == "part")
                    && request.scale.is_finite()
                    && request.scale > 0.
                    && request.center.iter().all(|value| value.is_finite())
            }
            Self::SetUiScale(_) => state.ui_scale.is_some(),
            Self::ShowHome | Self::Recent(_) => true,
            Self::ActivateWindow(_) | Self::CloseDocumentWindow(_) | Self::NewDocumentWindow => {
                !state.transaction_open
            }
            Self::SetSketchOptions(_) => state.sketch_options.is_some(),
            Self::LineCommand(_) => state.line_command.is_some(),
            Self::PrecisePoint(_) | Self::PreciseTriad(_) => state.precise_target.is_some(),
            Self::ToggleCenterPointFormat => state.authoring == GuiAuthoringContext::Sketch,
            Self::SetCenterPoint { .. } | Self::DeleteSketchRelation(_) => {
                state.sketch_at_ready && !state.transaction_open
            }
            Self::Inspect(crate::GuiInspectAction::StartMeasure) => {
                matches!(
                    state.authoring,
                    GuiAuthoringContext::Sketch | GuiAuthoringContext::Part
                ) && !state.transaction_open
            }
            Self::Inspect(crate::GuiInspectAction::FinishMeasure) => state.measure.is_some(),
            Self::Inspect(_) => state.authoring == GuiAuthoringContext::Sketch,
            Self::EditDimensionValue(_) => {
                (state.sketch_at_ready || state.dimension_tool_active) && !state.transaction_open
            }
            Self::SetPromptFrequency { .. } | Self::CaptureWindowPreview | Self::Exit => true,
            Self::CaptureIsometricPreview { document } => state.document == Some(*document),
            Self::ActivateDocument(request) => {
                !state.transaction_open
                    && state
                        .documents
                        .iter()
                        .any(|row| row.id == request.id && row.visible)
            }
            Self::EditThemeColor { name, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.editable() && theme.active_name() == name),
            Self::DuplicateTheme { base, .. } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.creation_available() && theme.active_name() == base),
            Self::DeleteTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.deletion_available() && theme.active_name() == name),
            Self::SaveTheme { name } => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.save_available() && theme.active_name() == name),
            Self::SetColorScheme(request) => state
                .color_scheme
                .as_ref()
                .is_some_and(|palette| palette.contains(&request.name)),
            Self::SetTheme(name) => state
                .theme
                .as_ref()
                .is_some_and(|theme| theme.contains(name)),
            Self::OpenDocument(_) | Self::OpenDeferredDrawing(_) | Self::CreateDocument(_) => {
                !state.transaction_open
            }
            Self::CloseDocument(request) => {
                (state.document == Some(request.id)
                    || state.documents.iter().any(|row| row.id == request.id))
                    && !state.transaction_open
            }
            Self::SaveDocumentAs(request) => {
                state.document == Some(request.document) && !state.transaction_open
            }
            Self::OrientView { .. }
            | Self::SetViewProjection(_)
            | Self::SetOrientationSelectionLock(_)
            | Self::ViewCommand(_) => state.document.is_some(),
            Self::InspectFile(_) | Self::KeepDocumentPicture { .. } => true,
            Self::ClickModel { document, .. } | Self::UpdateBodyProperties { document, .. } => {
                state.document == Some(*document)
            }
            Self::AssignPartAsset { document, .. } => {
                state.document == Some(*document)
                    && state.authoring == GuiAuthoringContext::Part
                    && !state.transaction_open
            }
            Self::SetBodyAppearance { document, .. } => {
                state.document == Some(*document) && !state.transaction_open
            }
            Self::EditFeature { document, .. }
            | Self::SetSketchLineStyle { document, .. }
            | Self::BrowserChoice { document, .. } => {
                state.document == Some(*document) && !state.transaction_open
            }
            Self::SelectBrowserFeature { document, feature } => {
                state.document == Some(*document)
                    && !state.transaction_open
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser.features.iter().any(|row| row.id == *feature)
                    })
            }
            Self::SelectBrowserSketch {
                document, sketch, ..
            }
            | Self::SetSketchVisibility {
                document, sketch, ..
            } => {
                state.document == Some(*document)
                    && !state.transaction_open
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::EditSketch { document, sketch } => {
                state.document == Some(*document)
                    && state.authoring != GuiAuthoringContext::Sketch
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::OpenExtrude => {
                state.extrude_available
                    && state.authoring == GuiAuthoringContext::Part
                    && !state.transaction_open
                    && state.document.is_some()
                    && state.browser.is_some()
            }
            Self::Extrude {
                document,
                sketch,
                profile,
                distance,
            } => {
                state.extrude_available
                    && state.document == Some(*document)
                    && state.authoring == GuiAuthoringContext::Part
                    && !state.transaction_open
                    && *profile >= 0
                    && !distance.trim().is_empty()
                    && !distance.contains('\0')
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sketches.iter().any(|row| row.id == *sketch))
            }
            Self::PreviewExtrude(None) => true,
            Self::ValidateExtrudeDistance { document, .. } => {
                state.document == Some(*document) && state.extrude_available
            }
            Self::PreviewExtrude(Some(preview)) => Self::Extrude {
                document: preview.document,
                sketch: preview.sketch,
                profile: preview.profile,
                distance: preview.distance.clone(),
            }
            .enabled(state),
            Self::Parameter { .. } => state.document.is_some(),
            Self::UpdatePart => {
                state.document_type == Some(oblikovati_api::types::DOCUMENT_PART)
                    && !state.transaction_open
            }
            Self::OpenHistoryBrowser => !state.histories.is_empty(),
            Self::JumpHistoryTo { document, position } => {
                !state.transaction_open
                    && state.histories.iter().any(|history| {
                        history.document == *document
                            && *position >= 0
                            && usize::try_from(*position).ok().is_some_and(|position| {
                                position <= history.entries.as_ref().map_or(0, Vec::len)
                            })
                    })
            }
            Self::SetSelectionMode(_)
            | Self::OpenSelectionFilter
            | Self::SetSelectionEnabled { .. }
            | Self::MoveSelectionKind { .. }
            | Self::ResetSelectionFilter => {
                state.document.is_some()
                    && !state.transaction_open
                    && state.authoring != crate::GuiAuthoringContext::Sketch
                    && matches!(
                        state.document_type,
                        Some(
                            oblikovati_api::types::DOCUMENT_PART
                                | oblikovati_api::types::DOCUMENT_ASSEMBLY
                        )
                    )
            }
            Self::Undo => state.undo_label.is_some(),
            Self::Redo => state.redo_label.is_some(),
            Self::ChooseSketchPlane
            | Self::ChooseSketchPlaneForExtrude
            | Self::StartSketchOn(_)
            | Self::StartSketchOnWorkPlane(_)
            | Self::StartSketchAt { .. } => state.authoring == GuiAuthoringContext::Part,
            Self::DeleteSketchSelection => {
                state.authoring == GuiAuthoringContext::Sketch
                    && state.sketch_delete_available
                    && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Start) => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Apply { id, .. }) => {
                state
                    .dimension_draft
                    .as_ref()
                    .is_some_and(|draft| draft.id == *id)
                    && !state.transaction_open
            }
            Self::Dimension(crate::GuiDimensionAction::Cancel { id }) => state
                .dimension_draft
                .as_ref()
                .is_some_and(|draft| draft.id == *id),
            Self::ConstrainSketch(_) => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::ToggleSketchFormat(_) => {
                state.authoring == GuiAuthoringContext::Sketch
                    && state.sketch_format_available
                    && !state.transaction_open
            }
            Self::PlaceSketchTool(tool) => {
                (*tool != crate::GuiSketchTool::Point || !state.sketch_formats.construction)
                    && state.authoring == GuiAuthoringContext::Sketch
                    && !state.transaction_open
            }
            Self::FinishSketch => {
                state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
            }
            Self::SetMarkingMenu(_)
            | Self::SetKeymap(_)
            | Self::NewPart
            | Self::Save
            | Self::Home => true,
        }
    }
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(&self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo | Self::EditSketch { .. });
        self.enabled(state) && !(history && state.transaction_open)
    }
    pub(crate) fn next_label<'a>(&self, state: &'a GuiCommandState) -> Option<&'a str> {
        match self {
            Self::Undo => state.undo_label.as_deref(),
            Self::Redo => state.redo_label.as_deref(),
            _ => None,
        }
    }
}
