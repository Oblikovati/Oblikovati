// SPDX-License-Identifier: GPL-2.0-only
//! Part representation rows use document-owned view identities (#4118).
use super::*;
use oblikovati_api::contract::PartDesignView as _;

impl BrowserTree<'_> {
    pub(super) fn representations(
        &self,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.browser.model_states.is_none() && self.browser.part_design_views.is_none() {
            return;
        }
        let Some(_representations) = self.branch("Representations", true) else {
            return;
        };
        self.model_states(view, actions);
        self.part_view_rows(view, actions);
    }

    fn part_view_rows(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        let Some(views) = &self.browser.part_design_views else {
            return;
        };
        view.part_view_edit.observe(self.state.document, views);
        let node = if view.part_view_edit.needs_open_branch() {
            self.ui
                .tree_node_config("View")
                .opened(true, imgui::Condition::Always)
                .push()
        } else {
            self.branch("View", false)
        };
        if let Some(active) = views.active() {
            self.ui.same_line_with_spacing(0., 16.);
            self.ui.text(format!("<{}>", active.name()));
        }
        let Some(_view) = node else {
            return;
        };
        for entry in views.list() {
            let name = entry.name();
            let _id = self.ui.push_id(format!("part-view-{}", entry.id()));
            view.part_view_edit
                .rename_key(self.ui, self.state.document, entry);
            if view
                .part_view_edit
                .draw(self.ui, self.state.document, entry, views, actions)
            {
                continue;
            }
            let position = self.ui.cursor_screen_pos();
            if crate::widget_abi::selectable(self.ui, "##part-view", entry.active(), true) {
                view.part_view_edit
                    .select(self.state.document.unwrap_or(0), entry.id());
            }
            if self.ui.is_item_hovered() && self.ui.is_mouse_clicked(MouseButton::Right) {
                view.part_view_edit
                    .select(self.state.document.unwrap_or(0), entry.id());
                self.ui.open_popup("##part-view-menu");
            }
            if !entry.active()
                && self.ui.is_item_hovered()
                && self.ui.is_mouse_double_clicked(MouseButton::Left)
            {
                actions.push(GuiShellAction::BrowserChoice {
                    document: self.state.document.unwrap_or(0),
                    choice: crate::GuiBrowserChoice::ActivatePartView(entry.id()),
                });
            }
            self.part_view_menu(entry, views, view, actions);
            if entry.active() {
                browser_model_state::active_state_mark(self.ui, position);
            }
            crate::widget_abi::literal_text(
                self.ui,
                [
                    position[0] + 16. * self.ui.current_font_size() / 13.,
                    position[1],
                ],
                self.ui.style_color(StyleColor::Text),
                &name,
            );
            if entry.locked() {
                let text_x = position[0] + 16. * self.ui.current_font_size() / 13.;
                let width = self.ui.calc_text_size(&name)[0];
                draw_part_view_lock(self.ui, [text_x + width + 6., position[1]]);
            }
        }
    }
    fn part_view_menu(
        &self,
        entry: &oblikovati_model::part_design_view::PartDesignView,
        views: &oblikovati_model::part_design_view::PartDesignViews,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = self.ui.begin_popup("##part-view-menu") else {
            return;
        };
        let document = self.state.document.unwrap_or(0);
        if self
            .ui
            .menu_item_config("Activate")
            .enabled(!entry.active())
            .build()
        {
            actions.push(GuiShellAction::BrowserChoice {
                document,
                choice: crate::GuiBrowserChoice::ActivatePartView(entry.id()),
            });
        }
        if self.ui.menu_item("Copy") {
            view.part_view_edit.request_copy(document, views);
            actions.push(GuiShellAction::BrowserChoice {
                document,
                choice: crate::GuiBrowserChoice::CopyPartView(entry.id()),
            });
        }
        if entry.active() {
            for (label, visible) in [("All Visible", true), ("All Hidden", false)] {
                if self.ui.menu_item(label) {
                    actions.push(GuiShellAction::BrowserChoice {
                        document,
                        choice: crate::GuiBrowserChoice::PartViewBodyVisibility {
                            id: entry.id(),
                            visible,
                        },
                    });
                }
            }
        }
        self.ui.menu("Camera View", || {
            if self
                .ui
                .menu_item_config("Save Current Camera")
                .enabled(entry.active() && !entry.locked())
                .build()
            {
                actions.push(GuiShellAction::BrowserChoice {
                    document,
                    choice: crate::GuiBrowserChoice::SavePartViewCamera(entry.id()),
                });
            }
            if self
                .ui
                .menu_item_config("Restore Saved Camera")
                .enabled(entry.camera().is_some())
                .build()
            {
                actions.push(GuiShellAction::BrowserChoice {
                    document,
                    choice: crate::GuiBrowserChoice::RestorePartViewCamera(entry.id()),
                });
            }
            if self
                .ui
                .menu_item_config("Autosave Camera")
                .selected(entry.autosave_camera())
                .build()
            {
                actions.push(GuiShellAction::BrowserChoice {
                    document,
                    choice: crate::GuiBrowserChoice::PartViewAutosave {
                        id: entry.id(),
                        enabled: !entry.autosave_camera(),
                    },
                });
            }
        });
        if self
            .ui
            .menu_item_config("Publish")
            .selected(entry.published())
            .build()
        {
            actions.push(GuiShellAction::BrowserChoice {
                document,
                choice: crate::GuiBrowserChoice::PublishPartView {
                    id: entry.id(),
                    published: !entry.published(),
                },
            });
        }
        if self
            .ui
            .menu_item_config("Lock")
            .selected(entry.locked())
            .enabled(!entry.primary())
            .build()
        {
            actions.push(GuiShellAction::BrowserChoice {
                document,
                choice: crate::GuiBrowserChoice::LockPartView {
                    id: entry.id(),
                    locked: !entry.locked(),
                },
            });
        }
    }
}

fn draw_part_view_lock(ui: &Ui, position: [f32; 2]) {
    let scale = ui.current_font_size() / 13.;
    let [x, y] = position;
    let color = ui.style_color(StyleColor::Text);
    let draw = ui.get_window_draw_list();
    draw.add_rect(
        [x, y + 6. * scale],
        [x + 9. * scale, y + 12. * scale],
        color,
    )
    .build();
    draw.add_line(
        [x + 2. * scale, y + 6. * scale],
        [x + 2. * scale, y + 3. * scale],
        color,
    )
    .build();
    draw.add_line(
        [x + 2. * scale, y + 3. * scale],
        [x + 7. * scale, y + 3. * scale],
        color,
    )
    .build();
    draw.add_line(
        [x + 7. * scale, y + 3. * scale],
        [x + 7. * scale, y + 6. * scale],
        color,
    )
    .build();
}
