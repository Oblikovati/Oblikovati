// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Slot creation and editing through the certified part replay owner.
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        hole::{HoleExtent, HoleFastenerSpec, HoleKind, HolePosition},
        slot::{SlotCenterMarks, SlotDefinition, SlotEndMill, SlotRecipe, SlotSeat, SlotShape},
        work_geometry::DatumLength,
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Seat parameters for desktop/automation Slot creation and editing.
#[derive(Clone, Debug, Default, PartialEq)]
pub enum SlotSeatInput {
    #[default]
    None,
    Counterbore {
        width: String,
        depth: String,
    },
    Spotface {
        width: String,
        depth: String,
    },
    Countersink {
        width: String,
        angle_deg: f64,
    },
}

/// Typed desktop/automation inputs for creating or editing a Part Modify Slot.
#[derive(Clone, Debug, PartialEq)]
pub struct SlotFeatureInputs {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
    pub shape: SlotShape,
    pub length: String,
    pub width: String,
    pub rotation_deg: f64,
    pub termination: crate::HoleTerminationInput,
    pub kind: HoleKind,
    pub seat: SlotSeatInput,
    pub end_mill: SlotEndMill,
    pub center_marks: SlotCenterMarks,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub imate: bool,
    pub fastener: Option<HoleFastenerSpec>,
}

/// A validated Slot recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedSlot {
    name: String,
    definition: SlotDefinition,
}

impl PreparedSlot {
    /// Inspect the validated Slot definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &SlotDefinition {
        &self.definition
    }

    /// Add the prepared Slot recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_slot(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Slot definition from desktop/automation inputs.
pub fn build_slot_definition(
    snapshot: &DocumentSnapshot,
    inputs: &SlotFeatureInputs,
) -> Result<SlotDefinition, Error> {
    let length = crate::extrusion::positive_length(snapshot, &inputs.length, "slot length")?;
    let width = crate::extrusion::positive_length(snapshot, &inputs.width, "slot width")?;
    let extent = match &inputs.termination {
        crate::HoleTerminationInput::ThroughAll => HoleExtent::ThroughAll,
        crate::HoleTerminationInput::Distance(expr) => HoleExtent::Distance(
            crate::extrusion::positive_length(snapshot, expr, "slot depth")?,
        ),
        crate::HoleTerminationInput::ToSurface(surface) => HoleExtent::ToSurface(surface.clone()),
    };
    let seat = match &inputs.seat {
        SlotSeatInput::None => SlotSeat::None,
        SlotSeatInput::Counterbore { width, depth } => SlotSeat::Counterbore {
            width: crate::extrusion::positive_length(snapshot, width, "slot counterbore width")?,
            depth: crate::extrusion::positive_length(snapshot, depth, "slot counterbore depth")?,
        },
        SlotSeatInput::Spotface { width, depth } => SlotSeat::Spotface {
            width: crate::extrusion::positive_length(snapshot, width, "slot spotface width")?,
            depth: crate::extrusion::positive_length(snapshot, depth, "slot spotface depth")?,
        },
        SlotSeatInput::Countersink { width, angle_deg } => SlotSeat::Countersink {
            width: crate::extrusion::positive_length(snapshot, width, "slot countersink width")?,
            angle_deg: *angle_deg,
        },
    };
    let position = match inputs.position {
        Some(point) => HolePosition::ModelPoint(point.map(DatumLength::Constant)),
        None => HolePosition::FaceCentroid,
    };
    let recipe = SlotRecipe {
        version: 1,
        body: inputs.body.clone(),
        face: inputs.face.clone(),
        position,
        shape: inputs.shape,
        length,
        width,
        rotation_deg: inputs.rotation_deg,
        extent,
        kind: inputs.kind,
        seat,
        end_mill: inputs.end_mill,
        center_marks: inputs.center_marks,
        direction_flipped: inputs.direction_flipped,
        extend_start: inputs.extend_start,
        imate: inputs.imate,
        fastener: inputs.fastener.clone(),
    };
    let definition = SlotDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve(&snapshot.parameters).map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Slot feature without mutating the document.
pub fn prepare_slot_definition(
    snapshot: &DocumentSnapshot,
    inputs: &SlotFeatureInputs,
) -> Result<PreparedSlot, Error> {
    let definition = build_slot_definition(snapshot, inputs)?;
    Ok(PreparedSlot {
        name: crate::fillet::next_name(snapshot, "Slot")?,
        definition,
    })
}

/// Replace an existing Slot feature's definition inside a transaction/preview candidate.
pub fn replace_slot(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SlotDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_slot(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
