// SPDX-License-Identifier: GPL-2.0-only

use crate::workspace::{WorkspaceError, WorkspaceSaveError};
use oblikovati_model::identity::RecipeIdError;
use oblikovati_persistence::{load::PackageLoadError, package::SnapshotPackageError};
use std::{error::Error, fmt, io, path::PathBuf};

/// A failed file operation; no new binding or successful save was adopted.
#[derive(Debug)]
pub enum FileWorkspaceError {
    DestinationInUse {
        path: PathBuf,
        owner: u64,
    },
    MissingFile(u64),
    RelativeCanonical(PathBuf),
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
            Self::DestinationInUse { path, owner } => write!(
                output,
                "destination {path:?} belongs to document {owner}: expected an unowned path or this document\'s current path"
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
            Self::Path { cause, .. } => Some(cause),
            Self::Load(error) => Some(error),
            Self::History(error) => Some(error),
            Self::Workspace(error) => Some(error),
            Self::Save(error) => Some(error),
            Self::MissingFile(_)
            | Self::RelativeCanonical(_)
            | Self::OpenVeto { .. }
            | Self::DestinationInUse { .. } => None,
        }
    }
}
