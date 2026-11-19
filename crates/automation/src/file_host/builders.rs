// SPDX-License-Identifier: GPL-2.0-only
//! Optional services for the file host.
use super::*;
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Set the current display producer identity.
    /// Example: `host.with_assembly_producer(Ok(digest))`.
    pub fn with_assembly_producer(mut self, producer: Result<String, String>) -> Self {
        self.context.get_mut().assembly_producer = Some(producer);
        self
    }
    /// Set the active display quality for saved assembly views.
    /// Example: `host.with_assembly_display_limits((0.05, 0.5))`.
    pub fn with_assembly_display_limits(mut self, limits: (f64, f64)) -> Self {
        self.context.get_mut().assembly_display_limits = Some(limits);
        self
    }
    /// Inject part replay ownership; example: `host.with_part_update(&mut replay)`.
    pub fn with_part_update(
        mut self,
        replay: &'a mut dyn crate::PartUpdateAccess<O, W, F>,
    ) -> Self {
        self.context.get_mut().part_update = Some(replay);
        self
    }
    /// Share the desktop's marking menus; example: `host.with_marking_menus(&mut menus)`.
    pub fn with_marking_menus(
        mut self,
        menus: &'a mut oblikovati_model::marking_menu::MarkingMenus,
    ) -> Self {
        self.context.get_mut().marking_menus = Some(menus);
        self
    }
    /// Share marking-menu storage; example: `host.with_marking_store(&mut store)`.
    pub fn with_marking_store(
        mut self,
        store: &'a mut dyn oblikovati_model::marking_menu::MarkingMenuPreferenceAccess,
    ) -> Self {
        self.context.get_mut().marking_store = Some(store);
        self
    }
    /// Share the application keymap; example: `host.with_keymap(&mut keymap)`.
    pub fn with_keymap(mut self, keymap: &'a mut oblikovati_model::keymap::Keymap) -> Self {
        self.context.get_mut().keymap = Some(keymap);
        self
    }
    /// Share keymap storage; example: `host.with_keymap_store(&mut store)`.
    pub fn with_keymap_store(
        mut self,
        store: &'a mut dyn oblikovati_model::keymap::KeymapPreferenceAccess,
    ) -> Self {
        self.context.get_mut().keymap_store = Some(store);
        self
    }
    /// Supply exact drawing projection; example: `host.with_drawing_views(&backend)`.
    pub fn with_drawing_views(
        mut self,
        backend: &'a dyn crate::DrawingViewProjectionAccess,
    ) -> Self {
        self.context.get_mut().drawing_views = Some(backend);
        self
    }
    /// Supply profile discovery and candidate validation; example: `host.with_extrusion(&backend)`.
    pub fn with_extrusion(mut self, backend: &'a dyn crate::ExtrudeRegionAccess) -> Self {
        self.context.get_mut().extrusion = Some(backend);
        self
    }
    /// Supply exact Fillet body resolution and validation; example: `host.with_fillet(&backend)`.
    pub fn with_fillet(mut self, backend: &'a dyn crate::FilletAccess) -> Self {
        self.context.get_mut().fillet = Some(backend);
        self
    }
    /// Borrow the application viewport palettes; example: `host.with_color_schemes(&mut registry)`.
    pub fn with_color_schemes(
        mut self,
        registry: &'a mut oblikovati_model::color_scheme::ColorSchemeRegistry,
    ) -> Self {
        self.context.get_mut().color_schemes = Some(registry);
        self
    }
    /// Share the application theme library; example: `host.with_themes(&library)`.
    pub fn with_themes(mut self, library: &'a oblikovati_model::theme::ThemeLibrary) -> Self {
        self.context.get_mut().themes = Some(library);
        self
    }
    /// Inject the native capture queue; example: `host.with_window_capture(&mut captures)`.
    pub fn with_window_capture(mut self, capture: &'a mut dyn crate::WindowCaptureAccess) -> Self {
        self.context.get_mut().window_capture = Some(capture);
        self
    }
    /// Supply current desktop edit state without changing the document recipe.
    /// Example: `host.with_sketch_edit(Some(edit.clone()))`.
    pub fn with_sketch_edit(mut self, edit: Option<crate::SketchInspectionEdit>) -> Self {
        self.context.get_mut().sketch_edit = edit;
        self
    }
    /// Share the native exact-reference owner; example: `host.with_part_topology(&mut scenes)`.
    pub fn with_part_topology(
        mut self,
        references: &'a mut dyn crate::PartTopologyAccess<O, W, F>,
    ) -> Self {
        self.context.get_mut().part_topology = Some(references);
        self
    }
    /// Inject the desktop camera for part-view transitions; example: `host.with_part_view_display(&mut display)`.
    pub fn with_part_view_display(
        mut self,
        display: &'a mut dyn crate::PartViewDisplayAccess,
    ) -> Self {
        self.context.get_mut().part_view_display = Some(display);
        self
    }
}
