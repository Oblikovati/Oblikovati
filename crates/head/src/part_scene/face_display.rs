// SPDX-License-Identifier: GPL-2.0-only
//! Cache derived face meshes by document generation for selection presentation (#4048).
use super::*;
use oblikovati_model::{feature::FeatureId, history::DocumentEditHistory, identity::DocumentGuid};
use std::collections::BTreeMap;

pub(super) type FaceSelection = (u64, u64, Vec<(FeatureId, Vec<u8>)>, [u8; 3]);
type NamedBodies = BTreeMap<FeatureId, Result<Vec<oblikovati_martensite::NamedFaceMesh>, String>>;

pub(super) struct FaceMeshCache {
    pub source: (u64, u64),
    pub bodies: NamedBodies,
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
            .health()
            .iter()
            .filter_map(|record| {
                evaluation.body(record.feature).map(|body| {
                    (
                        record.feature,
                        body.named_face_meshes(history, document, DISPLAY_MESH),
                    )
                })
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
        if !cache.bodies.contains_key(feature) {
            result.status.scene_errors.push(format!(
                "selected face {key:?} feature {feature:?}: expected a current display body"
            ));
        }
    }
    for record in evaluation.health() {
        let Some(body) = evaluation.body(record.feature) else {
            continue;
        };
        let part = style(body).and_then(|color| match color {
            Some(color) => build_body_scene(cache, selection, record.feature, color).map(Some),
            None => Ok(None),
        });
        append_named_body(&mut result, record.feature, part);
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
        Ok(Some(mut scene)) => {
            result.status.visible_bodies += 1;
            result.scene.triangles.append(&mut scene.triangles);
        }
        Ok(None) => {}
        Err(cause) => result
            .status
            .presentation_declines
            .push(PartPresentationDecline { feature, cause }),
    }
}

fn build_body_scene(
    cache: &FaceMeshCache,
    selection: &FaceSelection,
    feature: FeatureId,
    body_color: [f32; 4],
) -> Result<ViewportScene, String> {
    let meshes = cache
        .bodies
        .get(&feature)
        .ok_or_else(|| format!("display feature {feature:?}: expected cached named faces"))?
        .as_ref()
        .map_err(Clone::clone)?;
    validate_selected_meshes(meshes, selection, feature)?;
    let selected = selection.3.map(|channel| f32::from(channel) / 255.);
    let selected = [selected[0], selected[1], selected[2], 1.];
    let triangles = meshes
        .iter()
        .flat_map(|face| {
            let color = if selection
                .2
                .iter()
                .any(|(id, key)| *id == feature && *key == face.key)
            {
                selected
            } else {
                body_color
            };
            face.triangles
                .iter()
                .map(move |triangle| (*triangle, color))
        })
        .collect::<Vec<_>>();
    ViewportScene::from_colored_triangles([0., 0., 1.], &triangles)
}

fn validate_selected_meshes(
    meshes: &[oblikovati_martensite::NamedFaceMesh],
    selection: &FaceSelection,
    feature: FeatureId,
) -> Result<(), String> {
    for (_, key) in selection.2.iter().filter(|(id, _)| *id == feature) {
        if !meshes.iter().any(|face| face.key == *key) {
            return Err(format!(
                "display feature {feature:?} face {key:?}: expected one named face mesh"
            ));
        }
    }
    Ok(())
}

impl DesktopPartScenes {
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
    ) -> Result<(), String> {
        let source = (document, session.generation());
        if self
            .face_mesh_cache
            .as_ref()
            .is_some_and(|cache| cache.source != source)
        {
            self.face_mesh_cache = None;
        }
        if mode != AssemblyViewMode::Live
            || self
                .face_selection
                .as_ref()
                .is_none_or(|item| (item.0, item.1) != source)
        {
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
        };
        let cache = FaceMeshCache {
            source: (1, 2),
            bodies: BTreeMap::from([(feature, Ok(vec![face(1), face(2)]))]),
        };
        let selection = (1, 2, vec![(feature, vec![2])], [0, 200, 160]);
        let body_color = [0.5, 0.5, 0.5, 1.];
        let scene = build_body_scene(&cache, &selection, feature, body_color).unwrap();
        assert_eq!(scene.triangles[0].color, body_color);
        assert_eq!(scene.triangles[1].color, [0., 200. / 255., 160. / 255., 1.]);
        let unknown = (1, 2, vec![(feature, vec![3])], [0, 200, 160]);
        assert!(
            build_body_scene(&cache, &unknown, feature, body_color)
                .unwrap_err()
                .contains("expected one named face mesh")
        );
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
