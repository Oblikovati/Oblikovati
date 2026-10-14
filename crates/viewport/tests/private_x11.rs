// SPDX-License-Identifier: GPL-2.0-only
use rustix::event::{PollFd, PollFlags, Timespec, poll};
use std::io::{BufRead, BufReader};
use std::process::{Child, Command, Stdio};
pub(super) struct PrivateX11Server {
    child: Child,
    display: String,
}
impl PrivateX11Server {
    pub(super) fn open_compose_fixture() -> Result<Self, String> {
        let program = std::env::var_os("OBLIKOVATI_XVFB_BINARY").unwrap_or_else(|| "Xvfb".into());
        let child = Command::new(&program).args(["-displayfd", "1", "-screen", "0", "640x480x24", "-nolisten", "tcp", "-noreset"])
            .stdin(Stdio::null()).stdout(Stdio::piped()).stderr(Stdio::null()).spawn()
            .map_err(|cause| format!("private X11 executable {program:?}: expected Xvfb (or OBLIKOVATI_XVFB_BINARY): {cause}"))?;
        let mut server = Self {
            child,
            display: String::new(),
        };
        server.display = server.await_display()?;
        server.configure_compose_map()?;
        Ok(server)
    }
    fn await_display(&mut self) -> Result<String, String> {
        let output = self
            .child
            .stdout
            .take()
            .ok_or("private Xvfb stdout: expected an owned display-number pipe")?;
        require_display_pipe_readiness(&output)?;
        let mut line = String::new();
        BufReader::new(output)
            .read_line(&mut line)
            .map_err(|cause| format!("private Xvfb display line: expected an integer: {cause}"))?;
        let number: u32 = line.trim().parse().map_err(|cause| {
            format!("private Xvfb display {line:?}: expected a nonnegative display number: {cause}")
        })?;
        Ok(format!(":{number}"))
    }
    fn configure_compose_map(&self) -> Result<(), String> {
        let result = Command::new("setxkbmap")
            .args([
                "-display",
                &self.display,
                "-layout",
                "us",
                "-variant",
                "intl",
                "-option",
                "",
            ])
            .output()
            .map_err(|cause| format!("private XKB mapping: expected setxkbmap: {cause}"))?;
        if !result.status.success() {
            return Err(format!(
                "private XKB mapping {} {:?}: expected US international compose fixture",
                result.status, result.stderr
            ));
        }
        Ok(())
    }
    pub(super) fn display(&self) -> &str {
        &self.display
    }
}
impl Drop for PrivateX11Server {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

fn require_display_pipe_readiness(output: &std::process::ChildStdout) -> Result<(), String> {
    let timeout = Timespec::try_from(std::time::Duration::from_secs(3)).unwrap();
    let mut descriptors = [PollFd::new(output, PollFlags::IN)];
    poll(&mut descriptors, Some(&timeout))
        .map_err(|cause| format!("private Xvfb display readiness: expected live pipe: {cause}"))?;
    if !descriptors[0].revents().contains(PollFlags::IN) {
        return Err(format!(
            "private Xvfb pipe {:?}: expected a ready display within three seconds",
            descriptors[0].revents()
        ));
    }
    Ok(())
}
