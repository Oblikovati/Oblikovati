// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::{DocumentSnapshot, DocumentSnapshotTarget};
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    package::ParameterPackageTarget,
};
use std::{io, path::Path};

#[derive(Default)]
struct RefuseScalarFiles {
    stage_calls: usize,
}
struct RefuseScalarStage;
impl AtomicFiles for RefuseScalarFiles {
    type Stage = RefuseScalarStage;
    fn stage(&mut self, _destination: &Path) -> io::Result<Self::Stage> {
        self.stage_calls += 1;
        Err(io::Error::other("unexpected storage staging"))
    }
}
impl StagedFile for RefuseScalarStage {
    fn write_contents(&mut self, _contents: &[u8]) -> io::Result<()> {
        unreachable!("staging is refused")
    }
    fn synchronize(&mut self) -> io::Result<()> {
        unreachable!("staging is refused")
    }
    fn commit(self, _destination: &Path) -> io::Result<()> {
        unreachable!("staging is refused")
    }
}
fn snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: sequence(),
        parameters: parameters(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        properties: Default::default(),
    }
}
#[test]
fn invalid_owned_scalar_save_fails_before_any_storage_change() {
    let mut snapshot = snapshot();
    snapshot.parameters.remove_for_owner(9).unwrap();
    let mut package = stored();
    let before = package.clone();
    let mut files = RefuseScalarFiles::default();
    let mut target = ParameterPackageTarget::new(&mut package, Path::new("owned.opd"), &mut files);
    let result = target.prepare(&snapshot, 0, 9, Default::default(), 3);
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("existing Model parameter")
    );
    drop(target);
    assert_eq!(files.stage_calls, 0);
    assert_eq!(package, before);
}
