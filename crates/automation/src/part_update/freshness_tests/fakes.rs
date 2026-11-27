// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::{
    history::DocumentSnapshot,
    identity::{DocumentGuid, GuidSource, IdentityRegistration, IdentityTransitionError},
};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::DocumentFiles,
    load::PackageFileReader,
};
use oblikovati_session::{TransactionObserver, workspace::WorkspaceObserver};
use std::{
    io,
    path::{Path, PathBuf},
};

pub struct FakeFiles;
pub struct FakeStage;
fn unexpected() -> io::Error {
    io::Error::other("unexpected file access in coordinate-frame test")
}
impl PackageFileReader for FakeFiles {
    fn read(&mut self, _: &Path) -> io::Result<Vec<u8>> {
        Err(unexpected())
    }
}
impl AtomicFiles for FakeFiles {
    type Stage = FakeStage;
    fn stage(&mut self, _: &Path) -> io::Result<FakeStage> {
        Err(unexpected())
    }
}
impl StagedFile for FakeStage {
    fn write_contents(&mut self, _: &[u8]) -> io::Result<()> {
        Err(unexpected())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Err(unexpected())
    }
    fn commit(self, _: &Path) -> io::Result<()> {
        Err(unexpected())
    }
}
impl DocumentFiles for FakeFiles {
    fn save_path(&mut self, _: &Path) -> io::Result<PathBuf> {
        Err(unexpected())
    }
    fn canonical_path(&mut self, _: &Path) -> io::Result<PathBuf> {
        Err(unexpected())
    }
}
pub struct FakeGuids;
impl GuidSource for FakeGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        Ok("11111111-2222-4333-8444-555555555555".parse().unwrap())
    }
}
pub struct FakeObserver;
impl TransactionObserver for FakeObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}
impl WorkspaceObserver for FakeObserver {
    fn interest_client_present(&self, _: &str) -> bool {
        false
    }
    fn unavailable_interest_clients(&mut self, _: u64, _: &str, _: &[String]) {}
    fn created(&mut self, _: u64, _: IdentityRegistration) {}
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {}
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn saved(&mut self, _: u64) {}
    fn opened(&mut self, _: u64, _: IdentityRegistration) {}
    fn activated(&mut self, _: u64) {}
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {}
}
