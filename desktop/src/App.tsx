import { useState, useEffect } from "react";
import "./App.css";

interface Profile {
  id: string;
  port: number;
  auth_mode: string;
  remote_server_url: string;
  env: Record<string, string>;
  allow_tools: string[];
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

function App() {
  const [theme, setTheme] = useState<"light" | "dark">("dark");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [onboardingRequired, setOnboardingRequired] = useState(false);
  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [drawer, setDrawer] = useState<{ type: string; data?: any } | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([
    { timestamp: new Date().toLocaleTimeString(), level: "INFO", message: "MCP Scooter Command Center initialized." }
  ]);
  const [status, setStatus] = useState({ connected: true, uptime: "0h 0m", latency: "12ms" });

  // Interactive UI State
  const [newProfile, setNewProfile] = useState({ id: "", port: 6277 });
  const [toolInput, setToolInput] = useState("{}");

  const CONTROL_API = "http://localhost:6200/api";

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  useEffect(() => {
    fetchProfiles();
    const interval = setInterval(fetchProfiles, 5000);

    // Simulate uptime
    const startTime = Date.now();
    const uptimeInterval = setInterval(() => {
      const diff = Math.floor((Date.now() - startTime) / 1000);
      const h = Math.floor(diff / 3600);
      const m = Math.floor((diff % 3600) / 60);
      setStatus(s => ({ ...s, uptime: `${h}h ${m}m` }));
    }, 60000);

    return () => {
      clearInterval(interval);
      clearInterval(uptimeInterval);
    };
  }, []);

  const fetchProfiles = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/profiles`);
      const data = await res.json();
      const updatedProfiles = data.profiles || [];
      setProfiles(updatedProfiles);
      setOnboardingRequired(data.onboarding_required);
      if (updatedProfiles.length > 0 && !selectedProfileId) {
        setSelectedProfileId(updatedProfiles[0].id);
      }
      setStatus(s => ({ ...s, connected: true }));
    } catch (err) {
      console.error("Failed to fetch profiles", err);
      setStatus(s => ({ ...s, connected: false }));
    } finally {
      setLoading(false);
    }
  };

  const addLog = (message: string, level: string = "INFO") => {
    setLogs(prev => [{ timestamp: new Date().toLocaleTimeString(), level, message }, ...prev].slice(0, 50));
  };

  const selectedProfile = profiles.find(p => p.id === selectedProfileId);

  const toggleTheme = () => setTheme(prev => prev === "light" ? "dark" : "light");

  const deleteProfile = async (id: string) => {
    if (!confirm(`Are you sure you want to delete profile "${id}"?`)) return;
    try {
      const res = await fetch(`${CONTROL_API}/profiles?id=${id}`, { method: "DELETE" });
      if (res.ok) {
        addLog(`Deleted profile: ${id}`, "INFO");
        fetchProfiles();
        if (selectedProfileId === id) setSelectedProfileId("");
      } else {
        addLog(`Failed to delete profile: ${id}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error deleting profile: ${err}`, "ERROR");
    }
  };

  const createProfile = async () => {
    if (!newProfile.id) return;
    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...newProfile, auth_mode: "none" }),
      });
      if (res.ok) {
        addLog(`Created profile: ${newProfile.id}`, "INFO");
        fetchProfiles();
        setDrawer(null);
        setNewProfile({ id: "", port: 6277 });
      } else {
        const text = await res.text();
        addLog(`Failed to create profile: ${text}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error creating profile: ${err}`, "ERROR");
    }
  };

  const invokeTool = async () => {
    if (!selectedProfile || !drawer?.data) return;
    try {
      const args = JSON.parse(toolInput);
      addLog(`Invoking ${drawer.data}...`, "INFO");
      const res = await fetch(`http://localhost:${selectedProfile.port}/message`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "call_tool",
          params: { name: drawer.data, arguments: args },
          id: Date.now()
        }),
      });
      const result = await res.json();
      if (result.error) {
        addLog(`Tool error: ${result.error.message}`, "ERROR");
      } else {
        addLog(`Tool response: ${JSON.stringify(result.result)}`, "INFO");
      }
    } catch (err: any) {
      addLog(`Invocation failed: ${err.message}`, "ERROR");
    }
  };

  const startFresh = async () => {
    try {
      addLog("Initializing default workspace...", "INFO");
      const res = await fetch(`${CONTROL_API}/onboarding/start-fresh`, { method: "POST" });
      if (res.ok) {
        addLog("Workspace ready!", "INFO");
        fetchProfiles();
      } else {
        const text = await res.text();
        addLog(`Failed to start fresh: ${text}`, "ERROR");
      }
    } catch (err: any) {
      addLog(`Error starting fresh: ${err.message}`, "ERROR");
    }
  };

  const handleImport = async () => {
    // In a real Tauri app, we would use open() from @tauri-apps/api/dialog
    // For this simulation/web context, we'll use a file input click or alert
    addLog("Import feature requires native file dialog.", "INFO");
    alert("Please select your profiles.yaml file for import.");
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.yaml,.yml,.json';
    input.onchange = async (e: any) => {
      const file = e.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = async (event: any) => {
        try {
          addLog(`Importing ${file.name}...`, "INFO");
          const res = await fetch(`${CONTROL_API}/onboarding/import`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ profiles: [] }) // Placeholder for parsed profiles
          });
          if (res.ok) {
            addLog("Profiles imported successfully.", "INFO");
            fetchProfiles();
          } else {
            addLog("Import failed. Starting fresh instead...", "WARNING");
            startFresh();
          }
        } catch (err: any) {
          addLog(`Import error: ${err.message}`, "ERROR");
        }
      };
      reader.readAsText(file);
    };
    input.click();
  };

  if (loading && profiles.length === 0 && !onboardingRequired) {
    return <div className="loading-screen">Initializing MCP Scooter...</div>;
  }

  if (onboardingRequired) {
    return (
      <div className="window-frame onboarding-overlay">
        <div className="onboarding-card">
          <div className="onboarding-glow"></div>
          <div className="onboarding-content">
            <h1 className="onboarding-title">Welcome to MCP Scout</h1>
            <p className="onboarding-subtitle">Your universal gateway for the Model Context Protocol. Let's get you set up.</p>

            <div className="onboarding-options">
              <div className="onboarding-option" onClick={startFresh}>
                <div className="option-icon">🚀</div>
                <div className="option-text">
                  <h3>Start Fresh</h3>
                  <p>Create a default workspace and start using tools immediately.</p>
                </div>
              </div>

              <div className="onboarding-option" onClick={handleImport}>
                <div className="option-icon">📥</div>
                <div className="option-text">
                  <h3>Import Config</h3>
                  <p>Already have a profiles.yaml? Bring your existing setup.</p>
                </div>
              </div>
            </div>

            <footer className="onboarding-footer">
              Returning user? Ensure your config is in the correct directory.
            </footer>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="window-frame">
      <div className="command-center">
        {/* Profile Strip */}
        <div className="profile-strip">
          {profiles.map(p => (
            <div
              key={p.id}
              className={`profile-chip ${selectedProfileId === p.id ? "active" : ""}`}
              onClick={() => setSelectedProfileId(p.id)}
            >
              <span className={`status-dot ${selectedProfileId === p.id ? "active" : ""}`}></span>
              {p.id}
              {selectedProfileId === p.id && (
                <button
                  className="icon-btn"
                  style={{ color: "inherit", padding: "0 4px", fontSize: "10px", marginLeft: "4px" }}
                  onClick={(e) => { e.stopPropagation(); deleteProfile(p.id); }}
                >
                  ✕
                </button>
              )}
            </div>
          ))}
          <div className="profile-chip" onClick={() => setDrawer({ type: "add-profile" })}>
            <span>+</span> Add Profile
          </div>
        </div>

        {/* Main Grid */}
        <div className="main-grid">
          {/* Tools Section */}
          <section className="section-container">
            <header className="section-header">
              <span>Active Tools</span>
              <span className="badge">{selectedProfile?.allow_tools?.length || 0} Loaded</span>
            </header>
            <div className="scroll-section">
              <div className="card-grid">
                {selectedProfile?.allow_tools?.map(tool => (
                  <div key={tool} className="compact-card">
                    <div className="card-info">
                      <span className="card-title">{tool}</span>
                      <span className="card-subtitle">Ready for tool calls</span>
                    </div>
                    <div className="card-actions">
                      <button onClick={() => setDrawer({ type: "test-tool", data: tool })}>Test</button>
                    </div>
                  </div>
                ))}
                {!selectedProfile?.allow_tools?.length && (
                  <div className="card-subtitle" style={{ textAlign: "center", padding: "20px" }}>
                    No tools enabled for this profile.
                  </div>
                )}
              </div>
            </div>
          </section>

          {/* Integrations & Logs Column */}
          <div className="section-container" style={{ gap: "16px" }}>
            {/* Integrations Area */}
            <section className="section-container" style={{ flex: "0 1 auto", minHeight: "200px" }}>
              <header className="section-header">One-Click Setup</header>
              <div className="scroll-section">
                <div className="card-grid">
                  {[
                    { id: "cursor", name: "Cursor" },
                    { id: "claude-desktop", name: "Claude Desktop" },
                    { id: "claude-code", name: "Claude Code" },
                    { id: "vscode", name: "VS Code" },
                    { id: "antigravity", name: "Google Antigravity" },
                    { id: "gemini-cli", name: "Gemini CLI" },
                    { id: "codex", name: "Codex" },
                    { id: "zed", name: "Zed" }
                  ].map(integration => (
                    <div key={integration.id} className="compact-card">
                      <div className="card-info">
                        <span className="card-title">{integration.name}</span>
                      </div>
                      <button
                        className="primary"
                        onClick={async () => {
                          try {
                            const res = await fetch(`${CONTROL_API}/integrations/install`, {
                              method: "POST",
                              headers: { "Content-Type": "application/json" },
                              body: JSON.stringify({ target: integration.id, profile: selectedProfileId })
                            });
                            if (res.ok) {
                              addLog(`Successfully synced ${integration.name}.`, "INFO");
                              alert(`Successfully synced ${integration.name}!`);
                            } else {
                              const err = await res.text();
                              addLog(`Failed to sync ${integration.name}: ${err}`, "ERROR");
                              alert(`Failed to sync ${integration.name}: ${err}`);
                            }
                          } catch (err: any) {
                            addLog(`Network error syncing ${integration.name}: ${err.message}`, "ERROR");
                            alert(`Network error syncing ${integration.name}`);
                          }
                        }}
                      >
                        Sync
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </section>

            {/* Log Stream */}
            <section className="section-container">
              <header className="section-header">Real-time Stream</header>
              <div className="scroll-section" style={{ background: "rgba(0,0,0,0.1)", borderRadius: "6px", padding: "8px" }}>
                {logs.map((log, i) => (
                  <div key={i} style={{ fontFamily: "JetBrains Mono", fontSize: "10px", marginBottom: "4px" }}>
                    <span style={{ opacity: 0.5 }}>[{log.timestamp}]</span>{" "}
                    <span style={{ color: log.level === "ERROR" ? "#ff4d4d" : "inherit" }}>{log.message}</span>
                  </div>
                ))}
              </div>
            </section>
          </div>
        </div>
      </div>

      {/* Persistent Bottom Toolbar */}
      <footer className="bottom-toolbar">
        <div className="toolbar-group">
          <div className="stat-item">
            <span className={`status-dot ${status.connected ? "active" : "error"}`}></span>
            <span className="stat-label">API:</span>
            <span className="stat-value">localhost:6200</span>
          </div>
          <div className="stat-item">
            <span className="stat-label">Latency:</span>
            <span className="stat-value">{status.latency}</span>
          </div>
        </div>

        <div className="toolbar-group">
          <div className="stat-item">
            <span className="stat-label">Uptime:</span>
            <span className="stat-value">{status.uptime}</span>
          </div>
          <div className="toolbar-button" onClick={toggleTheme}>
            {theme === "dark" ? "🌙 Dark" : "☀️ Light"}
          </div>
        </div>
      </footer>

      {/* Drawer Overlay */}
      {drawer && (
        <div className="drawer-overlay" onClick={() => setDrawer(null)}>
          <div className="drawer-content" onClick={e => e.stopPropagation()}>
            <div className="drawer-header">
              <span className="drawer-title">
                {drawer.type === "test-tool" ? `Test Tool: ${drawer.data}` : "Add Profile"}
              </span>
              <button className="icon-btn" onClick={() => setDrawer(null)}>✕</button>
            </div>

            {drawer.type === "test-tool" && (
              <div className="form-field">
                <label>Input Parameters (JSON)</label>
                <textarea
                  rows={10}
                  value={toolInput}
                  onChange={(e) => setToolInput(e.target.value)}
                  placeholder="{}"
                  style={{ fontFamily: "JetBrains Mono" }}
                ></textarea>
                <button className="primary" style={{ marginTop: "8px" }} onClick={invokeTool}>Invoke Tool</button>
              </div>
            )}

            {drawer.type === "add-profile" && (
              <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
                <div className="form-field">
                  <label>Profile Name</label>
                  <input
                    type="text"
                    value={newProfile.id}
                    onChange={(e) => setNewProfile({ ...newProfile, id: e.target.value })}
                    placeholder="e.g. dev-mode"
                  />
                </div>
                <div className="form-field">
                  <label>Port</label>
                  <input
                    type="number"
                    value={newProfile.port}
                    onChange={(e) => setNewProfile({ ...newProfile, port: parseInt(e.target.value) })}
                    placeholder="6277"
                  />
                </div>
                <button className="primary" onClick={createProfile}>Create Profile</button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
