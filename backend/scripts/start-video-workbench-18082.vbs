Option Explicit

Dim shell, executable, backendDirectory
Set shell = CreateObject("WScript.Shell")

backendDirectory = "D:\WorkSpace\AISpace\subpoilt-account\SubPilot-Private-Dev-20260724\projects\gavin2api\backend"
executable = "D:\WorkSpace\AISpace\subpoilt-account\releases\gavin2api\20260828-a2c0c345f\gavin2api.exe"

shell.Environment("Process")("CONFIG_FILE") = "D:\WorkSpace\AISpace\subpoilt-account\SubPilot-Private-Dev-20260724\projects\gavin2api\backend\config.yaml"
shell.Environment("Process")("DATA_DIR") = "D:\WorkSpace\AISpace\subpoilt-account\SubPilot-Private-Dev-20260724\projects\gavin2api\backend"
shell.Environment("Process")("SERVER_HOST") = "127.0.0.1"
shell.Environment("Process")("SERVER_PORT") = "18082"
shell.Environment("Process")("VIDEO_API_PUBLIC_BASE_URL") = "http://127.0.0.1:18082"
shell.Environment("Process")("SKIP_SETUP") = "true"
shell.Environment("Process")("SUB2API_SKIP_DATABASE_MIGRATIONS") = "true"
shell.CurrentDirectory = backendDirectory
' Run with a hidden window. Using Run (rather than a piped Exec) lets the
' server start normally without blocking on a full stdout/stderr pipe.
shell.Run Chr(34) & executable & Chr(34), 0, False
