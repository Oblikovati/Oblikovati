// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::protocol::read_frame;
use std::io::{BufReader, Read};
pub(super) fn start(
    child: &mut Child,
    events: mpsc::SyncSender<Result<WorkerFrame, String>>,
) -> Result<Vec<JoinHandle<()>>, String> {
    let output = child
        .stdout
        .take()
        .ok_or("worker stdout: expected piped protocol")?;
    let errors = child
        .stderr
        .take()
        .ok_or("worker stderr: expected piped diagnostics")?;
    let protocol = thread::spawn(move || receive_protocol(BufReader::new(output), events));
    let diagnostics = thread::spawn(move || {
        let mut collected = Vec::new();
        let _ = errors.take(64 << 10).read_to_end(&mut collected);
    });
    Ok(vec![protocol, diagnostics])
}
fn receive_protocol(
    mut reader: impl std::io::BufRead,
    events: mpsc::SyncSender<Result<WorkerFrame, String>>,
) {
    loop {
        let frame = match next_protocol(&mut reader) {
            Ok(frame) => frame,
            Err(error) => {
                let _ = events.send(Err(error));
                break;
            }
        };
        if matches!(frame, WorkerFrame::Finished { .. }) {
            let outcome = finish_stream(&mut reader, frame);
            let _ = events.send(outcome);
            break;
        }
        if events.send(Ok(frame)).is_err() {
            break;
        }
    }
}

fn finish_stream(
    reader: &mut impl std::io::BufRead,
    frame: WorkerFrame,
) -> Result<WorkerFrame, String> {
    // Wait for process EOF so natural shutdown/profile flushing completes; the parent deadline still guards a stuck worker.
    match read_frame::<WorkerFrame>(reader)? {
        None => Ok(frame),
        Some(_) => Err("worker frame after Finished: expected immediate stream EOF".into()),
    }
}

fn next_protocol(reader: &mut impl std::io::BufRead) -> Result<WorkerFrame, String> {
    read_frame(reader)?.ok_or_else(|| "worker exited: expected Finished frame".into())
}
