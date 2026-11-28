// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration, hex-key decoding, and input building for Direct Edit (#3918).
use oblikovati_automation::DirectEditFeatureInputs;
use oblikovati_gui::{GuiDirectEditEditSeed, GuiDirectEditRequest};
use oblikovati_model::{
    feature::{
        FeatureId,
        direct_edit::{
            DirectEditDefinition, DirectEditMode, DirectEditOperation, DirectEditOperationKind,
            DirectEditScaleMode, DirectEditSelectionFilter, validate_scale_factor,
        },
        part::PartFeatureDefinition,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDirectEditEditSeed> {
    let mut seeds = BTreeMap::new();
    for feature in snapshot.work_geometry.features() {
        let PartFeatureDefinition::DirectEdit(definition) = feature.definition() else {
            continue;
        };
        seeds.insert(
            feature.id(),
            GuiDirectEditEditSeed {
                body: definition.body().to_vec(),
                operations: definition.operations().to_vec(),
            },
        );
    }
    seeds
}

pub(super) fn decode_hex_key(hex: &str) -> Result<Vec<u8>, String> {
    if !hex.len().is_multiple_of(2) {
        return Err(format!(
            "direct edit body key {hex:?}: expected even-length hex"
        ));
    }
    (0..hex.len())
        .step_by(2)
        .map(|i| {
            u8::from_str_radix(&hex[i..i + 2], 16)
                .map_err(|cause| format!("direct edit body key {hex:?}: {cause}"))
        })
        .collect()
}

pub(super) fn request_has_selection(request: &GuiDirectEditRequest) -> bool {
    match request.filter {
        DirectEditSelectionFilter::Faces => !request.faces.is_empty() && request.body.is_some(),
        DirectEditSelectionFilter::Solids => request.body.is_some(),
    }
}

pub(super) fn parse_positive_factor(text: &str, label: &str) -> Result<f64, String> {
    let trimmed = text.trim();
    let value: f64 = trimmed.parse().map_err(|_| {
        format!("direct edit scale {label} {trimmed:?}: expected a positive numeric factor")
    })?;
    validate_scale_factor(value, label)?;
    Ok(value)
}

pub(super) fn build_operation_kind(
    request: &GuiDirectEditRequest,
) -> Result<DirectEditOperationKind, String> {
    match request.mode {
        DirectEditMode::Move => Ok(DirectEditOperationKind::Move {
            filter: request.filter,
            faces: request.faces.clone(),
            translation: [0.0, 0.0, 0.0],
        }),
        DirectEditMode::Size => Ok(DirectEditOperationKind::Size {
            faces: request.faces.clone(),
            offset: 0.0,
        }),
        DirectEditMode::Scale => {
            let uniform = parse_positive_factor(&request.uniform_factor, "Uniform")?;
            let factors = match request.scale_mode {
                DirectEditScaleMode::Uniform => [uniform, uniform, uniform],
                DirectEditScaleMode::NonUniform => [
                    parse_positive_factor(&request.scale_factors[0], "X")?,
                    parse_positive_factor(&request.scale_factors[1], "Y")?,
                    parse_positive_factor(&request.scale_factors[2], "Z")?,
                ],
            };
            Ok(DirectEditOperationKind::Scale {
                mode: request.scale_mode,
                uniform,
                factors,
            })
        }
        DirectEditMode::Rotate => Ok(DirectEditOperationKind::Rotate {
            filter: request.filter,
            faces: request.faces.clone(),
            coordinate: request.coordinate_mode,
            origin: [0.0, 0.0, 0.0],
            axis: [0.0, 0.0, 1.0],
            angle_deg: 0.0,
        }),
        DirectEditMode::Delete => Ok(DirectEditOperationKind::Delete {
            faces: request.faces.clone(),
        }),
    }
}

pub(super) fn build_feature_inputs(
    snapshot: &DocumentSnapshot,
    request: &GuiDirectEditRequest,
    body: Vec<u8>,
) -> Result<DirectEditFeatureInputs, String> {
    let kind = build_operation_kind(request)?;
    if let Some(feature_id) = request.feature
        && let Ok(feature) = snapshot.work_geometry.get(feature_id)
        && let PartFeatureDefinition::DirectEdit(existing) = feature.definition()
    {
        let mut operations = existing.operations().to_vec();
        if let Some(idx) = request.editing_op_index
            && idx < operations.len()
        {
            operations[idx].kind = kind;
        } else {
            let name = DirectEditDefinition::next_operation_name(&operations, request.mode);
            operations.push(DirectEditOperation { name, kind });
        }
        return Ok(DirectEditFeatureInputs {
            body: existing.body().to_vec(),
            operations,
        });
    }
    let name = DirectEditDefinition::next_operation_name(&[], request.mode);
    Ok(DirectEditFeatureInputs {
        body,
        operations: vec![DirectEditOperation { name, kind }],
    })
}
