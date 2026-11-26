# SPDX-License-Identifier: GPL-2.0-only
param([Parameter(Mandatory)][string] $NativeLog)

$ErrorActionPreference = 'Stop'
$test = 'application::document_creation::tests::opening_the_chooser_does_not_create_a_document_and_escape_cancels'
$testPattern = '^test ' + [regex]::Escape($test) + ' \.\.\.\s*(?:$|Assertion failed:)'
$abortPattern = '(?<executable>[A-Z]:[^\x60]+[\\/]oblikovati_gui-[a-f0-9]+\.exe)[^\x60]*\x60 \(exit code: 0xc0000409'
$abort = Select-String -LiteralPath $NativeLog -Pattern $abortPattern | Select-Object -First 1
if ($null -eq $abort -or -not (Select-String -LiteralPath $NativeLog -Pattern $testPattern -Quiet)) {
    Write-Host 'The native test log has no matching New dialog abort.'
    return
}
$executable = $abort.Matches[0].Groups['executable'].Value
if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
    throw "Expected the native GUI test executable at $executable"
}

. "$PSScriptRoot/windows_process.ps1"
$hostAccess = [BuildBootstrapHost]::new()
$debugger = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\Debuggers\arm64\cdb.exe'
if (-not (Test-Path -LiteralPath $debugger -PathType Leaf)) {
    # The ARM64 runner starts with the SDK and leaves its debugger component optional.
    $sdk = Get-ItemProperty 'HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like 'Windows Software Development Kit*' -and $_.BundleCachePath } |
        Sort-Object DisplayVersion -Descending |
        Select-Object -First 1
    if ($null -eq $sdk) { throw 'Expected a cached Windows SDK installer for the ARM64 debugger' }
    $publisher = $hostAccess.MicrosoftSigner($sdk.BundleCachePath)
    if ($publisher -notmatch '(?:^|,\s*)O=Microsoft Corporation(?:,|$)') {
        throw "Expected Microsoft Corporation as the SDK installer signer, got $publisher"
    }
    $result = $hostAccess.Run($sdk.BundleCachePath, @('/features', 'OptionId.WindowsDesktopDebuggers', '/q', '/norestart'))
    if ($result -notin 0, 3010 -or -not (Test-Path -LiteralPath $debugger -PathType Leaf)) {
        throw "Expected ARM64 CDB at $debugger after SDK installer exit $result"
    }
}

$commands = Join-Path $env:RUNNER_TEMP 'oblikovati-gui-debug.txt'
@(
    'bu ucrtbase!_wassert "kv 40; q"'
    'sxe -c "kv 40; q" c0000409'
    'g'
) | Set-Content -LiteralPath $commands -Encoding ascii
$symbols = Split-Path -Parent $executable
$result = $hostAccess.Run($debugger, @('-G', '-y', $symbols, '-cf', $commands, $executable, $test, '--exact', '--nocapture', '--test-threads=1'))
if ($result -ne 0) { throw "ARM64 CDB exited with status $result" }
