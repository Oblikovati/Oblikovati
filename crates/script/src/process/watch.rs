// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn start(
    child: Arc<Mutex<Child>>,
    finished: Arc<AtomicBool>,
    stop: ScriptStop,
    deadline: Instant,
) -> JoinHandle<()> {
    thread::spawn(move || {
        while !finished.load(Ordering::Acquire) {
            if stop.0.load(Ordering::Acquire) || Instant::now() >= deadline {
                terminate(&child);
                break;
            }
            thread::sleep(Duration::from_millis(2));
        }
    })
}
pub(super) fn terminate(child: &Mutex<Child>) {
    if let Ok(mut child) = child.lock() {
        let _ = child.kill();
        let _ = child.wait();
    }
}
