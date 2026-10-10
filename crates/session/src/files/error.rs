// SPDX-License-Identifier: GPL-2.0-only

use crate::workspace::{WorkspaceError, WorkspaceSaveError};
use oblikovati_model::identity::{IdentityTransitionError, RecipeIdError};
use oblikovati_persistence::{
    load::PackageLoadError,
    package::{PackageCodecError, SnapshotPackageError},
};
use std::{error::Error, fmt, io, path::PathBuf};

/// A failed file operation; no new binding or successful save was adopted.
#[derive(Debug)]
pub enum FileWorkspaceError {
    ObserverFactory(String),
    AmbiguousSource(String),
    SourceValue {
        source: String,
        parameter: String,
        reason: String,
    },
    Project(oblikovati_persistence::project::ProjectLookupError),
    MissingTemplate(u32),
    TemplateKind {
        expected: u32,
        actual: u32,
    },
    CreationMetadata {
        field: &'static str,
        value: String,
    },
    TemplateDestination(PathBuf),
    DocumentType(u32),
    Package(PackageCodecError),
    Identity(IdentityTransitionError),
    TemplateIdentity(String),
    DestinationInUse {
        path: PathBuf,
        owner: u64,
    },
    MissingFile(u64),
    UnboundDocument(u64),
    RelativeCanonical(PathBuf),
    NonUtf8ResponsePath(PathBuf),
    Path {
        requested: PathBuf,
        cause: io::Error,
    },
    OpenVeto {
        path: PathBuf,
        reason: String,
    },
    Load(PackageLoadError),
    History(RecipeIdError),
    Workspace(WorkspaceError),
    Save(WorkspaceSaveError<SnapshotPackageError>),
}
impl fmt::Display for FileWorkspaceError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::SourceValue {
                source,
                parameter,
                reason,
            } => write!(
                output,
                "source {source:?} parameter {parameter:?}: {reason}"
            ),
            Self::AmbiguousSource(name) => write!(
                output,
                "source document {name:?}: expected one unique open document"
            ),
            Self::ObserverFactory(reason) => write!(
                output,
                "document observer {reason:?}: expected an available observer factory"
            ),
            Self::NonUtf8ResponsePath(path) => write!(
                output,
                "save destination {path:?}: expected UTF-8 for the canonical API response"
            ),
            Self::Project(error) => error.fmt(output),
            Self::MissingTemplate(kind) => write!(
                output,
                "project template for document type {kind}: expected an existing regular template file"
            ),
            Self::TemplateKind { expected, actual } => write!(
                output,
                "template document type {actual}: expected requested document type {expected}"
            ),
            Self::CreationMetadata { field, value } => write!(
                output,
                "new-file {field} {value:?}: expected a nonempty explicit file path"
            ),
            Self::TemplateDestination(path) => write!(
                output,
                "template destination {path:?}: expected a different file from the source template"
            ),
            Self::DocumentType(value) => write!(
                output,
                "document type {value}: expected Part, Assembly, Drawing, or Presentation"
            ),
            Self::Package(error) => error.fmt(output),
            Self::Identity(error) => error.fmt(output),
            Self::TemplateIdentity(reason) => write!(
                output,
                "template identity {reason}: expected a valid identity distinct from the new document"
            ),
            Self::DestinationInUse { path, owner } => write!(
                output,
                "destination {path:?} belongs to document {owner}: expected a destination not owned by an open document"
            ),
            Self::UnboundDocument(id) => write!(
                output,
                "document {id}: expected a storage path; use Save As first"
            ),
            Self::MissingFile(id) => {
                write!(output, "document {id}: expected an owned file binding")
            }
            Self::RelativeCanonical(path) => write!(
                output,
                "resolved path {path:?}: expected an absolute canonical path"
            ),
            Self::Path { requested, cause } => write!(
                output,
                "resolve {requested:?}: expected an existing canonical file path: {cause}"
            ),
            Self::OpenVeto { path, reason } => write!(
                output,
                "open {path:?} veto {reason:?}: expected policy approval"
            ),
            Self::Load(error) => error.fmt(output),
            Self::History(error) => error.fmt(output),
            Self::Workspace(error) => error.fmt(output),
            Self::Save(error) => error.fmt(output),
        }
    }
}
impl Error for FileWorkspaceError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Project(error) => Some(error),
            Self::Package(error) => Some(error),
            Self::Identity(error) => Some(error),
            Self::Path { cause, .. } => Some(cause),
            Self::Load(error) => Some(error),
            Self::History(error) => Some(error),
            Self::Workspace(error) => Some(error),
            Self::Save(error) => Some(error),
            Self::SourceValue { .. }
            | Self::AmbiguousSource(_)
            | Self::ObserverFactory(_)
            | Self::MissingTemplate(_)
            | Self::TemplateKind { .. }
            | Self::CreationMetadata { .. }
            | Self::TemplateDestination(_)
            | Self::DocumentType(_)
            | Self::TemplateIdentity(_)
            | Self::UnboundDocument(_)
            | Self::MissingFile(_)
            | Self::NonUtf8ResponsePath(_)
            | Self::RelativeCanonical(_)
            | Self::OpenVeto { .. }
            | Self::DestinationInUse { .. } => None,
        }
    }
}
