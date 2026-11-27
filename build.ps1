# SPDX-License-Identifier: GPL-2.0-only
#requires -Version 5.1
[CmdletBinding()]
param(
    [ValidateSet('build', 'run', 'check', 'help')][string] $Mode = 'build',
    [Parameter(ValueFromRemainingArguments = $true)][string[]] $GuiArguments = @()
)
$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/tools/bootstrap/windows_process.ps1"
. "$PSScriptRoot/tools/bootstrap/windows_dependencies.ps1"
. "$PSScriptRoot/tools/bootstrap/windows_build.ps1"
try {
    $hostAccess = [BuildBootstrapHost]::new()
    exit (Invoke-OblikovatiBootstrap -Root $PSScriptRoot -Mode $Mode -GuiArguments $GuiArguments -HostAccess $hostAccess)
} catch {
    [Console]::Error.WriteLine("build.ps1: $($_.Exception.Message)")
    exit 1
}
