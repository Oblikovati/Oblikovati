// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Import commands (STL mesh and STEP BRep) and features.add "import".
use oblikovati_api::{Error, wire::AddFeatureArgs};
use oblikovati_model::{
    feature::{
        FeatureId,
        imported_model::{
            ImportMode, ImportedFileKind, ImportedModelDefinition, ImportedModelRecipe,
            StepImportOptions, StlImportOptions, StlImportUnits,
        },
        part::PartFeatureDefinition,
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::path::Path;

/// Validated Imported Model ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedImport {
    name: String,
    definition: ImportedModelDefinition,
}

impl PreparedImport {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_imported_model(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &ImportedModelDefinition {
        &self.definition
    }

    /// Inspect the assigned browser feature name. Example: `prepared.name()`.
    pub fn name(&self) -> &str {
        &self.name
    }
}

/// Typed inputs for preparing an Imported Model definition.
pub struct ImportInputs<'a> {
    pub source_path: &'a str,
    pub source_name: &'a str,
    pub file_kind: ImportedFileKind,
    pub import_mode: ImportMode,
    pub body_names: Vec<String>,
    pub link_suppressed: bool,
    pub link_broken: bool,
    pub unit_scale: f64,
    pub stl_options: Option<StlImportOptions>,
    pub step_options: Option<StepImportOptions>,
}

/// Build and validate an Imported Model definition from authored inputs.
pub fn build_import_definition(inputs: ImportInputs<'_>) -> Result<ImportedModelDefinition, Error> {
    let source_name = if inputs.source_name.trim().is_empty() {
        default_source_name(inputs.source_path)
    } else {
        inputs.source_name.trim().to_string()
    };
    ImportedModelDefinition::from_recipe(&ImportedModelRecipe {
        version: 1,
        source_path: inputs.source_path.trim().to_string(),
        source_name,
        file_kind: inputs.file_kind,
        import_mode: inputs.import_mode,
        body_names: inputs.body_names,
        link_suppressed: inputs.link_suppressed,
        link_broken: inputs.link_broken,
        unit_scale: inputs.unit_scale,
        stl_options: inputs.stl_options,
        step_options: inputs.step_options,
    })
    .map_err(Error)
}

/// Prepare a named Imported Model feature following format and mode browser naming rules.
pub fn prepare_import_definition(
    snapshot: &DocumentSnapshot,
    inputs: ImportInputs<'_>,
) -> Result<PreparedImport, Error> {
    let definition = build_import_definition(inputs)?;
    let preferred = match (definition.file_kind(), definition.import_mode()) {
        (ImportedFileKind::Step, ImportMode::ConvertModel) => definition
            .body_names()
            .first()
            .map(|body| format!("{body}:1"))
            .unwrap_or_else(|| "Solid1:1".into()),
        _ => definition.source_name().to_owned(),
    };
    let name = next_unique_name(snapshot, &preferred);
    Ok(PreparedImport { name, definition })
}

/// Replace an existing Imported Model feature definition in place.
pub fn replace_imported_model(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ImportedModelDefinition,
) -> Result<(), Error> {
    let reads = definition.source_reads();
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::ImportedModel(Box::new(definition)),
            reads,
        )
        .map_err(api_error)
        .map(|_| ())
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct WireImportArgs {
    source_path: String,
    #[serde(default)]
    source_name: String,
    #[serde(default)]
    file_kind: Option<ImportedFileKind>,
    #[serde(default)]
    import_mode: ImportMode,
    #[serde(default)]
    body_names: Vec<String>,
    #[serde(default)]
    stl_units: Option<StlImportUnits>,
    #[serde(default)]
    unit_scale: Option<f64>,
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input = serde_json::from_value::<WireImportArgs>(request.args.0.unwrap_or_default())
        .map_err(|cause| Error(format!("import arguments: {cause}")))?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("import: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("import: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
    let file_kind = input
        .file_kind
        .unwrap_or_else(|| infer_file_kind(&input.source_path));
    let template_scale = snapshot.parameters.units().working_scale();
    let unit_scale = input.unit_scale.unwrap_or_else(|| match file_kind {
        ImportedFileKind::Stl => input
            .stl_units
            .unwrap_or(StlImportUnits::TemplateUnits)
            .scale_to_cm(template_scale),
        ImportedFileKind::Step => 0.1,
    });
    let prepared = prepare_import_definition(
        snapshot,
        ImportInputs {
            source_path: &input.source_path,
            source_name: &input.source_name,
            file_kind,
            import_mode: input.import_mode,
            body_names: input.body_names,
            link_suppressed: false,
            link_broken: false,
            unit_scale,
            stl_options: (file_kind == ImportedFileKind::Stl).then(StlImportOptions::default),
            step_options: (file_kind == ImportedFileKind::Step).then(StepImportOptions::default),
        },
    )?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Import",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn infer_file_kind(path: &str) -> ImportedFileKind {
    match Path::new(path)
        .extension()
        .and_then(|ext| ext.to_str())
        .map(str::to_ascii_lowercase)
        .as_deref()
    {
        Some("stp" | "step") => ImportedFileKind::Step,
        _ => ImportedFileKind::Stl,
    }
}

fn default_source_name(path: &str) -> String {
    Path::new(path)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("ImportedModel")
        .to_string()
}

fn next_unique_name(snapshot: &DocumentSnapshot, base: &str) -> String {
    let existing = |candidate: &str| {
        snapshot
            .work_geometry
            .features()
            .iter()
            .any(|feature| feature.name() == candidate)
    };
    if !existing(base) {
        return base.to_owned();
    }
    let mut index = 2usize;
    loop {
        let candidate = format!("{base} ({index})");
        if !existing(&candidate) {
            return candidate;
        }
        index += 1;
    }
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
