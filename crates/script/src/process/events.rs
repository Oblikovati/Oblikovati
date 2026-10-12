// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::panic::{AssertUnwindSafe, catch_unwind};
impl ScriptExecution {
    pub(super) fn event<C: Caller>(
        &mut self,
        frame: WorkerFrame,
        host: &C,
        output: &mut dyn Write,
    ) -> Result<Option<ScriptReport>, String> {
        match frame {
            WorkerFrame::Call { method, arguments } => self.host_call(method, arguments, host),
            WorkerFrame::Print { line } => self.print_line(line, output),
            WorkerFrame::Finished { outcome } => {
                self.finish();
                outcome.map(Some)
            }
        }
    }
    fn host_call<C: Caller>(
        &mut self,
        method: String,
        arguments: Vec<u8>,
        host: &C,
    ) -> Result<Option<ScriptReport>, String> {
        self.host_calls += 1;
        if self.host_calls > self.limits.host_calls {
            return self.fail(format!(
                "host calls {}: expected at most {}",
                self.host_calls, self.limits.host_calls
            ));
        }
        let reply = match invoke_host(host, &method, &arguments) {
            Some(reply) => reply,
            None => return self.fail(format!("host method {method:?} panicked: expected a normal result; new script calls stopped")),
        };
        if let Err(error) = self.require_live() {
            return self.fail(error);
        }
        if let Err(error) = self.write_reply(reply) {
            return self.fail(error);
        }
        Ok(None)
    }
    fn write_reply(&self, reply: Result<Vec<u8>, String>) -> Result<(), String> {
        self.worker
            .input
            .lock()
            .map_err(|_| "worker input poisoned: expected a live pipe".to_owned())
            .and_then(|mut input| write_frame(&mut *input, &ParentFrame::Reply { outcome: reply }))
    }
    fn print_line(
        &mut self,
        line: String,
        output: &mut dyn Write,
    ) -> Result<Option<ScriptReport>, String> {
        self.output_bytes = self.output_bytes.saturating_add(line.len() + 1);
        if self.output_bytes > self.limits.output_bytes {
            return self.fail(format!(
                "script output {} bytes: expected at most {}",
                self.output_bytes, self.limits.output_bytes
            ));
        }
        if let Err(error) = writeln!(output, "{line}") {
            return self.fail(format!(
                "script print sink: expected writable output: {error}"
            ));
        }
        Ok(None)
    }
}

fn invoke_host<C: Caller>(
    host: &C,
    method: &str,
    arguments: &[u8],
) -> Option<Result<Vec<u8>, String>> {
    catch_unwind(AssertUnwindSafe(|| host.call(method, arguments)))
        .ok()
        .map(|reply| reply.map_err(|error| error.0))
}
