// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::{error::Error, fmt};

/// A batch-wide refusal detected before any file is written.
#[derive(Debug)]
pub enum BatchSaveError {
    Operation(String),
    DuplicateDestination(PathBuf),
    Source(FileWorkspaceError),
}

impl fmt::Display for BatchSaveError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Operation(value) => write!(
                output,
                "batch operation {value:?}: expected save, saveAs, or saveCopyAs"
            ),
            Self::DuplicateDestination(path) => write!(
                output,
                "batch destination {path:?}: expected one write per resolved destination"
            ),
            Self::Source(error) => error.fmt(output),
        }
    }
}

impl Error for BatchSaveError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Source(error) => Some(error),
            Self::Operation(_) | Self::DuplicateDestination(_) => None,
        }
    }
}
