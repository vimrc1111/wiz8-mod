# wiz8-run.ps1: called by wiz8-run.sh via 32-bit PowerShell (so WOW64 process modules are visible).
# ASCII only: Windows PowerShell 5 reads BOM-less files as the ANSI codepage (cp950), which breaks multibyte comments.
# Launches Wiz8.exe, takes shots / sends keys / left-clicks (screen pixels) per schedule, lists modules, always Stop-Process at the end. Writes only $Out.
param([string]$Dir, [string]$Steps, [int]$Duration, [string]$GameArgs, [string]$Out)
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
Add-Type @"
using System; using System.Runtime.InteropServices;
public class W {
  [DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr h, IntPtr dc, uint f);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern int GetSystemMetrics(int i);
  [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint f, uint dx, uint dy, uint d, UIntPtr e);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int L,T,R,B; }
}
"@
[W]::SetProcessDPIAware() | Out-Null
New-Item -ItemType Directory -Force $Out | Out-Null
$log = "$Out\events.txt"
function Ev($s) { "{0:F1} {1}" -f ((Get-Date) - $script:t0).TotalSeconds, $s | Add-Content $log }
function Shot($name, $h) {
  # live size: Screen.PrimaryScreen caches and misses fullscreen mode switches (no message loop here)
  $b = New-Object System.Drawing.Rectangle 0, 0, ([W]::GetSystemMetrics(0)), ([W]::GetSystemMetrics(1))
  $bmp = New-Object System.Drawing.Bitmap $b.Width, $b.Height; $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($b.Location, [System.Drawing.Point]::Empty, $b.Size); $bmp.Save("$Out\$name.screen.png"); $g.Dispose(); $bmp.Dispose()
  $r = New-Object W+RECT
  if ([W]::GetWindowRect($h, [ref]$r) -and $r.R -gt $r.L -and $r.B -gt $r.T) {
    $bmp = New-Object System.Drawing.Bitmap ($r.R - $r.L), ($r.B - $r.T); $g = [System.Drawing.Graphics]::FromImage($bmp); $dc = $g.GetHdc()
    [W]::PrintWindow($h, $dc, 2) | Out-Null; $g.ReleaseHdc($dc); $bmp.Save("$Out\$name.window.png"); $g.Dispose(); $bmp.Dispose()
  }
  Ev "shot $name rect=$($r.L),$($r.T),$($r.R),$($r.B)"
}
$plan = ConvertFrom-Json -InputObject (Get-Content -Raw $Steps)  # not $steps: PS names are case-insensitive and $Steps is [string]
$env:__COMPAT_LAYER = "RunAsInvoker"
$script:t0 = Get-Date
# scenario args (wiz8-run.sh validates /[A-Z]+ only)
# Launch via WMI, not Start-Process: children of WSL's wslhost.exe inherit the PreferSystem32 image-load mitigation,
# so Wiz8.exe's DDRAW.DLL import skips the game dir and the proxy never loads (seen 2026-10-12). WMI-created processes don't inherit it.
# EnvironmentVariables replaces the whole block, so pass the full current environment (TEMP is where the proxy logs).
$ev = [string[]](Get-ChildItem Env: | ForEach-Object { "$($_.Name)=$($_.Value)" })
$si = New-CimInstance -ClassName Win32_ProcessStartup -ClientOnly -Property @{ EnvironmentVariables = $ev; ShowWindow = [uint16]1 }
$cl = "`"$Dir\Wiz8.exe`""; if ($GameArgs) { $cl += " $GameArgs" }
$r = Invoke-CimMethod Win32_Process -MethodName Create -Arguments @{ CommandLine = $cl; CurrentDirectory = $Dir; ProcessStartupInformation = $si }
if ($r.ReturnValue -ne 0) { throw "Win32_Process.Create rv=$($r.ReturnValue)" }
$p = Get-Process -Id $r.ProcessId
Ev "start pid=$($p.Id)"
try {
  foreach ($s in $plan) {
    while (((Get-Date) - $t0).TotalSeconds -lt $s.t) { Start-Sleep -Milliseconds 100 }
    $p.Refresh(); if ($p.HasExited) { Ev "exited code=$($p.ExitCode)"; break }
    $h = $p.MainWindowHandle
    if ($s.shot) { Shot $s.shot $h }
    if ($s.keys -or $s.click) {
      $fg = [W]::GetForegroundWindow(); $fpid = 0; [W]::GetWindowThreadProcessId($fg, [ref]$fpid) | Out-Null
      if ($fpid -ne $p.Id) { Ev "input-skipped keys=$($s.keys) click=$($s.click) foreground-pid=$fpid" }
      elseif ($s.keys) { [System.Windows.Forms.SendKeys]::SendWait($s.keys); Ev "keys $($s.keys)" }
      else {
        [W]::SetCursorPos($s.click[0], $s.click[1]) | Out-Null; Start-Sleep -Milliseconds 100
        [W]::mouse_event(2, 0, 0, 0, [UIntPtr]::Zero); Start-Sleep -Milliseconds 80; [W]::mouse_event(4, 0, 0, 0, [UIntPtr]::Zero)
        Ev "click $($s.click[0]),$($s.click[1])"
      }
    }
  }
  while (((Get-Date) - $t0).TotalSeconds -lt $Duration -and -not $p.HasExited) { Start-Sleep -Milliseconds 200; $p.Refresh() }
  if (-not $p.HasExited) {
    Ev "alive"
    $p.Modules | ForEach-Object { $_.ModuleName } | Set-Content "$Out\modules.txt"
  }
} finally {
  if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force; Ev "killed" }
}
