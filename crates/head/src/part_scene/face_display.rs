// SPDX-License-Identifier: GPL-2.0-only
//! Cache derived face meshes by document generation for selection presentation (#4048).
use super::*;
use oblikovati_model::{feature::FeatureId, history::DocumentEditHistory, identity::DocumentGuid};

pub(super) type FaceSelection = (u64, u64, Vec<(FeatureId, Vec<u8>)>, [u8; 3]);
pub(crate) const THREAD_OVERLAY_COLOR: [f32; 4] = [0.22, 0.25, 0.30, 1.0];
type NamedFaces = Result<Vec<oblikovati_martensite::NamedFaceMesh>, String>;

pub(super) fn append_thread_overlays(scene: &mut ViewportScene, body: &NamedSolid) {
    for tri in body.thread_overlay_triangles() {
        scene.triangles.push(oblikovati_viewport::ViewportTriangle {
            positions: tri.positions.map(|p| p.map(|c| c as f32)),
            normals: tri.normals.map(|ns| ns.map(|n| n.map(|c| c as f32))),
            color: THREAD_OVERLAY_COLOR,
            face_index: None,
        });
    }
}

pub(crate) fn render_part_scene(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    source: (u64, u64),
    mut style: impl FnMut(&NamedSolid) -> Result<Option<[f32; 4]>, String>,
) -> Result<HeadSceneUpdate, String> {
    let mut overlay_tris = Vec::new();
    let mut presentation = ViewportScene::from_part_evaluation_styled(
        evaluated,
        DISPLAY_MESH,
        [0., 0., 1.],
        |body| {
            let color = style(body)?;
            if color.is_some() {
                for tri in body.thread_overlay_triangles() {
                    overlay_tris.push(oblikovati_viewport::ViewportTriangle {
                        positions: tri.positions.map(|p| p.map(|c| c as f32)),
                        normals: tri.normals.map(|ns| ns.map(|n| n.map(|c| c as f32))),
                        color: THREAD_OVERLAY_COLOR,
                        face_index: None,
                    });
                }
            }
            Ok(color)
        },
    )?;
    presentation.scene.triangles.extend(overlay_tris);
    Ok(HeadSceneUpdate {
        scene: presentation.scene,
        status: HeadSceneStatus {
            source: Some(source),
            cached_assembly: false,
            visible_bodies: presentation.visible_bodies,
            health: evaluated.health().to_vec(),
            presentation_declines: presentation.declines,
            scene_errors: Vec::new(),
        },
    })
}

pub(super) struct FaceMeshCache {
    pub source: (u64, u64),
    pub bodies: Vec<(FeatureId, NamedFaces)>,
}

impl FaceMeshCache {
    pub fn build(
        source: (u64, u64),
        evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<
            NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Self {
        let bodies = evaluation
            .ordered_body_outputs()
            .map(|(feature, body)| {
                (
                    feature,
                    body.named_face_meshes(history, document, DISPLAY_MESH),
                )
            })
            .collect();
        Self { source, bodies }
    }
}

pub(super) fn selected_face_scene(
    evaluation: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    cache: &FaceMeshCache,
    selection: &FaceSelection,
    mut style: impl FnMut(&NamedSolid) -> Result<Option<[f32; 4]>, String>,
) -> Result<HeadSceneUpdate, String> {
    let mut result = empty_named_scene(cache.source, evaluation.health())?;
    for (feature, key) in &selection.2 {
        if !has_selected_mesh(cache, *feature, key) {
            result.status.scene_errors.push(format!(
                "selected face {key:?} feature {feature:?}: expected one named face mesh"
            ));
        }
    }
    for ((feature, body), (cached_feature, meshes)) in
        evaluation.ordered_body_outputs().zip(&cache.bodies)
    {
        if feature != *cached_feature {
            return Err(format!(
                "display feature {feature:?}: expected its current cached output"
            ));
        }
        let part = style(body).and_then(|color| match color {
            Some(color) => build_body_scene(meshes, selection, feature, color).map(|mut s| {
                append_thread_overlays(&mut s, body);
                Some(s)
            }),
            None => Ok(None),
        });
        append_named_body(&mut result, feature, part);
    }
    Ok(result)
}

fn empty_named_scene(
    source: (u64, u64),
    health: &[FeatureHealthRecord<PartFeatureFailure<oblikovati_martensite::EvaluationFailure>>],
) -> Result<HeadSceneUpdate, String> {
    Ok(HeadSceneUpdate {
        scene: ViewportScene::from_colored_triangles([0., 0., 1.], &[])?,
        status: HeadSceneStatus {
            source: Some(source),
            cached_assembly: false,
            visible_bodies: 0,
            health: health.to_vec(),
            presentation_declines: Vec::new(),
            scene_errors: Vec::new(),
        },
    })
}

fn append_named_body(
    result: &mut HeadSceneUpdate,
    feature: FeatureId,
    part: Result<Option<ViewportScene>, String>,
) {
    match part {
        Ok(Some(scene)) => {
            result.status.visible_bodies += 1;
            result.scene.append_triangles(scene);
        }
        Ok(None) => {}
        Err(cause) => result
            .status
            .presentation_declines
            .push(PartPresentationDecline { feature, cause }),
    }
}

fn build_body_scene(
    meshes: &NamedFaces,
    selection: &FaceSelection,
    feature: FeatureId,
    body_color: [f32; 4],
) -> Result<ViewportScene, String> {
    let meshes = meshes.as_ref().map_err(Clone::clone)?;
    let selected = selection.3.map(|channel| f32::from(channel) / 255.);
    let selected = [selected[0], selected[1], selected[2], 1.];
    let triangles = meshes.iter().enumerate().flat_map(|(index, face)| {
        let color = if selection
            .2
            .iter()
            .any(|(id, key)| *id == feature && *key == face.key)
        {
            selected
        } else {
            body_color
        };
        let has_normals = face.normals.len() == face.triangles.len();
        face.triangles
            .iter()
            .enumerate()
            .map(move |(tri_index, triangle)| {
                (
                    *triangle,
                    has_normals.then(|| face.normals[tri_index]),
                    color,
                    Some(index),
                )
            })
    });
    ViewportScene::from_shaded_triangles([0., 0., 1.], triangles)
}

fn has_selected_mesh(cache: &FaceMeshCache, feature: FeatureId, key: &[u8]) -> bool {
    cache.bodies.iter().any(|(owner, meshes)| {
        *owner == feature
            && meshes
                .as_ref()
                .is_ok_and(|meshes| meshes.iter().any(|face| face.key == key))
    })
}

impl DesktopPartScenes {
    #[cfg(all(test, target_os = "linux"))]
    pub(crate) fn face_mesh_builds(&self) -> usize {
        self.face_mesh_builds
    }

    pub(crate) fn set_selected_faces(&mut self, selection: Option<FaceSelection>) {
        if self.face_selection != selection {
            self.face_selection = selection;
            self.invalidate_presentation();
        }
    }

    pub(super) fn prepare_face_mesh_cache<O: TransactionObserver>(
        &mut self,
        session: &oblikovati_session::DocumentSession<O>,
        document: u64,
        guid: DocumentGuid,
        mode: AssemblyViewMode,
        active: bool,
    ) -> Result<(), String> {
        let source = (document, session.generation());
        if self
            .face_mesh_cache
            .as_ref()
            .is_some_and(|cache| cache.source != source)
        {
            self.face_mesh_cache = None;
        }
        if mode != AssemblyViewMode::Live || !active {
            return Ok(());
        }
        if self
            .face_mesh_cache
            .as_ref()
            .is_some_and(|cache| cache.source == source)
        {
            return Ok(());
        }
        let cache = FaceMeshCache::build(source, self.current(session)?, session.history(), guid);
        self.face_mesh_cache = Some(cache);
        #[cfg(test)]
        {
            self.face_mesh_builds += 1;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn colors_only_exact_selected_face_and_refuses_unknown_key() {
        let feature = FeatureId(7);
        let triangle = [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]];
        let face = |key| oblikovati_martensite::NamedFaceMesh {
            key: vec![key],
            triangles: vec![triangle],
            normals: Vec::new(),
        };
        let cache = FaceMeshCache {
            source: (1, 2),
            bodies: vec![(feature, Ok(vec![face(1)])), (feature, Ok(vec![face(2)]))],
        };
        let selection = (1, 2, vec![(feature, vec![2])], [0, 200, 160]);
        let body_color = [0.5, 0.5, 0.5, 1.];
        let primary =
            build_body_scene(&cache.bodies[0].1, &selection, feature, body_color).unwrap();
        let sibling =
            build_body_scene(&cache.bodies[1].1, &selection, feature, body_color).unwrap();
        assert_eq!(primary.triangles[0].color, body_color);
        assert_eq!(
            sibling.triangles[0].color,
            [0., 200. / 255., 160. / 255., 1.]
        );
        assert!(has_selected_mesh(&cache, feature, &[2]));
        assert!(!has_selected_mesh(&cache, feature, &[3]));
    }
    #[test]
    fn selection_changes_invalidate_once() {
        let mut scenes = DesktopPartScenes::default();
        let selection = Some((1, 2, vec![(FeatureId(3), vec![4])], [0, 200, 160]));
        scenes.set_selected_faces(selection.clone());
        assert!(scenes.presentation_invalidated);
        scenes.presentation_invalidated = false;
        scenes.set_selected_faces(selection);
        assert!(!scenes.presentation_invalidated);
        scenes.set_selected_faces(None);
        assert!(scenes.presentation_invalidated);
    }
}
