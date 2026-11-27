// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Derived Part dialog and component file picker built to UI Design System v1.1.
mod picker;
mod types;
pub use types::{
    GuiDeriveAction, GuiDeriveEditSeed, GuiDerivePreview, GuiDerivePreviewState, GuiDeriveRequest,
    GuiDeriveSourceSummary,
};

use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_error_badge, draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    derived_part::{DeriveMirrorPlane, DeriveStyle},
};
use picker::{ComponentPicker, PickerOutcome};
use std::path::Path;

const DERIVE_STYLES: [(DeriveStyle, &str); 4] = [
    (
        DeriveStyle::SingleSolid,
        "Single solid body merging out seams",
    ),
    (DeriveStyle::SolidWithSeams, "Solid body keep seams"),
    (
        DeriveStyle::MaintainSolids,
        "Maintain each solid as a solid body",
    ),
    (DeriveStyle::SingleComposite, "Single composite feature"),
];

const MIRROR_PLANES: [(DeriveMirrorPlane, &str); 3] = [
    (DeriveMirrorPlane::XY, "XY Plane"),
    (DeriveMirrorPlane::YZ, "YZ Plane"),
    (DeriveMirrorPlane::XZ, "XZ Plane"),
];

#[derive(Default)]
pub(crate) struct DeriveDialog {
    dock: crate::dockable_lifecycle::DockableState,
    picker: ComponentPicker,
    document: Option<u64>,
    feature: Option<FeatureId>,
    attempted: Option<GuiDeriveRequest>,
    submitted: Option<GuiDeriveRequest>,
    source_path: String,
    source_name: String,
    style: DeriveStyle,
    include_bodies: bool,
    include_sketches: bool,
    include_work_geometry: bool,
    include_parameters: bool,
    scale: String,
    mirror: bool,
    mirror_plane: DeriveMirrorPlane,
    link_face_color: bool,
    link_sketch_format: bool,
    show_all_objects: bool,
    linked: bool,
    model_state: String,
    design_view: String,
    preview_hidden: bool,
    presets_shown: bool,
    popup_was_open: bool,
}

impl DockableWindow for DeriveDialog {}

impl DeriveDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Derived Part", &self.dock)
    }

    pub(crate) fn is_opening(&self) -> bool {
        self.picker.is_opening()
    }

    pub(super) fn open(
        &mut self,
        state: &GuiCommandState,
        feature: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = state.document else {
            return;
        };
        match feature {
            Some(id) => {
                let Some(seed) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.derive_edits.get(&id))
                    .cloned()
                else {
                    return;
                };
                self.picker.close();
                self.load_seed(document, Some(id), seed, actions);
            }
            None => {
                let initial_dir = state
                    .document_storage
                    .get(&document)
                    .and_then(|storage| storage.path.as_deref())
                    .and_then(|path| Path::new(path).parent());
                self.document = Some(document);
                self.feature = None;
                self.dock.set_open(false);
                self.attempted = None;
                self.submitted = None;
                self.picker.open_in(initial_dir);
            }
        }
    }

    fn load_seed(
        &mut self,
        document: u64,
        feature: Option<FeatureId>,
        seed: GuiDeriveEditSeed,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.document = Some(document);
        self.feature = feature;
        self.source_path = seed.source_path.clone();
        self.source_name = seed.source_name;
        self.style = seed.style;
        self.include_bodies = seed.include_bodies;
        self.include_sketches = seed.include_sketches;
        self.include_work_geometry = seed.include_work_geometry;
        self.include_parameters = seed.include_parameters;
        self.scale = seed.scale;
        self.mirror = seed.mirror;
        self.mirror_plane = seed.mirror_plane;
        self.link_face_color = seed.link_face_color;
        self.link_sketch_format = seed.link_sketch_format;
        self.show_all_objects = seed.show_all_objects;
        self.linked = seed.linked;
        self.model_state = seed.model_state;
        self.design_view = seed.design_view;
        self.preview_hidden = false;
        self.attempted = None;
        self.submitted = None;
        self.dock.set_open(true);
        actions.push(GuiShellAction::Derive(GuiDeriveAction::InspectSource {
            document,
            path: seed.source_path,
        }));
    }

    pub(super) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        let was_open = self.document.take().is_some() || self.picker.is_open();
        self.picker.close();
        if was_open {
            self.submitted = None;
            self.attempted = None;
            self.dock.set_open(false);
            actions.push(GuiShellAction::Derive(GuiDeriveAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_none() && !self.picker.is_open() {
            return;
        }
        if self.document != state.document || !GuiDeriveAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if state.derive_preview.is_none()
            && let Some(request) = self.submitted.take()
            && state.document_generation != Some(request.generation)
        {
            self.document = None;
            self.attempted = None;
            self.dock.set_open(false);
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if self.picker.is_open() {
            match self.picker.draw(ui) {
                PickerOutcome::Pending => return,
                PickerOutcome::Cancelled => {
                    self.close(actions);
                    return;
                }
                PickerOutcome::Selected(path) => {
                    let source_name = Path::new(&path)
                        .file_name()
                        .and_then(|name| name.to_str())
                        .unwrap_or("DerivedPart.opd")
                        .to_string();
                    let seed = GuiDeriveEditSeed {
                        source_path: path,
                        source_name,
                        ..GuiDeriveEditSeed::default()
                    };
                    self.load_seed(document, None, seed, actions);
                }
            }
        }
        if !self.dock.is_open() {
            return;
        }
        let Some(generation) = state.document_generation else {
            return;
        };
        let current = self.build_request(document, generation);
        let current_result = state
            .derive_preview
            .as_ref()
            .filter(|entry| entry.request == current)
            .map(|entry| &entry.result);
        let ok_enabled = matches!(current_result, Some(GuiDerivePreview::Ready));

        let escape_closes = !self.popup_was_open
            && !crate::widget_abi::any_popup_open(ui)
            && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if escape_closes {
            self.close(actions);
            return;
        }

        let mut keep_open = true;
        let mut commit_now = false;
        let mut close_now = false;

        widget_abi::fit_viewport_properties(ui, "Derived Part", area, [340., 460.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Derived Part", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let title = format!("Derived Part › {}", self.source_name);
            let header = draw_properties_header(
                ui,
                icons,
                &title,
                "##derive-advanced-settings",
                &mut self.presets_shown,
                Some((self.preview_hidden, ok_enabled || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let body_height = (ui.content_region_avail()[1] - 44.).max(160.);
            let body = widget_abi::scroll_region(ui, "##derive-body", [0., body_height]);
            if body.visible {
                self.draw_tabs(ui, state.derive_source.as_ref(), current_result);
            }
            drop(body);
            let footer = draw_properties_footer(ui, ok_enabled);
            commit_now = footer.ok;
            close_now = footer.cancel;
        });
        self.popup_was_open = crate::widget_abi::any_popup_open(ui);

        if !keep_open || close_now {
            self.close(actions);
            return;
        }
        let next = self.build_request(document, generation);
        if commit_now && ok_enabled && next == current {
            self.submitted = Some(next.clone());
            actions.push(GuiShellAction::Derive(GuiDeriveAction::Commit(Box::new(
                next,
            ))));
            return;
        }
        if self.attempted.as_ref() != Some(&next) {
            self.attempted = Some(next.clone());
            actions.push(GuiShellAction::Derive(GuiDeriveAction::Preview(Box::new(
                next,
            ))));
        }
    }

    fn draw_tabs(
        &mut self,
        ui: &Ui,
        summary: Option<&GuiDeriveSourceSummary>,
        result: Option<&GuiDerivePreview>,
    ) {
        if let Some(_tabs) = ui.tab_bar("##derive-tabs") {
            if let Some(_tab) = ui.tab_item("Objects") {
                self.draw_objects_tab(ui, summary);
            }
            if let Some(_tab) = ui.tab_item("Options") {
                self.draw_options_tab(ui);
            }
            if let Some(_tab) = ui.tab_item("Representation") {
                self.draw_representation_tab(ui, summary);
            }
        }
        if let Some(GuiDerivePreview::Failed(message)) = result {
            ui.dummy([0., 4.]);
            draw_error_badge(ui, message);
        }
    }

    fn draw_objects_tab(&mut self, ui: &Ui, summary: Option<&GuiDeriveSourceSummary>) {
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Derive Style");
        }
        for (style, label) in DERIVE_STYLES {
            if ui.radio_button_bool(label, self.style == style) {
                self.style = style;
            }
        }
        ui.separator();
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Objects");
        }
        let empty = Vec::new();
        let bodies = summary.map_or(&empty, |s| &s.solid_bodies);
        let sketches = summary.map_or(&empty, |s| &s.sketches);
        let work = summary.map_or(&empty, |s| &s.work_geometry);
        let params = summary.map_or(&empty, |s| &s.parameters);
        draw_object_group(ui, "Solid Bodies", &mut self.include_bodies, bodies);
        draw_object_group(ui, "Sketches", &mut self.include_sketches, sketches);
        draw_object_group(ui, "Work Geometry", &mut self.include_work_geometry, work);
        draw_object_group(ui, "Parameters", &mut self.include_parameters, params);
    }

    fn draw_options_tab(&mut self, ui: &Ui) {
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Links");
        }
        ui.checkbox(
            "Link face color from source component",
            &mut self.link_face_color,
        );
        ui.checkbox(
            "Link sketch formatting from source component",
            &mut self.link_sketch_format,
        );
        ui.checkbox("Show all objects", &mut self.show_all_objects);
        ui.checkbox("Associative link", &mut self.linked);
        ui.separator();
        {
            let _semi = crate::ui_scale::ui_semibold_font(ui);
            ui.text("Transform");
        }
        crate::theme::property_label(ui, "Scale Factor");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##derive-scale", &mut self.scale).build();
        }
        ui.checkbox("Mirror part", &mut self.mirror);
        if self.mirror {
            crate::theme::property_label(ui, "Mirror plane");
            ui.set_next_item_width(-1.);
            let preview = mirror_plane_label(self.mirror_plane);
            if let Some(_combo) = ui.begin_combo("##derive-mirror-plane", preview) {
                for (plane, label) in MIRROR_PLANES {
                    if ui
                        .selectable_config(label)
                        .selected(self.mirror_plane == plane)
                        .build()
                    {
                        self.mirror_plane = plane;
                    }
                }
            }
        }
    }

    fn draw_representation_tab(&mut self, ui: &Ui, summary: Option<&GuiDeriveSourceSummary>) {
        let default_states = vec!["Primary".to_string()];
        let states = summary
            .filter(|s| !s.model_states.is_empty())
            .map_or(&default_states, |s| &s.model_states);
        let views = summary
            .filter(|s| !s.design_views.is_empty())
            .map_or(&default_states, |s| &s.design_views);

        crate::theme::property_label(ui, "Model State");
        ui.set_next_item_width(-1.);
        let state_display = representation_display(&self.model_state);
        if let Some(_combo) = ui.begin_combo("##derive-model-state", &state_display) {
            for state_name in states {
                let display = representation_display(state_name);
                if ui
                    .selectable_config(&display)
                    .selected(&self.model_state == state_name)
                    .build()
                {
                    self.model_state = state_name.clone();
                }
            }
        }

        crate::theme::property_label(ui, "Design View");
        ui.set_next_item_width(-1.);
        let view_display = representation_display(&self.design_view);
        if let Some(_combo) = ui.begin_combo("##derive-design-view", &view_display) {
            for view_name in views {
                let display = representation_display(view_name);
                if ui
                    .selectable_config(&display)
                    .selected(&self.design_view == view_name)
                    .build()
                {
                    self.design_view = view_name.clone();
                }
            }
        }
    }

    fn build_request(&self, document: u64, generation: u64) -> GuiDeriveRequest {
        GuiDeriveRequest {
            document,
            generation,
            feature: self.feature,
            source_path: self.source_path.clone(),
            source_name: self.source_name.clone(),
            style: self.style,
            include_bodies: self.include_bodies,
            include_sketches: self.include_sketches,
            include_work_geometry: self.include_work_geometry,
            include_parameters: self.include_parameters,
            scale: self.scale.clone(),
            mirror: self.mirror,
            mirror_plane: self.mirror_plane,
            link_face_color: self.link_face_color,
            link_sketch_format: self.link_sketch_format,
            show_all_objects: self.show_all_objects,
            linked: self.linked,
            model_state: self.model_state.clone(),
            design_view: self.design_view.clone(),
            preview_hidden: self.preview_hidden,
        }
    }
}

fn draw_object_group(ui: &Ui, label: &str, included: &mut bool, items: &[String]) {
    let _id = ui.push_id(label);
    ui.checkbox("##include", included);
    ui.same_line_with_spacing(0., 6.);
    if let Some(_node) = ui
        .tree_node_config(label)
        .default_open(label == "Solid Bodies")
        .push()
    {
        if items.is_empty() {
            ui.text_disabled("(none)");
        } else {
            for item in items {
                ui.bullet_text(item);
            }
        }
    }
}

fn mirror_plane_label(plane: DeriveMirrorPlane) -> &'static str {
    match plane {
        DeriveMirrorPlane::XY => "XY Plane",
        DeriveMirrorPlane::YZ => "YZ Plane",
        DeriveMirrorPlane::XZ => "XZ Plane",
    }
}

fn representation_display(name: &str) -> String {
    let trimmed = name.trim();
    if trimmed.is_empty() || trimmed.eq_ignore_ascii_case("primary") {
        "[Primary]".into()
    } else {
        trimmed.to_string()
    }
}
