// SPDX-License-Identifier: GPL-2.0-only
//! Map GUI commands to viewport shortcuts.
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::ViewportKey;

pub(super) fn action_key(action: &GuiShellAction) -> Option<ViewportKey> {
    Some(match action {
        GuiShellAction::OpenInferenceScope
        | GuiShellAction::SelectInferenceGeometry(_)
        | GuiShellAction::SetInferenceScope(_)
        | GuiShellAction::SelectBlockGeometry(_)
        | GuiShellAction::CreateSketchBlock(_)
        | GuiShellAction::PlaceSketchBlock(_) => {
            return None;
        }
        GuiShellAction::SetMarkingMenu(_)
        | GuiShellAction::SetKeymap(_)
        | GuiShellAction::SetUiScale(_)
        | GuiShellAction::SetDocumentUnits { .. }
        | GuiShellAction::CreateDrawingBaseView(_)
        | GuiShellAction::PlaceComponent { .. }
        | GuiShellAction::PlaceComponentFile { .. }
        | GuiShellAction::PreviewComponent(_)
        | GuiShellAction::PreviewComponentFile { .. }
        | GuiShellAction::PreviewDrawingBaseView(_)
        | GuiShellAction::SetSketchOptions(_)
        | GuiShellAction::SetSketchIsolation(_)
        | GuiShellAction::SetPartOptions(_)
        | GuiShellAction::LineCommand(_)
        | GuiShellAction::PrecisePoint(_)
        | GuiShellAction::PreciseTriad(_)
        | GuiShellAction::ToggleCenterPointFormat
        | GuiShellAction::SetCenterPoint { .. }
        | GuiShellAction::DeleteSketchRelation(_)
        | GuiShellAction::SelectSketchContextCandidate { .. }
        | GuiShellAction::DismissSketchContext { .. }
        | GuiShellAction::SelectContextConstraintGlyph { .. }
        | GuiShellAction::HideContextConstraintGlyph { .. }
        | GuiShellAction::SetContextConstraintGlyphs { .. }
        | GuiShellAction::ExplodeSketchContext { .. }
        | GuiShellAction::Inspect(_)
        | GuiShellAction::MeasureSelectedFace { .. }
        | GuiShellAction::OpenFaceProperties { .. }
        | GuiShellAction::CloseFaceProperties
        | GuiShellAction::EditDimensionValue(_)
        | GuiShellAction::SetPromptFrequency { .. }
        | GuiShellAction::CaptureWindowPreview
        | GuiShellAction::CaptureIsometricPreview { .. }
        | GuiShellAction::Exit
        | GuiShellAction::EditThemeColor { .. }
        | GuiShellAction::DuplicateTheme { .. }
        | GuiShellAction::DeleteTheme { .. }
        | GuiShellAction::SaveTheme { .. }
        | GuiShellAction::SetColorScheme(_)
        | GuiShellAction::SetTheme(_)
        | GuiShellAction::Parameter { .. }
        | GuiShellAction::SetDocumentProperty(_)
        | GuiShellAction::OpenIProperties
        | GuiShellAction::OpenModelStateTable
        | GuiShellAction::SetModelStateTable { .. }
        | GuiShellAction::VerifyModelStateTable { .. }
        | GuiShellAction::UpdatePart
        | GuiShellAction::SetSelectionMode(_)
        | GuiShellAction::OpenSelectionFilter
        | GuiShellAction::OpenHistoryBrowser
        | GuiShellAction::JumpHistoryTo { .. }
        | GuiShellAction::SetSelectionEnabled { .. }
        | GuiShellAction::MoveSelectionKind { .. }
        | GuiShellAction::ResetSelectionFilter
        | GuiShellAction::ActivateDocument(_)
        | GuiShellAction::OpenDocument(_)
        | GuiShellAction::LoadFullDocument(_)
        | GuiShellAction::OpenDeferredDrawing(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::SetSketchVisibility { .. }
        | GuiShellAction::SelectBrowserSketch { .. }
        | GuiShellAction::SelectBrowserBlock { .. }
        | GuiShellAction::ExplodeSketchBlock { .. }
        | GuiShellAction::SelectBrowserFeature { .. }
        | GuiShellAction::BrowserChoice { .. }
        | GuiShellAction::SetSketchLineStyle { .. }
        | GuiShellAction::EditFeature { .. }
        | GuiShellAction::OpenExtrudeEdit(_)
        | GuiShellAction::PreviewExtrudeEdit(_)
        | GuiShellAction::EditExtrudeDistance(_)
        | GuiShellAction::ClickModel { .. }
        | GuiShellAction::ContextClickModel { .. }
        | GuiShellAction::InspectFile(_)
        | GuiShellAction::KeepDocumentPicture { .. }
        | GuiShellAction::OrientView { .. }
        | GuiShellAction::SetViewProjection(_)
        | GuiShellAction::SetOrientationSelectionLock(_)
        | GuiShellAction::ViewCommand(_)
        | GuiShellAction::UpdateBodyProperties { .. }
        | GuiShellAction::AssignPartAsset { .. }
        | GuiShellAction::SetBodyAppearance { .. }
        | GuiShellAction::ClearSelectedFaceAppearance { .. }
        | GuiShellAction::ClearSelectedMixedAppearance { .. }
        | GuiShellAction::ClearSelectedBodyAppearance { .. }
        | GuiShellAction::OpenClearBodyAppearance { .. }
        | GuiShellAction::EditSketch { .. }
        | GuiShellAction::Dimension(_)
        | GuiShellAction::PreviewSketchPattern(_)
        | GuiShellAction::CreateSketchPattern(_)
        | GuiShellAction::OpenSketchPattern { .. }
        | GuiShellAction::PreviewSketchPatternEdit { .. }
        | GuiShellAction::EditSketchPattern { .. }
        | GuiShellAction::SketchMove(_)
        | GuiShellAction::ConstrainSketch(_)
        | GuiShellAction::PlaceSketchTool(_)
        | GuiShellAction::ToggleSketchFormat(_)
        | GuiShellAction::DeleteSketchSelection
        | GuiShellAction::CreateDocument(_)
        | GuiShellAction::NewPart
        | GuiShellAction::ChooseSketchPlane
        | GuiShellAction::ChooseSketchPlaneForExtrude
        | GuiShellAction::ShowHome
        | GuiShellAction::Recent(_)
        | GuiShellAction::ActivateWindow(_)
        | GuiShellAction::NewDocumentWindow
        | GuiShellAction::CloseDocumentWindow(_)
        | GuiShellAction::StartSketchOn(_)
        | GuiShellAction::StartSketchOnWorkPlane(_)
        | GuiShellAction::StartSketchOnSelectedFace { .. }
        | GuiShellAction::StartSketchAt { .. }
        | GuiShellAction::FinishSketch
        | GuiShellAction::OpenExtrude
        | GuiShellAction::OpenMoveBody
        | GuiShellAction::PreviewMoveBody(_)
        | GuiShellAction::CreateMoveBody(_)
        | GuiShellAction::OpenShellFeature
        | GuiShellAction::PreviewShellFeature(_)
        | GuiShellAction::CreateShellFeature(_)
        | GuiShellAction::OpenHole
        | GuiShellAction::PreviewHole(_)
        | GuiShellAction::CreateHole(_)
        | GuiShellAction::OpenFillet
        | GuiShellAction::OpenChamfer
        | GuiShellAction::OpenFilletEdit(_)
        | GuiShellAction::OpenChamferEdit(_)
        | GuiShellAction::PreviewFilletEdit(_)
        | GuiShellAction::EditFilletRadius(_)
        | GuiShellAction::PreviewChamferEdit(_)
        | GuiShellAction::EditChamferDistance(_)
        | GuiShellAction::LookAtSelection
        | GuiShellAction::PreviewFillet(_)
        | GuiShellAction::CreateFillet(_)
        | GuiShellAction::PreviewChamfer(_)
        | GuiShellAction::CreateChamfer(_)
        | GuiShellAction::OpenOffsetPlane
        | GuiShellAction::ValidateOffsetPlane { .. }
        | GuiShellAction::CreateOffsetPlane { .. }
        | GuiShellAction::ValidateExtrudeDistance { .. }
        | GuiShellAction::PreviewExtrude(_)
        | GuiShellAction::Extrude { .. } => {
            return None;
        }
        GuiShellAction::Save => ViewportKey::Save,
        GuiShellAction::Undo => ViewportKey::Undo,
        GuiShellAction::Redo => ViewportKey::Redo,
        GuiShellAction::Home => ViewportKey::Home,
    })
}
