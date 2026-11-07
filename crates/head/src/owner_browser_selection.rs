// SPDX-License-Identifier: GPL-2.0-only
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
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(Some(sketch), mode, self.sketch_tool.is_some())
            .map(|count| count.is_some())
            .map_err(|cause| cause.to_string())
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
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
            .map_err(|cause| cause.to_string())?;
        self.model_face = self
            .part_scenes
            .face_on_line(&self.files, document, line)?
            .map(|(feature, role)| ChosenModelFace {
                document,
                feature,
                role,
            });
        Ok(true)
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
            BrowserOutlineRow::Face(feature, role) => {
                self.part_scenes.feature_edges(session, feature, Some(role))
            }
            BrowserOutlineRow::Sketch(sketch) => sketch_outline(session.snapshot(), sketch),
        }
    }
    fn require_browser_document(&self, document: u64) -> Result<(), String> {
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
                oblikovati_session::selection::SketchPickTarget::Entity(entity) => {
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
