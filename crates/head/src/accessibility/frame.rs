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
}

#[derive(Clone, Debug, PartialEq)]
pub(crate) struct FrameSemantics {
    pub title: String,
    pub maximized: bool,
    pub bounds: [[f64; 4]; 3],
    pub focused: Option<FrameControl>,
}

impl FrameSemantics {
    pub fn tree(&self) -> TreeUpdate {
        let mut root = Node::new(Role::Window);
        root.set_label(&self.title);
        root.set_children([MINIMIZE, MAXIMIZE, CLOSE]);
        let names = [
            "Minimize",
            if self.maximized {
                "Restore"
            } else {
                "Maximize"
            },
            "Close",
        ];
        let nodes = [MINIMIZE, MAXIMIZE, CLOSE]
            .into_iter()
            .zip(names)
            .zip(self.bounds)
            .map(|((id, name), bounds)| (id, frame_button(name, bounds)));
        TreeUpdate {
            nodes: std::iter::once((WINDOW, root)).chain(nodes).collect(),
            tree: Some(TreeInfo::new(WINDOW)),
            tree_id: TreeId::ROOT,
            focus: self.focused.map_or(WINDOW, control_id),
        }
    }
}

fn frame_button(name: &str, bounds: [f64; 4]) -> Node {
    let mut node = Node::new(Role::Button);
    node.set_label(name);
    node.set_bounds(Rect::new(bounds[0], bounds[1], bounds[2], bounds[3]));
    node.add_action(Action::Click);
    node.add_action(Action::Focus);
    node
}

pub(crate) fn control_id(control: FrameControl) -> NodeId {
    match control {
        FrameControl::Minimize => MINIMIZE,
        FrameControl::Maximize => MAXIMIZE,
        FrameControl::Close => CLOSE,
    }
}

pub(crate) fn control_for_id(id: NodeId) -> Option<FrameControl> {
    match id {
        MINIMIZE => Some(FrameControl::Minimize),
        MAXIMIZE => Some(FrameControl::Maximize),
        CLOSE => Some(FrameControl::Close),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn controls_have_names_bounds_actions_and_restored_state() {
        let mut frame = FrameSemantics {
            title: "Part1".into(),
            maximized: false,
            bounds: [[1., 2., 3., 4.]; 3],
            focused: Some(FrameControl::Close),
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
        assert_eq!(control_for_id(MAXIMIZE), Some(FrameControl::Maximize));
        assert_eq!(control_for_id(NodeId(42)), None);
    }
}
