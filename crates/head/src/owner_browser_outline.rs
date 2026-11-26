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
    #[cfg(all(test, target_os = "linux"))]
    pub(crate) fn face_mesh_builds(&self) -> usize {
        self.part_scenes.face_mesh_builds()
    }

    pub(in crate::owner) fn current_selected_edge_outlines(&mut self) -> Vec<Vec<[[f64; 3]; 2]>> {
        if self.replacement_preview_active() && !self.fillet_input_preview_active() {
            return Vec::new();
        }
        let Some(source) = self.browser_outline_source() else {
            return Vec::new();
        };
        let BrowserOutlineRow::Edges(edges) = &source.row else {
            return Vec::new();
        };
        if let Some(cache) = &self.selected_edge_outlines
            && cache.source == source
        {
            return cache.groups.clone();
        }
        let groups = match self.edge_outline_groups(source.document, edges) {
            Ok(groups) => groups,
            Err(cause) => {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                return Vec::new();
            }
        };
        self.selected_edge_outlines = Some(EdgeOutlineCache {
            source,
            groups: groups.clone(),
        });
        groups
    }
    fn edge_outline_groups(
        &self,
        document: u64,
        edges: &[(oblikovati_model::feature::FeatureId, Vec<u8>)],
    ) -> Result<Vec<Vec<[[f64; 3]; 2]>>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        edges
            .iter()
            .map(|(feature, key)| self.part_scenes.edge_outline(session, guid, *feature, key))
            .collect()
    }
    /// The chosen row's outline; worked out once per model state, then read every frame.
    pub(in crate::owner) fn current_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
        if self.replacement_preview_active() {
            return Vec::new();
        }
        let Some(source) = self.browser_outline_source() else {
            return Vec::new();
        };
        if matches!(source.row, BrowserOutlineRow::Edges(_)) {
            return self
                .current_selected_edge_outlines()
                .into_iter()
                .flatten()
                .collect();
        }
        if let Some((known, outline)) = &self.browser_outline
            && *known == source
        {
            return outline.clone();
        }
        // A row that cannot be outlined is reported where its body or sketch shows; it has no mark.
        let outline = self.browser_outline_of(&source).unwrap_or_default();
        self.browser_outline = Some((source, outline.clone()));
        outline
    }
    fn replacement_preview_active(&self) -> bool {
        self.extrusion_preview
            .as_ref()
            .is_some_and(|(document, generation, _, kind)| {
                *kind != super::owner_extrusion_preview::PreviewKind::Additive
                    && self.files.workspace().active_document() == Some(*document)
                    && self
                        .files
                        .workspace()
                        .session(*document)
                        .is_ok_and(|session| session.generation() == *generation)
            })
    }
    fn fillet_input_preview_active(&self) -> bool {
        self.extrusion_preview
            .as_ref()
            .is_some_and(|(_, _, _, kind)| {
                *kind == super::owner_extrusion_preview::PreviewKind::BlendInput
            })
    }
    fn browser_outline_source(&self) -> Option<BrowserOutlineSource> {
        let document = self.files.workspace().active_document()?;
        let session = self.files.workspace().session(document).ok()?;
        let row = match (
            session.selected_browser_feature(),
            session.selected_browser_sketches().first(),
        ) {
            _ if !self.chosen_model_edges(document).is_empty() => BrowserOutlineRow::Edges(
                self.chosen_model_edges(document)
                    .iter()
                    .map(|edge| (edge.feature, edge.key.clone()))
                    .collect(),
            ),
            _ if !session.selected_browser_bodies().is_empty()
                && !self.current_face_appearance_picks(document).is_empty() =>
            {
                BrowserOutlineRow::Mixed {
                    bodies: session
                        .selected_browser_bodies()
                        .iter()
                        .map(|body| body.feature)
                        .collect(),
                    faces: self
                        .current_face_appearance_picks(document)
                        .iter()
                        .map(|pick| (pick.face.feature, pick.face.key.clone()))
                        .collect(),
                }
            }
            _ if session.browser_root_selected() => BrowserOutlineRow::Root,
            _ if session.selected_browser_body().is_some() => {
                BrowserOutlineRow::Feature(session.selected_browser_body()?.1)
            }
            (Some(feature), _) => BrowserOutlineRow::Feature(feature),
            (None, None) => {
                let picks = self.current_face_appearance_picks(document);
                let faces = if picks.is_empty() {
                    let face = self.chosen_model_face(document)?;
                    vec![(face.feature, face.key)]
                } else {
                    picks
                        .iter()
                        .map(|pick| (pick.face.feature, pick.face.key.clone()))
                        .collect()
                };
                BrowserOutlineRow::Faces(faces)
            }
            // A sketch in view already reads as chosen by its colour; only a hidden one needs this.
            (None, Some(sketch))
                if self.sketch_scenes.overlays_visible()
                    && self.active_sketch_edit().is_none()
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
        document: oblikovati_model::identity::DocumentGuid,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let mut outline = Vec::new();
        for feature in session.snapshot().work_geometry.features() {
            outline.extend(self.part_scenes.feature_edges(
                session,
                document,
                feature.id(),
                None,
            )?);
        }
        Ok(outline)
    }
    fn browser_outline_of(
        &self,
        source: &BrowserOutlineSource,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let session = self
            .files
            .workspace()
            .session(source.document)
            .map_err(|cause| cause.to_string())?;
        let document = self
            .files
            .workspace()
            .identity(source.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        match &source.row {
            BrowserOutlineRow::Feature(feature) => self
                .part_scenes
                .feature_edges(session, document, *feature, None),
            BrowserOutlineRow::Root => self.outline_of_every_feature(session, document),
            BrowserOutlineRow::Edges(edges) => self
                .edge_outline_groups(source.document, edges)
                .map(|groups| groups.into_iter().flatten().collect()),
            BrowserOutlineRow::Faces(faces) => {
                let mut outline = Vec::new();
                for (feature, face) in faces {
                    outline.extend(self.part_scenes.feature_edges(
                        session,
                        document,
                        *feature,
                        Some(face),
                    )?);
                }
                Ok(outline)
            }
            BrowserOutlineRow::Mixed { bodies, faces } => {
                let mut outline = Vec::new();
                for feature in bodies {
                    outline.extend(
                        self.part_scenes
                            .feature_edges(session, document, *feature, None)?,
                    );
                }
                for (feature, face) in faces {
                    outline.extend(self.part_scenes.feature_edges(
                        session,
                        document,
                        *feature,
                        Some(face),
                    )?);
                }
                Ok(outline)
            }
            BrowserOutlineRow::Sketch(sketch) => sketch_outline(session.snapshot(), *sketch),
        }
    }
}
/// Which row an outline shows, and the model state it was worked out from.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(in crate::owner) struct BrowserOutlineSource {
    document: u64,
    generation: u64,
    row: BrowserOutlineRow,
}
pub(in crate::owner) struct EdgeOutlineCache {
    source: BrowserOutlineSource,
    groups: Vec<Vec<[[f64; 3]; 2]>>,
}
#[derive(Clone, Debug, PartialEq, Eq)]
enum BrowserOutlineRow {
    Root,
    Edges(Vec<(oblikovati_model::feature::FeatureId, Vec<u8>)>),
    Mixed {
        bodies: Vec<oblikovati_model::feature::FeatureId>,
        faces: Vec<(oblikovati_model::feature::FeatureId, Vec<u8>)>,
    },
    Feature(oblikovati_model::feature::FeatureId),
    Faces(Vec<(oblikovati_model::feature::FeatureId, Vec<u8>)>),
    Sketch(DocumentSketchId),
}
/// A sketch's curves as segments in model space, as finely as they are drawn when edited.
fn sketch_outline(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: DocumentSketchId,
) -> Result<Vec<[[f64; 3]; 2]>, String> {
    use oblikovati_martensite::planar_curve::sketch::{
        SketchDisplayPrimitive, SketchSamplingOptionsV1, sample_sketch_display,
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
            working_scale: snapshot.parameters.units().working_scale(),
            deflection: OUTLINE_DEFLECTION,
            maximum_curve_points: 4096,
        },
    )?;
    let mut segments = Vec::new();
    for entity in report.entities {
        let paths = match &entity.primitive {
            SketchDisplayPrimitive::Point(_) => continue,
            SketchDisplayPrimitive::Curve(points) => std::slice::from_ref(points),
            SketchDisplayPrimitive::GlyphEdges(paths) => paths,
        };
        for pair in paths.iter().flat_map(|points| points.windows(2)) {
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
