// SPDX-License-Identifier: GPL-2.0-only
//! Track explicit tool-window visibility independently of whether a parent is drawn.
use std::cell::{Cell, RefCell};
use std::collections::BTreeMap;
use std::rc::Rc;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum DockVisibility {
    #[default]
    Closed,
    Open,
    Suspended,
}

#[derive(Debug, Default)]
struct DockStateValue {
    visibility: Cell<DockVisibility>,
    close_pending: Cell<bool>,
    title: RefCell<Option<std::ffi::CString>>,
}

/// A window owner retains this state even while its UI is suspended.
#[derive(Clone, Debug, Default)]
pub(crate) struct DockableState(Rc<DockStateValue>);
impl DockableState {
    pub(crate) fn is_open(&self) -> bool {
        self.0.visibility.get() == DockVisibility::Open
    }
    pub(crate) fn set_open(&self, open: bool) {
        self.set_visibility(if open {
            DockVisibility::Open
        } else {
            DockVisibility::Closed
        });
    }
    pub(crate) fn set_title(&self, title: &str) -> Result<(), String> {
        if self
            .0
            .title
            .borrow()
            .as_ref()
            .is_some_and(|current| current.as_bytes() == title.as_bytes())
        {
            return Ok(());
        }
        *self.0.title.borrow_mut() = Some(validate_title(title)?);
        Ok(())
    }
    pub(crate) fn set_suspended(&self, suspended: bool) {
        match (self.0.visibility.get(), suspended) {
            (DockVisibility::Open, true) => self.set_visibility(DockVisibility::Suspended),
            (DockVisibility::Suspended, false) => self.set_visibility(DockVisibility::Open),
            _ => {}
        }
    }
    pub(crate) fn set_visibility(&self, next: DockVisibility) {
        let previous = self.0.visibility.replace(next);
        if next == DockVisibility::Closed && previous != DockVisibility::Closed {
            self.0.close_pending.set(true);
        } else if next != DockVisibility::Closed {
            self.0.close_pending.set(false);
        }
    }
}

struct DockRegistryEntry {
    title: std::ffi::CString,
    state: Rc<DockStateValue>,
    was_open: bool,
    #[cfg(test)]
    owner_type: &'static str,
}

/// The GUI owner records closed windows after all controls finish drawing.
#[derive(Default)]
pub(crate) struct DockRegistry {
    windows: Vec<DockRegistryEntry>,
}
impl DockRegistry {
    pub(crate) fn register<T: crate::dockable_window::DockableWindow + 'static>(
        &mut self,
        title: &str,
        state: &DockableState,
    ) -> Result<(), String> {
        let title = validate_title(title)?;
        let state = Rc::clone(&state.0);
        if self
            .windows
            .iter()
            .any(|entry| Rc::ptr_eq(&entry.state, &state))
        {
            return Err(format!(
                "dockable window {title:?}: expected one registration per owner"
            ));
        }
        self.windows.push(DockRegistryEntry {
            title,
            state,
            was_open: false,
            #[cfg(test)]
            owner_type: std::any::type_name::<T>(),
        });
        Ok(())
    }
    pub(crate) fn reconcile(
        &mut self,
        mut closed: impl FnMut(&std::ffi::CStr),
    ) -> Result<bool, String> {
        let active = self.active_identities()?;
        let mut changed = false;
        self.windows.retain_mut(|entry| {
            let dropped = Rc::strong_count(&entry.state) == 1;
            let state = &entry.state;
            let old_identity = crate::widget_abi::dock_window_identity(&entry.title);
            let next_title = state.title.borrow().clone();
            let next_identity = next_title
                .as_ref()
                .map(|title| crate::widget_abi::dock_window_identity(title));
            let renamed = entry.was_open && next_identity.is_some_and(|id| id != old_identity);
            let pending = state.close_pending.replace(false);
            let dropped_open =
                dropped && (entry.was_open || state.visibility.get() != DockVisibility::Closed);
            if renamed && !active.contains_key(&old_identity) {
                closed(&entry.title);
                changed = true;
            }
            let final_title = next_title.as_ref().unwrap_or(&entry.title);
            let final_identity = crate::widget_abi::dock_window_identity(final_title);
            if (pending || dropped_open) && !active.contains_key(&final_identity) {
                closed(final_title);
                changed = true;
            }
            if let Some(title) = next_title {
                entry.title = title;
            }
            entry.was_open = state.visibility.get() != DockVisibility::Closed;
            !dropped
        });
        Ok(changed)
    }
    #[cfg(test)]
    pub(crate) fn owner_instance_counts(&self) -> BTreeMap<&'static str, usize> {
        let mut counts = BTreeMap::new();
        for entry in &self.windows {
            let owner = entry
                .owner_type
                .rsplit("::")
                .next()
                .unwrap_or(entry.owner_type);
            *counts.entry(owner).or_insert(0) += 1;
        }
        counts
    }
    #[cfg(test)]
    pub(crate) fn owner_types(&self) -> std::collections::BTreeSet<&'static str> {
        self.windows
            .iter()
            .map(|entry| {
                entry
                    .owner_type
                    .rsplit("::")
                    .next()
                    .unwrap_or(entry.owner_type)
            })
            .collect()
    }
    fn active_identities(&self) -> Result<BTreeMap<u32, usize>, String> {
        let mut active = BTreeMap::new();
        for (index, entry) in self.windows.iter().enumerate() {
            let state = &entry.state;
            if Rc::strong_count(state) == 1 || state.visibility.get() == DockVisibility::Closed {
                continue;
            }
            let title = state.title.borrow();
            let title = title.as_ref().unwrap_or(&entry.title);
            let identity = crate::widget_abi::dock_window_identity(title);
            if let Some(previous) = active.insert(identity, index) {
                return Err(format!(
                    "dockable window title {:?}, identity {identity:#x}: expected unique active owners, found {previous} and {index}",
                    title.to_string_lossy()
                ));
            }
        }
        Ok(active)
    }
}
fn validate_title(title: &str) -> Result<std::ffi::CString, String> {
    std::ffi::CString::new(title)
        .map_err(|_| format!("dockable window title {title:?}: expected NUL-free text"))
}

#[cfg(test)]
#[path = "../tests/dockable_lifecycle/unit.rs"]
mod tests;
