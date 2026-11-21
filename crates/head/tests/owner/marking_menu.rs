// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[derive(Clone, Default)]
struct FakeMarkingStore {
    saved: std::sync::Arc<std::sync::Mutex<Vec<MarkingMenuView>>>,
    fail: std::sync::Arc<std::sync::atomic::AtomicBool>,
    classic: bool,
}
impl oblikovati_model::marking_menu::MarkingMenuPreferenceAccess for FakeMarkingStore {
    fn classic(&self) -> bool {
        self.classic
    }
    fn menus(&self) -> Vec<MarkingMenuView> {
        self.saved.lock().unwrap().clone()
    }
    fn save(&mut self, menus: &oblikovati_model::marking_menu::MarkingMenus) -> Result<(), String> {
        if self.fail.load(std::sync::atomic::Ordering::SeqCst) {
            return Err("marking menu write: expected writable preferences".into());
        }
        *self.saved.lock().unwrap() = menus.all();
        Ok(())
    }
}
#[test]
fn marking_menu_save_failure_preserves_live_and_saved_state() {
    use oblikovati_api::types;
    let store = FakeMarkingStore::default();
    let (desktop, awake) = owner();
    let mut desktop = desktop
        .with_marking_menu_preferences(Box::new(store.clone()))
        .unwrap();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Rectangle".into(),
        }]),
        overflow: None,
    };
    let saved = menu.clone();
    invoke(&mut desktop, &awake, move |client| {
        client.ui().set_marking_menu(&SetMarkingMenuArgs { menu })
    })
    .unwrap();
    store.fail.store(true, std::sync::atomic::Ordering::SeqCst);
    let mut changed = saved.clone();
    changed.quadrants.as_mut().unwrap()[0].command_id = "Sketch.Line".into();
    assert!(
        Client::new(desktop.document_host())
            .ui()
            .set_marking_menu(&SetMarkingMenuArgs { menu: changed })
            .is_err()
    );
    let current = Client::new(desktop.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert_eq!(current, saved);
    let (fresh, _) = owner();
    let mut fresh = fresh
        .with_marking_menu_preferences(Box::new(store))
        .unwrap();
    let restored = Client::new(fresh.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert_eq!(restored, saved);
}
