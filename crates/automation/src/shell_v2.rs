// SPDX-License-Identifier: GPL-2.0-only
//! Typed v2 Shell creation and editing (directions, overrides, rounded type, approximation, #4106).
use crate::PreparedShell;
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        shell::{ShellDefinition, ShellDirection, ShellFaceThickness, ShellOptions},
    },
    history::DocumentSnapshot,
};

/// One face-specific thickness override input for Shell v2.
#[derive(Clone, Debug, PartialEq)]
pub struct ShellV2OverrideInput {
    pub face: Vec<u8>,
    pub thickness: String,
}

/// Complete v1/v2 Shell inputs shared by preview and commit.
#[derive(Clone, Debug, PartialEq)]
pub struct ShellV2Inputs {
    pub body: Vec<u8>,
    pub openings: Vec<Vec<u8>>,
    pub thickness: String,
    pub direction: ShellDirection,
    pub overrides: Vec<ShellV2OverrideInput>,
    pub options: ShellOptions,
}

/// Build a validated v1/v2 Shell definition from desktop/automation inputs.
pub fn build_shell_v2_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ShellV2Inputs,
) -> Result<ShellDefinition, Error> {
    if inputs.openings.is_empty() {
        return Err(Error("shell: expected at least one opening face".into()));
    }
    let thickness =
        crate::extrusion::positive_length(snapshot, &inputs.thickness, "shell thickness")?;
    let mut face_thicknesses = Vec::with_capacity(inputs.overrides.len());
    for entry in &inputs.overrides {
        let ov_t = crate::extrusion::positive_length(
            snapshot,
            &entry.thickness,
            "shell override thickness",
        )?;
        face_thicknesses.push(ShellFaceThickness {
            face: entry.face.clone(),
            thickness: ov_t,
        });
    }
    ShellDefinition::new_with_options(
        inputs.body.clone(),
        inputs.openings.clone(),
        thickness,
        inputs.direction,
        face_thicknesses,
        inputs.options,
    )
    .map_err(Error)
}

/// Prepare a validated v2 Shell draft for creation.
pub fn prepare_shell_v2(
    snapshot: &DocumentSnapshot,
    inputs: &ShellV2Inputs,
) -> Result<PreparedShell, Error> {
    let definition = build_shell_v2_definition(snapshot, inputs)?;
    let name = crate::fillet::next_name(snapshot, "Shell")?;
    Ok(PreparedShell::from_parts(name, definition))
}

/// Replace an existing Shell feature in-place inside a candidate or transaction.
pub fn replace_shell(
    candidate: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ShellDefinition,
) -> Result<(), Error> {
    candidate
        .edit_shell(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
