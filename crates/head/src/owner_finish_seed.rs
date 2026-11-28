// SPDX-License-Identifier: GPL-2.0-only
//! Browser Edit Feature hydration and warning detection for Part Modify Finish (#3918).
use oblikovati_gui::GuiFinishEditSeed;
use oblikovati_martensite::{EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus, FeatureId,
        part::{PartFeatureDefinition, PartFeatureEvaluation},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::DocumentGuid,
};
use std::collections::BTreeMap;

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    evaluation: Option<&PartFeatureEvaluation<NamedSolid, EvaluationFailure>>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> BTreeMap<FeatureId, GuiFinishEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Finish(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let has_warning = evaluation.is_some_and(|ev| {
                ev.health().iter().any(|record| {
                    record.feature == feature.id()
                        && matches!(
                            record.status,
                            FeatureEvaluationStatus::Blocked { .. }
                                | FeatureEvaluationStatus::Failed(_)
                                | FeatureEvaluationStatus::ConditionFailed(_)
                        )
                })
            });
            let live = match evaluation {
                Some(ev) => resolve_live_finish_geometry(ev, history, guid, &recipe),
                None => LiveFinishGeometry {
                    body: None,
                    included_faces: Vec::new(),
                    excluded_faces: Vec::new(),
                    area_mm2: 0.0,
                },
            };
            let missing_original = live.included_faces.is_empty();
            Some((
                feature.id(),
                GuiFinishEditSeed {
                    body: live.body,
                    included_faces: live.included_faces,
                    excluded_faces: live.excluded_faces,
                    finish_type: recipe.finish_type,
                    process: recipe.process,
                    appearance: recipe.appearance,
                    thickness: recipe.thickness,
                    hardness: recipe.hardness,
                    depth: recipe.depth,
                    description: recipe.description,
                    short_description: recipe.short_description,
                    comment: recipe.comment,
                    export_all_parameters: recipe.export_all_parameters,
                    has_warning: has_warning || missing_original,
                    area_mm2: live.area_mm2,
                },
            ))
        })
        .collect()
}

struct LiveFinishGeometry {
    body: Option<Vec<u8>>,
    included_faces: Vec<Vec<u8>>,
    excluded_faces: Vec<Vec<u8>>,
    area_mm2: f64,
}

fn resolve_live_finish_geometry(
    evaluation: &PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
    recipe: &oblikovati_model::feature::finish::FinishRecipe,
) -> LiveFinishGeometry {
    for (_, solid) in evaluation.ordered_body_outputs() {
        let Ok(keys) = solid.face_reference_keys(history, guid) else {
            continue;
        };
        let live_included: Vec<Vec<u8>> = recipe
            .included_faces
            .iter()
            .filter(|face| keys.iter().any(|k| k == *face))
            .cloned()
            .collect();
        if live_included.is_empty() {
            continue;
        }
        let live_excluded: Vec<Vec<u8>> = recipe
            .excluded_faces
            .iter()
            .filter(|face| keys.iter().any(|k| k == *face))
            .cloned()
            .collect();
        let body = solid
            .body_reference_key(history, guid)
            .ok()
            .map(<[u8]>::to_vec);
        let area_mm2 = solid
            .finish_area_mm2(history, guid, &live_included, &live_excluded)
            .unwrap_or(0.0);
        return LiveFinishGeometry {
            body,
            included_faces: live_included,
            excluded_faces: live_excluded,
            area_mm2,
        };
    }
    LiveFinishGeometry {
        body: None,
        included_faces: Vec::new(),
        excluded_faces: Vec::new(),
        area_mm2: 0.0,
    }
}
