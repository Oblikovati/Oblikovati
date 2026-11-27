# SPDX-License-Identifier: GPL-2.0-only
param(
    [Parameter(Mandatory)][string] $NativeLog,
    [Parameter(Mandatory)][ValidateSet('arm64', 'x64')][string] $Architecture
)

$ErrorActionPreference = 'Stop'
$test = 'application::document_creation::tests::opening_the_chooser_does_not_create_a_document_and_escape_cancels'
$binary = 'oblikovati_gui'
$exception = 'c0000409'
$exceptionMode = 'sxe'
if ($Architecture -eq 'x64') {
    $test = 'native::gui_composition_tests::native_owned_imgui_packet_composes_with_the_document_after_context_destruction'
    $binary = 'oblikovati_head'
    $exception = 'c000001d'
    # Let the application handle first-chance exceptions before recording an unhandled fault.
    $exceptionMode = 'sxd'
}
$testPattern = '^test ' + [regex]::Escape($test) + ' \.\.\.(?!\s*(?:ok|FAILED|ignored)\b)'
$abortPattern = '(?<executable>[A-Z]:[^\x60]+[\\/]' + $binary + '-[a-f0-9]+\.exe)[^\x60]*\x60 \(exit code: 0x' + $exception
$abort = Select-String -LiteralPath $NativeLog -Pattern $abortPattern | Select-Object -First 1
if ($null -eq $abort -or -not (Select-String -LiteralPath $NativeLog -Pattern $testPattern -Quiet)) {
    Write-Host "The native test log has no matching $test crash."
    return
}
$executable = $abort.Matches[0].Groups['executable'].Value
if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
    throw "Expected the native GUI test executable at $executable"
}

. "$PSScriptRoot/windows_process.ps1"
$hostAccess = [BuildBootstrapHost]::new()
$debugger = Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\Debuggers\$Architecture\cdb.exe"
if (-not (Test-Path -LiteralPath $debugger -PathType Leaf)) {
    # The runner starts with the SDK and may leave its debugger component optional.
    $sdk = Get-ItemProperty 'HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like 'Windows Software Development Kit*' -and $_.BundleCachePath } |
        Sort-Object DisplayVersion -Descending |
        Select-Object -First 1
    if ($null -eq $sdk) { throw "Expected a cached Windows SDK installer for the $Architecture debugger" }
    $publisher = $hostAccess.MicrosoftSigner($sdk.BundleCachePath)
    if ($publisher -notmatch '(?:^|,\s*)O=Microsoft Corporation(?:,|$)') {
        throw "Expected Microsoft Corporation as the SDK installer signer, got $publisher"
    }
    $result = $hostAccess.Run($sdk.BundleCachePath, @('/features', 'OptionId.WindowsDesktopDebuggers', '/q', '/norestart'))
    if ($result -notin 0, 3010 -or -not (Test-Path -LiteralPath $debugger -PathType Leaf)) {
        throw "Expected $Architecture CDB at $debugger after SDK installer exit $result"
    }
}

$commands = Join-Path $env:RUNNER_TEMP 'oblikovati-gui-debug.txt'
$breakpoints = @()
if ($Architecture -eq 'arm64') { $breakpoints += 'bu ucrtbase!abort "kv 40; q"' }
($breakpoints + @(
    # Record the failing thread even when the exception arrives only at second chance.
    "$exceptionMode -c2 `"kv 40; q`" $exception"
    'g'
)) | Set-Content -LiteralPath $commands -Encoding ascii
$symbols = Split-Path -Parent $executable
$result = $hostAccess.Run($debugger, @('-G', '-y', $symbols, '-cf', $commands, $executable, $test, '--exact', '--nocapture', '--test-threads=1'))
if ($result -ne 0) { throw "$Architecture CDB exited with status $result" }
