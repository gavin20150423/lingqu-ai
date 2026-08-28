$ErrorActionPreference = 'Stop'
$mainBackend = 'D:\WorkSpace\AISpace\subpoilt-account\SubPilot-Private-Dev-20260724\projects\gavin2api\backend'
$runtimeBackend = 'D:\WorkSpace\AISpace\subpoilt-account\.worktrees\gavin-video-workbench-starter\backend'
$env:CONFIG_FILE = "$mainBackend\config.yaml"
$env:DATA_DIR = $mainBackend
$env:SERVER_HOST = '127.0.0.1'
$env:SERVER_PORT = '18082'
$env:VIDEO_API_PUBLIC_BASE_URL = 'http://127.0.0.1:18082'
$env:SKIP_SETUP = 'true'
$out = Join-Path $runtimeBackend 'video-workbench-restrictions-live.out.log'
$err = Join-Path $runtimeBackend 'video-workbench-restrictions-live.err.log'
Start-Process -FilePath (Join-Path $runtimeBackend 'gavin2api-video-workbench-starter.restrictions.exe') -WorkingDirectory $runtimeBackend -WindowStyle Hidden -RedirectStandardOutput $out -RedirectStandardError $err
