// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[derive(Default)]
pub(super) struct RecordedStore {
    pub files: BTreeMap<PathBuf, Vec<u8>>,
    pub events: Vec<&'static str>,
}
#[derive(Default)]
pub(super) struct RecordedFiles {
    pub store: Rc<RefCell<RecordedStore>>,
    failure: Option<&'static str>,
}
impl RecordedFiles {
    pub fn with_failure(phase: &'static str) -> Self {
        Self {
            failure: Some(phase),
            ..Default::default()
        }
    }
}
fn observe(
    store: &Rc<RefCell<RecordedStore>>,
    failure: Option<&'static str>,
    phase: &'static str,
) -> io::Result<()> {
    store.borrow_mut().events.push(phase);
    if failure == Some(phase) {
        return Err(io::Error::other(format!("{phase} refused")));
    }
    Ok(())
}
impl PackageFileReader for RecordedFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        observe(&self.store, self.failure, "read")?;
        self.store.borrow().files.get(path).cloned().ok_or_else(|| {
            io::Error::new(
                io::ErrorKind::NotFound,
                format!("file {path:?}: expected saved bytes"),
            )
        })
    }
}
impl DocumentFiles for RecordedFiles {
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        observe(&self.store, self.failure, "path")?;
        Ok(requested.to_path_buf())
    }
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        observe(&self.store, self.failure, "canonical")?;
        Ok(requested.to_path_buf())
    }
}
pub(super) struct RecordedStage {
    store: Rc<RefCell<RecordedStore>>,
    failure: Option<&'static str>,
    contents: Vec<u8>,
}
impl AtomicFiles for RecordedFiles {
    type Stage = RecordedStage;
    fn stage(&mut self, _: &Path) -> io::Result<Self::Stage> {
        observe(&self.store, self.failure, "stage")?;
        Ok(RecordedStage {
            store: self.store.clone(),
            failure: self.failure,
            contents: Vec::new(),
        })
    }
}
impl StagedFile for RecordedStage {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        observe(&self.store, self.failure, "write")?;
        self.contents = contents.into();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        observe(&self.store, self.failure, "sync")
    }
    fn commit(self, destination: &Path) -> io::Result<()> {
        observe(&self.store, self.failure, "commit")?;
        self.store
            .borrow_mut()
            .files
            .insert(destination.into(), self.contents);
        Ok(())
    }
}
#[derive(Default)]
pub(super) struct SequenceGuids {
    pub issued: u32,
    pub fail_at: Option<u32>,
}
impl GuidSource for SequenceGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.issued += 1;
        if self.fail_at == Some(self.issued) {
            return Err(IdentityTransitionError::Randomness(
                "GUID source refused".into(),
            ));
        }
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.issued)
            .parse()
            .unwrap())
    }
}
