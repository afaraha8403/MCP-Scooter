use tauri::{
    menu::{Menu, MenuItem},
    tray::{TrayIconBuilder, TrayIconEvent},
    Manager, WebviewWindow,
};
use serde::{Serialize, Deserialize};
use std::process::Command;
use sysinfo::{System, Pid};

#[derive(Serialize, Deserialize, Clone)]
pub struct ProcessInfo {
    pid: u32,
    name: String,
}

#[tauri::command]
async fn check_port_usage(port: u16) -> Result<Option<ProcessInfo>, String> {
    #[cfg(target_os = "windows")]
    {
        let output = Command::new("cmd")
            .args(&["/C", &format!("netstat -ano | findstr :{}", port)])
            .output()
            .map_err(|e| e.to_string())?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        for line in stdout.lines() {
            if line.contains("LISTENING") {
                let parts: Vec<&str> = line.split_whitespace().collect();
                if let Some(pid_str) = parts.last() {
                    if let Ok(pid) = pid_str.parse::<u32>() {
                        let mut sys = System::new_all();
                        sys.refresh_all();
                        if let Some(process) = sys.process(Pid::from(pid as usize)) {
                            return Ok(Some(ProcessInfo {
                                pid,
                                name: process.name().to_string_lossy().to_string(),
                            }));
                        }
                    }
                }
            }
        }
    }

    #[cfg(not(target_os = "windows"))]
    {
        let output = Command::new("lsof")
            .args(&["-ti", &format!(":{}", port)])
            .output()
            .map_err(|e| e.to_string())?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        if let Some(pid_str) = stdout.lines().next() {
            if let Ok(pid) = pid_str.parse::<u32>() {
                let mut sys = System::new_all();
                sys.refresh_all();
                if let Some(process) = sys.process(Pid::from(pid as usize)) {
                    return Ok(Some(ProcessInfo {
                        pid,
                        name: process.name().to_string_lossy().to_string(),
                    }));
                }
            }
        }
    }

    Ok(None)
}

#[tauri::command]
async fn kill_process(pid: u32) -> Result<bool, String> {
    let mut sys = System::new_all();
    sys.refresh_all();
    if let Some(process) = sys.process(Pid::from(pid as usize)) {
        return Ok(process.kill());
    }
    Err("Process not found".to_string())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![check_port_usage, kill_process])
        .setup(|app| {
            let quit_i = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
            let show_i = MenuItem::with_id(app, "show", "Show Dashboard", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&show_i, &quit_i])?;

            let _tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .on_menu_event(|app: &tauri::AppHandle, event| match event.id.as_ref() {
                    "quit" => {
                        app.exit(0);
                    }
                    "show" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let window: WebviewWindow = window;
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray: &tauri::tray::TrayIcon, event| {
                    if let TrayIconEvent::Click { .. } = event {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            let window: WebviewWindow = window;
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                })
                .build(app)?;

            Ok(())
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                // Instead of closing, we just hide the window
                window.hide().unwrap();
                api.prevent_close();
            }
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
