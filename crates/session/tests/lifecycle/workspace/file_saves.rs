// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn save_uses_only_the_bound_package_and_destination() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let _other = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    host.save(id, "beta", &mut source).unwrap();
    let files = host.files().contents.borrow();
    let written = DocumentPackage::decode(&files[&path("a.obk")]).unwrap();
    assert_eq!(
        written
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(files[&path("b.obk")], fixture(2));
    assert!(!host.workspace().needs_save(id).unwrap());
}

fn stored_picture(host: &Host) -> Option<oblikovati_persistence::package::SavedPicture> {
    let files = host.files().contents.borrow();
    let written = DocumentPackage::decode(&files[&path("a.obk")]).unwrap();
    written.restore_preview_picture().unwrap()
}
#[test]
fn a_kept_picture_reaches_the_next_save_and_a_removal_takes_it_out() {
    use oblikovati_persistence::package::SavedPicture;
    use oblikovati_session::files::FilePicture;
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let picture = SavedPicture {
        extent: [1, 1],
        rgb: vec![[7, 8, 9]],
    };
    host.keep_file_picture(id, FilePicture::Store(picture.clone()));
    host.save(id, "beta", &mut source).unwrap();
    assert_eq!(stored_picture(&host), Some(picture.clone()));
    host.save(id, "beta", &mut source).unwrap();
    assert_eq!(stored_picture(&host), Some(picture));
    host.keep_file_picture(id, FilePicture::Remove);
    host.save(id, "beta", &mut source).unwrap();
    assert_eq!(stored_picture(&host), None);
}
