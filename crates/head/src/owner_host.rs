// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn borrowed_host<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &'a mut FileWorkspace<O, W, F>,
    guids: &'a mut dyn GuidSource,
    version: &'a str,
    factory: &'a mut dyn DocumentObserverFactory<Observer = O>,
    scenes: &'a mut crate::part_scene::DesktopPartScenes,
    construction: &'a oblikovati_martensite::planar_curve::MartensiteSketchConstruction,
    captures: &'a mut Option<crate::DesktopWindowCaptures>,
) -> FileHost<'a, O, W, F> {
    let host = FileHost::new(files, guids, version)
        .with_observer_factory(factory)
        .with_part_topology(scenes)
        .with_assembly_display_limits((
            crate::part_scene::DISPLAY_MESH.chord_error,
            crate::part_scene::DISPLAY_MESH.angular_error,
        ))
        .with_sketch_construction(construction, Default::default())
        .with_sketch_corner_construction(construction, Default::default());
    match captures {
        Some(captures) => host.with_window_capture(captures),
        None => host,
    }
}
