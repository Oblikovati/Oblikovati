import json
import sys
from dataclasses import replace
from pathlib import Path
sys.path.insert(0, '/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/tools/bdd')
from local_snapshot_evidence import committed_object, scoped_selection
from pinned_runner import PinnedRunnerProcess
from sandbox_build_probe import runner_path, desktop_path
from sealed_execution import SealedRequest, file_identity, run_sealed
from sealed_report_check import check_sealed_report
root = Path('/tmp/obk-sealed-docking-mM1ojG')
repository = Path('/home/vmiguel/git/oblikovati-workspace/oblikovati-rust')
revision = '5909058cfb127689b3176978f761e1e819fba910'
receipt = 'architecture/rust-port/receipts/2026-11-20-docking-local-snapshot-3978'
runner = runner_path(root/'build-logs/cargo.jsonl', root/'target')
desktop = desktop_path(root/'build-logs/desktop.jsonl', root/'target')
request = SealedRequest(root/'runtime', runner, root/'scenarios', root/'reports', root/'native-retry5-proc.log', root/'bwrap', 'application', 'docking-system', 'Restore the saved panel arrangement after restarting the application', desktop)
metadata = committed_object(repository, revision, receipt, 'snapshot.json')
inventory = committed_object(repository, revision, receipt, 'inventory.json')
row, verify = scoped_selection(request, metadata, inventory)
runner_hash = file_identity(runner)['sha256']
desktop_hash = file_identity(desktop)['sha256']
process = PinnedRunnerProcess(runner, runner_hash, desktop=(desktop, desktop_hash))
def proc_diagnostic(command, log):
    index = command.index("--chdir")
    return process(command[:index] + ["--proc", "/proc"] + command[index:], log)
result = run_sealed(request, proc_diagnostic, verify)
result['report_check'] = check_sealed_report(request, row, verify(request.checkout), None)
print(json.dumps(result, sort_keys=True))
