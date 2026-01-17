import { useState, useEffect, useRef } from "react";
import { FluentProvider, webLightTheme, webDarkTheme } from "@fluentui/react-components";
import { SettingsModal } from "./components/SettingsModal";
import { ProfileSelectionModal } from "./components/ProfileSelectionModal";
import { invoke } from "@tauri-apps/api/core";
import "./App.css";

interface Profile {
  id: string;
  remote_auth_mode: string;
  remote_server_url: string;
  env: Record<string, string>;
  allow_tools: string[];
}

interface Settings {
  control_port: number;
  mcp_port: number;
  enable_beta: boolean;
  gateway_api_key: string;
}

interface ProcessInfo {
  pid: number;
  name: string;
}

interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

interface ToolDefinition {
  name: string;
  description: string;
  category?: string;
  source: string;
  installed: boolean;
  icon?: string;
}

interface ClientDefinition {
  id: string;
  name: string;
  icon: string;
  description: string;
  manual_instructions: string;
}

function App() {
  const [theme, setTheme] = useState<"light" | "dark">("dark");
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [allTools, setAllTools] = useState<ToolDefinition[]>([]);
  const [allClients, setAllClients] = useState<ClientDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [onboardingRequired, setOnboardingRequired] = useState(false);
  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [drawer, setDrawer] = useState<{ type: string; data?: any } | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([
    { timestamp: new Date().toLocaleTimeString(), level: "INFO", message: "MCP Scooter Command Center initialized." }
  ]);
  const [status, setStatus] = useState({ connected: true, uptime: "0h 0m", latency: "12ms" });
  const [appSettings, setAppSettings] = useState<Settings>({ 
    control_port: 6200, 
    mcp_port: 6277, 
    enable_beta: false,
    gateway_api_key: ""
  });
  const [portConflicts, setPortConflicts] = useState<{ port: number; process: ProcessInfo }[]>([]);

  // Track logged messages to avoid duplicates in splash screen
  const loggedMessages = useRef<Set<string>>(new Set());
  const lastConnectionState = useRef<boolean | null>(null);

  // Interactive UI State
  const [activeTab, setActiveTab] = useState<"active" | "catalog" | "clients">("active");
  const [searchQuery, setSearchQuery] = useState("");
  const [newProfile, setNewProfile] = useState({ id: "" });
  const [newTool, setNewTool] = useState({ name: "", description: "", category: "utility", source: "community" });
  const [toolInput, setToolInput] = useState("{}");
  const [showSettings, setShowSettings] = useState(false);
  const [showProfileModal, setShowProfileModal] = useState(false);

  const CONTROL_API = `http://localhost:${appSettings.control_port}/api`;

  // Splash screen helper
  const splashLog = (message: string, type: string = 'normal', once: boolean = false) => {
    if (once && loggedMessages.current.has(message)) {
      return;
    }
    
    if (typeof window !== 'undefined' && (window as any).splashLog) {
      (window as any).splashLog(message, type);
      if (once) {
        loggedMessages.current.add(message);
      }
    }
  };

  const hideSplash = () => {
    if (typeof window !== 'undefined' && (window as any).hideSplash) {
      (window as any).hideSplash();
    }
  };

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  useEffect(() => {
    splashLog('Checking port availability...', 'active', true);
    
    const checkConflicts = async () => {
      try {
        const conflicts: { port: number; process: ProcessInfo }[] = [];
        
        splashLog(`Scanning port ${appSettings.control_port}...`, 'normal', true);
        const controlUsage = await invoke<ProcessInfo | null>("check_port_usage", { port: appSettings.control_port });
        if (controlUsage && controlUsage.name !== "scooter.exe" && controlUsage.name !== "main.exe" && controlUsage.name !== "desktop.exe") {
          conflicts.push({ port: appSettings.control_port, process: controlUsage });
          splashLog(`Port ${appSettings.control_port} in use by ${controlUsage.name}`, 'active', true);
        } else {
          splashLog(`Port ${appSettings.control_port} available`, 'success', true);
        }

        splashLog(`Scanning port ${appSettings.mcp_port}...`, 'normal', true);
        const mcpUsage = await invoke<ProcessInfo | null>("check_port_usage", { port: appSettings.mcp_port });
        if (mcpUsage && mcpUsage.name !== "scooter.exe" && mcpUsage.name !== "main.exe" && mcpUsage.name !== "desktop.exe") {
          conflicts.push({ port: appSettings.mcp_port, process: mcpUsage });
          splashLog(`Port ${appSettings.mcp_port} in use by ${mcpUsage.name}`, 'active', true);
        } else {
          splashLog(`Port ${appSettings.mcp_port} available`, 'success', true);
        }

        setPortConflicts(conflicts);
      } catch (err) {
        console.error("Port conflict check failed:", err);
        splashLog('Port check skipped (native API unavailable)');
        setPortConflicts([]);
      }
    };

    checkConflicts();
  }, [appSettings.control_port, appSettings.mcp_port]);

  useEffect(() => {
    splashLog('Connecting to backend...', 'active', true);
    fetchProfiles();
    fetchAllTools();
    fetchClients();
    const interval = setInterval(() => {
      fetchProfiles();
      fetchAllTools();
      fetchClients();
    }, 5000);

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
  }, [appSettings.control_port]);

  const fetchProfiles = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/profiles`);
      const data = await res.json();
      const updatedProfiles = data.profiles || [];
      setProfiles(updatedProfiles);
      if (data.settings && (data.settings.control_port !== appSettings.control_port || data.settings.mcp_port !== appSettings.mcp_port)) {
        setAppSettings(data.settings);
      }
      setOnboardingRequired(data.onboarding_required);
      if (updatedProfiles.length > 0 && !selectedProfileId) {
        setSelectedProfileId(updatedProfiles[0].id);
      }
      
      if (lastConnectionState.current !== true) {
        setStatus(s => ({ ...s, connected: true }));
        splashLog('Backend connected!', 'success');
        lastConnectionState.current = true;
      }
      
      // Hide splash after successful connection
      setTimeout(() => hideSplash(), 500);
    } catch (err) {
      console.error("Failed to fetch profiles", err);
      if (lastConnectionState.current !== false) {
        setStatus(s => ({ ...s, connected: false }));
        splashLog('Waiting for backend...', 'active');
        lastConnectionState.current = false;
      }
    } finally {
      setLoading(false);
    }
  };

  const handleKillProcess = async (pid: number) => {
    try {
      const success = await invoke<boolean>("kill_process", { pid });
      if (success) {
        addLog(`Successfully killed process ${pid}`, "INFO");
        // Re-check conflicts after a short delay
        setTimeout(() => {
          // Trigger re-check by refreshing settings or similar
          setAppSettings({ ...appSettings });
        }, 1000);
      } else {
        addLog(`Failed to kill process ${pid}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error killing process: ${err}`, "ERROR");
    }
  };

  const updateGlobalSettings = async (newSettings: Settings) => {
    try {
      const res = await fetch(`${CONTROL_API}/settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newSettings),
      });
      if (res.ok) {
        setAppSettings(newSettings);
        addLog("Global settings updated. Please restart the backend to apply changes.", "INFO");
        setShowSettings(false);
      }
    } catch (err) {
      addLog(`Error updating settings: ${err}`, "ERROR");
    }
  };

  const handleResetApp = async () => {
    try {
      addLog("Resetting application...", "INFO");
      const res = await fetch(`${CONTROL_API}/reset`, { method: "POST" });
      if (res.ok) {
        addLog("Application reset successful.", "INFO");
        setProfiles([]);
        setSelectedProfileId("");
        setOnboardingRequired(true);
        setShowSettings(false);
        // Force a reload to ensure clean state and return to onboarding
        setTimeout(() => {
          window.location.reload();
        }, 1000);
      } else {
        const text = await res.text();
        addLog(`Reset failed: ${text}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error resetting application: ${err}`, "ERROR");
    }
  };

  const fetchAllTools = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/tools`);
      const data = await res.json();
      setAllTools(data.tools || []);
    } catch (err) {
      console.error("Failed to fetch all tools", err);
    }
  };

  const fetchClients = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/clients`);
      const data = await res.json();
      setAllClients(data.clients || []);
    } catch (err) {
      console.error("Failed to fetch clients", err);
    }
  };

  const addLog = (message: string, level: string = "INFO") => {
    setLogs(prev => [{ timestamp: new Date().toLocaleTimeString(), level, message }, ...prev].slice(0, 50));
  };

  const filteredTools = allTools
    .filter(t => 
      (t.name || "").toLowerCase().includes(searchQuery.toLowerCase()) || 
      (t.description || "").toLowerCase().includes(searchQuery.toLowerCase())
    )
    .sort((a, b) => {
      const aName = a.name || "";
      const bName = b.name || "";
      const aNameMatch = aName.toLowerCase().includes(searchQuery.toLowerCase());
      const bNameMatch = bName.toLowerCase().includes(searchQuery.toLowerCase());
      if (aNameMatch && !bNameMatch) return -1;
      if (!aNameMatch && bNameMatch) return 1;
      return 0;
    });

  const toolsByCategory = filteredTools.reduce((acc, tool) => {
    const cat = tool.category || "uncategorized";
    if (!acc[cat]) acc[cat] = [];
    acc[cat].push(tool);
    return acc;
  }, {} as Record<string, ToolDefinition[]>);

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

  const updateProfileTools = async (profileId: string, tools: string[]) => {
    const profile = profiles.find(p => p.id === profileId);
    if (!profile) return;

    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...profile, allow_tools: tools }),
      });
      if (res.ok) {
        addLog(`Updated tools for profile: ${profileId}`, "INFO");
        fetchProfiles();
      } else {
        addLog(`Failed to update tools for profile: ${profileId}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error updating profile tools: ${err}`, "ERROR");
    }
  };

  const createProfile = async () => {
    if (!newProfile.id) return;
    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...newProfile, remote_auth_mode: "none" }),
      });
      if (res.ok) {
        addLog(`Created profile: ${newProfile.id}`, "INFO");
        fetchProfiles();
        setDrawer(null);
        setNewProfile({ id: "" });
      } else {
        const text = await res.text();
        addLog(`Failed to create profile: ${text}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error creating profile: ${err}`, "ERROR");
    }
  };

  const registerTool = async () => {
    if (!newTool.name) return;
    try {
      const res = await fetch(`${CONTROL_API}/tools`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...newTool, installed: true }),
      });
      if (res.ok) {
        addLog(`Registered tool: ${newTool.name}`, "INFO");
        fetchAllTools();
        setDrawer(null);
        setNewTool({ name: "", description: "", source: "community" });
      } else {
        const text = await res.text();
        addLog(`Failed to register tool: ${text}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error registering tool: ${err}`, "ERROR");
    }
  };

  const invokeTool = async () => {
    if (!selectedProfile || !drawer?.data) return;
    try {
      const args = JSON.parse(toolInput);
      addLog(`Invoking ${drawer.data}...`, "INFO");
      
      // Use the unified gateway port and include profile ID in path
      const url = `http://localhost:6277/profiles/${selectedProfile.id}/message`;
      
      const res = await fetch(url, {
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

  if (portConflicts.length > 0) {
    return (
      <FluentProvider theme={theme === "light" ? webLightTheme : webDarkTheme}>
        <div className={`window-frame onboarding-overlay ${theme}`}>
          <div className="onboarding-card conflict-card" style={{ maxWidth: '500px' }}>
            <div className="onboarding-header">
              <span style={{ fontSize: '40px' }}>⚠️</span>
              <h2 style={{ fontSize: '24px', margin: '12px 0' }}>Port Conflict Detected</h2>
              <p style={{ color: 'var(--text-secondary)', marginBottom: '24px' }}>
                MCP Scooter needs specific ports to operate, but they are currently in use by other applications.
              </p>
            </div>
            
            <div className="conflict-list" style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '24px' }}>
              {portConflicts.map(c => (
                <div key={c.port} className="conflict-item" style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '16px', borderRadius: '8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div className="conflict-info">
                    <div style={{ fontWeight: 600, color: 'var(--accent-primary)' }}>Port {c.port}</div>
                    <div style={{ fontSize: '12px', opacity: 0.8 }}>Process: <strong>{c.process.name}</strong> (PID: {c.process.pid})</div>
                  </div>
                  <button className="danger-btn" style={{ width: 'auto', padding: '8px 16px', borderRadius: '4px' }} onClick={() => handleKillProcess(c.process.pid)}>
                    Kill Process
                  </button>
                </div>
              ))}
            </div>

            <div className="conflict-actions" style={{ borderTop: '1px solid var(--border-subtle)', paddingTop: '24px' }}>
              <p style={{ fontSize: '13px', marginBottom: '12px' }}>Or change ports and restart:</p>
              <div style={{ display: 'flex', gap: '12px', marginBottom: '16px' }}>
                <div className="form-field" style={{ flex: 1 }}>
                  <label>Control Port</label>
                  <input 
                    type="number" 
                    value={appSettings.control_port} 
                    onChange={e => setAppSettings({ ...appSettings, control_port: parseInt(e.target.value) })} 
                    style={{ width: '100%', padding: '8px', background: 'var(--background-card)', border: '1px solid var(--border-subtle)', color: 'var(--text-primary)' }}
                  />
                </div>
                <div className="form-field" style={{ flex: 1 }}>
                  <label>MCP Port</label>
                  <input 
                    type="number" 
                    value={appSettings.mcp_port} 
                    onChange={e => setAppSettings({ ...appSettings, mcp_port: parseInt(e.target.value) })} 
                    style={{ width: '100%', padding: '8px', background: 'var(--background-card)', border: '1px solid var(--border-subtle)', color: 'var(--text-primary)' }}
                  />
                </div>
              </div>
              <button className="primary" style={{ width: '100%' }} onClick={() => setPortConflicts([])}>
                Bypass & Try Anyway
              </button>
            </div>
          </div>
        </div>
      </FluentProvider>
    );
  }

  if (loading && profiles.length === 0 && !onboardingRequired) {
    return (
      <FluentProvider theme={theme === "dark" ? webDarkTheme : webLightTheme} style={{ background: "transparent" }}>
        <div className={`window-frame ${theme} loading-screen`}>
          <div className="splash-logo-container">
            <img src={theme === 'dark' ? '/logo/logo-dark.svg' : '/logo/logo-light.svg'} className="splash-logo" alt="MCP Scooter" />
            <div className="splash-title">MCP SCOOTER</div>
            <div className="loading-dots">Initializing...</div>
          </div>
        </div>
      </FluentProvider>
    );
  }

  if (onboardingRequired) {
    return (
      <FluentProvider theme={theme === "dark" ? webDarkTheme : webLightTheme} style={{ background: "transparent" }}>
        <div className={`window-frame onboarding-overlay ${theme}`}>
          <div className="onboarding-card">
            <div className="onboarding-glow"></div>
            <div className="onboarding-content">
              <div className="onboarding-header-visual">
                <img src={theme === 'dark' ? '/logo/logo-dark.svg' : '/logo/logo-light.svg'} className="onboarding-logo" alt="MCP Scooter" />
              </div>
              <h1 className="onboarding-title">Create your first profile</h1>
              <p className="onboarding-subtitle">
                Welcome to MCP Scooter. Let's get you set up with a default profile to start managing your AI tools.
              </p>

              <div className="onboarding-main-action">
                <button className="primary large-btn" onClick={startFresh}>
                  <span className="btn-icon">🚀</span>
                  <div className="btn-text">
                    <h3>Create Default Profile</h3>
                    <p>Creates "work" profile on port 6277</p>
                  </div>
                </button>
              </div>

              <div className="onboarding-separator">
                <span>or</span>
              </div>

              <div className="onboarding-secondary-options">
                <button className="secondary-btn" onClick={handleImport}>
                  <span className="btn-icon">📥</span>
                  <span>Import Existing Config</span>
                </button>
              </div>

              <footer className="onboarding-footer">
                MCP Scooter acts as a universal gateway for your MCPs.
              </footer>
            </div>
          </div>
        </div>
      </FluentProvider>
    );
  }

  return (
    <FluentProvider theme={theme === "dark" ? webDarkTheme : webLightTheme} style={{ background: "transparent" }}>
      <div className={`window-frame ${theme}`}>
        <div className="command-center">
        {/* Profile Strip */}
        <div className="profile-strip">
          <div className="app-logo-mini">
            <img src={theme === 'dark' ? '/logo/logo-dark.svg' : '/logo/logo-light.svg'} alt="S" />
          </div>
          
          <div className="profile-selector-container" onClick={() => setShowProfileModal(true)} style={{ cursor: 'pointer' }}>
            <span className="profile-label">Profile</span>
            <div className="profile-select-display" style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingRight: '8px' }}>
              <span style={{ fontWeight: 700, fontSize: '14px' }}>{selectedProfileId || 'Select Profile'}</span>
              <span style={{ fontSize: '10px', opacity: 0.5 }}>▼</span>
            </div>
          </div>

          <button className="add-profile-btn" onClick={() => setDrawer({ type: "add-profile" })} title="Add Profile">
            +
          </button>
        </div>

        {/* Main Grid */}
        <div className={`main-grid ${activeTab !== 'active' ? 'catalog-full' : ''}`}>
          {/* Main Content Section */}
          <section className="section-container">
            <header className="section-header">
              <div style={{ display: 'flex', gap: '12px' }}>
                <span 
                  className={`tab-link ${activeTab === 'active' ? 'active' : ''}`}
                  onClick={() => setActiveTab('active')}
                >
                  Active Tools
                </span>
                <span 
                  className={`tab-link ${activeTab === 'catalog' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('catalog');
                    setSearchQuery(""); // Clear search when switching
                  }}
                >
                  Catalog
                </span>
                <span 
                  className={`tab-link ${activeTab === 'clients' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('clients');
                    setSearchQuery(""); 
                  }}
                >
                  Clients
                </span>
              </div>
              <span className="badge">
                {activeTab === 'active' 
                  ? `${selectedProfile?.allow_tools?.length || 0} Loaded` 
                  : activeTab === 'catalog'
                  ? `${(filteredTools || []).length} Available`
                  : `${(allClients || []).length} Configurable`}
              </span>
            </header>

            {activeTab === 'catalog' && (
              <div className="search-container">
                <span className="search-icon">🔍</span>
                <input 
                  type="text" 
                  className="search-input" 
                  placeholder="Search tools by name or description..." 
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            )}

            <div className="scroll-section">
              <div className={activeTab !== 'active' ? "card-grid grid-layout" : "card-grid"}>
                {activeTab === 'active' && (
                  <>
                    {selectedProfile?.allow_tools?.map(toolName => {
                      const tool = allTools.find(t => t.name === toolName);
                      return (
                        <div key={toolName} className="compact-card">
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            {tool?.icon ? (
                              <img src={tool.icon} alt={toolName} style={{ width: '24px', height: '24px', objectFit: 'contain' }} />
                            ) : (
                              <div style={{ width: '24px', height: '24px', background: 'var(--border-subtle)', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px' }}>
                                🛠️
                              </div>
                            )}
                            <div className="card-info">
                              <span className="card-title">{toolName}</span>
                              <span className="card-subtitle">{tool?.description || "Ready for tool calls"}</span>
                            </div>
                          </div>
                          <div className="card-actions">
                            <button onClick={() => setDrawer({ type: "test-tool", data: toolName })}>Test</button>
                            <button 
                              className="secondary"
                              onClick={() => {
                                const tools = (selectedProfile.allow_tools || []).filter(t => t !== toolName);
                                updateProfileTools(selectedProfile.id, tools);
                              }}
                            >
                              Deactivate
                            </button>
                          </div>
                        </div>
                      );
                    })}
                    {!selectedProfile?.allow_tools?.length && (
                      <div className="card-subtitle" style={{ textAlign: "center", padding: "20px" }}>
                        No tools enabled for this profile.
                      </div>
                    )}
                  </>
                )}

                {activeTab === 'catalog' && (
                  <>
                    {(Object.entries(toolsByCategory || {})).map(([category, tools]) => (
                      <div key={category} className="category-section" style={{ gridColumn: '1 / -1' }}>
                        <div className="category-title">{category}</div>
                        <div className="card-grid grid-layout">
                          {(tools || []).map(tool => {
                            const isActive = selectedProfile?.allow_tools?.includes(tool.name);
                            return (
                              <div key={tool.name} className="compact-card">
                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                  {tool.icon ? (
                                    <img src={tool.icon} alt={tool.name} style={{ width: '24px', height: '24px', objectFit: 'contain' }} />
                                  ) : (
                                    <div style={{ width: '24px', height: '24px', background: 'var(--border-subtle)', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px' }}>
                                      🛠️
                                    </div>
                                  )}
                                  <div className="card-info">
                                    <span className="card-title">{tool.name}</span>
                                    <span className="card-subtitle">{tool.description}</span>
                                  </div>
                                </div>
                                <div className="card-actions">
                                  {isActive ? (
                                    <button disabled style={{ opacity: 0.5 }}>Active</button>
                                  ) : (
                                    <button 
                                      className="primary"
                                      onClick={() => {
                                        if (!selectedProfile) return;
                                        const tools = [...(selectedProfile.allow_tools || []), tool.name];
                                        updateProfileTools(selectedProfile.id, tools);
                                      }}
                                    >
                                      Activate
                                    </button>
                                  )}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                    <div className="compact-card" style={{ borderStyle: 'dashed', justifyContent: 'center', cursor: 'pointer', gridColumn: '1 / -1' }} onClick={() => setDrawer({ type: "add-custom-tool" })}>
                      <span className="card-title">+ Bring Your Own Tool</span>
                    </div>
                  </>
                )}

                {activeTab === 'clients' && (
                  <>
                    {allClients.map(client => (
                      <div key={client.id} className="compact-card" style={{ alignItems: 'flex-start', flexDirection: 'column', gap: '12px' }}>
                        <div style={{ display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            {client.icon ? (
                              <img src={client.icon} alt={client.name} style={{ width: '32px', height: '32px', objectFit: 'contain' }} />
                            ) : (
                              <div style={{ width: '32px', height: '32px', background: 'var(--border-subtle)', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '16px' }}>
                                📱
                              </div>
                            )}
                            <div className="card-info">
                              <span className="card-title" style={{ fontSize: '15px' }}>{client.name}</span>
                              <span className="card-subtitle">{client.description}</span>
                            </div>
                          </div>
                          <button
                            className="primary"
                            onClick={async () => {
                              try {
                                const res = await fetch(`${CONTROL_API}/clients/sync`, {
                                  method: "POST",
                                  headers: { "Content-Type": "application/json" },
                                  body: JSON.stringify({ target: client.id, profile: selectedProfileId })
                                });
                                if (res.ok) {
                                  addLog(`Successfully synced ${client.name}.`, "INFO");
                                  alert(`Successfully synced ${client.name}!`);
                                } else {
                                  const err = await res.text();
                                  addLog(`Failed to sync ${client.name}: ${err}`, "ERROR");
                                  alert(`Failed to sync ${client.name}: ${err}`);
                                }
                              } catch (err: any) {
                                addLog(`Network error syncing ${client.name}: ${err.message}`, "ERROR");
                                alert(`Network error syncing ${client.name}`);
                              }
                            }}
                          >
                            Auto-Sync
                          </button>
                        </div>
                        
                        <div style={{ width: '100%', padding: '12px', background: 'var(--log-bg)', borderRadius: '6px', fontSize: '11px' }}>
                          <div style={{ fontWeight: 700, marginBottom: '6px', textTransform: 'uppercase', opacity: 0.7 }}>Manual Setup</div>
                          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                            {client.manual_instructions.replace('{profile}', selectedProfileId || 'work')}
                          </pre>
                        </div>
                      </div>
                    ))}
                  </>
                )}
              </div>
            </div>
          </section>

          {/* Logs Column */}
          {activeTab === 'active' && (
            <div className="section-container" style={{ gap: "16px" }}>
              {/* Log Stream */}
              <section className="section-container">
                <header className="section-header">Real-time Stream</header>
                <div className="scroll-section log-stream">
                  {logs.map((log, i) => (
                    <div key={i} style={{ fontFamily: "Google Sans Code, JetBrains Mono, monospace", fontSize: "10px", marginBottom: "4px" }}>
                      <span style={{ opacity: 0.5 }}>[{log.timestamp}]</span>{" "}
                      <span style={{ color: log.level === "ERROR" ? "#ff4d4d" : "inherit" }}>{log.message}</span>
                    </div>
                  ))}
                </div>
              </section>
            </div>
          )}
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
          <div className="toolbar-button" onClick={() => setShowSettings(true)}>
            ⚙️ Settings
          </div>
          <div className="toolbar-button" onClick={toggleTheme}>
            {theme === "dark" ? "🌙 Dark" : "☀️ Light"}
          </div>
        </div>
      </footer>

      <SettingsModal 
        isOpen={showSettings} 
        onClose={() => setShowSettings(false)}
        profiles={profiles}
        settings={appSettings}
        onUpdateSettings={updateGlobalSettings}
        onDeleteProfile={deleteProfile}
        onReset={handleResetApp}
      />

      <ProfileSelectionModal
        isOpen={showProfileModal}
        onClose={() => setShowProfileModal(false)}
        profiles={profiles}
        selectedProfileId={selectedProfileId}
        onSelectProfile={setSelectedProfileId}
        onCreateProfile={() => setDrawer({ type: "add-profile" })}
      />

      {/* Drawer Overlay */}
      {drawer && (
        <div className="drawer-overlay" onClick={() => setDrawer(null)}>
          <div className="drawer-content" onClick={e => e.stopPropagation()}>
            <div className="drawer-header">
              <span className="drawer-title">
                {drawer.type === "test-tool" ? `Test Tool: ${drawer.data}` : 
                 drawer.type === "add-custom-tool" ? "Bring Your Own Tool" : "Add Profile"}
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

            {drawer.type === "add-custom-tool" && (
              <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
                <div className="form-field">
                  <label>Tool Name</label>
                  <input
                    type="text"
                    value={newTool.name}
                    onChange={(e) => setNewTool({ ...newTool, name: e.target.value })}
                    placeholder="e.g. my-custom-tool"
                  />
                </div>
                <div className="form-field">
                  <label>Description</label>
                  <input
                    type="text"
                    value={newTool.description}
                    onChange={(e) => setNewTool({ ...newTool, description: e.target.value })}
                    placeholder="What does this tool do?"
                  />
                </div>
                <div className="form-field">
                  <label>Category</label>
                  <input
                    type="text"
                    value={newTool.category}
                    onChange={(e) => setNewTool({ ...newTool, category: e.target.value })}
                    placeholder="e.g. search, development, utility"
                  />
                </div>
                <div className="form-field">
                  <label>Source Type</label>
                  <select 
                    value={newTool.source}
                    onChange={(e) => setNewTool({ ...newTool, source: e.target.value })}
                  >
                    <option value="community">Community (Remote)</option>
                    <option value="local">Local (WASM)</option>
                  </select>
                </div>
                <button className="primary" onClick={registerTool}>Register Tool</button>
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
                <button className="primary" onClick={createProfile}>Create Profile</button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  </FluentProvider>
  );
}

export default App;
