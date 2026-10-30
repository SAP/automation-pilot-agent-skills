# Verify the Windows binary without executing it or loading user PowerShell profiles.
$ErrorActionPreference = 'Stop'
try {
    $target = 'windows-amd64/autopi.exe'
    $binary = Join-Path $PSScriptRoot 'bin\windows-amd64\autopi.exe'
    $sums = Join-Path $PSScriptRoot 'bin\SHA256SUMS'
    $selectedHashes = @()
    foreach ($line in [System.IO.File]::ReadAllLines($sums)) {
        if ($line -cnotmatch '^([0-9a-f]{64})[ \t]+(\S+)$') {
            throw 'SHA256SUMS contains an invalid entry'
        }
        if ($Matches[2] -ceq $target) {
            $selectedHashes += $Matches[1]
        }
    }
    if ($selectedHashes.Count -ne 1) {
        throw "SHA256SUMS must contain one valid entry for $target"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $binary).Hash.ToLowerInvariant()
    if ($actual -cne $selectedHashes[0]) {
        throw "SHA-256 mismatch for $target"
    }
} catch {
    [Console]::Error.WriteLine('autopi: binary verification failed: ' + $_.Exception.Message)
    [Console]::Error.WriteLine('        restore the complete bundle from a trusted source; do not bypass this check')
    exit 1
}
exit 0
