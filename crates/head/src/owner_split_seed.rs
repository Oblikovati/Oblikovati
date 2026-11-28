// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration, pre-split face count recovery, and body naming helpers for Split (#3918).
use oblikovati_automation::SplitFeatureInputs;
use oblikovati_gui::{GuiSplitEditSeed, GuiSplitRequest};
use oblikovati_martensite::{EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::{PartFeatureDefinition, PartFeatureEvaluation},
        split::{SplitMode, SplitTarget},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DocumentGuid, ReferenceEnvelope},
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    evaluation: Option<&PartFeatureEvaluation<NamedSolid, EvaluationFailure>>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> BTreeMap<FeatureId, GuiSplitEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Split(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let (body, all_faces, all_faces_count, faces) = match recipe.target {
                SplitTarget::AllFaces { body } => {
                    let count = evaluation
                        .and_then(|eval| eval.body(feature.id()))
                        .map(|solid| pre_split_face_count(solid, history, guid))
                        .unwrap_or(0);
                    (body, true, count, Vec::new())
                }
                SplitTarget::Faces { body, faces } => (body, false, 0, faces),
                SplitTarget::Solid { body } => (body, false, 0, Vec::new()),
            };
            Some((
                feature.id(),
                GuiSplitEditSeed {
                    mode: recipe.mode,
                    tool: recipe.tool,
                    body,
                    all_faces,
                    all_faces_count,
                    faces,
                    keep_side: recipe.keep_side,
                },
            ))
        })
        .collect()
}

pub(super) fn pre_split_face_count(
    solid: &NamedSolid,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> usize {
    let live_faces = solid
        .face_reference_keys(history, guid)
        .map(|keys| keys.len())
        .unwrap_or(0);
    let Some(splits) = solid.split_topology_keys() else {
        return live_faces;
    };
    let mut extra = 0usize;
    for (parent, children) in splits {
        if ReferenceEnvelope::decode(parent).is_ok_and(|env| env.kind == 1) && children.len() > 1 {
            extra = extra.saturating_add(children.len() - 1);
        }
    }
    live_faces.saturating_sub(extra)
}

/// Map a split-descendant face key back to its parent face key when editing an existing Split.
pub(super) fn remap_split_descendant_face(solid: &NamedSolid, face: &[u8]) -> Vec<u8> {
    let Some(splits) = solid.split_topology_keys() else {
        return face.to_vec();
    };
    for (parent, children) in splits {
        if ReferenceEnvelope::decode(parent).is_ok_and(|env| env.kind == 1)
            && children.iter().any(|child| child.as_slice() == face)
        {
            return parent.clone();
        }
    }
    face.to_vec()
}

pub(super) fn request_to_inputs(request: &GuiSplitRequest) -> Option<SplitFeatureInputs> {
    let tool = request.tool.clone()?;
    let body = request.body.clone()?;
    let target = match request.mode {
        SplitMode::Faces if request.all_faces => SplitTarget::AllFaces { body },
        SplitMode::Faces => {
            if request.faces.is_empty() {
                return None;
            }
            SplitTarget::Faces {
                body,
                faces: request.faces.clone(),
            }
        }
        SplitMode::Solid => SplitTarget::Solid { body },
    };
    Some(SplitFeatureInputs {
        mode: request.mode,
        tool,
        target,
        keep_side: request.keep_side,
    })
}

pub(super) fn max_solid_number(
    rows: &[crate::part_scene::BodyRow],
    stored: &BTreeMap<String, String>,
) -> usize {
    rows.iter()
        .map(|row| row.name.as_str())
        .chain(stored.values().map(String::as_str))
        .filter_map(parse_solid_number)
        .max()
        .unwrap_or(0)
}

fn parse_solid_number(name: &str) -> Option<usize> {
    name.trim().strip_prefix("Solid")?.parse::<usize>().ok()
}

pub(super) fn encode_hex_key(bytes: &[u8]) -> String {
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}

pub(super) fn decode_hex_key(hex: &str) -> Result<Vec<u8>, String> {
    if !hex.len().is_multiple_of(2) {
        return Err(format!("split body key {hex:?}: expected even hex length"));
    }
    hex.as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            std::str::from_utf8(pair)
                .ok()
                .and_then(|text| u8::from_str_radix(text, 16).ok())
                .ok_or_else(|| format!("split body key {hex:?}: expected hex bytes"))
        })
        .collect()
}
