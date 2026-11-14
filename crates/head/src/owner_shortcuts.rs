// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::keymap::Keymap;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey};

impl<
    O: super::TransactionObserver,
    W: super::WorkspaceObserver,
    F: super::DocumentFiles,
    G: super::GuidSource,
    R: super::DocumentObserverFactory<Observer = O> + super::HeadInvalidation,
    S: super::DispatchWakeup,
> super::DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn route_sketch_alias(
        &mut self,
        event: ViewportEvent,
    ) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        if self.active_sketch_edit().is_none() || self.sketch_tool.is_some() {
            self.sketch_alias.clear();
            return (false, None);
        }
        match event {
            ViewportEvent::Input(ViewportInput::Text(ch))
                if !ch.is_control() && self.sketch_alias.len() + ch.len_utf8() <= 128 =>
            {
                self.sketch_alias.push(ch);
                (true, None)
            }
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Backspace,
                pressed: true,
                ..
            }) if !self.sketch_alias.is_empty() => {
                self.sketch_alias.pop();
                (true, None)
            }
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Enter,
                pressed: true,
                ..
            }) if !self.sketch_alias.is_empty() => self.submit_sketch_alias(),
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Escape,
                pressed: true,
                ..
            }) if !self.sketch_alias.is_empty() => {
                self.sketch_alias.clear();
                (true, None)
            }
            ViewportEvent::Focus(false) | ViewportEvent::Mapped(false) => {
                self.sketch_alias.clear();
                (false, None)
            }
            _ => (false, None),
        }
    }
    fn submit_sketch_alias(&mut self) -> (bool, Option<oblikovati_gui::GuiShellAction>) {
        let typed = std::mem::take(&mut self.sketch_alias);
        let action = self.keymap.resolve_alias(&typed).and_then(alias_action);
        if action.is_none() {
            self.command_notice = Some(super::DesktopCommandNotice::AuthoringFailed {
                cause: format!("command alias {typed:?}: expected a registered sketch command"),
            });
        }
        (true, action)
    }
}
fn alias_action(id: &str) -> Option<oblikovati_gui::GuiShellAction> {
    use oblikovati_gui::GuiShellAction;
    Some(match id {
        "edit.undo" => GuiShellAction::Undo,
        "edit.redo" => GuiShellAction::Redo,
        "file.save" => GuiShellAction::Save,
        "view.home" => GuiShellAction::Home,
        "view.previous" => GuiShellAction::ViewCommand(oblikovati_gui::GuiViewCommand::Previous),
        _ => return sketch_action(id),
    })
}

impl<O, W, F, G, R, S: super::DispatchWakeup> super::DesktopOwner<O, W, F, G, R, S> {
    /// Load and attach keymap storage; example: `owner.with_keymap_preferences(store)?`.
    pub fn with_keymap_preferences(
        mut self,
        store: Box<dyn oblikovati_model::keymap::KeymapPreferenceAccess>,
    ) -> Result<Self, String> {
        self.keymap.restore_saved(&store.keymap())?;
        self.keymap_store = Some(store);
        Ok(self)
    }
}

pub(super) struct ShortcutKeys {
    held: Vec<(ViewportPhysicalKey, Option<ViewportKey>)>,
    action: Option<oblikovati_gui::GuiShellAction>,
    focused: bool,
    mapped: bool,
}
impl Default for ShortcutKeys {
    fn default() -> Self {
        Self {
            held: Vec::new(),
            action: None,
            focused: true,
            mapped: true,
        }
    }
}
impl ShortcutKeys {
    pub(super) fn take_action(&mut self) -> Option<oblikovati_gui::GuiShellAction> {
        self.action.take()
    }
    pub(super) fn resolve(&mut self, map: &Keymap, event: ViewportEvent) -> ViewportEvent {
        self.action = None;
        match event {
            ViewportEvent::Focus(focused) => {
                self.focused = focused;
                if !focused {
                    self.held.clear();
                }
                event
            }
            ViewportEvent::Mapped(mapped) => {
                self.mapped = mapped;
                if !mapped {
                    self.held.clear();
                }
                event
            }
            ViewportEvent::CloseRequested => {
                self.held.clear();
                self.focused = false;
                event
            }
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key,
                command,
                pressed,
                modifiers,
            }) => {
                if !self.focused || !self.mapped {
                    return event;
                }
                if pressed && !self.held.iter().any(|(held, _)| *held == key) {
                    self.action = action_id(map, layout_key, modifiers).and_then(chord_action);
                }
                let command = self.transition(map, key, layout_key, command, pressed, modifiers);
                ViewportEvent::Input(ViewportInput::PhysicalKey {
                    key,
                    layout_key,
                    command,
                    pressed,
                    modifiers,
                })
            }
            _ => event,
        }
    }
    fn transition(
        &mut self,
        map: &Keymap,
        key: ViewportPhysicalKey,
        layout: ViewportPhysicalKey,
        _original: Option<ViewportKey>,
        pressed: bool,
        modifiers: oblikovati_viewport::ViewportModifiers,
    ) -> Option<ViewportKey> {
        if let Some(index) = self.held.iter().position(|(held, _)| *held == key) {
            return if pressed {
                None
            } else {
                self.held.remove(index).1
            };
        }
        if !pressed {
            return None;
        }
        let command = shortcut(map, layout, modifiers);
        self.held.push((key, command));
        command
    }
}
fn chord_action(id: &str) -> Option<oblikovati_gui::GuiShellAction> {
    if id == "Create.Extrude" {
        return Some(oblikovati_gui::GuiShellAction::OpenExtrude);
    }
    sketch_action(id)
}
fn shortcut(
    map: &Keymap,
    key: ViewportPhysicalKey,
    modifiers: oblikovati_viewport::ViewportModifiers,
) -> Option<ViewportKey> {
    match action_id(map, key, modifiers)? {
        "edit.undo" => Some(ViewportKey::Undo),
        "edit.redo" => Some(ViewportKey::Redo),
        "file.save" => Some(ViewportKey::Save),
        "tool.cancel" => Some(ViewportKey::Cancel),
        "view.pan" => Some(ViewportKey::Pan),
        "view.zoom" => Some(ViewportKey::Zoom),
        "view.orbit" => Some(ViewportKey::Orbit),
        "view.previous" => Some(ViewportKey::Previous),
        "view.home" => Some(ViewportKey::Home),
        _ => None,
    }
}
fn action_id(
    map: &Keymap,
    key: ViewportPhysicalKey,
    modifiers: oblikovati_viewport::ViewportModifiers,
) -> Option<&str> {
    if modifiers.super_key {
        return None;
    }
    let chord = oblikovati_api::types::KeyChord {
        key: key.chord_token(),
        ctrl: modifiers.control,
        alt: modifiers.alt,
        shift: modifiers.shift,
    };
    map.resolve_chord(&chord)
}
fn sketch_action(id: &str) -> Option<oblikovati_gui::GuiShellAction> {
    use oblikovati_gui::GuiSketchTool;
    if id == "dimension" {
        return Some(oblikovati_gui::GuiShellAction::Dimension(
            oblikovati_gui::GuiDimensionAction::Start,
        ));
    }
    if let Some(constraint) = constraint_action(id) {
        return Some(oblikovati_gui::GuiShellAction::ConstrainSketch(constraint));
    }
    let tool = match id {
        "sketch-line" => GuiSketchTool::Line,
        "sketch-rectangle" => GuiSketchTool::Rectangle,
        "sketch-circle" => GuiSketchTool::Circle,
        "sketch-point" => GuiSketchTool::Point,
        "sketch-center-arc" => GuiSketchTool::CenterPointArc,
        "sketch-ellipse" => GuiSketchTool::Ellipse,
        "sketch-center-rectangle" => GuiSketchTool::CenterRectangle,
        "sketch-three-point-rectangle" => GuiSketchTool::ThreePointRectangle,
        "sketch-three-point-center-rectangle" => GuiSketchTool::ThreePointCenterRectangle,
        "sketch-three-point-circle" => GuiSketchTool::ThreePointCircle,
        "sketch-three-point-arc" => GuiSketchTool::ThreePointArc,
        _ => return None,
    };
    Some(oblikovati_gui::GuiShellAction::PlaceSketchTool(tool))
}
fn constraint_action(id: &str) -> Option<oblikovati_gui::GuiSketchConstraint> {
    use oblikovati_gui::GuiSketchConstraint::*;
    Some(match id {
        "constraint-horizontal" => Horizontal,
        "constraint-vertical" => Vertical,
        "constraint-coincident" => Coincident,
        "constraint-fix" => Fix,
        "constraint-parallel" => Parallel,
        "constraint-perpendicular" => Perpendicular,
        "constraint-collinear" => Collinear,
        "constraint-equal" => Equal,
        "constraint-concentric" => Concentric,
        "constraint-tangent" => Tangent,
        _ => return None,
    })
}

impl<
    O: super::TransactionObserver,
    W: super::WorkspaceObserver,
    F: super::DocumentFiles,
    G: super::GuidSource,
    R: super::DocumentObserverFactory<Observer = O> + super::HeadInvalidation,
    S: super::DispatchWakeup,
> super::DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_keymap_action(
        &mut self,
        delta: &oblikovati_api::wire::KeymapExport,
    ) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .keymap()
            .import(delta)
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_viewport::ViewportModifiers;
    fn key(key: ViewportPhysicalKey, pressed: bool) -> ViewportEvent {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            command: None,
            pressed,
            modifiers: ViewportModifiers {
                control: true,
                ..Default::default()
            },
        })
    }
    fn command(event: ViewportEvent) -> Option<ViewportKey> {
        match event {
            ViewportEvent::Input(ViewportInput::PhysicalKey { command, .. }) => command,
            _ => None,
        }
    }
    #[test]
    fn remapped_shortcut_dispatches_once_and_preserves_release() {
        let mut map = Keymap::desktop_defaults();
        map.set_chord("edit.undo", "Ctrl+U").unwrap();
        let mut keys = ShortcutKeys::default();
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::Z, true))),
            None
        );
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, true))),
            Some(ViewportKey::Undo)
        );
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, true))),
            None
        );
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, false))),
            Some(ViewportKey::Undo)
        );
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, true))),
            Some(ViewportKey::Undo)
        );
        keys.resolve(&map, ViewportEvent::Focus(false));
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, true))),
            None
        );
        keys.resolve(&map, ViewportEvent::Focus(true));
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::U, true))),
            Some(ViewportKey::Undo)
        );
    }
    #[test]
    fn sketch_shortcut_suppresses_repeat_until_release() {
        let mut map = Keymap::desktop_defaults();
        map.set_chord("sketch-rectangle", "Ctrl+R").unwrap();
        let mut keys = ShortcutKeys::default();
        keys.resolve(&map, key(ViewportPhysicalKey::R, true));
        assert_eq!(
            keys.take_action(),
            Some(oblikovati_gui::GuiShellAction::PlaceSketchTool(
                oblikovati_gui::GuiSketchTool::Rectangle
            ))
        );
        keys.resolve(&map, key(ViewportPhysicalKey::R, true));
        assert_eq!(keys.take_action(), None);
        keys.resolve(&map, key(ViewportPhysicalKey::R, false));
        assert_eq!(keys.take_action(), None);
        keys.resolve(&map, key(ViewportPhysicalKey::R, true));
        assert!(keys.take_action().is_some());
    }
    #[test]
    fn every_registered_sketch_command_has_a_dispatch_action() {
        let bindings = Keymap::desktop_defaults().catalog();
        for binding in bindings.iter().filter(|binding| binding.kind == "command") {
            assert!(
                sketch_action(&binding.action_id).is_some(),
                "{}",
                binding.action_id
            );
        }
    }
    #[test]
    fn remapped_pan_does_not_leave_the_old_platform_shortcut_active() {
        let mut map = Keymap::desktop_defaults();
        let mut keys = ShortcutKeys::default();
        let f2 = |pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::F2,
                layout_key: ViewportPhysicalKey::F2,
                command: Some(ViewportKey::Pan),
                pressed,
                modifiers: Default::default(),
            })
        };
        assert_eq!(
            command(keys.resolve(&map, f2(true))),
            Some(ViewportKey::Pan)
        );
        assert_eq!(
            command(keys.resolve(&map, f2(false))),
            Some(ViewportKey::Pan)
        );
        map.set_chord("view.pan", "Ctrl+P").unwrap();
        assert_eq!(command(keys.resolve(&map, f2(true))), None);
        keys.resolve(&map, f2(false));
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::P, true))),
            Some(ViewportKey::Pan)
        );
        assert_eq!(
            command(keys.resolve(&map, key(ViewportPhysicalKey::P, false))),
            Some(ViewportKey::Pan)
        );
    }
}
