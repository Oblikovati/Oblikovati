// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Derived Part commands and the features.add "derive" API through certified Part replay.
use oblikovati_api::{Error, wire::AddFeatureArgs};
use oblikovati_model::{
    feature::{
        FeatureId,
        derived_part::{DeriveMirrorPlane, DeriveStyle, DerivedPartDefinition, DerivedPartRecipe},
        part::PartFeatureDefinition,
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::path::Path;

/// Validated Derived Part ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedDerivedPart {
    name: String,
    definition: DerivedPartDefinition,
}

impl PreparedDerivedPart {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_derived_part(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &DerivedPartDefinition {
        &self.definition
    }

    /// Inspect the assigned browser feature name. Example: `prepared.name()`.
    pub fn name(&self) -> &str {
        &self.name
    }
}

/// Typed inputs for preparing a Derived Part definition from desktop or automation callers.
pub struct DeriveInputs<'a> {
    pub source_path: &'a str,
    pub source_name: &'a str,
    pub style: DeriveStyle,
    pub include_bodies: bool,
    pub include_sketches: bool,
    pub include_work_geometry: bool,
    pub include_parameters: bool,
    pub scale: &'a str,
    pub mirror: bool,
    pub mirror_plane: DeriveMirrorPlane,
    pub link_face_color: bool,
    pub link_sketch_format: bool,
    pub show_all_objects: bool,
    pub linked: bool,
    pub model_state: &'a str,
    pub design_view: &'a str,
}

/// Build and validate a Derived Part definition from authored inputs.
pub fn build_derived_part_definition(
    inputs: DeriveInputs<'_>,
) -> Result<DerivedPartDefinition, Error> {
    let scale = parse_positive_scale(inputs.scale)?;
    let source_name = if inputs.source_name.trim().is_empty() {
        default_source_name(inputs.source_path)
    } else {
        inputs.source_name.trim().to_string()
    };
    let model_state = if inputs.model_state.trim().is_empty() {
        "Primary"
    } else {
        inputs.model_state.trim()
    };
    let design_view = if inputs.design_view.trim().is_empty() {
        "Primary"
    } else {
        inputs.design_view.trim()
    };
    DerivedPartDefinition::from_recipe(&DerivedPartRecipe {
        version: 1,
        source_path: inputs.source_path.trim().to_string(),
        source_name,
        style: inputs.style,
        include_bodies: inputs.include_bodies,
        include_sketches: inputs.include_sketches,
        include_work_geometry: inputs.include_work_geometry,
        include_parameters: inputs.include_parameters,
        scale,
        mirror: inputs.mirror,
        mirror_plane: inputs.mirror_plane,
        link_face_color: inputs.link_face_color,
        link_sketch_format: inputs.link_sketch_format,
        show_all_objects: inputs.show_all_objects,
        linked: inputs.linked,
        model_state: model_state.to_string(),
        design_view: design_view.to_string(),
    })
    .map_err(Error)
}

/// Prepare a named Derived Part feature with a unique browser name matching the native source.
pub fn prepare_derived_part_definition(
    snapshot: &DocumentSnapshot,
    inputs: DeriveInputs<'_>,
) -> Result<PreparedDerivedPart, Error> {
    let definition = build_derived_part_definition(inputs)?;
    let name = next_derived_name(snapshot, definition.source_name());
    Ok(PreparedDerivedPart { name, definition })
}

/// Replace an existing Derived Part feature definition in place.
pub fn replace_derived_part(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: DerivedPartDefinition,
) -> Result<(), Error> {
    let reads = definition.source_reads();
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::DerivedPart(Box::new(definition)),
            reads,
        )
        .map_err(api_error)
        .map(|_| ())
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct WireDeriveArgs {
    source_path: String,
    #[serde(default)]
    source_name: String,
    #[serde(default)]
    style: DeriveStyle,
    #[serde(default = "default_true")]
    include_bodies: bool,
    #[serde(default)]
    include_sketches: bool,
    #[serde(default)]
    include_work_geometry: bool,
    #[serde(default)]
    include_parameters: bool,
    #[serde(default = "default_scale")]
    scale: String,
    #[serde(default)]
    mirror: bool,
    #[serde(default)]
    mirror_plane: DeriveMirrorPlane,
    #[serde(default = "default_true")]
    link_face_color: bool,
    #[serde(default = "default_true")]
    link_sketch_format: bool,
    #[serde(default)]
    show_all_objects: bool,
    #[serde(default = "default_true")]
    linked: bool,
    #[serde(default)]
    model_state: String,
    #[serde(default)]
    design_view: String,
}

fn default_true() -> bool {
    true
}

fn default_scale() -> String {
    "1".into()
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input = serde_json::from_value::<WireDeriveArgs>(request.args.0.unwrap_or_default())
        .map_err(|cause| Error(format!("derive arguments: expected Derive object: {cause}")))?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("derive: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("derive: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    files
        .sync_derived_sources(document, &[&input.source_path])
        .map_err(Error)?;
    let (guid, prepared) = {
        let workspace = files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(api_error)?
            .internal_name;
        let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
        let prepared = prepare_derived_part_definition(
            snapshot,
            DeriveInputs {
                source_path: &input.source_path,
                source_name: &input.source_name,
                style: input.style,
                include_bodies: input.include_bodies,
                include_sketches: input.include_sketches,
                include_work_geometry: input.include_work_geometry,
                include_parameters: input.include_parameters,
                scale: &input.scale,
                mirror: input.mirror,
                mirror_plane: input.mirror_plane,
                link_face_color: input.link_face_color,
                link_sketch_format: input.link_sketch_format,
                show_all_objects: input.show_all_objects,
                linked: input.linked,
                model_state: &input.model_state,
                design_view: &input.design_view,
            },
        )?;
        (guid, prepared)
    };
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Derive",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn parse_positive_scale(raw: &str) -> Result<f64, Error> {
    let normalized = raw.trim().replace(',', ".");
    let value = normalized.parse::<f64>().map_err(|_| {
        Error(format!(
            "derived part scale {raw:?}: expected a positive numeric scale factor"
        ))
    })?;
    if !value.is_finite() || value <= 0.0 {
        return Err(Error(format!(
            "The expression is out of the expected range: expected a positive scale factor, got {raw:?}"
        )));
    }
    Ok(value)
}

fn default_source_name(path: &str) -> String {
    Path::new(path)
        .file_name()
        .and_then(|name| name.to_str())
        .filter(|name| !name.is_empty())
        .unwrap_or("DerivedPart.opd")
        .to_string()
}

fn next_derived_name(snapshot: &DocumentSnapshot, base: &str) -> String {
    let base = base.trim();
    if snapshot
        .work_geometry
        .features()
        .iter()
        .all(|feature| feature.name() != base)
    {
        return base.to_string();
    }
    for ordinal in 2..=10_000 {
        let candidate = format!("{base}:{ordinal}");
        if snapshot
            .work_geometry
            .features()
            .iter()
            .all(|feature| feature.name() != candidate)
        {
            return candidate;
        }
    }
    format!("{base}:{}", snapshot.work_geometry.features().len() + 1)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
