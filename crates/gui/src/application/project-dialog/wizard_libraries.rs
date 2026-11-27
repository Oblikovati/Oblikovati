// SPDX-License-Identifier: GPL-2.0-only
//! Transfer registered reference libraries into the new project in search order.
use crate::GuiProject;
use imgui::Ui;
use std::path::PathBuf;

const TRANSFER_WIDTH: f32 = 42.;
const LIST_HEIGHT: f32 = 110.;

#[derive(Default)]
pub(super) struct WizardLibraries {
    pub(super) roots: Vec<PathBuf>,
    pub(super) redraw: bool,
    available_selection: Option<PathBuf>,
    new_selection: Option<PathBuf>,
}

impl WizardLibraries {
    pub(super) fn panel(&mut self, ui: &Ui, projects: &[GuiProject]) {
        ui.text("Select Libraries");
        let spacing = ui.clone_style().item_spacing[0];
        let width = ((ui.content_region_avail()[0] - TRANSFER_WIDTH - 2. * spacing) * 0.5).max(1.);
        let available = self.available(projects);
        self.redraw |= library_list(
            ui,
            "All Projects",
            width,
            &available,
            &mut self.available_selection,
        );
        ui.same_line();
        self.transfer_buttons(ui, &available);
        ui.same_line();
        let chosen: Vec<_> = self.roots.iter().collect();
        self.redraw |= library_list(ui, "New Project", width, &chosen, &mut self.new_selection);
    }
    fn available<'a>(&self, projects: &'a [GuiProject]) -> Vec<&'a PathBuf> {
        let mut available = Vec::new();
        for root in projects.iter().flat_map(|project| &project.library_roots) {
            if !self.roots.contains(root) && !available.contains(&root) {
                available.push(root);
            }
        }
        available
    }
    fn transfer_buttons(&mut self, ui: &Ui, available: &[&PathBuf]) {
        ui.group(|| {
            let add = self
                .available_selection
                .as_ref()
                .filter(|root| available.contains(root));
            if transfer_button(ui, ">>", add.is_some())
                && let Some(root) = add
            {
                self.roots.push(root.clone());
                self.available_selection = None;
                self.redraw = true;
            }
            if transfer_button(ui, "<<", self.new_selection.is_some())
                && let Some(root) = self.new_selection.take()
            {
                self.roots.retain(|chosen| chosen != &root);
                self.redraw = true;
            }
        });
    }
}

fn transfer_button(ui: &Ui, label: &str, enabled: bool) -> bool {
    let _disabled = ui.begin_disabled(!enabled);
    crate::widget_abi::button(ui, label, [TRANSFER_WIDTH, 0.])
}

fn library_list(
    ui: &Ui,
    title: &str,
    width: f32,
    roots: &[&PathBuf],
    selection: &mut Option<PathBuf>,
) -> bool {
    let mut changed = false;
    ui.group(|| {
        ui.text(title);
        ui.child_window(format!("##{title}"))
            .size([width, LIST_HEIGHT])
            .border(true)
            .build(|| {
                if roots.is_empty() {
                    ui.text_disabled("(none)");
                }
                for root in roots {
                    let label = root.to_string_lossy();
                    let _identity = ui.push_id(&label);
                    if super::super::choose_project_name(
                        ui,
                        &label,
                        selection.as_ref() == Some(*root),
                    ) {
                        *selection = Some((*root).clone());
                        changed = true;
                    }
                }
            });
    });
    changed
}
