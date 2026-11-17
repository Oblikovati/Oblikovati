// SPDX-License-Identifier: GPL-2.0-only
//! Semantic representation of the application window controls (#4064).
use accesskit::{Action, Node, NodeId, Rect, Role, TreeId, TreeInfo, TreeUpdate};

const WINDOW: NodeId = NodeId(1);
const MINIMIZE: NodeId = NodeId(2);
const MAXIMIZE: NodeId = NodeId(3);
const CLOSE: NodeId = NodeId(4);

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum FrameControl {
    Minimize,
    Maximize,
    Close,
    Answer(oblikovati_gui::GuiClosePromptChoice),
}

#[derive(Clone, Debug, PartialEq)]
pub(crate) struct FrameSemantics {
    pub title: String,
    pub maximized: bool,
    pub enabled: bool,
    pub bounds: [[f64; 4]; 3],
    pub focused: Option<FrameControl>,
    pub prompt: Option<oblikovati_gui::GuiClosePrompt>,
}

impl FrameSemantics {
    pub fn tree(&self) -> TreeUpdate {
        let mut nodes = self.frame_nodes();
        if let Some(prompt) = &self.prompt {
            nodes.extend(super::dialog::dialog_nodes(prompt));
        }
        TreeUpdate {
            nodes,
            tree: Some(TreeInfo::new(WINDOW)),
            tree_id: TreeId::ROOT,
            focus: self.prompt.as_ref().map_or_else(
                || self.focused.map_or(WINDOW, control_id),
                super::dialog::focused_dialog_control,
            ),
        }
    }
    fn frame_nodes(&self) -> Vec<(NodeId, Node)> {
        let mut root = Node::new(Role::Window);
        root.set_label(&self.title);
        let mut children = vec![MINIMIZE, MAXIMIZE, CLOSE];
        if self.prompt.is_some() {
            children.push(super::dialog::DIALOG);
        }
        root.set_children(children);
        let names = [
            "Minimize",
            if self.maximized {
                "Restore"
            } else {
                "Maximize"
            },
            "Close",
        ];
        let buttons = [MINIMIZE, MAXIMIZE, CLOSE]
            .into_iter()
            .zip(names)
            .zip(self.bounds)
            .map(|((id, name), bounds)| (id, frame_button(name, bounds, self.enabled)));
        std::iter::once((WINDOW, root)).chain(buttons).collect()
    }
}

pub(super) fn frame_button(name: &str, bounds: [f64; 4], enabled: bool) -> Node {
    let mut node = Node::new(Role::Button);
    node.set_label(name);
    node.set_bounds(Rect::new(bounds[0], bounds[1], bounds[2], bounds[3]));
    if enabled {
        node.add_action(Action::Click);
    } else {
        node.set_disabled();
    }
    node
}

pub(crate) fn control_id(control: FrameControl) -> NodeId {
    match control {
        FrameControl::Minimize => MINIMIZE,
        FrameControl::Maximize => MAXIMIZE,
        FrameControl::Close => CLOSE,
        FrameControl::Answer(choice) => match choice {
            oblikovati_gui::GuiClosePromptChoice::ContinueSketch => NodeId(6),
            oblikovati_gui::GuiClosePromptChoice::Save => NodeId(7),
            oblikovati_gui::GuiClosePromptChoice::Discard => NodeId(8),
            oblikovati_gui::GuiClosePromptChoice::Cancel => NodeId(9),
        },
    }
}

pub(crate) fn control_for_id(id: NodeId) -> Option<FrameControl> {
    match id {
        MINIMIZE => Some(FrameControl::Minimize),
        MAXIMIZE => Some(FrameControl::Maximize),
        CLOSE => Some(FrameControl::Close),
        NodeId(6) => Some(FrameControl::Answer(
            oblikovati_gui::GuiClosePromptChoice::ContinueSketch,
        )),
        NodeId(7) => Some(FrameControl::Answer(
            oblikovati_gui::GuiClosePromptChoice::Save,
        )),
        NodeId(8) => Some(FrameControl::Answer(
            oblikovati_gui::GuiClosePromptChoice::Discard,
        )),
        NodeId(9) => Some(FrameControl::Answer(
            oblikovati_gui::GuiClosePromptChoice::Cancel,
        )),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn guarded_prompt_has_modal_buttons_and_focus() {
        let prompt = oblikovati_gui::GuiClosePrompt {
            title: "Save changes",
            message: "Save changes to Part1 before closing?".into(),
            bounds: [10., 20., 300., 200.],
            buttons: vec![oblikovati_gui::GuiClosePromptButton {
                label: "Cancel",
                choice: oblikovati_gui::GuiClosePromptChoice::Cancel,
                bounds: [50., 60., 100., 80.],
                focused: true,
            }],
        };
        let frame = FrameSemantics {
            title: "Part1".into(),
            maximized: false,
            enabled: false,
            bounds: [[1., 2., 3., 4.]; 3],
            focused: None,
            prompt: Some(prompt),
        };
        let tree = frame.tree();
        assert_eq!(tree.focus, NodeId(9));
        let dialog = &tree.nodes[4].1;
        assert_eq!(dialog.role(), Role::Dialog);
        assert!(dialog.is_modal());
        assert_eq!(
            dialog.description(),
            Some("Save changes to Part1 before closing?")
        );
        assert_eq!(tree.nodes[5].1.label(), Some("Cancel"));
        assert!(tree.nodes[5].1.supports_action(Action::Click));
        assert!(tree.nodes[1].1.is_disabled());
    }
    #[test]
    fn controls_have_names_bounds_actions_and_restored_state() {
        let mut frame = FrameSemantics {
            title: "Part1".into(),
            maximized: false,
            enabled: true,
            bounds: [[1., 2., 3., 4.]; 3],
            focused: Some(FrameControl::Close),
            prompt: None,
        };
        let tree = frame.tree();
        assert_eq!(tree.focus, CLOSE);
        assert_eq!(tree.nodes[1].1.label(), Some("Minimize"));
        assert_eq!(tree.nodes[2].1.label(), Some("Maximize"));
        assert_eq!(tree.nodes[3].1.label(), Some("Close"));
        assert!(tree.nodes[3].1.supports_action(Action::Click));
        assert_eq!(tree.nodes[3].1.bounds(), Some(Rect::new(1., 2., 3., 4.)));
        frame.maximized = true;
        assert_eq!(frame.tree().nodes[2].1.label(), Some("Restore"));
        frame.enabled = false;
        let disabled = frame.tree();
        assert!(disabled.nodes[1].1.is_disabled());
        assert!(!disabled.nodes[1].1.supports_action(Action::Click));
        assert_eq!(control_for_id(MAXIMIZE), Some(FrameControl::Maximize));
        assert_eq!(control_for_id(NodeId(42)), None);
    }
}
