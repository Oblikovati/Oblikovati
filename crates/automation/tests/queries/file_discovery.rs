// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::FileQueryHost;
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::DocumentFiles,
    load::PackageFileReader,
    package::DocumentPackage,
};
use oblikovati_session::files::FileWorkspace;
use std::{cell::RefCell, collections::BTreeMap, io, path::PathBuf, rc::Rc};

type MemoryPackages = Rc<RefCell<BTreeMap<PathBuf, Vec<u8>>>>;
#[derive(Default)]
struct DiscoveryFiles {
    refuse_write: bool,
    save_aliases: RefCell<BTreeMap<PathBuf, PathBuf>>,
    packages: MemoryPackages,
}
struct DiscoveryStage {
    packages: MemoryPackages,
    contents: Vec<u8>,
}
impl PackageFileReader for DiscoveryFiles {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        self.packages
            .borrow()
            .get(path)
            .cloned()
            .ok_or_else(|| io::Error::other(format!("missing fixture {path:?}")))
    }
}
impl DocumentFiles for DiscoveryFiles {
    fn save_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        Ok(self
            .save_aliases
            .borrow()
            .get(requested)
            .cloned()
            .unwrap_or_else(|| requested.into()))
    }
    fn canonical_path(&mut self, requested: &Path) -> io::Result<PathBuf> {
        Ok(requested.into())
    }
}
impl AtomicFiles for DiscoveryFiles {
    type Stage = DiscoveryStage;
    fn stage(&mut self, _: &Path) -> io::Result<Self::Stage> {
        if self.refuse_write {
            return Err(io::Error::other("write refused"));
        }
        Ok(DiscoveryStage {
            packages: Rc::clone(&self.packages),
            contents: vec![],
        })
    }
}
impl StagedFile for DiscoveryStage {
    fn write_contents(&mut self, contents: &[u8]) -> io::Result<()> {
        self.contents = contents.into();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn commit(self, path: &Path) -> io::Result<()> {
        self.packages
            .borrow_mut()
            .insert(path.into(), self.contents);
        Ok(())
    }
}
type DiscoveryWorkspace = FileWorkspace<QueryObserver, QueryObserver, DiscoveryFiles>;

#[test]
fn file_query_host_discovers_types_names_subtypes_and_creation_order() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    assert_eq!(
        Client::new(FileQueryHost::new(&files))
            .documents()
            .list()
            .unwrap()
            .documents,
        Some(vec![])
    );
    for (kind, name) in [
        (DOCUMENT_PART, "part"),
        (DOCUMENT_ASSEMBLY, "assembly"),
        (DOCUMENT_DRAWING, "drawing"),
        (DOCUMENT_PRESENTATION, "presentation"),
    ] {
        let package = DocumentPackage::decode(format!("schemaVersion: 2\ndocumentType: {}\ndisplayName: 'Custom {name}'\nsubType: flavor\n", kind.0).as_bytes()).unwrap();
        let registered = files
            .create(
                &PathBuf::from(format!("/fixtures/{name}.obk")),
                package,
                QueryObserver::default(),
                "test",
                &mut guids,
            )
            .unwrap();
        assert_eq!(
            files.workspace().active_document(),
            Some(registered.session_id)
        );
    }
    let before = files.workspace().observer().notifications;
    let documents = Client::new(FileQueryHost::new(&files))
        .documents()
        .list()
        .unwrap()
        .documents
        .unwrap();
    assert_eq!(
        documents
            .iter()
            .map(|entry| entry.type_.as_str())
            .collect::<Vec<_>>(),
        ["part", "assembly", "drawing", "presentation"]
    );
    for (index, document) in documents.iter().enumerate() {
        assert_eq!(document.name, format!("Custom {}", document.type_));
        assert_eq!(document.sub_type, "flavor");
        assert!(document.dirty && document.visible);
        assert_eq!(document.active, index == 3);
    }
    assert_eq!(files.workspace().observer().notifications, before);
    assert!(files.document_info(999).is_err());
}

#[test]
fn discovery_tracks_save_undo_focus_and_reopened_names() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: [{name: width, kind: user, expression: '1 cm'}]\n").unwrap();
    let path = Path::new("/fixtures/part.opd");
    let id = files
        .create(path, package, QueryObserver::default(), "test", &mut guids)
        .unwrap()
        .session_id;
    files.save(id, "test", &mut guids).unwrap();
    assert!(!files.document_info(id).unwrap().dirty);
    files
        .commands(id)
        .unwrap()
        .run("width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    assert!(files.document_info(id).unwrap().dirty);
    files.commands(id).unwrap().undo().unwrap();
    assert!(!files.document_info(id).unwrap().dirty);
    let client = Client::new(FileQueryHost::new(&files));
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "width".into()
            })
            .unwrap()
            .expression,
        "1 cm"
    );
    assert_eq!(
        client
            .transactions()
            .history(&TransactionHistoryArgs::default())
            .unwrap()
            .name,
        "part"
    );
    assert!(client.transactions().undo().is_err());
    files.close_discarding_changes(id).unwrap();
    let reopened = files
        .open(
            path,
            QueryObserver::default(),
            "test",
            &mut guids,
            OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    let info = files.document_info(reopened).unwrap();
    assert_eq!(info.name, "part");
    assert!(!info.dirty && !info.active && info.visible);
    assert_eq!(
        Client::new(FileQueryHost::new(&files))
            .documents()
            .list()
            .unwrap()
            .documents
            .unwrap(),
        [info]
    );
}

#[path = "file_commands.rs"]
mod commands;
