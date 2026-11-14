// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::DOCUMENT_ASSEMBLY;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;
use oblikovati_persistence::package::DocumentPackage;

use crate::owner::tests::files::{MemoryDocumentFiles, OwnerThreadGuids};

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
        .run("Place", |snapshot, ids| -> Result<(), ()> {
            let mut first = occurrence("First", 0.);
            first.session_id = ids.allocate_occurrence().unwrap();
            let mut second = occurrence("Second", 10.);
            second.session_id = ids.allocate_occurrence().unwrap();
            snapshot.assembly = Some(vec![first, second]);
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
fn component_preview_is_transformed_without_an_assembly_edit() {
    let (mut files, part, assembly, _) = workspace();
    let generation = files.workspace().session(assembly).unwrap().generation();
    let before = files
        .workspace()
        .session(assembly)
        .unwrap()
        .snapshot()
        .assembly
        .clone();
    let placement = oblikovati_api::types::Matrix {
        cells: [
            1., 0., 0., 20., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
    };
    let preview = DesktopPartScenes::default()
        .component_preview(&mut files, part, &placement)
        .unwrap();
    assert!(!preview.triangles.is_empty());
    assert!(
        preview
            .triangles
            .iter()
            .flat_map(|triangle| triangle.positions)
            .all(|point| point[0] >= 20.)
    );
    assert_eq!(
        files.workspace().session(assembly).unwrap().generation(),
        generation
    );
    assert_eq!(
        files
            .workspace()
            .session(assembly)
            .unwrap()
            .snapshot()
            .assembly,
        before
    );
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
fn occurrence_selection_tints_only_its_placement_and_clears() {
    let (mut files, _, assembly, _) = workspace();
    let generation = files.workspace().session(assembly).unwrap().generation();
    let mut scenes = DesktopPartScenes::default();
    let original = scenes.refresh(&mut files).unwrap().unwrap();
    let half = original.scene.triangles.len() / 2;
    assert!(half > 0);
    scenes.set_occurrence_selection(Some(vec![(assembly, generation, 1)]));
    let selected = scenes.refresh(&mut files).unwrap().unwrap();
    assert!(
        selected.scene.triangles[..half]
            .iter()
            .all(|triangle| triangle.color != CHOSEN_BODY_COLOR)
    );
    assert!(
        selected.scene.triangles[half..]
            .iter()
            .all(|triangle| triangle.color == CHOSEN_BODY_COLOR)
    );
    scenes.set_occurrence_selection(None);
    let cleared = scenes.refresh(&mut files).unwrap().unwrap();
    assert!(
        cleared
            .scene
            .triangles
            .iter()
            .all(|triangle| triangle.color != CHOSEN_BODY_COLOR)
    );
}

#[test]
fn sight_lines_pick_the_correct_analytic_component_and_refuse_ties() {
    let (mut files, _, assembly, _) = workspace();
    let generation = files.workspace().session(assembly).unwrap().generation();
    let mut scenes = DesktopPartScenes::default();
    let first = scenes
        .occurrence_on_line(&mut files, assembly, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap()
        .unwrap();
    let second = scenes
        .occurrence_on_line(&mut files, assembly, [[11., 1., 10.], [0., 0., -1.]])
        .unwrap()
        .unwrap();
    assert_eq!(first, [(assembly, generation, 0)]);
    assert_eq!(second, [(assembly, generation, 1)]);
    assert!(
        scenes
            .occurrence_on_line(&mut files, assembly, [[30., 1., 10.], [0., 0., -1.]])
            .unwrap()
            .is_none()
    );
    files
        .commands(assembly)
        .unwrap()
        .run("Overlap", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly.as_mut().unwrap()[1].transform[3] = 0.;
            Ok(())
        })
        .unwrap();
    assert!(
        scenes
            .occurrence_on_line(&mut files, assembly, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap_err()
            .contains("unambiguous")
    );
    files
        .commands(assembly)
        .unwrap()
        .run(
            "Place nearer component",
            |snapshot, ids| -> Result<(), ()> {
                let mut nearer = occurrence("Near", 0.);
                nearer.session_id = ids.allocate_occurrence().unwrap();
                nearer.transform[11] = 5.;
                snapshot.assembly.as_mut().unwrap().push(nearer);
                Ok(())
            },
        )
        .unwrap();
    let generation = files.workspace().session(assembly).unwrap().generation();
    assert_eq!(
        scenes
            .occurrence_on_line(&mut files, assembly, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
            .unwrap(),
        [(assembly, generation, 2)]
    );
}

#[test]
fn analytic_pick_compares_reaches_in_owner_coordinates_after_scaling() {
    let (mut files, _, assembly, _) = workspace();
    files
        .commands(assembly)
        .unwrap()
        .run("Scale placement", |snapshot, _| -> Result<(), ()> {
            let matrix = &mut snapshot.assembly.as_mut().unwrap()[1].transform;
            matrix[3] = 0.;
            matrix[10] = 10.;
            matrix[11] = -19.;
            Ok(())
        })
        .unwrap();
    let generation = files.workspace().session(assembly).unwrap().generation();
    let hit = DesktopPartScenes::default()
        .occurrence_on_line(&mut files, assembly, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap()
        .unwrap();
    assert_eq!(hit, [(assembly, generation, 0)]);
}

#[test]
fn assembly_reports_unresolved_components_instead_of_showing_an_empty_scene() {
    let (mut files, part, assembly, _) = workspace();
    files
        .commands(assembly)
        .unwrap()
        .run("Missing", |snapshot, ids| -> Result<(), ()> {
            snapshot
                .assembly
                .as_mut()
                .unwrap()
                .push(AssemblyOccurrenceRecipe {
                    session_id: ids.allocate_occurrence().unwrap(),
                    name: "Missing".into(),
                    component: "missing.opd".into(),
                    ..Default::default()
                });
            Ok(())
        })
        .unwrap();
    let mut scenes = DesktopPartScenes::default();
    let failed = scenes.refresh(&mut files).unwrap().unwrap();
    assert_eq!(failed.status.visible_bodies, 2);
    assert!(!failed.scene.triangles.is_empty());
    let error = &failed.status.scene_errors[0];
    assert!(
        error.contains("Missing") && error.contains("missing.opd"),
        "{error}"
    );
    assert!(failed.status.title().contains("missing.opd"));
    assert!(scenes.refresh(&mut files).unwrap().is_none());
    files
        .attach_document_reference(assembly, "missing.opd".into(), part, false)
        .unwrap();
    let recovered = scenes.refresh(&mut files).unwrap().unwrap();
    assert!(recovered.status.scene_errors.is_empty());
    assert_eq!(recovered.status.visible_bodies, 3);
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
        .run("Place part", |snapshot, ids| -> Result<(), ()> {
            let mut child = occurrence("Child", 5.);
            child.session_id = ids.allocate_occurrence().unwrap();
            snapshot.assembly = Some(vec![child]);
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(nested, "part.opd".into(), part, false)
        .unwrap();
    files
        .commands(assembly)
        .unwrap()
        .run("Place nested", |snapshot, ids| -> Result<(), ()> {
            let mut nested = occurrence("Nested", 20.);
            nested.session_id = ids.allocate_occurrence().unwrap();
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
