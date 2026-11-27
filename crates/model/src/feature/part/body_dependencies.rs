// SPDX-License-Identifier: GPL-2.0-only
//! Add ordered body-replacement dependencies to a transient replay program.
use super::{PartFeatureDefinition, PartFeatureProgramError};
use crate::feature::{FeatureId, FeatureReadFootprint, FeatureReadKey, FeatureSequence};
use std::collections::BTreeMap;

pub(super) fn with_replacement_reads(
    mut sequence: FeatureSequence<PartFeatureDefinition>,
) -> Result<FeatureSequence<PartFeatureDefinition>, PartFeatureProgramError> {
    let entries: Vec<_> = sequence
        .features()
        .iter()
        .map(|feature| {
            (
                feature.id(),
                feature.definition().clone(),
                feature.reads().clone(),
            )
        })
        .collect();
    let mut owners = BTreeMap::new();
    for (id, definition, reads) in entries {
        let extended = replacement_footprint(&reads, &owners);
        if extended != reads {
            sequence
                .replace(id, definition.clone(), extended)
                .map_err(PartFeatureProgramError::Sequence)?;
        }
        track_replacement(&mut owners, id, &definition);
    }
    Ok(sequence)
}

fn replacement_footprint(
    reads: &FeatureReadFootprint,
    owners: &BTreeMap<FeatureId, FeatureId>,
) -> FeatureReadFootprint {
    let mut keys = reads.keys().clone();
    for key in reads.keys() {
        if let FeatureReadKey::Feature(source) = key
            && let Some(owner) = owners.get(source)
        {
            keys.insert(FeatureReadKey::Feature(*owner));
        }
    }
    FeatureReadFootprint::new(keys, reads.has_unattributed_reads())
}

fn track_replacement(
    owners: &mut BTreeMap<FeatureId, FeatureId>,
    id: FeatureId,
    definition: &PartFeatureDefinition,
) {
    let source = match definition {
        PartFeatureDefinition::SketchPrism(prism) => match prism.boolean_target() {
            Some(target) => target.source(),
            None => {
                owners.insert(id, id);
                return;
            }
        },
        PartFeatureDefinition::SketchLoft(_) => {
            owners.insert(id, id);
            return;
        }
        PartFeatureDefinition::SketchRevolve(revolve) => match revolve.boolean_target() {
            Some((_, _, source)) => source,
            None => {
                owners.insert(id, id);
                return;
            }
        },
        PartFeatureDefinition::SketchCoil(coil) => match coil.boolean_target() {
            Some((_, _, source)) => source,
            None => {
                owners.insert(id, id);
                return;
            }
        },
        PartFeatureDefinition::SketchEmboss(emboss) => emboss.target_body().1,
        PartFeatureDefinition::SketchRib(rib) => rib.target_body().1,
        PartFeatureDefinition::SketchSweep(sweep) => match sweep.boolean_target() {
            Some((_, _, source)) => source,
            None => {
                owners.insert(id, id);
                return;
            }
        },
        PartFeatureDefinition::Fillet(fillet) => fillet.body_source(),
        PartFeatureDefinition::Chamfer(chamfer) => chamfer.body_source(),
        PartFeatureDefinition::Hole(hole) => hole.body_source(),
        PartFeatureDefinition::Shell(shell) => shell.body_source(),
        PartFeatureDefinition::MoveBody(motion) => motion.body_source(),
        PartFeatureDefinition::Mirror(mirror) => mirror.recipe().sources[0],
        PartFeatureDefinition::CircularPattern(pattern) => pattern.recipe().sources[0],
        PartFeatureDefinition::RectangularPattern(pattern) => pattern.recipe().sources[0],
        PartFeatureDefinition::DeleteBody(delete) => delete.source(),
        _ => return,
    };
    let Some(previous) = owners.get(&source).copied() else {
        return;
    };
    for owner in owners.values_mut() {
        if *owner == previous {
            *owner = id;
        }
    }
    owners.insert(source, id);
    owners.insert(id, id);
}
