// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::package::{SavedAssemblyTriangle, SavedAssemblyView};

fn save_qualified_assembly(
    files: &mut TestFiles,
    part: u64,
    assembly: u64,
    guids: &mut OwnerThreadGuids,
) -> (std::path::PathBuf, Vec<SavedAssemblyTriangle>) {
    files.save(part, "test", guids).unwrap();
    let part_name = files.path(part).unwrap().to_str().unwrap().to_owned();
    files
        .commands(assembly)
        .unwrap()
        .run("Use the saved component", |snapshot, _| -> Result<(), ()> {
            for occurrence in snapshot.assembly.as_mut().unwrap() {
                occurrence.component.clone_from(&part_name);
            }
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(assembly, part_name, part, false)
        .unwrap();
    let mut scenes = DesktopPartScenes::default();
    let output = scenes
        .evaluate_assembly_for_view(files, assembly, AssemblyViewMode::Persisted)
        .unwrap();
    let triangles: Vec<_> = output
        .scene
        .triangles
        .iter()
        .map(|item| SavedAssemblyTriangle {
            positions: item.positions,
            color: item.color,
        })
        .collect();
    assert!(!triangles.is_empty());
    let path = files
        .path(part)
        .unwrap()
        .parent()
        .unwrap()
        .join("assembly.oad");
    let view = SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences: files
            .workspace()
            .session(assembly)
            .unwrap()
            .snapshot()
            .assembly
            .clone()
            .unwrap(),
        sources: files.assembly_saved_sources(assembly, &path).unwrap(),
        catalog_digest: files.assembly_view_catalog_digest().unwrap(),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: triangles.clone(),
    };
    files.stage_assembly_view(assembly, &path, view).unwrap();
    files.save_as(assembly, &path, "test", guids).unwrap();
    (path, triangles)
}

#[test]
fn express_scene_uses_saved_triangles_and_full_load_replaces_it() {
    let (mut files, part, assembly, mut guids) = workspace();
    let part_path = files.path(part).unwrap().to_owned();
    let (path, expected) = save_qualified_assembly(&mut files, part, assembly, &mut guids);
    files.close_discarding_changes(assembly).unwrap();
    files.close_discarding_changes(part).unwrap();
    let express = files
        .open_express_with_factory(
            &path,
            || Ok(crate::DesktopObservations::default()),
            "test",
            &mut guids,
            (DISPLAY_MESH.chord_error, DISPLAY_MESH.angular_error),
        )
        .unwrap()
        .session_id();
    assert_eq!(files.workspace().documents(), &[express]);
    let mut scenes = DesktopPartScenes::default();
    let cached = scenes.refresh(&mut files).unwrap().unwrap();
    assert!(cached.status.cached_assembly);
    assert!(cached.status.scene_errors.is_empty());
    assert_eq!(cached.scene.triangles.len(), expected.len());
    assert_eq!(cached.scene.triangles[0].positions, expected[0].positions);
    assert_eq!(cached.scene.triangles[0].color, expected[0].color);
    let part = files
        .open(
            &part_path,
            crate::DesktopObservations::default(),
            "test",
            &mut guids,
            oblikovati_session::workspace::OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    files
        .attach_document_reference(express, part_path.to_str().unwrap().into(), part, false)
        .unwrap();
    assert!(files.finish_express_assembly(express).unwrap());
    let full = scenes.refresh(&mut files).unwrap().unwrap();
    assert!(!full.status.cached_assembly);
    assert_eq!(full.status.visible_bodies, 2);
    assert!(!full.scene.triangles.is_empty());
}
