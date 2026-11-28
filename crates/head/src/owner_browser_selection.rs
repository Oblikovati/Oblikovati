// SPDX-License-Identifier: GPL-2.0-only
#[path = "owner_model_state.rs"]
mod model_state;
#[path = "owner_browser_outline.rs"]
mod outline;
#[path = "owner_viewport_selection.rs"]
mod viewport;
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::selection::SelectionClickMode;
pub(super) use outline::{BrowserOutlineSource, EdgeOutlineCache};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn select_browser_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let mode = SelectionClickMode::from_modifiers(toggle, false);
        let changed = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(Some(sketch), mode, self.sketch_tool.is_some())
            .map(|count| count.is_some())
            .map_err(|cause| cause.to_string())?;
        self.model_face = None;
        self.model_edges.clear();
        self.face_appearance_picks.clear();
        self.selected_occurrence = None;
        Ok(changed)
    }
    pub(super) fn select_browser_feature(
        &mut self,
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        if self.handle_work_axis_browser_feature(document, feature)? {
            return Ok(true);
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_feature_selection(feature)
            .map_err(|cause| cause.to_string())?;
        self.model_face = None;
        self.model_edges.clear();
        self.face_appearance_picks.clear();
        self.selected_occurrence = None;
        Ok(true)
    }
    pub(super) fn set_sketch_dimensions_shown(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        shown: bool,
    ) -> Result<bool, String> {
        let session = self.files.workspace().session(document);
        let session = session.map_err(|cause| cause.to_string())?;
        let placed = session.snapshot().sketches.get(sketch);
        let style = placed
            .map_err(|cause| cause.to_string())?
            .line_style()
            .clone();
        let style = oblikovati_model::document_sketch::SketchLineStyle {
            dimensions_hidden: !shown,
            ..style
        };
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_line_style(sketch, style)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// A saved, undoable change to a feature's name and to when it is suppressed.
    pub(super) fn edit_feature(
        &mut self,
        document: u64,
        edit: &oblikovati_gui::GuiFeatureEdit,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        oblikovati_automation::edit_part_with_checkpoints(
            &mut self.files,
            &mut self.part_scenes,
            document,
            "Feature Properties",
            |snapshot, _| apply_feature_properties(snapshot, edit).map_err(oblikovati_api::Error),
        )
        .map_err(|cause| cause.0)?;
        Ok(true)
    }
    /// A saved, undoable change to how a sketch's lines are drawn.
    pub(super) fn set_sketch_line_style(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        style: &oblikovati_gui::GuiSketchLineStyle,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let saved = oblikovati_model::document_sketch::SketchLineStyle {
            color: style.color,
            line_type: style.line_type.clone(),
            weight: style.weight,
            scale: style.scale,
            dimensions_hidden: self
                .files
                .workspace()
                .session(document)
                .ok()
                .and_then(|session| session.snapshot().sketches.get(sketch).ok())
                .is_some_and(|placed| placed.line_style().dimensions_hidden),
        };
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_sketch_line_style(sketch, saved)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// Read once, on selection: a frame loop must not touch the disk.
    pub(super) fn inspect_file(&mut self, path: &str) {
        let file = std::path::Path::new(path);
        let picture = oblikovati_persistence::package::preview_picture_of_file(file);
        self.inspected_file = Some(oblikovati_gui::GuiInspectedFile {
            path: path.to_owned(),
            design_views: oblikovati_persistence::package::design_view_names_of_file(file),
            properties: oblikovati_persistence::package::property_rows_of_file(file),
            picture: picture.map(|saved| oblikovati_gui::GuiPreviewImage {
                extent: saved.extent,
                rgb: std::sync::Arc::new(saved.rgb),
            }),
        });
    }
    /// Keep a document's picture for its next save; only saving writes it to the file.
    pub(super) fn keep_document_picture(
        &mut self,
        document: u64,
        picture: Option<&oblikovati_gui::GuiPreviewImage>,
    ) {
        use oblikovati_session::files::FilePicture;
        let kept = picture.map_or(FilePicture::Remove, |image| {
            FilePicture::Store(oblikovati_persistence::package::SavedPicture {
                extent: image.extent,
                rgb: image.rgb.as_ref().clone(),
            })
        });
        self.files.keep_file_picture(document, kept);
    }
    /// Show a running navigation command's prompt in the status area, or Ready without one.
    pub(super) fn navigation_prompt(&mut self, prompt: Option<&str>) -> Result<bool, String> {
        self.command_notice = Some(match prompt {
            Some(message) => DesktopCommandNotice::PlacementPrompt {
                message: message.into(),
            },
            None => DesktopCommandNotice::Ready,
        });
        Ok(true)
    }
    /// The outline of the face on a line of sight, leaving what is chosen as it was.
    pub(super) fn edges_of_face_on_line(
        &self,
        line: [[f64; 3]; 2],
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(Vec::new());
        };
        let Some((feature, face)) = self.part_scenes.face_on_line(&self.files, document, line)?
        else {
            return Ok(Vec::new());
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = self
            .files
            .workspace()
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        self.part_scenes
            .feature_edges(session, guid, feature, Some(&face))
    }
    /// The open sketch has something to delete, and nothing else is in the way.
    pub(super) fn sketch_delete_available(&self, document: u64) -> bool {
        let Ok(session) = self.files.workspace().session(document) else {
            return false;
        };
        !session.in_transaction()
            && self.sketch_tool.is_none()
            && self.active_sketch_edit().is_some_and(|edit| {
                edit.document == document && !session.selected_sketch_picks(edit.sketch).is_empty()
            })
    }
    /// Re-resolve one current face before an action consumes its identity.
    pub(super) fn checked_selected_named_face(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<ChosenModelFace, String> {
        self.require_browser_document(document)?;
        let face = self.only_selected_named_face(document, key)?;
        let resolved = self.part_scenes.selected_face_reference(
            &mut self.files,
            document,
            face.feature,
            &face.key,
        )?;
        if resolved != key {
            return Err(format!(
                "document {document}, selected face {key:?}: expected current face reference {resolved:?}"
            ));
        }
        Ok(face)
    }
    pub(super) fn only_selected_named_face(
        &self,
        document: u64,
        key: &str,
    ) -> Result<ChosenModelFace, String> {
        let [pick] = self.current_face_appearance_picks(document) else {
            return Err(format!(
                "document {document}, selected face {key:?}: expected exactly one current face"
            ));
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if pick.key != key
            || !self.chosen_model_edges(document).is_empty()
            || !session.selected_browser_bodies().is_empty()
        {
            return Err(format!(
                "document {document}, selected face {key:?}: expected the only current selected face"
            ));
        }
        Ok(pick.face.clone())
    }
    /// The face chosen in the view, while its document is the active one.
    pub(super) fn chosen_model_face(&self, document: u64) -> Option<ChosenModelFace> {
        self.model_face
            .clone()
            .filter(|face| face.document == document)
    }
    pub(super) fn chosen_model_edges(&self, document: u64) -> &[ChosenModelEdge] {
        let generation = self
            .files
            .workspace()
            .session(document)
            .ok()
            .map(|session| session.generation());
        if self
            .model_edges
            .iter()
            .all(|edge| edge.document == document && Some(edge.generation) == generation)
        {
            &self.model_edges
        } else {
            &[]
        }
    }
    pub(super) fn require_browser_document(&self, document: u64) -> Result<(), String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "browser document {document}: expected active document {:?}",
                self.files.workspace().active_document()
            ));
        }
        Ok(())
    }
    pub(super) fn selected_browser_sketches(&self, document: u64) -> Vec<DocumentSketchId> {
        self.files
            .workspace()
            .session(document)
            .map(|session| session.selected_browser_sketches())
            .unwrap_or_default()
    }
}
/// What the view marks as chosen: whole sketches from the browser, entities from the editor.
pub(super) fn gui_sketch_selection<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
) -> oblikovati_viewport::ViewportSketchSelection {
    oblikovati_viewport::ViewportSketchSelection {
        sketches: session.selected_browser_sketches(),
        entities: session
            .selected_editor_picks()
            .iter()
            .filter_map(|pick| match pick.target() {
                oblikovati_session::selection::SketchPickTarget::Entity(entity)
                | oblikovati_session::selection::SketchPickTarget::BlockInstance(entity) => {
                    Some((pick.sketch(), entity))
                }
                _ => None,
            })
            .collect(),
    }
}
/// A face a click chose in the view; it lasts until the next click, and no edit depends on it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(in crate::owner) struct ChosenModelFace {
    pub(in crate::owner) document: u64,
    pub(in crate::owner) feature: oblikovati_model::feature::FeatureId,
    pub(in crate::owner) key: Vec<u8>,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct ChosenModelEdge {
    pub(super) document: u64,
    pub(super) generation: u64,
    pub(super) feature: oblikovati_model::feature::FeatureId,
    pub(super) key: Vec<u8>,
}
const FEATURE_COMPARISONS: [oblikovati_model::feature::FeatureSuppressionComparison; 6] = {
    use oblikovati_model::feature::FeatureSuppressionComparison::{
        Equal, GreaterOrEqual, GreaterThan, LessOrEqual, LessThan, NotEqual,
    };
    [
        Equal,
        NotEqual,
        LessThan,
        LessOrEqual,
        GreaterThan,
        GreaterOrEqual,
    ]
};

fn apply_feature_properties(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    edit: &oblikovati_gui::GuiFeatureEdit,
) -> Result<(), String> {
    let condition = feature_condition(&snapshot.parameters, edit)?;
    let features = &mut snapshot.work_geometry;
    features
        .rename(edit.feature, &edit.name)
        .map_err(|cause| cause.to_string())?;
    features
        .set_suppression_condition(edit.feature, condition)
        .map_err(|cause| cause.to_string())?;
    snapshot
        .set_feature_suppressed(edit.feature, edit.always_suppressed)
        .map_err(|cause| cause.to_string())
}

/// The dialog names a parameter and types a value; a condition holds the identity and quantity.
fn feature_condition(
    parameters: &oblikovati_model::parameter::ParameterCollection<u64>,
    edit: &oblikovati_gui::GuiFeatureEdit,
) -> Result<Option<oblikovati_model::feature::FeatureSuppressionCondition>, String> {
    let Some(condition) = &edit.condition else {
        return Ok(None);
    };
    let parameter = parameters.find(&condition.parameter).ok_or_else(|| {
        format!(
            "suppression parameter {:?}: expected a parameter of this document",
            condition.parameter
        )
    })?;
    let threshold = parameters
        .evaluate_expression(&condition.value)
        .map_err(|cause| format!("suppression value {:?}: {cause}", condition.value))?;
    let comparison = FEATURE_COMPARISONS
        .get(condition.comparison)
        .copied()
        .ok_or_else(|| format!("comparison {}: expected one of six", condition.comparison))?;
    oblikovati_model::feature::FeatureSuppressionCondition::new(parameter, comparison, threshold)
        .map(Some)
        .map_err(|cause| cause.to_string())
}
