// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl ScriptExecution {
    pub(super) fn receive<C: Caller>(
        &mut self,
        blocking: bool,
        host: &C,
        output: &mut dyn Write,
    ) -> Result<Option<ScriptReport>, String> {
        let next = self.next_frame(blocking);
        if let Err(error) = self.require_live() {
            return self.fail(error);
        }
        match next {
            Ok(Some(frame)) => self.event(frame, host, output),
            Ok(None) => Ok(None),
            Err(error) => self.fail(error),
        }
    }
    fn next_frame(&self, blocking: bool) -> Result<Option<WorkerFrame>, String> {
        if !blocking {
            return match self.incoming.try_recv() {
                Ok(result) => result.map(Some),
                Err(mpsc::TryRecvError::Empty) => Ok(None),
                Err(mpsc::TryRecvError::Disconnected) => {
                    Err("worker stream disconnected: expected Finished".into())
                }
            };
        }
        match self.incoming.recv_timeout(Duration::from_millis(2)) {
            Ok(result) => result.map(Some),
            Err(mpsc::RecvTimeoutError::Timeout) => Ok(None),
            Err(mpsc::RecvTimeoutError::Disconnected) => {
                Err("worker stream disconnected: expected Finished".into())
            }
        }
    }
}
