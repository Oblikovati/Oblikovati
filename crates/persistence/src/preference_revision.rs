// SPDX-License-Identifier: GPL-2.0-only
//! Detect cached preference owners whose input changed before a write (#4008).
use crate::load::PackageFileReader;
use std::{io, path::Path};

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) enum PreferenceRevision {
    Absent,
    Contents(Vec<u8>),
}

impl PreferenceRevision {
    pub(crate) fn read(path: &Path, files: &mut impl PackageFileReader) -> io::Result<Self> {
        match files.read(path) {
            Ok(bytes) => Ok(Self::Contents(bytes)),
            Err(error) if error.kind() == io::ErrorKind::NotFound => Ok(Self::Absent),
            Err(error) => Err(error),
        }
    }

    pub(crate) fn contents(&self) -> &[u8] {
        match self {
            Self::Absent => &[],
            Self::Contents(bytes) => bytes,
        }
    }

    pub(crate) fn require_current(
        &self,
        path: &Path,
        files: &mut impl PackageFileReader,
    ) -> io::Result<()> {
        if Self::read(path, files)? != *self {
            return Err(io::Error::other(format!(
                "preference file {path:?} changed since load: expected reload before saving"
            )));
        }
        Ok(())
    }
}
