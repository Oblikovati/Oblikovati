// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchError, SketchPoint2D, SketchRelationId, compile, residual::PreparedConstraint,
};
use crate::parameter::ParameterCollection;
use oblikovati_solve::{
    ad::DualNumber,
    residual::{ResidualBlock, ResidualError, ResidualSource},
};
use std::collections::BTreeSet;
mod layout;
use layout::{GroundedLayout, GroundedSlot};

pub(super) struct GroundedSketch {
    pub values: Vec<f64>,
    pub origins: Vec<SketchRelationId>,
    pub parameter_dependencies: Vec<u64>,
    sources: Vec<GroundedResidual>,
    layout: GroundedLayout,
}
impl GroundedSketch {
    pub fn blocks(&self) -> Vec<ResidualBlock<'_>> {
        self.sources
            .iter()
            .map(|source| ResidualBlock {
                source,
                variables: &source.variables,
                equations: source.original.equations(),
            })
            .collect()
    }
    pub fn restore_freedoms(&self, values: &[bool]) -> Result<Vec<bool>, SketchError> {
        self.layout.restore_freedoms(values)
    }
    pub fn restore(&self, values: &[f64]) -> Result<Vec<f64>, SketchError> {
        self.layout.restore(values)
    }
}
pub(super) fn prepare(
    sketch: &Sketch2D,
    parameters: &ParameterCollection<u64>,
    anchors: &[SketchPoint2D],
) -> Result<GroundedSketch, SketchError> {
    // #3762: branch and contact preparation must see the current reference coordinates.
    let (seed, identities) = grounded_seed(sketch, anchors)?;
    let prepared = compile::prepare(&seed, parameters)?;
    let layout = GroundedLayout::new(&prepared.values, &grounded_columns(&seed, &identities));
    let sources = prepared
        .sources
        .into_iter()
        .map(|source| GroundedResidual::new(source, &layout))
        .collect();
    Ok(GroundedSketch {
        values: layout.active_values(&prepared.values),
        origins: prepared.origins,
        parameter_dependencies: prepared.parameter_dependencies,
        sources,
        layout,
    })
}
fn grounded_seed(
    sketch: &Sketch2D,
    anchors: &[SketchPoint2D],
) -> Result<(Sketch2D, BTreeSet<super::SketchEntityId>), SketchError> {
    let mut seed = sketch.clone();
    let mut identities = BTreeSet::new();
    for anchor in anchors {
        if !identities.insert(anchor.id) {
            return Err(SketchError::Definition(format!(
                "grounded sketch point {:?}: expected one current coordinate source per point",
                anchor.id
            )));
        }
        seed.set_position(anchor.id, anchor.position)?;
    }
    Ok((seed, identities))
}
fn grounded_columns(
    seed: &Sketch2D,
    identities: &BTreeSet<super::SketchEntityId>,
) -> BTreeSet<usize> {
    seed.points()
        .iter()
        .enumerate()
        .filter(|(_, point)| identities.contains(&point.id))
        .flat_map(|(index, _)| [index * 2, index * 2 + 1])
        .collect()
}
struct GroundedResidual {
    original: PreparedConstraint,
    slots: Option<Vec<GroundedSlot>>,
    variables: Vec<usize>,
}
impl GroundedResidual {
    fn new(original: PreparedConstraint, layout: &GroundedLayout) -> Self {
        let slots = layout.local_slots(&original.variables);
        let variables = slots
            .iter()
            .filter_map(|slot| match slot {
                GroundedSlot::Active(index) => Some(*index),
                GroundedSlot::Constant(_) => None,
            })
            .collect();
        Self {
            original,
            // Ordinary sketches retain the original residual input path without another allocation.
            slots: slots
                .iter()
                .any(|slot| matches!(slot, GroundedSlot::Constant(_)))
                .then_some(slots),
            variables,
        }
    }
}
impl ResidualSource for GroundedResidual {
    fn tolerance_kind(&self) -> oblikovati_solve::residual::ResidualTolerance {
        self.original.tolerance_kind()
    }

    fn equation_tolerance(&self, row: usize) -> oblikovati_solve::residual::ResidualTolerance {
        self.original.equation_tolerance(row)
    }
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        let Some(slots) = &self.slots else {
            return self.original.evaluate(variables);
        };
        if variables.len() != self.variables.len() {
            return Err(ResidualError::Domain(format!(
                "grounded residual has {} inputs: expected {} active coordinates",
                variables.len(),
                self.variables.len()
            )));
        }
        self.original
            .evaluate(&expand_grounded_inputs(slots, variables)?)
    }
}
fn expand_grounded_inputs(
    slots: &[GroundedSlot],
    variables: &[DualNumber],
) -> Result<Vec<DualNumber>, ResidualError> {
    let mut active = variables.iter();
    slots
        .iter()
        .map(|slot| match slot {
            GroundedSlot::Constant(value) => {
                DualNumber::constant(*value).map_err(ResidualError::from)
            }
            GroundedSlot::Active(_) => {
                Ok(active.next().expect("validated active input count").clone())
            }
        })
        .collect()
}

#[cfg(test)]
#[path = "../../tests/sketch/grounded.rs"]
mod tests;
