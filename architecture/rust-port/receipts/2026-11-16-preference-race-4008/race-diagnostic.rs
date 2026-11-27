// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::marking_menu::{MarkingMenuPreferenceAccess, MarkingMenus};
use oblikovati_persistence::{atomic::{AtomicFiles, StagedFile}, load::PackageFileReader, marking_menu_preferences::MarkingMenuPreferenceFile};
use std::{cell::RefCell, io, path::Path, rc::Rc};

struct RacingFiles(Rc<RefCell<Vec<u8>>>);
struct RacingStage { disk: Rc<RefCell<Vec<u8>>>, encoded: Vec<u8> }
impl PackageFileReader for RacingFiles {
    fn read(&mut self, _: &Path) -> io::Result<Vec<u8>> { Ok(self.0.borrow().clone()) }
}
impl AtomicFiles for RacingFiles {
    type Stage = RacingStage;
    fn stage(&mut self, _: &Path) -> io::Result<RacingStage> {
        *self.0.borrow_mut() = b"menus: []\nexternal: retained\n".to_vec();
        Ok(RacingStage { disk: self.0.clone(), encoded: Vec::new() })
    }
}
impl StagedFile for RacingStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> { self.encoded.extend_from_slice(bytes); Ok(()) }
    fn synchronize(&mut self) -> io::Result<()> { Ok(()) }
    fn commit(self, _: &Path) -> io::Result<()> { *self.disk.borrow_mut() = self.encoded; Ok(()) }
}
#[test]
fn change_between_check_and_staging_must_not_be_overwritten() {
    let disk = Rc::new(RefCell::new(b"menus: []\n".to_vec()));
    let mut file = MarkingMenuPreferenceFile::load(Path::new("menu.yaml"), RacingFiles(disk.clone())).unwrap();
    assert!(file.save(&MarkingMenus::default()).is_err());
    assert_eq!(*disk.borrow(), b"menus: []\nexternal: retained\n");
}

struct CommitRaceFiles(Rc<RefCell<Vec<u8>>>);
struct CommitRaceStage { disk: Rc<RefCell<Vec<u8>>>, encoded: Vec<u8> }
impl PackageFileReader for CommitRaceFiles {
    fn read(&mut self, _: &Path) -> io::Result<Vec<u8>> { Ok(self.0.borrow().clone()) }
}
impl AtomicFiles for CommitRaceFiles {
    type Stage = CommitRaceStage;
    fn stage(&mut self, _: &Path) -> io::Result<CommitRaceStage> {
        Ok(CommitRaceStage { disk: self.0.clone(), encoded: Vec::new() })
    }
}
impl StagedFile for CommitRaceStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> { self.encoded.extend_from_slice(bytes); Ok(()) }
    fn synchronize(&mut self) -> io::Result<()> { *self.disk.borrow_mut() = b"menus: []\nexternal: retained\n".to_vec(); Ok(()) }
    fn commit(self, _: &Path) -> io::Result<()> { *self.disk.borrow_mut() = self.encoded; Ok(()) }
}
#[test]
fn change_between_stage_and_commit_must_not_be_overwritten() {
    let disk = Rc::new(RefCell::new(b"menus: []\n".to_vec()));
    let mut file = MarkingMenuPreferenceFile::load(Path::new("menu.yaml"), CommitRaceFiles(disk.clone())).unwrap();
    assert!(file.save(&MarkingMenus::default()).is_err());
    assert_eq!(*disk.borrow(), b"menus: []\nexternal: retained\n");
}
