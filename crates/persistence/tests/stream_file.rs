// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    file_access::NativeDocumentFiles,
    load::PackageFileReader,
    package::DocumentPackage,
    stream_file::{read_stream_file, write_stream_file},
};
use std::{cell::RefCell, io, path::Path, rc::Rc};

#[derive(Default)]
struct StreamDisk {
    contents: Option<Vec<u8>>,
    read_error: Option<io::ErrorKind>,
    refuse_commit: bool,
    stages: usize,
}
struct StreamFiles(Rc<RefCell<StreamDisk>>);
struct StreamStage {
    disk: Rc<RefCell<StreamDisk>>,
    pending: Vec<u8>,
    synchronized: bool,
}
impl PackageFileReader for StreamFiles {
    fn read(&mut self, _: &Path) -> io::Result<Vec<u8>> {
        let disk = self.0.borrow();
        if let Some(kind) = disk.read_error {
            return Err(io::Error::new(kind, "injected read failure"));
        }
        disk.contents.clone().ok_or(io::ErrorKind::NotFound.into())
    }
}
impl AtomicFiles for StreamFiles {
    type Stage = StreamStage;
    fn stage(&mut self, _: &Path) -> io::Result<StreamStage> {
        self.0.borrow_mut().stages += 1;
        Ok(StreamStage {
            disk: self.0.clone(),
            pending: Vec::new(),
            synchronized: false,
        })
    }
}
impl StagedFile for StreamStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> {
        self.pending = bytes.to_vec();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        self.synchronized = true;
        Ok(())
    }
    fn commit(self, _: &Path) -> io::Result<()> {
        assert!(self.synchronized);
        if self.disk.borrow().refuse_commit {
            return Err(io::Error::other("injected commit failure"));
        }
        self.disk.borrow_mut().contents = Some(self.pending);
        Ok(())
    }
}

#[test]
fn missing_file_creates_manifest_free_container_and_preserves_other_streams() {
    let disk = Rc::new(RefCell::new(StreamDisk::default()));
    let mut files = StreamFiles(disk.clone());
    let path = Path::new("standalone.yml");
    write_stream_file(path, "../opaque", &[0, 255], &mut files).unwrap();
    write_stream_file(path, "", &[], &mut files).unwrap();
    assert_eq!(
        read_stream_file(path, "../opaque", &mut files).unwrap(),
        [0, 255]
    );
    assert!(read_stream_file(path, "", &mut files).unwrap().is_empty());
    let encoded = disk.borrow().contents.clone().unwrap();
    let root: serde_yaml_ng::Mapping = serde_yaml_ng::from_slice(&encoded).unwrap();
    assert_eq!(root.len(), 1);
    assert!(root.contains_key("data"));
    assert!(DocumentPackage::decode(&encoded).is_err());
    let error = read_stream_file(path, "absent", &mut files).unwrap_err();
    assert_eq!(error.kind(), io::ErrorKind::NotFound);
    assert!(error.to_string().contains("absent"));
    assert!(error.to_string().contains("standalone.yml"));
}

#[test]
fn document_sections_survive_stream_replacement_and_commit_failure() {
    let original = b"schemaVersion: 3\ndocumentType: 1\nmodel: {future: [1, 2]}\nattributes: [0, 0]\ndata: {keep: Zg==, replace: AA==}\n";
    let disk = Rc::new(RefCell::new(StreamDisk {
        contents: Some(original.to_vec()),
        refuse_commit: true,
        ..Default::default()
    }));
    let mut files = StreamFiles(disk.clone());
    let path = Path::new("part.opd");
    assert!(write_stream_file(path, "replace", b"new", &mut files).is_err());
    assert_eq!(disk.borrow().contents.as_deref().unwrap(), original);
    disk.borrow_mut().refuse_commit = false;
    write_stream_file(path, "replace", b"new", &mut files).unwrap();
    let before = DocumentPackage::decode(original).unwrap();
    let after = DocumentPackage::decode(disk.borrow().contents.as_ref().unwrap()).unwrap();
    assert_eq!(after.streams()["replace"], b"new");
    assert_eq!(after.streams()["keep"], b"f");
    for section in ["model", "attributes"] {
        assert_eq!(
            before.section_yaml(section).unwrap(),
            after.section_yaml(section).unwrap()
        );
    }
}

#[test]
fn unreadable_or_invalid_existing_files_are_never_replaced() {
    let path = Path::new("bad.yml");
    let disk = Rc::new(RefCell::new(StreamDisk::default()));
    let mut files = StreamFiles(disk.clone());
    for kind in [io::ErrorKind::PermissionDenied, io::ErrorKind::Interrupted] {
        disk.borrow_mut().read_error = Some(kind);
        assert_eq!(
            write_stream_file(path, "x", &[], &mut files)
                .unwrap_err()
                .kind(),
            kind
        );
        assert_eq!(
            read_stream_file(path, "x", &mut files).unwrap_err().kind(),
            kind
        );
    }
    disk.borrow_mut().read_error = None;
    for input in [
        "[",
        "unknown: 1",
        "schemaVersion: 99\ndocumentType: 1",
        "displayName: incomplete",
        "data: {x: '!'}",
        "schemaVersion: wrong",
    ] {
        disk.borrow_mut().contents = Some(input.as_bytes().to_vec());
        let error = write_stream_file(path, "x", &[], &mut files).unwrap_err();
        assert_eq!(error.kind(), io::ErrorKind::InvalidData);
        assert!(error.to_string().contains("bad.yml"));
        assert_eq!(disk.borrow().contents.as_deref().unwrap(), input.as_bytes());
    }
    assert_eq!(disk.borrow().stages, 0);
}

#[test]
fn linux_native_round_trip_reads_go_manifest_free_shape() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("streams.yml");
    // Go yamlcodec omits zero manifest fields and base64-encodes binary streams.
    std::fs::write(&path, include_bytes!("fixtures/go-stream-container.yml")).unwrap();
    let mut files = NativeDocumentFiles;
    assert_eq!(
        read_stream_file(&path, "binary", &mut files).unwrap(),
        [0, 255]
    );
    write_stream_file(&path, "text", "µm".as_bytes(), &mut files).unwrap();
    assert_eq!(
        read_stream_file(&path, "text", &mut files).unwrap(),
        "µm".as_bytes()
    );
    assert!(
        read_stream_file(&path, "empty", &mut files)
            .unwrap()
            .is_empty()
    );
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
}
