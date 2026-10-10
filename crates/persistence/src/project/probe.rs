// SPDX-License-Identifier: GPL-2.0-only

use std::{
    error::Error,
    fmt, io,
    path::{Path, PathBuf},
};

/// Project lookup's only filesystem dependency; it does not read or open document recipes.
pub trait ProjectFileProbe {
    /// Distinguish a regular file from absence or a directory; example: `probe.is_regular_file(path)?`.
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool>;
}

/// Native metadata lookup. Symlinks are followed; missing components are a normal miss.
pub struct NativeProjectFileProbe;

impl ProjectFileProbe for NativeProjectFileProbe {
    fn is_regular_file(&mut self, path: &Path) -> io::Result<bool> {
        match std::fs::metadata(path) {
            Ok(metadata) => Ok(metadata.is_file()),
            Err(error)
                if matches!(
                    error.kind(),
                    io::ErrorKind::NotFound | io::ErrorKind::NotADirectory
                ) =>
            {
                Ok(false)
            }
            Err(error) => Err(error),
        }
    }
}

/// Invalid project configuration or an inaccessible candidate; neither is a missing reference.
#[derive(Debug)]
pub enum ProjectLookupError {
    RelativeRoot(PathBuf),
    Probe { path: PathBuf, cause: io::Error },
}

impl fmt::Display for ProjectLookupError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::RelativeRoot(path) => write!(
                output,
                "project root {path:?}: expected an absolute directory path"
            ),
            Self::Probe { path, cause } => write!(
                output,
                "project file {path:?}: expected readable regular-file metadata: {cause}"
            ),
        }
    }
}

impl Error for ProjectLookupError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::RelativeRoot(_) => None,
            Self::Probe { cause, .. } => Some(cause),
        }
    }
}
