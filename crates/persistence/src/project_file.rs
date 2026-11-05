// SPDX-License-Identifier: GPL-2.0-only

//! Portable schema-1 project configuration, separate from document identity.
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    project::ProjectFileLookup,
};
use oblikovati_api::types::PROJECT_FILE_EXTENSION;
use serde::{Deserialize, Serialize};
use std::{
    io,
    path::{Path, PathBuf},
};

/// Recorded project locations; relative spelling is retained during save.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "camelCase")]
pub struct ProjectConfiguration {
    pub name: String,
    pub workspace: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub libraries: Vec<String>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub templates: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub design_data: String,
}

#[derive(Serialize)]
struct ProjectRecord {
    #[serde(rename = "schemaVersion")]
    schema_version: u64,
    #[serde(flatten)]
    configuration: ProjectConfiguration,
}

/// Resolved paths for consumers; persisted configuration remains unchanged.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct BoundProject {
    pub lookup: ProjectFileLookup,
    pub design_data: Option<PathBuf>,
}

impl ProjectConfiguration {
    /// Decode the Go schema-1 format without reading source files.
    /// Example: `let project = ProjectConfiguration::decode(bytes)?;`.
    pub fn decode(bytes: &[u8]) -> io::Result<Self> {
        let record = decode_record(bytes)?;
        if record.schema_version != 1 {
            return Err(invalid_record(format!(
                "schemaVersion {}: expected 1",
                record.schema_version
            )));
        }
        record.configuration.validate_paths()?;
        Ok(record.configuration)
    }

    /// Encode portable paths without substituting resolved absolute paths.
    /// Example: `let bytes = project.encode()?;`.
    pub fn encode(&self) -> io::Result<Vec<u8>> {
        self.validate_paths()?;
        let record = ProjectRecord {
            schema_version: 1,
            configuration: self.clone(),
        };
        serde_yaml_ng::to_string(&record)
            .map(String::into_bytes)
            .map_err(invalid_record)
    }

    /// Bind locations against an absolute project file, with no process-directory fallback.
    /// Example: `let bound = project.bind(Path::new("/work/gear.opj"))?;`.
    pub fn bind(&self, project_file: &Path) -> io::Result<BoundProject> {
        require_extension(project_file)?;
        self.validate_paths()?;
        let parent = project_parent(project_file)?;
        let libraries = self
            .libraries
            .iter()
            .map(|root| parent.join(root))
            .collect();
        let lookup = ProjectFileLookup::new(
            location(parent, &self.workspace),
            libraries,
            location(parent, &self.templates),
        )
        .map_err(invalid_record)?;
        Ok(BoundProject {
            lookup,
            design_data: location(parent, &self.design_data),
        })
    }

    fn validate_paths(&self) -> io::Result<()> {
        for path in [&self.workspace, &self.templates, &self.design_data]
            .into_iter()
            .chain(&self.libraries)
        {
            if path.contains('\0') {
                return Err(invalid_record(format!(
                    "project location {path:?}: expected a path without NUL"
                )));
            }
        }
        if self.libraries.iter().any(String::is_empty) {
            return Err(invalid_record(
                "library root \"\": expected a nonempty path",
            ));
        }
        Ok(())
    }
}

/// Read project YAML through injected file access; wrong extensions fail before I/O.
/// Example: `let project = read_project_file(path, &mut files)?;`.
pub fn read_project_file(
    path: &Path,
    files: &mut dyn PackageFileReader,
) -> io::Result<ProjectConfiguration> {
    require_extension(path)?;
    let bytes = files.read(path).map_err(|error| {
        io::Error::new(
            error.kind(),
            format!("project {path:?}: expected a readable file: {error}"),
        )
    })?;
    ProjectConfiguration::decode(&bytes)
        .map_err(|error| io::Error::new(error.kind(), format!("project {path:?}: {error}")))
}

/// Atomically write a schema-1 project with its recorded location spelling.
/// Example: `write_project_file(path, &project, &mut files)?;`.
pub fn write_project_file<F: AtomicFiles>(
    path: &Path,
    project: &ProjectConfiguration,
    files: &mut F,
) -> io::Result<()> {
    require_extension(path)?;
    replace_file(path, &project.encode()?, files).map_err(io::Error::other)
}

fn location(parent: &Path, recorded: &str) -> Option<PathBuf> {
    (!recorded.is_empty()).then(|| parent.join(recorded))
}

fn require_extension(path: &Path) -> io::Result<()> {
    if path
        .extension()
        .and_then(|extension| extension.to_str())
        .is_some_and(|extension| extension.eq_ignore_ascii_case(&PROJECT_FILE_EXTENSION[1..]))
    {
        return Ok(());
    }
    Err(io::Error::new(
        io::ErrorKind::InvalidInput,
        format!("project file {path:?}: expected extension {PROJECT_FILE_EXTENSION:?}"),
    ))
}

fn invalid_record(reason: impl std::fmt::Display) -> io::Error {
    io::Error::new(
        io::ErrorKind::InvalidData,
        format!("project configuration: expected schema-1 YAML: {reason}"),
    )
}

fn decode_record(bytes: &[u8]) -> io::Result<ProjectRecord> {
    let mut fields: serde_yaml_ng::Mapping =
        serde_yaml_ng::from_slice(bytes).map_err(invalid_record)?;
    let schema = fields
        .remove("schemaVersion")
        .ok_or_else(|| invalid_record("missing schemaVersion: expected 1"))?;
    let schema_version = serde_yaml_ng::from_value(schema).map_err(invalid_record)?;
    let configuration =
        serde_yaml_ng::from_value(serde_yaml_ng::Value::Mapping(fields)).map_err(invalid_record)?;
    Ok(ProjectRecord {
        schema_version,
        configuration,
    })
}

fn project_parent(project_file: &Path) -> io::Result<&Path> {
    project_file
        .parent()
        .filter(|_| project_file.is_absolute())
        .ok_or_else(|| {
            invalid_record(format!(
                "project path {project_file:?}: expected an absolute .opj file"
            ))
        })
}
