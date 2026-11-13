// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::DOCUMENT_ASSEMBLY;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
use oblikovati_persistence::package::DocumentPackage;

#[path = "owner/files.rs"]
mod files;
use files::{MemoryDocumentFiles, OwnerThreadGuids};

type TestFiles =
    FileWorkspace<crate::DesktopObservations, crate::DesktopObservations, MemoryDocumentFiles>;

fn workspace() -> (TestFiles, u64, u64, OwnerThreadGuids) {
    let mut files = FileWorkspace::new(
        crate::DesktopObservations::default(),
        MemoryDocumentFiles::default(),
    );
    let mut guids = OwnerThreadGuids::default();
    let fixture = crate::scene_fixture::selected_prism_document();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&fixture.parameters, Default::default())
        .unwrap()
        .with_sketches(&fixture.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&fixture.work_geometry, 1)
        .unwrap();
    let path = std::path::Path::new("/fixture/part.opd");
    files
        .files()
        .saved
        .borrow_mut()
        .insert(path.into(), package.encode().unwrap());
    let part = files
        .open(
            path,
            crate::DesktopObservations::default(),
            "test",
            &mut guids,
            oblikovati_session::workspace::OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let assembly = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly"),
            crate::DesktopObservations::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    files
        .commands(assembly)
        .unwrap()
        .run("Place", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(vec![occurrence("First", 0.), occurrence("Second", 10.)]);
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(assembly, "part.opd".into(), part, false)
        .unwrap();
    (files, part, assembly, guids)
}

fn occurrence(name: &str, x: f64) -> AssemblyOccurrenceRecipe {
    AssemblyOccurrenceRecipe {
        name: name.into(),
        component: "part.opd".into(),
        transform: vec![
            1., 0., 0., x, 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
        ..Default::default()
    }
}

#[test]
fn assembly_repeats_components_and_refreshes_after_component_edits() {
    let (mut files, part, assembly, _) = workspace();
    let mut scenes = DesktopPartScenes::default();
    let first = scenes.refresh(&mut files).unwrap().unwrap();
    assert_eq!(first.status.source.unwrap().0, assembly);
    assert_eq!(first.status.visible_bodies, 2);
    let xs: Vec<_> = first
        .scene
        .triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|p| p[0])
        .collect();
    assert!(xs.iter().copied().fold(f32::NEG_INFINITY, f32::max) > 10.);
    assert!(scenes.refresh(&mut files).unwrap().is_none());
    files
        .commands(part)
        .unwrap()
        .run("Depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    let changed = scenes.refresh(&mut files).unwrap().unwrap();
    let max_z = changed
        .scene
        .triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|p| p[2])
        .fold(f32::NEG_INFINITY, f32::max);
    assert_eq!(max_z, 3.);
}

#[test]
fn assembly_reports_unresolved_components_instead_of_showing_an_empty_scene() {
    let (mut files, _, assembly, _) = workspace();
    files
        .commands(assembly)
        .unwrap()
        .run("Missing", |snapshot, _| -> Result<(), ()> {
            snapshot
                .assembly
                .as_mut()
                .unwrap()
                .push(AssemblyOccurrenceRecipe {
                    name: "Missing".into(),
                    component: "missing.opd".into(),
                    ..Default::default()
                });
            Ok(())
        })
        .unwrap();
    let error = match DesktopPartScenes::default().refresh(&mut files) {
        Err(error) => error,
        Ok(_) => panic!("expected unresolved component failure"),
    };
    assert!(
        error.contains("Missing") && error.contains("missing.opd"),
        "{error}"
    );
}

#[test]
fn nested_placements_compose_and_suppressed_occurrences_are_hidden() {
    let (mut files, part, assembly, mut guids) = workspace();
    let nested = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Nested"),
            crate::DesktopObservations::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    files
        .commands(nested)
        .unwrap()
        .run("Place part", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(vec![occurrence("Child", 5.)]);
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(nested, "part.opd".into(), part, false)
        .unwrap();
    files
        .commands(assembly)
        .unwrap()
        .run("Place nested", |snapshot, _| -> Result<(), ()> {
            let mut nested = occurrence("Nested", 20.);
            nested.component = "nested.oad".into();
            snapshot.assembly.as_mut().unwrap().push(nested);
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(assembly, "nested.oad".into(), nested, false)
        .unwrap();
    files.activate(assembly).unwrap();
    let mut scenes = DesktopPartScenes::default();
    let scene = scenes.refresh(&mut files).unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 3);
    let max_x = scene
        .scene
        .triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|point| point[0])
        .fold(f32::NEG_INFINITY, f32::max);
    assert!(max_x > 25.);
    files
        .commands(assembly)
        .unwrap()
        .run("Suppress nested", |snapshot, _| -> Result<(), ()> {
            snapshot
                .assembly
                .as_mut()
                .unwrap()
                .last_mut()
                .unwrap()
                .suppressed = true;
            Ok(())
        })
        .unwrap();
    assert_eq!(
        scenes
            .refresh(&mut files)
            .unwrap()
            .unwrap()
            .status
            .visible_bodies,
        2
    );
}
