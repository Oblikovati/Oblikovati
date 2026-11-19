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

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Borrow the canonical caller for UI commands on this owner; example: `Client::new(owner.document_host())`.
    pub fn document_host(&mut self) -> FileHost<'_, O, W, F> {
        self.synchronize_sketch_edit();
        let editing = self.sketch_inspection_edit();
        let host = owner_host::borrowed_host(
            &mut self.files,
            &mut self.guids,
            &self.version,
            &mut self.factory,
            &mut self.part_scenes,
            &self.sketch_construction,
            &mut self.window_captures,
        )
        .with_drawing_views(&self.drawing_projection_access)
        .with_sketch_edit(editing)
        .with_marking_menus(&mut self.marking_menus)
        .with_keymap(&mut self.keymap)
        .with_color_schemes(&mut self.color_schemes)
        .with_startup_projection(self.sketch_options.auto_project_origin);
        let host = match self.part_store.as_mut() {
            Some(store) => host.with_part_preferences(store.as_mut()),
            None => host,
        };
        let host = match self.marking_store.as_mut() {
            Some(store) => host.with_marking_store(store.as_mut()),
            None => host,
        };
        let host = match self.keymap_store.as_mut() {
            Some(store) => host.with_keymap_store(store.as_mut()),
            None => host,
        };
        let host = match &self.extrusion_access {
            Some(backend) => host.with_extrusion(backend),
            None => host,
        };
        let host = match &self.fillet_access {
            Some(backend) => host.with_fillet(backend),
            None => host,
        };
        match &self.themes {
            Some(library) => host.with_themes(library),
            None => host,
        }
    }
}
