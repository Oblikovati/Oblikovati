// SPDX-License-Identifier: GPL-2.0-only

//! Standalone DataIO files; operations do not register a document in a session.

use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    package::DocumentPackage,
};
use std::{io, path::Path};

/// Read owned bytes from a named stream, including manifest-free Go containers.
/// Example: `let bytes = read_stream_file(path, "addin/state", &mut files)?;`.
pub fn read_stream_file(
    path: &Path,
    stream: &str,
    files: &mut dyn PackageFileReader,
) -> io::Result<Vec<u8>> {
    let encoded = files
        .read(path)
        .map_err(|error| read_failure(path, error))?;
    let package = decode_file(path, &encoded)?;
    package.streams().get(stream).cloned().ok_or_else(|| {
        io::Error::new(
            io::ErrorKind::NotFound,
            format!("stream {stream:?} in {path:?}: expected an existing named stream"),
        )
    })
}

/// Atomically replace one stream, preserving all other package sections.
/// Only a missing file creates a container; all other read errors stop the write.
/// Example: `write_stream_file(path, "addin/state", &[0, 255], &mut files)?;`.
pub fn write_stream_file<Files: PackageFileReader + AtomicFiles>(
    path: &Path,
    stream: &str,
    bytes: &[u8],
    files: &mut Files,
) -> io::Result<()> {
    let package = match files.read(path) {
        Ok(encoded) => decode_file(path, &encoded)?,
        Err(error) if error.kind() == io::ErrorKind::NotFound => DocumentPackage::empty_container(),
        Err(error) => return Err(read_failure(path, error)),
    };
    let encoded = package
        .with_stream(stream, bytes)
        .encode()
        .map_err(|error| {
            io::Error::other(format!(
                "package {path:?}: expected encodable streams: {error}"
            ))
        })?;
    replace_file(path, &encoded, files).map_err(io::Error::other)
}

fn decode_file(path: &Path, encoded: &[u8]) -> io::Result<DocumentPackage> {
    DocumentPackage::decode_container(encoded).map_err(|error| {
        io::Error::new(
            io::ErrorKind::InvalidData,
            format!("package {path:?}: expected a supported YAML container: {error}"),
        )
    })
}

fn read_failure(path: &Path, error: io::Error) -> io::Error {
    io::Error::new(
        error.kind(),
        format!("package {path:?}: expected a readable file: {error}"),
    )
}
