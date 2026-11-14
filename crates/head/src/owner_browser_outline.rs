// SPDX-License-Identifier: GPL-2.0-only
//! Cache and compute the visible outline for the selected browser row.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// The chosen row's outline; worked out once per model state, then read every frame.
    pub(in crate::owner) fn current_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
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
}
/// Which row an outline shows, and the model state it was worked out from.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(in crate::owner) struct BrowserOutlineSource {
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
