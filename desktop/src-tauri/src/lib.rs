use tauri::{
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::{TrayIconBuilder, TrayIconEvent},
    Manager,
};
use serde::{Serialize, Deserialize};
use std::process::Command;
use sysinfo::{System, Pid};
use std::time::Duration;

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct ProcessInfo {
    pid: u32,
    name: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
struct ToolStatus {
    name: String,
    status: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
struct ProfileStatus {
    id: String,
    running: bool,
    active_tools: i32,
    tool_status: Option<Vec<ToolStatus>>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
struct AppStatus {
    gateway_running: bool,
    control_port: u16,
    mcp_port: u16,
    active_profile_id: String,
    profiles: Vec<ProfileStatus>,
}

fn build_tray_menu<R: tauri::Runtime>(app: &tauri::AppHandle<R>, status: &Option<AppStatus>) -> tauri::Result<Menu<R>> {
    let mut items: Vec<Box<dyn tauri::menu::IsMenuItem<R>>> = Vec::new();

    if let Some(s) = status {
        let gateway_text = format!("Gateway: {} (Port {})", 
            if s.gateway_running { "Running" } else { "Stopped" },
            s.mcp_port
        );
        items.push(Box::new(MenuItem::with_id(app, "status_header", &gateway_text, false, None::<&str>)?));
        items.push(Box::new(PredefinedMenuItem::separator(app)?));

        let mut has_tools = false;
        
        // Show Active Profile Info
        if !s.active_profile_id.is_empty() {
            let active_text = format!("Active Profile: {}", s.active_profile_id);
            items.push(Box::new(MenuItem::with_id(app, "active_profile_header", &active_text, false, None::<&str>)?));
            items.push(Box::new(PredefinedMenuItem::separator(app)?));
        }

        for p in &s.profiles {
            let is_active = p.id == s.active_profile_id;
            let tools = p.tool_status.as_deref().unwrap_or_default();
            
            if p.running && !tools.is_empty() {
                has_tools = true;
                let profile_label = if is_active {
                    format!("● Profile: {}", p.id)
                } else {
                    format!("  Profile: {}", p.id)
                };
                
                items.push(Box::new(MenuItem::with_id(app, format!("profile_{}", p.id), &profile_label, false, None::<&str>)?));
                
                for tool in tools {
                    let icon = match tool.status.as_str() {
                        "ok" => "🟢",      // Running/Active
                        "idle" => "⚪",    // Enabled but not running
                        "warning" => "🟡",
                        _ => "🔴",
                    };
                    let tool_text = format!("    {} {}", icon, tool.name);
                    items.push(Box::new(MenuItem::with_id(app, format!("tool_{}", tool.name), &tool_text, true, None::<&str>)?));
                }
            }
        }
        
        if !has_tools {
            items.push(Box::new(MenuItem::with_id(app, "no_tools", "No tools enabled", false, None::<&str>)?));
        }
        
        items.push(Box::new(PredefinedMenuItem::separator(app)?));
        items.push(Box::new(MenuItem::with_id(app, "reload", "↻ Reload Gateway", true, None::<&str>)?));
        items.push(Box::new(PredefinedMenuItem::separator(app)?));
    } else {
        items.push(Box::new(MenuItem::with_id(app, "gateway_status", "Connecting to Gateway...", false, None::<&str>)?));
        items.push(Box::new(PredefinedMenuItem::separator(app)?));
    }

    items.push(Box::new(MenuItem::with_id(app, "show", "Open MCP Scooter Dashboard", true, None::<&str>)?));
    items.push(Box::new(MenuItem::with_id(app, "quit", "Quit MCP Scooter", true, None::<&str>)?));

    let ref_items: Vec<&dyn tauri::menu::IsMenuItem<R>> = items.iter().map(|i| i.as_ref()).collect();
    Menu::with_items(app, &ref_items)
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
            let handle = app.handle().clone();
            
            // Initial menu
            let menu = build_tray_menu(&handle, &None)?;

            let _tray = TrayIconBuilder::with_id("main-tray")
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .show_menu_on_left_click(true)
                .on_menu_event(move |app, event| {
                    match event.id.as_ref() {
                        "quit" => {
                            app.exit(0);
                        }
                        "show" => {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                        "reload" => {
                            tauri::async_runtime::spawn(async move {
                                let client = reqwest::Client::new();
                                let _ = client.post("http://127.0.0.1:6200/api/reset").send().await;
                            });
                        }
                        _ => {}
                    }
                })
                .on_tray_icon_event(|tray, event| {
                    match event {
                        TrayIconEvent::DoubleClick { .. } => {
                            let app = tray.app_handle();
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                        _ => {}
                    }
                })
                .build(app)?;

            // Background polling for status
            tauri::async_runtime::spawn(async move {
                let client = reqwest::Client::new();
                let mut last_status: Option<AppStatus> = None;

                // Initial delay to let backend start
                tokio::time::sleep(Duration::from_secs(2)).await;

                loop {
                    let status = match client.get("http://127.0.0.1:6200/api/status").send().await {
                        Ok(resp) => {
                            if resp.status().is_success() {
                                match resp.text().await {
                                    Ok(text) => {
                                        match serde_json::from_str::<AppStatus>(&text) {
                                            Ok(parsed) => Some(parsed),
                                            Err(_) => None
                                        }
                                    },
                                    Err(_) => None
                                }
                            } else {
                                None
                            }
                        },
                        Err(_) => None
                    };

                    // Check if status changed (simple check)
                    let status_changed = match (&status, &last_status) {
                        (Some(s), Some(ls)) => {
                            s.gateway_running != ls.gateway_running || 
                            s.active_profile_id != ls.active_profile_id ||
                            s.profiles.len() != ls.profiles.len() ||
                            s.profiles.iter().any(|p| {
                                ls.profiles.iter().find(|lp| lp.id == p.id)
                                    .map(|lp| lp.active_tools != p.active_tools || lp.tool_status != p.tool_status)
                                    .unwrap_or(true)
                            })
                        },
                        (None, None) => false,
                        _ => true,
                    };

                    if status_changed {
                        last_status = status.clone();
                        
                        // Update tray
                        if let Some(tray) = handle.tray_by_id("main-tray") {
                            // Update menu
                            if let Ok(new_menu) = build_tray_menu(&handle, &status) {
                                let _ = tray.set_menu(Some(new_menu));
                            }

                            // Update icon based on status
                            let icon_name = if let Some(s) = &status {
                                if !s.gateway_running {
                                    "tray-error.png"
                                } else if s.profiles.iter().any(|p| {
                                    p.tool_status.as_deref().unwrap_or_default().iter().any(|ts| ts.status != "ok")
                                }) {
                                    "tray-warning.png"
                                } else {
                                    "tray-ok.png"
                                }
                            } else {
                                "tray-error.png"
                            };

                            // Load icon based on status
                            let icon_path = std::path::Path::new("icons").join(icon_name);
                            let dev_icon_path = std::path::Path::new("desktop/src-tauri/icons").join(icon_name);

                            let final_path = if icon_path.exists() {
                                Some(icon_path)
                            } else if dev_icon_path.exists() {
                                Some(dev_icon_path)
                            } else {
                                None
                            };

                            if let Some(path) = final_path {
                                if let Ok(img) = tauri::image::Image::from_path(path) {
                                    let _ = tray.set_icon(Some(img));
                                }
                            }
                        }
                    }

                    tokio::time::sleep(Duration::from_secs(5)).await;
                }
            });

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
