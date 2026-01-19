import { useState, useEffect, useRef } from "react";
import { FluentProvider, webLightTheme, webDarkTheme } from "@fluentui/react-components";
import { 
  EyeRegular, 
  EyeOffRegular, 
  EditRegular, 
  CheckmarkRegular, 
  DismissRegular,
  CheckmarkCircleRegular,
  ErrorCircleRegular,
  SearchRegular,
  SettingsRegular,
  WarningRegular,
  HomeRegular,
  BoxRegular,
  BookRegular,
  OpenRegular,
  RocketRegular,
  ArrowDownloadRegular,
  KeyRegular,
  WrenchRegular,
  WeatherMoonRegular,
  WeatherSunnyRegular,
  ArrowLeftRegular,
  ChevronDownRegular,
  PhoneLaptopRegular,
  AddRegular
} from "@fluentui/react-icons";
import ReactMarkdown from "react-markdown";
import { SettingsModal } from "./components/SettingsModal";
import { ProfileSelectionModal } from "./components/ProfileSelectionModal";
import { invoke } from "@tauri-apps/api/core";
import { revealItemInDir } from "@tauri-apps/plugin-opener";
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
  last_profile_id?: string;
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
  title?: string;
  version?: string;
  description: string;
  category?: string;
  source: string;
  installed: boolean;
  icon?: string;
  about?: string;
  tags?: string[];
  homepage?: string;
  repository?: string;
  documentation?: string;
  authorization?: {
    type: string;
    required?: boolean;
    env_var?: string;
    display_name?: string;
    description?: string;
    help_url?: string;
    env_vars?: { name: string; display_name: string; description?: string; secret?: boolean; required?: boolean }[];
  };
  tools?: { 
    name: string; 
    title?: string;
    description: string; 
    inputSchema?: {
      type: string;
      properties?: Record<string, any>;
      required?: string[];
    };
    sampleInput?: Record<string, any>;
  }[];
  package?: {
    type: string;
    name?: string;
    version?: string;
  };
  runtime?: {
    transport?: string;
    command?: string;
    args?: string[];
  };
  metadata?: {
    author?: string;
    license?: string;
  };
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
  const [configPath, setConfigPath] = useState("");
  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [selectedTool, setSelectedTool] = useState<ToolDefinition | null>(null);
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
  const [selectedFunctionName, setSelectedFunctionName] = useState<string>("");
  const [testResult, setTestResult] = useState<{ status: 'idle' | 'loading' | 'success' | 'error', data: any } | null>(null);
  const [authInput, setAuthInput] = useState<Record<string, string>>({});
  const [revealedAuthKeys, setRevealedAuthKeys] = useState<Record<string, boolean>>({});
  const [editingAuthKey, setEditingAuthKey] = useState<string | null>(null);
  const [editingAuthValue, setEditingAuthValue] = useState("");
  const [showSettings, setShowSettings] = useState(false);
  const [showProfileModal, setShowProfileModal] = useState(false);
  const [savedToolParams, setSavedToolParams] = useState<Record<string, Record<string, any>>>({});

  // Load saved tool params on mount
  useEffect(() => {
    const loadSavedParams = async () => {
      try {
        const res = await fetch(`http://localhost:${appSettings.control_port}/api/tool-params`);
        if (res.ok) {
          const data = await res.json();
          setSavedToolParams(data || {});
        }
      } catch (err) {
        console.log("Could not load saved tool params:", err);
      }
    };
    loadSavedParams();
  }, [appSettings.control_port]);

  // Get the best input for a tool function: saved > sampleInput > generated
  const getToolInput = (tool: ToolDefinition | undefined, functionName: string) => {
    if (!tool) return "{}";
    
    // 1. Check for saved params first
    if (savedToolParams[functionName]) {
      return JSON.stringify(savedToolParams[functionName], null, 2);
    }
    
    // 2. Check for sampleInput in the tool definition
    const toolFunc = tool.tools?.find(t => t.name === functionName);
    if (toolFunc?.sampleInput) {
      return JSON.stringify(toolFunc.sampleInput, null, 2);
    }
    
    // 3. Fall back to generated default
    if (toolFunc?.inputSchema) {
      return generateDefaultJson(toolFunc.inputSchema);
    }
    
    return "{}";
  };

  // Save tool params when modified
  const saveToolParams = async (functionName: string, params: Record<string, any>) => {
    try {
      await fetch(`http://localhost:${appSettings.control_port}/api/tool-params`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tool_name: functionName, parameters: params }),
      });
      setSavedToolParams(prev => ({ ...prev, [functionName]: params }));
    } catch (err) {
      console.log("Could not save tool params:", err);
    }
  };

  useEffect(() => {
    if (drawer?.type === "test-tool") {
      setTestResult(null); // Reset result when opening drawer or changing tool
      const tool = allTools.find(t => t.name === drawer.data);
      if (tool && tool.tools && tool.tools.length > 0) {
        const firstFunc = tool.tools[0];
        setSelectedFunctionName(firstFunc.name);
        setToolInput(getToolInput(tool, firstFunc.name));
      } else if (tool) {
        setSelectedFunctionName(tool.name);
        setToolInput(getToolInput(tool, tool.name));
      }
    } else {
      setSelectedFunctionName("");
      setToolInput("{}");
      setTestResult(null);
    }
  }, [drawer, allTools, savedToolParams]);

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

  useEffect(() => {
    if (profiles.length > 0) {
      if (!selectedProfileId || !profiles.find(p => p.id === selectedProfileId)) {
        // Try to use the last profile from settings, otherwise use first profile
        const lastProfile = appSettings.last_profile_id;
        if (lastProfile && profiles.find(p => p.id === lastProfile)) {
          setSelectedProfileId(lastProfile);
        } else {
          setSelectedProfileId(profiles[0].id);
        }
      }
    } else if (!loading) {
      setSelectedProfileId("");
    }
  }, [profiles, selectedProfileId, loading, appSettings.last_profile_id]);

  const fetchProfiles = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/profiles`);
      const data = await res.json();
      const updatedProfiles = data.profiles || [];
      setProfiles(updatedProfiles);
      setConfigPath(data.config_path || "");
      
      if (data.settings) {
        setAppSettings(current => {
          if (data.settings.control_port !== current.control_port || data.settings.mcp_port !== current.mcp_port) {
            return data.settings;
          }
          return current;
        });
      }
      
      setOnboardingRequired(data.onboarding_required);
      
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

  const handleSelectProfile = async (profileId: string) => {
    setSelectedProfileId(profileId);
    // Save the last selected profile to settings
    try {
      const newSettings = { ...appSettings, last_profile_id: profileId };
      await fetch(`${CONTROL_API}/settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newSettings),
      });
      setAppSettings(newSettings);
    } catch (err) {
      // Silent fail - not critical if we can't save the preference
      console.log("Could not save last profile preference:", err);
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
      const tools = data.tools || [];
      setAllTools(prev => {
        if (JSON.stringify(prev) !== JSON.stringify(tools)) {
          return tools;
        }
        return prev;
      });
    } catch (err) {
      console.error("Failed to fetch all tools", err);
    }
  };

  const fetchClients = async () => {
    try {
      const res = await fetch(`${CONTROL_API}/clients`);
      const data = await res.json();
      const clients = data.clients || [];
      setAllClients(prev => {
        if (JSON.stringify(prev) !== JSON.stringify(clients)) {
          return clients;
        }
        return prev;
      });
    } catch (err) {
      console.error("Failed to fetch clients", err);
    }
  };

  const addLog = (message: string, level: string = "INFO") => {
    console.log(`[${level}] ${message}`);
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

  const updateProfileEnv = async (profileId: string, env: Record<string, string>) => {
    const profile = profiles.find(p => p.id === profileId);
    if (!profile) return;

    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...profile, env: { ...(profile.env || {}), ...env } }),
      });
      if (res.ok) {
        addLog(`Updated environment for profile: ${profileId}`, "INFO");
        fetchProfiles();
      } else {
        addLog(`Failed to update environment for profile: ${profileId}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error updating profile environment: ${err}`, "ERROR");
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
        setNewTool({ name: "", description: "", category: "utility", source: "community" });
      } else {
        const text = await res.text();
        addLog(`Failed to register tool: ${text}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error registering tool: ${err}`, "ERROR");
    }
  };

  const invokeTool = async () => {
    if (!selectedProfile || !drawer?.data || !selectedFunctionName) {
      addLog("Please select a tool/function to invoke.", "WARNING");
      return;
    }
    setTestResult({ status: 'loading', data: null });
    try {
      const args = JSON.parse(toolInput);
      addLog(`Invoking ${selectedFunctionName}...`, "INFO");
      
      // Use the unified gateway port and include profile ID in path
      const url = `http://localhost:${appSettings.mcp_port}/profiles/${selectedProfile.id}/message`;
      
      const res = await fetch(url, {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          "X-Scooter-Internal": "true"
        },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "call_tool",
          params: { name: selectedFunctionName, arguments: args },
          id: Date.now()
        }),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(`Server returned ${res.status}: ${text}`);
      }

      const result = await res.json();
      if (result.error) {
        addLog(`Tool error: ${result.error.message}`, "ERROR");
        setTestResult({ status: 'error', data: result.error });
      } else {
        addLog(`Tool response received`, "INFO");
        setTestResult({ status: 'success', data: result.result });
      }
    } catch (err: any) {
      addLog(`Invocation failed: ${err.message}`, "ERROR");
      setTestResult({ status: 'error', data: err.message });
    }
  };

  const generateDefaultJson = (schema: any) => {
    if (!schema || !schema.properties) return "{}";
    const obj: any = {};
    Object.keys(schema.properties).forEach(key => {
      const prop = schema.properties[key];
      if (prop.default !== undefined) {
        obj[key] = prop.default;
      } else {
        switch (prop.type) {
          case 'string': obj[key] = ""; break;
          case 'number':
          case 'integer': obj[key] = 0; break;
          case 'boolean': obj[key] = false; break;
          case 'object': obj[key] = {}; break;
          case 'array': obj[key] = []; break;
          default: obj[key] = null;
        }
      }
    });
    return JSON.stringify(obj, null, 2);
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
    addLog("Import feature requires native file dialog.", "INFO");
    alert("Please select your profiles.yaml file for import.");
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.yaml,.yml,.json';
    input.onchange = async (e: any) => {
      const file = e.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = async () => {
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
              <WarningRegular style={{ fontSize: '48px', color: '#ffcc00' }} />
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
                  <span className="btn-icon"><RocketRegular /></span>
                  <div className="btn-text">
                    <h3>Create Default Profile</h3>
                    <p>Creates "work" profile on port {appSettings.mcp_port}</p>
                  </div>
                </button>
              </div>

              <div className="onboarding-separator">
                <span>or</span>
              </div>

              <div className="onboarding-secondary-options">
                <button className="secondary-btn" onClick={handleImport}>
                  <span className="btn-icon"><ArrowDownloadRegular /></span>
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
              <ChevronDownRegular style={{ fontSize: '12px', opacity: 0.5 }} />
            </div>
          </div>

          <button className="add-profile-btn" onClick={() => setDrawer({ type: "add-profile" })} title="Add Profile">
            <AddRegular />
          </button>
        </div>

        {/* Main Grid */}
        <div className={`main-grid ${activeTab !== 'active' ? 'catalog-full' : ''}`}>
          {/* Detail View */}
          {selectedTool && (
            <section className="section-container detail-view">
              <header className="section-header">
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <button className="icon-btn" onClick={() => setSelectedTool(null)} title="Back">
                    <ArrowLeftRegular />
                  </button>
                  <span style={{ fontSize: '18px', fontWeight: 700, textTransform: 'none', color: 'var(--text-primary)' }}>
                    {selectedTool.title || selectedTool.name}
                    {selectedTool.title && selectedTool.title !== selectedTool.name && (
                      <span style={{ fontSize: '14px', opacity: 0.5, marginLeft: '8px', fontWeight: 400 }}>
                        ({selectedTool.name})
                      </span>
                    )}
                  </span>
                </div>
                <div className="card-actions">
                  {selectedTool.authorization && (
                    <button 
                      onClick={(e) => { e.stopPropagation(); setDrawer({ type: "auth-config", data: selectedTool.name }); }}
                      title="Manage Authentication"
                      style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
                    >
                      <KeyRegular style={{ fontSize: '16px' }} />
                      <span>Auth</span>
                    </button>
                  )}
                  {selectedProfile?.allow_tools?.includes(selectedTool.name) ? (
                    <button 
                      className="secondary"
                      onClick={(e) => {
                        e.stopPropagation();
                        const tools = (selectedProfile.allow_tools || []).filter(t => t !== selectedTool.name);
                        updateProfileTools(selectedProfile.id, tools);
                      }}
                    >
                      Deactivate
                    </button>
                  ) : (
                    <button 
                      className="primary"
                      onClick={(e) => {
                        e.stopPropagation();
                        if (!selectedProfile) return;
                        const tools = [...(selectedProfile.allow_tools || []), selectedTool.name];
                        updateProfileTools(selectedProfile.id, tools);
                      }}
                    >
                      Activate
                    </button>
                  )}
                  <button onClick={(e) => { e.stopPropagation(); setDrawer({ type: "test-tool", data: selectedTool.name }); }}>Test</button>
                </div>
              </header>

              <div className="scroll-section detail-content" style={{ padding: '24px' }}>
                <div style={{ display: 'flex', gap: '32px', marginBottom: '32px' }}>
                  {selectedTool.icon ? (
                    <img src={selectedTool.icon} alt={selectedTool.name} style={{ width: '80px', height: '80px', objectFit: 'contain' }} />
                  ) : (
                    <div style={{ width: '80px', height: '80px', background: 'var(--border-subtle)', borderRadius: '12px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <WrenchRegular style={{ fontSize: '40px', opacity: 0.5 }} />
                    </div>
                  )}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <span className="badge">{selectedTool.category || 'Utility'}</span>
                      <span className="badge" style={{ background: 'var(--accent-primary)', color: 'white' }}>{selectedTool.source}</span>
                      {selectedTool.version && <span style={{ fontSize: '12px', opacity: 0.5 }}>v{selectedTool.version}</span>}
                    </div>
                    <p style={{ fontSize: '16px', lineHeight: '1.5', margin: 0, color: 'var(--text-primary)', fontWeight: 500 }}>
                      {selectedTool.description}
                    </p>
                    {selectedTool.tags && selectedTool.tags.length > 0 && (
                      <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginTop: '4px' }}>
                        {selectedTool.tags.map(tag => (
                          <span key={tag} style={{ fontSize: '11px', padding: '2px 8px', background: 'var(--background-card)', border: '1px solid var(--border-subtle)', borderRadius: '12px', color: 'var(--text-secondary)' }}>
                            #{tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                  {selectedTool.authorization && selectedProfile && (
                    (() => {
                      const auth = selectedTool.authorization;
                      const envVars = auth.type === 'custom' ? (auth.env_vars || []) : (auth.env_var ? [{ name: auth.env_var, display_name: auth.display_name || auth.env_var, description: auth.description, required: true }] : []);
                      const missingVars = envVars.filter(v => v.required && !selectedProfile.env?.[v.name]);
                      const configuredVars = envVars.filter(v => selectedProfile.env?.[v.name]);

                      return (
                        <div style={{ marginBottom: '32px' }}>
                          {missingVars.length > 0 && (
                            <div style={{ fontSize: '12px', background: 'rgba(255, 204, 0, 0.1)', border: '1px solid rgba(255, 204, 0, 0.3)', padding: '24px', borderRadius: '8px', color: '#b38600', display: 'flex', flexDirection: 'column', gap: '16px', marginBottom: configuredVars.length > 0 ? '12px' : '0' }}>
                              <div style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px', fontSize: '14px' }}>
                                <span><WarningRegular style={{ verticalAlign: 'middle', marginRight: '4px', color: '#ffcc00' }} /> Configuration Required</span>
                              </div>
                              <div style={{ fontSize: '13px', lineHeight: '1.4' }}>
                                This tool requires additional setup to function in the <strong>{selectedProfile.id}</strong> profile.
                              </div>
                              
                              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                {missingVars.map(v => (
                                  <div key={v.name} style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                      <label style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', opacity: 0.8 }}>
                                        {v.display_name}
                                      </label>
                                      <code style={{ fontSize: '10px', opacity: 0.6 }}>{v.name}</code>
                                    </div>
                                    {v.description && <p style={{ margin: '0', fontSize: '12px', opacity: 0.8 }}>{v.description}</p>}
                                    <div style={{ display: 'flex', gap: '8px' }}>
                                      <input 
                                        type={v.secret !== false ? "password" : "text"}
                                        placeholder={`Enter ${v.display_name}...`}
                                        value={authInput[v.name] || ''}
                                        onChange={(e) => setAuthInput({ ...authInput, [v.name]: e.target.value })}
                                        style={{ flex: 1, padding: '10px', borderRadius: '6px', border: '1px solid rgba(179, 134, 0, 0.3)', background: 'var(--background-card)', color: 'var(--text-primary)' }}
                                      />
                                      <button 
                                        className="primary"
                                        onClick={() => {
                                          if (authInput[v.name]) {
                                            updateProfileEnv(selectedProfile.id, { [v.name]: authInput[v.name] });
                                            const newAuthInput = { ...authInput };
                                            delete newAuthInput[v.name];
                                            setAuthInput(newAuthInput);
                                          }
                                        }}
                                        disabled={!authInput[v.name]}
                                        style={{ whiteSpace: 'nowrap' }}
                                      >
                                        Save Key
                                      </button>
                                    </div>
                                  </div>
                                ))}
                              </div>

                              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px', borderTop: '1px solid rgba(179, 134, 0, 0.1)', paddingTop: '12px' }}>
                                {auth.help_url ? (
                                  <a 
                                    href={auth.help_url} 
                                    target="_blank" 
                                    rel="noopener noreferrer"
                                    style={{ color: '#b38600', textDecoration: 'none', fontSize: '12px', fontWeight: 500 }}
                                  >
                                    Need help getting started? ↗
                                  </a>
                                ) : <div></div>}
                                <button 
                                  style={{ background: 'transparent', border: 'none', color: '#b38600', fontSize: '12px', cursor: 'pointer', opacity: 0.7, textDecoration: 'underline' }}
                                  onClick={async () => {
                                    if (configPath) {
                                      try {
                                        await revealItemInDir(configPath);
                                      } catch (err) {
                                        console.error("Failed to open config file location:", err);
                                        setShowProfileModal(true);
                                      }
                                    } else {
                                      setShowProfileModal(true);
                                    }
                                  }}
                                >
                                  Manage all variables in profile
                                </button>
                              </div>
                            </div>
                          )}

                          {configuredVars.length > 0 && missingVars.length > 0 && (
                            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '12px' }}>
                              {configuredVars.map(v => (
                                <div key={v.name} style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 10px', background: 'var(--background-card)', border: '1px solid var(--border-subtle)', borderRadius: '20px', fontSize: '10px' }}>
                                  <span style={{ opacity: 0.6 }}>{v.display_name}:</span>
                                  <code>••••••</code>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      );
                    })()
                  )}

                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 300px', gap: '32px' }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
                    {selectedTool.about && (
                      <div className="detail-section">
                        <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '16px' }}>About</h3>
                        <div className="markdown-content" style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '24px', borderRadius: '8px', lineHeight: '1.6', fontSize: '14px' }}>
                          <ReactMarkdown>{selectedTool.about}</ReactMarkdown>
                        </div>
                      </div>
                    )}

                    <div className="detail-section">
                      <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '16px' }}>Capabilities</h3>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                        {selectedTool.tools && selectedTool.tools.length > 0 ? (
                          selectedTool.tools.map(t => (
                            <div key={t.name} style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '16px', borderRadius: '8px' }}>
                              <div style={{ fontWeight: 600, marginBottom: '4px', display: 'flex', justifyContent: 'space-between' }}>
                                <span>{t.title || t.name}</span>
                                <code style={{ fontSize: '11px', opacity: 0.5 }}>{t.name}</code>
                              </div>
                              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', lineHeight: '1.4' }}>{t.description}</div>
                            </div>
                          ))
                        ) : (
                          <div key={selectedTool.name} style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '16px', borderRadius: '8px' }}>
                            <div style={{ fontWeight: 600, marginBottom: '4px' }}>{selectedTool.name}</div>
                            <div style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{selectedTool.description}</div>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                    <div className="detail-section">
                      <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '12px' }}>Details</h3>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: '13px' }}>
                        {selectedTool.metadata?.author && (
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ color: 'var(--text-secondary)' }}>Author</span>
                            <span style={{ fontWeight: 500 }}>{selectedTool.metadata.author}</span>
                          </div>
                        )}
                        {selectedTool.metadata?.license && (
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ color: 'var(--text-secondary)' }}>License</span>
                            <span style={{ fontWeight: 500 }}>{selectedTool.metadata.license}</span>
                          </div>
                        )}
                        {selectedTool.package?.type && (
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ color: 'var(--text-secondary)' }}>Registry</span>
                            <span style={{ fontWeight: 500 }}>{selectedTool.package.type.toUpperCase()}</span>
                          </div>
                        )}
                        {selectedTool.package?.name && (
                          <div style={{ display: 'flex', justifyContent: 'space-between', flexDirection: 'column', gap: '4px' }}>
                            <span style={{ color: 'var(--text-secondary)' }}>Package</span>
                            <code style={{ background: 'var(--background-card)', padding: '4px 8px', borderRadius: '4px', fontSize: '11px', overflow: 'hidden', textOverflow: 'ellipsis' }}>{selectedTool.package.name}</code>
                          </div>
                        )}
                      </div>
                    </div>

                    {(selectedTool.homepage || selectedTool.repository || selectedTool.documentation) && (
                      <div className="detail-section">
                        <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '12px' }}>Links</h3>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '13px' }}>
                          {selectedTool.homepage && (
                            <a href={selectedTool.homepage} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-primary)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '8px' }}>
                              <HomeRegular style={{ fontSize: '16px' }} />
                              <span>Homepage</span>
                              <OpenRegular style={{ fontSize: '12px', opacity: 0.6 }} />
                            </a>
                          )}
                          {selectedTool.repository && (
                            <a href={selectedTool.repository} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-primary)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '8px' }}>
                              <BoxRegular style={{ fontSize: '16px' }} />
                              <span>Repository</span>
                              <OpenRegular style={{ fontSize: '12px', opacity: 0.6 }} />
                            </a>
                          )}
                          {selectedTool.documentation && (
                            <a href={selectedTool.documentation} target="_blank" rel="noopener noreferrer" style={{ color: 'var(--accent-primary)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: '8px' }}>
                              <BookRegular style={{ fontSize: '16px' }} />
                              <span>Documentation</span>
                              <OpenRegular style={{ fontSize: '12px', opacity: 0.6 }} />
                            </a>
                          )}
                        </div>
                      </div>
                    )}

                    {selectedTool.authorization && (
                      <div className="detail-section">
                        <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '12px' }}>Security</h3>
                        <div style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '12px', borderRadius: '8px', fontSize: '13px' }}>
                          <div style={{ fontWeight: 600, marginBottom: '4px' }}>{selectedTool.authorization.type === 'api_key' ? 'API Key Required' : selectedTool.authorization.type}</div>
                          {selectedTool.authorization.env_var && (
                            <div style={{ fontSize: '11px', opacity: 0.7, marginBottom: '8px' }}>
                              Env: <code>{selectedTool.authorization.env_var}</code>
                            </div>
                          )}
                          {selectedTool.authorization.description && (
                            <div style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.4' }}>{selectedTool.authorization.description}</div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </section>
          )}

          {/* Main List View */}
          <section className="section-container" style={{ display: selectedTool ? 'none' : 'flex' }}>
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
                <span className="search-icon"><SearchRegular /></span>
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
                        <div 
                          key={toolName} 
                          className="compact-card clickable"
                          onClick={() => {
                            if (tool) setSelectedTool(tool);
                          }}
                        >
                          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                            {tool?.icon ? (
                              <img src={tool.icon} alt={toolName} style={{ width: '40px', height: '40px', objectFit: 'contain' }} />
                            ) : (
                              <div style={{ width: '40px', height: '40px', background: 'var(--border-subtle)', borderRadius: '6px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                <WrenchRegular style={{ fontSize: '20px', opacity: 0.5 }} />
                              </div>
                            )}
                            <div className="card-info">
                              <span className="card-title">{tool?.title || toolName}</span>
                              <span className="card-subtitle">{tool?.description || "Ready for tool calls"}</span>
                            </div>
                          </div>
                          <div className="card-actions">
                            <button onClick={(e) => { e.stopPropagation(); setDrawer({ type: "test-tool", data: toolName }); }}>Test</button>
                            <button 
                              className="secondary"
                              onClick={(e) => {
                                e.stopPropagation();
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
                              <div 
                                key={tool.name} 
                                className="compact-card clickable"
                                onClick={() => setSelectedTool(tool)}
                              >
                                <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                                  {tool.icon ? (
                                    <img src={tool.icon} alt={tool.name} style={{ width: '40px', height: '40px', objectFit: 'contain' }} />
                                  ) : (
                                    <div style={{ width: '40px', height: '40px', background: 'var(--border-subtle)', borderRadius: '6px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                      <WrenchRegular style={{ fontSize: '20px', opacity: 0.5 }} />
                                    </div>
                                  )}
                                  <div className="card-info">
                                    <span className="card-title">{tool.title || tool.name}</span>
                                    <span className="card-subtitle">{tool.description}</span>
                                  </div>
                                </div>
                                <div className="card-actions">
                                  {isActive ? (
                                    <button disabled style={{ opacity: 0.5 }} onClick={(e) => e.stopPropagation()}>Active</button>
                                  ) : (
                                    <button 
                                      className="primary"
                                      onClick={(e) => {
                                        e.stopPropagation();
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
                              <div style={{ width: '32px', height: '32px', background: 'var(--border-subtle)', borderRadius: '4px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                <PhoneLaptopRegular style={{ fontSize: '20px', opacity: 0.5 }} />
                              </div>
                            )}
                            <div className="card-info">
                              <span className="card-title" style={{ fontSize: '15px' }}>{client.name}</span>
                              <span className="card-subtitle">{client.description}</span>
                            </div>
                          </div>
                          <button
                            className="primary"
                            onClick={async (e) => {
                              e.stopPropagation();
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
                            {client.manual_instructions
                              .replace('{profile}', selectedProfileId || 'work')
                              .replace('6277', appSettings.mcp_port.toString())}
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
          {activeTab === 'active' && !selectedTool && (
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
            <span className="stat-value">localhost:{appSettings.control_port}</span>
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
            <SettingsRegular style={{ marginRight: '4px' }} /> Settings
          </div>
          <div className="toolbar-button" onClick={toggleTheme}>
            {theme === "dark" ? (
              <><WeatherMoonRegular style={{ marginRight: '4px' }} /> Dark</>
            ) : (
              <><WeatherSunnyRegular style={{ marginRight: '4px' }} /> Light</>
            )}
          </div>
        </div>
      </footer>

      <SettingsModal 
        isOpen={showSettings} 
        onClose={() => setShowSettings(false)}
        settings={appSettings}
        onUpdateSettings={updateGlobalSettings}
        onReset={handleResetApp}
      />

      <ProfileSelectionModal
        isOpen={showProfileModal}
        onClose={() => setShowProfileModal(false)}
        profiles={profiles}
        selectedProfileId={selectedProfileId}
        onSelectProfile={handleSelectProfile}
        onDeleteProfile={deleteProfile}
        onCreateProfile={() => setDrawer({ type: "add-profile" })}
        configPath={configPath}
      />

      {/* Drawer Overlay */}
      {drawer && (
        <div className="drawer-overlay" onClick={() => setDrawer(null)}>
          <div className="drawer-content" onClick={e => e.stopPropagation()}>
            <div className="drawer-header">
              <span className="drawer-title">
                {drawer.type === "test-tool" ? `Test Tool: ${drawer.data}` : 
                 drawer.type === "auth-config" ? "Manage Authentication" :
                 drawer.type === "add-custom-tool" ? "Bring Your Own Tool" : "Add Profile"}
              </span>
              <button className="icon-btn" onClick={() => setDrawer(null)}><DismissRegular /></button>
            </div>

            {drawer.type === "auth-config" && (() => {
              const tool = allTools.find(t => t.name === drawer.data);
              if (!tool || !tool.authorization || !selectedProfile) return null;
              
              const auth = tool.authorization;
              const envVars = auth.type === 'custom' ? (auth.env_vars || []) : (auth.env_var ? [{ name: auth.env_var, display_name: auth.display_name || auth.env_var, description: auth.description, required: true }] : []);
              
              return (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                  <div style={{ background: 'var(--background-card)', border: '1px solid var(--border-subtle)', padding: '16px', borderRadius: '8px' }}>
                    <div style={{ fontWeight: 600, fontSize: '14px', marginBottom: '4px' }}>{tool.title || tool.name}</div>
                    <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Auth Type: <strong>{auth.type.toUpperCase()}</strong></div>
                  </div>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    {envVars.map(v => {
                      const isConfigured = !!selectedProfile.env?.[v.name];
                      return (
                        <div key={v.name} className="form-field">
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                            <label style={{ marginBottom: 0 }}>{v.display_name}</label>
                            <code style={{ fontSize: '10px', opacity: 0.6 }}>{v.name}</code>
                          </div>
                          
                          {isConfigured ? (
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '10px', background: 'var(--log-bg)', border: '1px solid var(--border-subtle)', borderRadius: '6px' }}>
                              {editingAuthKey === v.name ? (
                                <>
                                  <input 
                                    type="text"
                                    value={editingAuthValue}
                                    onChange={(e) => setEditingAuthValue(e.target.value)}
                                    style={{ flex: 1, padding: '2px 8px', fontSize: '12px', height: '24px' }}
                                    autoFocus
                                    onKeyDown={(e) => {
                                      if (e.key === 'Enter') {
                                        updateProfileEnv(selectedProfile.id, { [v.name]: editingAuthValue });
                                        setEditingAuthKey(null);
                                      } else if (e.key === 'Escape') {
                                        setEditingAuthKey(null);
                                      }
                                    }}
                                  />
                                  <button 
                                    className="icon-btn" 
                                    onClick={() => {
                                      updateProfileEnv(selectedProfile.id, { [v.name]: editingAuthValue });
                                      setEditingAuthKey(null);
                                    }}
                                    title="Save"
                                  >
                                    <CheckmarkRegular style={{ fontSize: '16px', color: 'var(--accent-primary)' }} />
                                  </button>
                                  <button 
                                    className="icon-btn" 
                                    onClick={() => setEditingAuthKey(null)}
                                    title="Cancel"
                                  >
                                    <DismissRegular style={{ fontSize: '16px' }} />
                                  </button>
                                </>
                              ) : (
                                <>
                                  <code style={{ flex: 1, fontSize: '12px', color: 'var(--accent-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                    {revealedAuthKeys[v.name] ? selectedProfile.env[v.name] : '••••••••••••'}
                                  </code>
                                  <button 
                                    className="icon-btn" 
                                    onClick={() => setRevealedAuthKeys({ ...revealedAuthKeys, [v.name]: !revealedAuthKeys[v.name] })}
                                    title={revealedAuthKeys[v.name] ? "Hide" : "Reveal"}
                                  >
                                    {revealedAuthKeys[v.name] ? <EyeOffRegular style={{ fontSize: '16px' }} /> : <EyeRegular style={{ fontSize: '16px' }} />}
                                  </button>
                                  <button 
                                    className="icon-btn" 
                                    onClick={() => {
                                      setEditingAuthKey(v.name);
                                      setEditingAuthValue(selectedProfile.env[v.name]);
                                    }}
                                    title="Edit"
                                  >
                                    <EditRegular style={{ fontSize: '16px' }} />
                                  </button>
                                </>
                              )}
                            </div>
                          ) : (
                            <div style={{ display: 'flex', gap: '8px' }}>
                              <input 
                                type="password"
                                placeholder={`Enter ${v.display_name}...`}
                                value={authInput[v.name] || ''}
                                onChange={(e) => setAuthInput({ ...authInput, [v.name]: e.target.value })}
                                style={{ flex: 1 }}
                              />
                              <button 
                                className="primary"
                                onClick={() => {
                                  if (authInput[v.name]) {
                                    updateProfileEnv(selectedProfile.id, { [v.name]: authInput[v.name] });
                                    const newAuthInput = { ...authInput };
                                    delete newAuthInput[v.name];
                                    setAuthInput(newAuthInput);
                                  }
                                }}
                                disabled={!authInput[v.name]}
                              >
                                Save
                              </button>
                            </div>
                          )}
                          {v.description && <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginTop: '4px' }}>{v.description}</div>}
                        </div>
                      );
                    })}
                  </div>

                  {auth.help_url && (
                    <a 
                      href={auth.help_url} 
                      target="_blank" 
                      rel="noopener noreferrer"
                      className="tab-link"
                      style={{ textAlign: 'center', padding: '12px', background: 'var(--card-hover)', borderRadius: '6px', textDecoration: 'none', color: 'var(--accent-primary)', fontSize: '12px', fontWeight: 600 }}
                    >
                      Need help getting an API key? ↗
                    </a>
                  )}

                  <button 
                    style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', fontSize: '11px', cursor: 'pointer', opacity: 0.7, textDecoration: 'underline' }}
                    onClick={async () => {
                      if (configPath) {
                        try {
                          await revealItemInDir(configPath);
                        } catch (err) {
                          console.error("Failed to open config file location:", err);
                          setShowProfileModal(true);
                        }
                      } else {
                        setShowProfileModal(true);
                      }
                    }}
                  >
                    Manage all profile variables manually
                  </button>
                </div>
              );
            })()}

            {drawer.type === "test-tool" && (() => {
              const tool = allTools.find(t => t.name === drawer.data);
              return (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                  <div className="form-field">
                    <label>Select Tool/Function</label>
                    {tool?.authorization && selectedProfile && (
                      (() => {
                        const auth = tool.authorization;
                        const envVars = auth.type === 'custom' ? (auth.env_vars || []) : (auth.env_var ? [{ name: auth.env_var, display_name: auth.display_name || auth.env_var, required: true }] : []);
                        const missingVars = envVars.filter(v => v.required && !selectedProfile.env?.[v.name]);
                        const configuredVars = envVars.filter(v => selectedProfile.env?.[v.name]);
                        
                        return (
                          <div style={{ marginBottom: '12px' }}>
                            {missingVars.length > 0 && (
                              <div style={{ fontSize: '11px', color: '#b38600', marginBottom: '8px', padding: '12px', background: 'rgba(255, 204, 0, 0.05)', borderRadius: '6px', border: '1px solid rgba(255, 204, 0, 0.2)', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                <div style={{ fontWeight: 600 }}><WarningRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> Missing Setup</div>
                                {missingVars.map(v => (
                                  <div key={v.name} style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                      <span>{v.display_name}</span>
                                      <code style={{ opacity: 0.6 }}>{v.name}</code>
                                    </div>
                                    <div style={{ display: 'flex', gap: '4px' }}>
                                      <input 
                                        type="password"
                                        placeholder="Enter value..."
                                        value={authInput[v.name] || ''}
                                        onChange={(e) => setAuthInput({ ...authInput, [v.name]: e.target.value })}
                                        style={{ flex: 1, padding: '4px 8px', fontSize: '11px', borderRadius: '4px', border: '1px solid var(--border-subtle)', background: 'var(--background-card)' }}
                                      />
                                      <button 
                                        className="primary"
                                        onClick={() => {
                                          if (authInput[v.name]) {
                                            updateProfileEnv(selectedProfile.id, { [v.name]: authInput[v.name] });
                                            const newAuthInput = { ...authInput };
                                            delete newAuthInput[v.name];
                                            setAuthInput(newAuthInput);
                                          }
                                        }}
                                        style={{ padding: '2px 8px', fontSize: '10px' }}
                                      >
                                        Save
                                      </button>
                                    </div>
                                  </div>
                                ))}
                                <button 
                                  style={{ background: 'transparent', border: 'none', color: '#b38600', fontSize: '11px', cursor: 'pointer', opacity: 0.7, textDecoration: 'underline', alignSelf: 'flex-end', marginTop: '4px' }}
                                  onClick={async () => {
                                    if (configPath) {
                                      try {
                                        await revealItemInDir(configPath);
                                      } catch (err) {
                                        console.error("Failed to open config file location:", err);
                                        setShowProfileModal(true);
                                      }
                                    } else {
                                      setShowProfileModal(true);
                                    }
                                  }}
                                >
                                  Manage all variables in profile
                                </button>
                              </div>
                            )}

                            {configuredVars.length > 0 && (
                              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                                {configuredVars.map(v => (
                                  <div key={v.name} style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '4px 8px', background: 'var(--background-card)', border: '1px solid var(--border-subtle)', borderRadius: '12px', fontSize: '10px' }}>
                                    <span style={{ opacity: 0.6 }}>{v.display_name}:</span>
                                    <code>{revealedAuthKeys[v.name] ? selectedProfile.env[v.name] : '••••••'}</code>
                                    <button 
                                      style={{ background: 'transparent', border: 'none', padding: '0', cursor: 'pointer', opacity: 0.5, display: 'flex', alignItems: 'center' }}
                                      onClick={() => setRevealedAuthKeys({ ...revealedAuthKeys, [v.name]: !revealedAuthKeys[v.name] })}
                                    >
                                      {revealedAuthKeys[v.name] ? <EyeOffRegular style={{ fontSize: '14px' }} /> : <EyeRegular style={{ fontSize: '14px' }} />}
                                    </button>
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                        );
                      })()
                    )}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '200px', overflowY: 'auto', padding: '4px' }}>
                      {tool?.tools && tool.tools.length > 0 ? (
                        tool.tools.map(t => (
                          <div 
                            key={t.name}
                            onClick={() => {
                              setSelectedFunctionName(t.name);
                              setToolInput(getToolInput(tool, t.name));
                            }}
                            style={{ 
                              padding: '10px', 
                              borderRadius: '6px', 
                              border: '1px solid var(--border-subtle)', 
                              background: selectedFunctionName === t.name ? 'var(--card-hover)' : 'var(--background-card)',
                              borderColor: selectedFunctionName === t.name ? 'var(--accent-primary)' : 'var(--border-subtle)',
                              cursor: 'pointer',
                              transition: 'all 0.2s'
                            }}
                          >
                            <div style={{ fontWeight: 600, fontSize: '13px' }}>{t.title || t.name}</div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginTop: '2px' }}>{t.description}</div>
                          </div>
                        ))
                      ) : (
                        <div style={{ padding: '10px', borderRadius: '6px', border: '1px solid var(--accent-primary)', background: 'var(--card-hover)' }}>
                          <div style={{ fontWeight: 600, fontSize: '13px' }}>{tool?.name}</div>
                          <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginTop: '2px' }}>{tool?.description}</div>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="form-field">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                      <label style={{ marginBottom: 0 }}>Input Parameters (JSON)</label>
                      {selectedFunctionName && (
                        <code style={{ fontSize: '10px', opacity: 0.7 }}>{selectedFunctionName}</code>
                      )}
                    </div>
                    <textarea
                      rows={12}
                      value={toolInput}
                      onChange={(e) => setToolInput(e.target.value)}
                      onBlur={() => {
                        // Save params when user finishes editing
                        if (selectedFunctionName && toolInput) {
                          try {
                            const parsed = JSON.parse(toolInput);
                            saveToolParams(selectedFunctionName, parsed);
                          } catch {
                            // Invalid JSON, don't save
                          }
                        }
                      }}
                      placeholder="{}"
                      style={{ 
                        fontFamily: "Google Sans Code, monospace", 
                        fontSize: '12px',
                        background: 'var(--log-bg)',
                        border: '1px solid var(--border-subtle)',
                        borderRadius: '6px',
                        padding: '12px'
                      }}
                    ></textarea>
                    <button 
                      className="primary" 
                      style={{ marginTop: "12px", padding: '10px', fontSize: '14px', position: 'relative' }} 
                      onClick={invokeTool}
                      disabled={!selectedFunctionName || testResult?.status === 'loading'}
                    >
                      {testResult?.status === 'loading' ? 'Invoking...' : 'Invoke Tool'}
                    </button>
                  </div>

                  {testResult && (
                    <div className="detail-section" style={{ marginTop: '0' }}>
                      <h3 style={{ fontSize: '11px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '8px' }}>
                        Result {testResult.status === 'success' ? <CheckmarkCircleRegular style={{ color: 'var(--accent-primary)', verticalAlign: 'middle', marginLeft: '4px' }} /> : testResult.status === 'error' ? <ErrorCircleRegular style={{ color: '#ff4d4d', verticalAlign: 'middle', marginLeft: '4px' }} /> : ''}
                      </h3>
                      <div style={{ 
                        background: testResult.status === 'error' ? 'rgba(255, 77, 77, 0.05)' : 'var(--background-card)',
                        border: `1px solid ${testResult.status === 'error' ? 'rgba(255, 77, 77, 0.2)' : 'var(--border-subtle)'}`,
                        borderRadius: '6px',
                        padding: '12px',
                        maxHeight: '300px',
                        overflowY: 'auto',
                        fontFamily: "Google Sans Code, monospace",
                        fontSize: '12px'
                      }}>
                        {testResult.status === 'success' ? (
                          <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--text-primary)' }}>
                            {JSON.stringify(testResult.data, null, 2)}
                          </pre>
                        ) : testResult.status === 'error' ? (
                          <div style={{ color: '#ff4d4d' }}>
                            {typeof testResult.data === 'string' ? testResult.data : JSON.stringify(testResult.data, null, 2)}
                            {testResult.data?.message === "tool not found: " + selectedFunctionName && (
                              <div style={{ marginTop: '8px', fontSize: '11px', opacity: 0.8 }}>
                                Tip: Make sure this tool is <strong>Activated</strong> in your profile before testing.
                              </div>
                            )}
                          </div>
                        ) : null}
                      </div>
                    </div>
                  )}

                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)', background: 'var(--background-card)', padding: '12px', borderRadius: '6px', border: '1px solid var(--border-subtle)' }}>
                    {selectedProfile ? (
                      <>
                        <strong>Note:</strong> Tool calls are routed through the <code>{selectedProfile.id}</code> profile on port <code>{appSettings.mcp_port}</code>.
                      </>
                    ) : (
                      <strong style={{ color: '#ff4d4d' }}><WarningRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> No profile selected. Please select or create a profile first.</strong>
                    )}
                  </div>
                </div>
              );
            })()}

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
