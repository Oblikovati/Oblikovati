// SPDX-License-Identifier: GPL-2.0-only
#[path = "owner_model_state.rs"]
mod model_state;
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::selection::SelectionClickMode;
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
        self.selected_occurrence = None;
        Ok(changed)
    }
    pub(super) fn select_browser_feature(
        &mut self,
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_feature_selection(feature)
            .map_err(|cause| cause.to_string())?;
        self.model_face = None;
        self.selected_occurrence = None;
        Ok(true)
    }
    /// A click on the view: the face on the line of sight is chosen, and every browser row is
    /// let go. On nothing, nothing stays chosen.
    pub(super) fn click_model(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        if self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .assembly
            .is_some()
        {
            if !self
                .ambient_selection
                .filter()
                .accepts(oblikovati_session::selection::SelectionKind::Occurrence)
            {
                return Ok(false);
            }
            return self.click_assembly(document, line);
        }
        if self
            .part_measure
            .as_ref()
            .is_none_or(|measure| measure.document != document)
            && !self
                .ambient_selection
                .filter()
                .accepts(oblikovati_session::selection::SelectionKind::Face)
        {
            return Ok(false);
        }
        self.selected_occurrence = None;
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
            .map_err(|cause| cause.to_string())?;
        let picked = self.part_scenes.face_on_line(&self.files, document, line)?;
        self.measure_part_face(document, picked)?;
        self.model_face = picked.map(|(feature, role)| ChosenModelFace {
            document,
            feature,
            role,
        });
        Ok(true)
    }
    fn click_assembly(&mut self, document: u64, line: [[f64; 3]; 2]) -> Result<bool, String> {
        let picked = self
            .part_scenes
            .occurrence_on_line(&mut self.files, document, line);
        let choice = match picked {
            Ok(Some(path)) => oblikovati_gui::GuiBrowserChoice::Occurrence(path),
            Ok(None) => oblikovati_gui::GuiBrowserChoice::Nothing,
            Err(cause) => {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                return Ok(true);
            }
        };
        self.browser_choice(document, &choice)
    }
    fn set_sketch_dimensions_shown(
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
        let edit = edit.clone();
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Feature Properties", move |snapshot, _| {
                let condition = feature_condition(&snapshot.parameters, &edit)?;
                let features = &mut snapshot.work_geometry;
                features
                    .rename(edit.feature, &edit.name)
                    .map_err(|cause| cause.to_string())?;
                features
                    .set_suppression_condition(edit.feature, condition)
                    .map_err(|cause| cause.to_string())?;
                snapshot
                    .set_feature_suppressed(edit.feature, edit.always_suppressed)
                    .map_err(|cause| cause.to_string())?;
                Ok::<(), String>(())
            })
            .map_err(|cause| cause.to_string())?;
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
    pub(super) fn browser_choice(
        &mut self,
        document: u64,
        choice: &oblikovati_gui::GuiBrowserChoice,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiBrowserChoice::{
            ActivateModelState, Body, BodyVisible, CopyModelState, CreateModelState, Nothing,
            Occurrence, RenameModelState, Root, SetOccurrenceSuppressed, ShowFeatureDimensions,
            SketchDimensionsShown, ToggleModelStateEditScope,
        };
        self.require_browser_document(document)?;
        if let SetOccurrenceSuppressed { path, suppressed } = choice {
            self.document_host()
                .set_occurrence_suppressed(document, path, *suppressed)
                .map_err(|cause| cause.to_string())?;
            self.selected_occurrence = None;
            self.model_face = None;
            return Ok(true);
        }
        if let Occurrence(path) = choice {
            let browser = self.gui_browser_snapshot(document)?;
            if !super::model_browser::occurrences::contains_occurrence(&browser.occurrences, path) {
                return Err(format!(
                    "occurrence path {path:?}: expected a current assembly occurrence"
                ));
            }
            self.files
                .commands(document)
                .map_err(|cause| cause.to_string())?
                .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                .map_err(|cause| cause.to_string())?;
            self.selected_occurrence = Some((document, path.clone()));
            self.model_face = None;
            return Ok(true);
        }
        if let RenameModelState { id, name } = choice {
            return self.rename_model_state(*id, name);
        }
        if matches!(choice, CreateModelState) {
            return self.create_active_model_state(document);
        }
        if matches!(choice, ToggleModelStateEditScope) {
            return self.toggle_model_state_scope(document);
        }
        if let CopyModelState(id) = choice {
            return self.copy_model_state(*id);
        }
        if let ActivateModelState(id) = choice {
            oblikovati_api::client::Client::new(self.document_host())
                .model_states()
                .activate(&oblikovati_api::wire::RepRef { id: *id })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        if let ShowFeatureDimensions(feature) = choice {
            self.shown_feature_dimensions = feature.map(|feature| (document, feature));
            return Ok(true);
        }
        if let SketchDimensionsShown { sketch, shown } = choice {
            return self.set_sketch_dimensions_shown(document, *sketch, *shown);
        }
        let mut commands = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?;
        match choice {
            CreateModelState
            | CopyModelState(_)
            | RenameModelState { .. }
            | ActivateModelState(_)
            | ToggleModelStateEditScope
            | ShowFeatureDimensions(_)
            | SketchDimensionsShown { .. }
            | SetOccurrenceSuppressed { .. }
            | Occurrence(_) => {}
            Nothing => {
                commands
                    .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                    .map_err(|cause| cause.to_string())?;
            }
            Root => commands.set_browser_root_selection(),
            Body { key, feature } => commands.set_browser_body_selection(key, *feature),
            BodyVisible { key, visible } => {
                commands.set_body_visible(key, *visible)?;
            }
        }
        if matches!(choice, Nothing | Root | Body { .. }) {
            self.model_face = None;
            self.selected_occurrence = None;
        }
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
        let Some((feature, role)) = self.part_scenes.face_on_line(&self.files, document, line)?
        else {
            return Ok(Vec::new());
        };
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        self.part_scenes.feature_edges(session, feature, Some(role))
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
    /// The face chosen in the view, while its document is the active one.
    pub(super) fn chosen_model_face(&self, document: u64) -> Option<ChosenModelFace> {
        self.model_face.filter(|face| face.document == document)
    }
    /// The chosen row's outline; worked out once per model state, then read every frame.
    pub(super) fn current_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
        let Some(source) = self.browser_outline_source() else {
            return Vec::new();
        };
        if let Some((known, outline)) = &self.browser_outline
            && *known == source
        {
            return outline.clone();
        }
        // A row that cannot be outlined is reported where its body or sketch shows; it has no mark.
        let outline = self.browser_outline_of(source).unwrap_or_default();
        self.browser_outline = Some((source, outline.clone()));
        outline
    }
    fn browser_outline_source(&self) -> Option<BrowserOutlineSource> {
        let document = self.files.workspace().active_document()?;
        let session = self.files.workspace().session(document).ok()?;
        let row = match (
            session.selected_browser_feature(),
            session.selected_browser_sketches().first(),
        ) {
            _ if session.browser_root_selected() => BrowserOutlineRow::Root,
            _ if session.selected_browser_body().is_some() => {
                BrowserOutlineRow::Feature(session.selected_browser_body()?.1)
            }
            (Some(feature), _) => BrowserOutlineRow::Feature(feature),
            (None, None) => {
                let face = self.chosen_model_face(document)?;
                BrowserOutlineRow::Face(face.feature, face.role)
            }
            // A sketch in view already reads as chosen by its colour; only a hidden one needs this.
            (None, Some(sketch))
                if self.active_sketch_edit().is_none()
                    && session
                        .snapshot()
                        .sketches
                        .get(*sketch)
                        .is_ok_and(|placed| !placed.visible()) =>
            {
                BrowserOutlineRow::Sketch(*sketch)
            }
            _ => return None,
        };
        Some(BrowserOutlineSource {
            document,
            generation: session.generation(),
            row,
        })
    }
    fn outline_of_every_feature(
        &self,
        session: &oblikovati_session::DocumentSession<O>,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let mut outline = Vec::new();
        for feature in session.snapshot().work_geometry.features() {
            outline.extend(
                self.part_scenes
                    .feature_edges(session, feature.id(), None)?,
            );
        }
        Ok(outline)
    }
    fn browser_outline_of(
        &self,
        source: BrowserOutlineSource,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let session = self
            .files
            .workspace()
            .session(source.document)
            .map_err(|cause| cause.to_string())?;
        match source.row {
            BrowserOutlineRow::Feature(feature) => {
                self.part_scenes.feature_edges(session, feature, None)
            }
            BrowserOutlineRow::Root => self.outline_of_every_feature(session),
            BrowserOutlineRow::Face(feature, role) => {
                self.part_scenes.feature_edges(session, feature, Some(role))
            }
            BrowserOutlineRow::Sketch(sketch) => sketch_outline(session.snapshot(), sketch),
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
/// Which row an outline shows, and the model state it was worked out from.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct BrowserOutlineSource {
    document: u64,
    generation: u64,
    row: BrowserOutlineRow,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BrowserOutlineRow {
    Root,
    Feature(oblikovati_model::feature::FeatureId),
    Face(
        oblikovati_model::feature::FeatureId,
        oblikovati_martensite::PrismFaceRole,
    ),
    Sketch(DocumentSketchId),
}
/// A sketch's curves as segments in model space, as finely as they are drawn when edited.
fn sketch_outline(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: DocumentSketchId,
) -> Result<Vec<[[f64; 3]; 2]>, String> {
    use oblikovati_martensite::planar_curve::sketch::{
        SketchDisplayPrimitive::Curve, SketchSamplingOptionsV1, sample_sketch_display,
    };
    let geometry = snapshot
        .sketches
        .get(sketch)
        .map_err(|cause| cause.to_string())?
        .geometry();
    let plane = snapshot
        .resolve_sketch_plane(sketch)
        .map_err(|cause| cause.to_string())?;
    let report = sample_sketch_display(
        geometry,
        SketchSamplingOptionsV1 {
            deflection: OUTLINE_DEFLECTION,
            maximum_curve_points: 4096,
        },
    )?;
    let mut segments = Vec::new();
    for entity in report.entities {
        let Curve(points) = entity.primitive else {
            continue;
        };
        for pair in points.windows(2) {
            segments.push([
                plane.to_model(pair[0]).map_err(|cause| cause.to_string())?,
                plane.to_model(pair[1]).map_err(|cause| cause.to_string())?,
            ]);
        }
    }
    Ok(segments)
}
/// Display fineness of an outlined sketch, in working units.
const OUTLINE_DEFLECTION: f64 = 0.01;
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
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct ChosenModelFace {
    document: u64,
    feature: oblikovati_model::feature::FeatureId,
    role: oblikovati_martensite::PrismFaceRole,
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
