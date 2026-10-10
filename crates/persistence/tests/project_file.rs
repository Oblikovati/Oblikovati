// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::{
    atomic::{AtomicFiles, StagedFile},
    load::PackageFileReader,
    project_file::{ProjectConfiguration, read_project_file, write_project_file},
};
use std::{
    cell::RefCell,
    io,
    path::{Path, PathBuf},
    rc::Rc,
};

#[derive(Default)]
struct ProjectDisk {
    bytes: Vec<u8>,
    reads: usize,
    stages: usize,
    fail_read: bool,
    fail_commit: bool,
}
struct ProjectFiles(Rc<RefCell<ProjectDisk>>);
struct ProjectStage {
    disk: Rc<RefCell<ProjectDisk>>,
    bytes: Vec<u8>,
}
impl PackageFileReader for ProjectFiles {
    fn read(&mut self, _: &Path) -> io::Result<Vec<u8>> {
        let mut disk = self.0.borrow_mut();
        disk.reads += 1;
        if disk.fail_read {
            return Err(io::Error::new(
                io::ErrorKind::PermissionDenied,
                "project denied",
            ));
        }
        Ok(disk.bytes.clone())
    }
}
impl AtomicFiles for ProjectFiles {
    type Stage = ProjectStage;
    fn stage(&mut self, _: &Path) -> io::Result<Self::Stage> {
        self.0.borrow_mut().stages += 1;
        Ok(ProjectStage {
            disk: self.0.clone(),
            bytes: vec![],
        })
    }
}
impl StagedFile for ProjectStage {
    fn write_contents(&mut self, bytes: &[u8]) -> io::Result<()> {
        self.bytes = bytes.to_vec();
        Ok(())
    }
    fn synchronize(&mut self) -> io::Result<()> {
        Ok(())
    }
    fn commit(self, _: &Path) -> io::Result<()> {
        if self.disk.borrow().fail_commit {
            return Err(io::Error::other("project commit denied"));
        }
        self.disk.borrow_mut().bytes = self.bytes;
        Ok(())
    }
}
fn sample() -> ProjectConfiguration {
    ProjectConfiguration {
        name: "Acme Gearbox".into(),
        workspace: "workspace".into(),
        libraries: vec!["../fasteners".into(), "/opt/bearings".into()],
        templates: "templates".into(),
        design_data: "design".into(),
    }
}

#[test]
fn go_project_round_trip_keeps_recorded_paths_and_order() {
    let project = ProjectConfiguration::decode(include_bytes!("fixtures/go-project.opj")).unwrap();
    assert_eq!(project, sample());
    assert_eq!(
        ProjectConfiguration::decode(&project.encode().unwrap()).unwrap(),
        project
    );
    let empty = ProjectConfiguration::decode(b"schemaVersion: 1\n").unwrap();
    assert_eq!(empty, ProjectConfiguration::default());
    assert_eq!(
        ProjectConfiguration::decode(&empty.encode().unwrap()).unwrap(),
        empty
    );
}

#[test]
fn bound_paths_use_project_parent_and_leave_configuration_unchanged() {
    let project = sample();
    let bound = project.bind(Path::new("/work/acme/gear.OPJ")).unwrap();
    assert_eq!(
        bound.lookup.search_paths().collect::<Vec<_>>(),
        [
            Path::new("/work/acme/workspace"),
            Path::new("/work/fasteners"),
            Path::new("/opt/bearings")
        ]
    );
    assert_eq!(bound.design_data, Some(PathBuf::from("/work/acme/design")));
    assert_eq!(project, sample());
    assert!(
        project
            .bind(Path::new("gear.opj"))
            .unwrap_err()
            .to_string()
            .contains("absolute")
    );
    let empty = ProjectConfiguration::default()
        .bind(Path::new("/work/empty.opj"))
        .unwrap();
    assert_eq!(empty.lookup.search_paths().count(), 0);
    assert!(empty.design_data.is_none());
}

#[test]
fn malformed_schema_fields_and_locations_are_refused() {
    for bytes in [
        "schemaVersion: 2",
        "name: missing",
        "schemaVersion: 1\nfuture: true",
        "schemaVersion: 1\nworkspace: []",
        "schemaVersion: 1\nlibraries: ['']",
        "schemaVersion: 1\nschemaVersion: 1",
        "schemaVersion: 1\nname: a\nname: b",
        "schemaVersion: 1\nworkspace: \"bad\\0path\"",
    ] {
        assert!(
            ProjectConfiguration::decode(bytes.as_bytes()).is_err(),
            "accepted {bytes:?}"
        );
    }
    let mut invalid = sample();
    invalid.libraries.push(String::new());
    assert!(invalid.encode().is_err());
    assert!(invalid.bind(Path::new("/work/gear.opj")).is_err());
}

#[test]
fn project_file_io_checks_extensions_and_preserves_old_file_on_failed_commit() {
    let disk = Rc::new(RefCell::new(ProjectDisk::default()));
    let mut files = ProjectFiles(disk.clone());
    let wrong = Path::new("gear.opd");
    assert!(write_project_file(wrong, &sample(), &mut files).is_err());
    assert!(read_project_file(wrong, &mut files).is_err());
    assert_eq!((disk.borrow().reads, disk.borrow().stages), (0, 0));
    let path = Path::new("gear.OPJ");
    write_project_file(path, &sample(), &mut files).unwrap();
    assert_eq!(read_project_file(path, &mut files).unwrap(), sample());
    let saved = disk.borrow().bytes.clone();
    disk.borrow_mut().fail_commit = true;
    assert!(
        write_project_file(path, &ProjectConfiguration::default(), &mut files)
            .unwrap_err()
            .to_string()
            .contains("commit")
    );
    assert_eq!(disk.borrow().bytes, saved);
    disk.borrow_mut().fail_read = true;
    assert_eq!(
        read_project_file(path, &mut files).unwrap_err().kind(),
        io::ErrorKind::PermissionDenied
    );
    disk.borrow_mut().fail_read = false;
    disk.borrow_mut().bytes = b"schemaVersion: 999".to_vec();
    assert!(
        read_project_file(path, &mut files)
            .unwrap_err()
            .to_string()
            .contains("gear.OPJ")
    );
}

#[test]
fn loaded_project_drives_native_attachment_and_template_lookup() {
    use oblikovati_api::types::DOCUMENT_PART;
    use oblikovati_model::attachment::ExternalAttachmentFiles;
    use oblikovati_persistence::{
        attachment_files::NativeAttachmentFiles,
        file_access::NativeDocumentFiles,
        project::{NativeProjectFileProbe, ProjectAttachmentFiles},
    };
    let directory = tempfile::tempdir().unwrap();
    std::fs::create_dir(directory.path().join("library")).unwrap();
    std::fs::create_dir(directory.path().join("templates")).unwrap();
    std::fs::write(directory.path().join("library/source.bin"), [0, 255]).unwrap();
    let template = directory.path().join("templates/part.opd");
    std::fs::write(&template, b"template").unwrap();
    let configuration = ProjectConfiguration {
        workspace: "workspace".into(),
        libraries: vec!["library".into()],
        templates: "templates".into(),
        ..Default::default()
    };
    let path = directory.path().join("gear.opj");
    write_project_file(&path, &configuration, &mut NativeDocumentFiles).unwrap();
    let loaded = read_project_file(&path, &mut NativeDocumentFiles).unwrap();
    let bound = loaded.bind(&path).unwrap();
    assert_eq!(
        bound
            .lookup
            .template_file(DOCUMENT_PART, &mut NativeProjectFileProbe)
            .unwrap(),
        Some(template)
    );
    let mut native = NativeAttachmentFiles;
    let mut attachments = ProjectAttachmentFiles::new(&bound.lookup, &mut native);
    assert_eq!(attachments.read("source.bin").unwrap(), [0, 255]);
    assert_eq!(loaded, configuration);
}
