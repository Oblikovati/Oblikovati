// SPDX-License-Identifier: GPL-2.0-only
//! Validated radial menus shared by the desktop and public API.
use oblikovati_api::{
    types::{self, Environment, ScreenQuadrant},
    wire::{MarkingMenuItem, MarkingMenuView},
};
use std::collections::{BTreeMap, BTreeSet};

#[path = "marking_menu_display.rs"]
mod display;
pub use display::{MarkingMenuDisplay, MarkingMenuOverflow, MarkingMenuStyle};

/// Storage for a user's marking-menu assignments.
pub trait MarkingMenuPreferenceAccess {
    /// Read the classic context-menu choice; example: `store.classic()`.
    fn classic(&self) -> bool;
    /// Read the saved menus; example: `store.menus()`.
    fn menus(&self) -> Vec<MarkingMenuView>;
    /// Read the saved presentation choices and classic-menu preference.
    fn display(&self) -> MarkingMenuDisplay {
        MarkingMenuDisplay {
            classic: self.classic(),
            ..Default::default()
        }
    }
    /// Save validated menus; example: `store.save(&menus)?`.
    fn save(&mut self, menus: &MarkingMenus) -> Result<(), String>;
    /// Save assignments and the classic-menu choice together.
    fn save_with_classic(&mut self, menus: &MarkingMenus, classic: bool) -> Result<(), String> {
        if classic != self.classic() {
            return Err("marking menu storage does not support changing the classic choice".into());
        }
        self.save(menus)
    }
    /// Save the menu assignments and presentation choices together.
    fn save_with_display(
        &mut self,
        menus: &MarkingMenus,
        display: MarkingMenuDisplay,
    ) -> Result<(), String> {
        let current = self.display();
        if current.style != display.style || current.overflow != display.overflow {
            return Err("marking menu storage does not support changing the presentation".into());
        }
        self.save_with_classic(menus, display.classic)
    }
}

/// Per-environment radial command assignments.
#[derive(Clone, Debug, PartialEq)]
pub struct MarkingMenus {
    menus: BTreeMap<u8, MarkingMenuView>,
}
impl Default for MarkingMenus {
    fn default() -> Self {
        let menus = [base_menu(), sketch_menu(), sketch_3d_menu()]
            .into_iter()
            .map(|menu| (menu.environment.0, menu))
            .collect();
        Self { menus }
    }
}
impl MarkingMenus {
    /// Copy configured environments in stable order; example: `menus.all()`.
    pub fn all(&self) -> Vec<MarkingMenuView> {
        self.menus.values().cloned().collect()
    }
    /// Read one environment, including empty unconfigured environments.
    /// Example: `menus.get(types::SKETCH_ENVIRONMENT)`.
    pub fn get(&self, environment: Environment) -> MarkingMenuView {
        self.menus
            .get(&environment.0)
            .cloned()
            .unwrap_or(MarkingMenuView {
                environment,
                ..Default::default()
            })
    }
    /// Replace one menu after validating every radial slot.
    /// Example: `menus.set(view)?`.
    pub fn set(&mut self, menu: MarkingMenuView) -> Result<(), String> {
        validate(&menu)?;
        self.menus.insert(menu.environment.0, menu);
        Ok(())
    }
}
fn validate(menu: &MarkingMenuView) -> Result<(), String> {
    let mut seen = BTreeSet::new();
    for item in menu.quadrants.iter().flatten() {
        if item.quadrant.0 > types::QUADRANT_NORTH_WEST.0 || !seen.insert(item.quadrant.0) {
            return Err(format!(
                "marking menu quadrant {}: expected a unique value in 0..=7",
                item.quadrant.0
            ));
        }
    }
    Ok(())
}
fn menu(environment: Environment, slots: &[(ScreenQuadrant, &str)]) -> MarkingMenuView {
    MarkingMenuView {
        environment,
        quadrants: Some(
            slots
                .iter()
                .map(|(quadrant, id)| MarkingMenuItem {
                    quadrant: *quadrant,
                    command_id: (*id).into(),
                })
                .collect(),
        ),
        overflow: None,
    }
}
fn base_menu() -> MarkingMenuView {
    let mut part = menu(
        types::BASE_ENVIRONMENT,
        &[
            (types::QUADRANT_NORTH, "Modify.Fillet"),
            (types::QUADRANT_NORTH_EAST, "Create.Extrude"),
            (types::QUADRANT_EAST, "Create.Revolve"),
            (types::QUADRANT_SOUTH_EAST, "Modify.Hole"),
            (types::QUADRANT_SOUTH, "Sketch.Create2D"),
            (types::QUADRANT_SOUTH_WEST, "WorkPlane.Offset"),
            (types::QUADRANT_WEST, "edit.undo"),
            (types::QUADRANT_NORTH_WEST, "Inspect.Measure"),
        ],
    );
    part.overflow = Some(vec!["Modify.Chamfer".into()]);
    part
}
fn sketch_menu() -> MarkingMenuView {
    menu(
        types::SKETCH_ENVIRONMENT,
        &[
            (types::QUADRANT_NORTH, "Sketch.Line"),
            (types::QUADRANT_NORTH_EAST, "Sketch.Dimension"),
            (types::QUADRANT_EAST, "Sketch.Circle"),
            (types::QUADRANT_SOUTH, "Sketch.Rectangle"),
            (types::QUADRANT_WEST, "Sketch.Arc"),
            (types::QUADRANT_NORTH_WEST, "Sketch.Finish"),
        ],
    )
}

fn sketch_3d_menu() -> MarkingMenuView {
    menu(
        types::SKETCH3_D_ENVIRONMENT,
        &[
            (types::QUADRANT_NORTH, "Sketch3D.Line"),
            (types::QUADRANT_NORTH_EAST, "Sketch3D.Arc"),
            (types::QUADRANT_EAST, "Sketch3D.Helix"),
            (types::QUADRANT_SOUTH_EAST, "Sketch3D.Include"),
            (types::QUADRANT_SOUTH, "Sketch3D.Finish"),
            (types::QUADRANT_SOUTH_WEST, "Sketch3D.Dimension"),
            (types::QUADRANT_WEST, "edit.undo"),
            (types::QUADRANT_NORTH_WEST, "Sketch3D.Bend"),
        ],
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn go_sketch_defaults_and_unknown_environment_are_distinct() {
        let menus = MarkingMenus::default();
        let sketch = menus.get(types::SKETCH_ENVIRONMENT);
        assert!(
            sketch
                .quadrants
                .unwrap()
                .iter()
                .any(|slot| slot.quadrant == types::QUADRANT_SOUTH
                    && slot.command_id == "Sketch.Rectangle")
        );
        assert!(menus.get(Environment(99)).quadrants.is_none());
    }
    #[test]
    fn invalid_replacement_leaves_previous_menu_intact() {
        let mut menus = MarkingMenus::default();
        let before = menus.get(types::SKETCH_ENVIRONMENT);
        for slots in [vec![(0, "one"), (0, "two")], vec![(8, "bad")]] {
            let invalid = menu(
                types::SKETCH_ENVIRONMENT,
                &slots
                    .into_iter()
                    .map(|(slot, id)| (ScreenQuadrant(slot), id))
                    .collect::<Vec<_>>(),
            );
            assert!(menus.set(invalid).is_err());
            assert_eq!(menus.get(types::SKETCH_ENVIRONMENT), before);
        }
        let replacement = menu(
            types::SKETCH_ENVIRONMENT,
            &[(types::QUADRANT_WEST, "future.command")],
        );
        menus.set(replacement.clone()).unwrap();
        assert_eq!(menus.get(types::SKETCH_ENVIRONMENT), replacement);
    }
}
