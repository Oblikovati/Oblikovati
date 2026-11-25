// SPDX-License-Identifier: GPL-2.0-only

use std::{
    error::Error,
    fmt, io,
    path::{Path, PathBuf},
};

mod native;
pub use native::{NativeAtomicFiles, NativeStagedFile};

/// A sibling staging file; dropping it before commit must remove the temporary.
pub trait StagedFile {
    /// Write all bytes before synchronization. Example: `stage.write_contents(bytes)?`.
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()>;
    /// Synchronize staged bytes before replacement. Example: `stage.synchronize()?`.
    fn synchronize(&mut self) -> io::Result<()>;
    /// Replace the destination atomically. Example: `stage.commit(path)?`.
    /// An error must leave the old destination intact.
    fn commit(self, destination: &Path) -> io::Result<()>;
    /// Publish at an unused destination, for example `stage.commit_new(path)?`.
    fn commit_new(self, destination: &Path) -> io::Result<()>
    where
        Self: Sized,
    {
        Err(io::Error::new(
            io::ErrorKind::Unsupported,
            format!(
                "destination {destination:?}: expected a filesystem that supports atomic creation"
            ),
        ))
    }
}

/// Project-owned filesystem boundary for same-directory atomic replacement.
pub trait AtomicFiles {
    type Stage: StagedFile;
    /// Create a temporary beside the destination. Example: `files.stage(path)?`.
    fn stage(&mut self, destination: &Path) -> io::Result<Self::Stage>;
}

/// Failed phase and destination, with the underlying I/O error retained.
#[derive(Debug)]
pub struct AtomicWriteError {
    pub destination: PathBuf,
    pub phase: &'static str,
    cause: io::Error,
}

/// Replace a file only after its complete new contents have been synchronized.
/// There is no fallible step after commit, so memory can safely adopt a save.
/// Directory metadata durability across power loss is outside this operation.
///
/// ```no_run
/// use oblikovati_persistence::atomic::{replace_file, NativeAtomicFiles};
/// replace_file(std::path::Path::new("part.obk"), b"encoded snapshot", &mut NativeAtomicFiles)?;
/// # Ok::<(), oblikovati_persistence::atomic::AtomicWriteError>(())
/// ```
pub fn replace_file<Files: AtomicFiles>(
    destination: &Path,
    contents: &[u8],
    files: &mut Files,
) -> Result<(), AtomicWriteError> {
    let staged = prepare_file(destination, contents, files)?;
    staged
        .commit(destination)
        .map_err(|cause| AtomicWriteError {
            destination: destination.to_owned(),
            phase: "commit",
            cause,
        })
}

/// Publish complete bytes at a new path, for example `create_file(path, bytes, &mut NativeAtomicFiles)?`.
pub(crate) fn create_file<Files: AtomicFiles>(
    destination: &Path,
    contents: &[u8],
    files: &mut Files,
) -> io::Result<()> {
    let staged = prepare_file(destination, contents, files).map_err(io::Error::other)?;
    staged.commit_new(destination)
}

fn prepare_file<Files: AtomicFiles>(
    destination: &Path,
    contents: &[u8],
    files: &mut Files,
) -> Result<Files::Stage, AtomicWriteError> {
    let failure = |phase, cause| AtomicWriteError {
        destination: destination.to_owned(),
        phase,
        cause,
    };
    let mut staged = files
        .stage(destination)
        .map_err(|error| failure("stage", error))?;
    staged
        .write_contents(contents)
        .map_err(|error| failure("write", error))?;
    staged
        .synchronize()
        .map_err(|error| failure("sync", error))?;
    Ok(staged)
}

impl fmt::Display for AtomicWriteError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "{} {:?}: expected a complete atomic file replacement: {}",
            self.phase, self.destination, self.cause
        )
    }
}

impl Error for AtomicWriteError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(&self.cause)
    }
}
