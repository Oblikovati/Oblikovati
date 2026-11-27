# SPDX-License-Identifier: GPL-2.0-only
param(
    [Parameter(Mandatory)][string] $DriverManifest,
    [string] $ValidationManifest
)

$ErrorActionPreference = 'Stop'
if ($env:GITHUB_ACTIONS -ne 'true') {
    throw 'Vulkan machine registration is only used on the disposable Windows CI runner'
}

function Register-VulkanManifest {
    param([Microsoft.Win32.RegistryKey] $Registry, [string] $Kind, [string] $Manifest)
    $key = $Registry.CreateSubKey("SOFTWARE\Khronos\Vulkan\$Kind")
    try { $key.SetValue($Manifest, 0, [Microsoft.Win32.RegistryValueKind]::DWord) }
    finally { $key.Dispose() }
}

$driver = (Get-Item -LiteralPath $DriverManifest).FullName
$validation = if ($ValidationManifest) { (Get-Item -LiteralPath $ValidationManifest).FullName }
# The Vulkan loader ignores per-process manifest overrides on elevated hosted runners.
$registry = [Microsoft.Win32.RegistryKey]::OpenBaseKey('LocalMachine', 'Registry64')
try {
    Register-VulkanManifest $registry 'Drivers' $driver
    if ($validation) { Register-VulkanManifest $registry 'ExplicitLayers' $validation }
} finally { $registry.Dispose() }
