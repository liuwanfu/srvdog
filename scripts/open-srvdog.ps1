Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$HostName = "107.174.48.241"
$Port = 45678
$User = "root"
$LocalHost = "127.0.0.1"
$LocalPort = 8090
$RemoteHost = "127.0.0.1"
$RemotePort = 8090
$Url = "http://127.0.0.1:8090"
$IdentityFile = $env:SRVDOG_IDENTITY_FILE

if (-not $IdentityFile) {
    $defaultIdentity = Join-Path $HOME ".ssh\id_ed25519_racknerd_107_174_48_241"
    if (Test-Path $defaultIdentity) {
        $IdentityFile = $defaultIdentity
    }
}

function Test-PortInUse {
    param([int]$PortNumber)

    $listeners = [System.Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().GetActiveTcpListeners()
    foreach ($listener in $listeners) {
        if ($listener.Port -eq $PortNumber) {
            return $true
        }
    }
    return $false
}

$sshCommand = Get-Command ssh -ErrorAction SilentlyContinue
if (-not $sshCommand) {
    Write-Error "ssh was not found in PATH. Install OpenSSH client or add it to PATH."
}

if (Test-PortInUse -PortNumber $LocalPort) {
    Write-Host "Local port $LocalPort is already in use. Close the conflicting process or change the launcher config." -ForegroundColor Red
    exit 1
}

$sshArgs = @("-N")
if ($IdentityFile) {
    $sshArgs += @("-i", $IdentityFile)
}
$sshArgs += @(
    "-o", "IdentitiesOnly=yes",
    "-L", "$LocalHost`:$LocalPort`:$RemoteHost`:$RemotePort",
    "-p", "$Port",
    "$User@$HostName"
)

Write-Host "Opening srvdog tunnel on $Url"
Write-Host "Target: ${User}@${HostName}:${Port} -> ${RemoteHost}:${RemotePort}"
Write-Host "Close this terminal to stop the tunnel."

$browserJob = Start-Job -ScriptBlock {
    param($OpenUrl)
    Start-Sleep -Seconds 2
    try {
        Start-Process $OpenUrl | Out-Null
    } catch {
        Write-Host "Failed to open browser automatically. Open $OpenUrl manually."
    }
} -ArgumentList $Url

try {
    & $sshCommand.Source @sshArgs
} finally {
    if ($browserJob) {
        Receive-Job -Job $browserJob -Keep | Out-Null
        Remove-Job -Job $browserJob -Force -ErrorAction SilentlyContinue
    }
}
