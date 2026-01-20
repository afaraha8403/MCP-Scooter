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
  WeatherMoonRegular,
  WeatherSunnyRegular,
  ArrowLeftRegular,
  ChevronDownRegular,
  CopyRegular,
  PhoneLaptopRegular,
  AddRegular,
  BuildingRegular,
  PeopleRegular,
  WindowRegular,
  LaptopRegular,
  DesktopRegular,
  PhoneRegular,
  GlobeRegular,
  DeleteRegular,
  FolderOpenRegular
} from "@fluentui/react-icons";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { SettingsModal } from "./components/SettingsModal";
import { ProfileSelectionModal } from "./components/ProfileSelectionModal";
import { JsonEditor } from "./components/JsonEditor";
import "vanilla-jsoneditor/themes/jse-theme-dark.css";
import { invoke } from "@tauri-apps/api/core";
import { revealItemInDir } from "@tauri-apps/plugin-opener";
import "./App.css";

interface Profile {
  id: string;
  remote_auth_mode: string;
  remote_server_url: string;
  env: Record<string, string>;
  allow_tools: string[];
  disabled_system_tools?: string[];
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
  icon_background?: {
    light?: string;
    dark?: string;
  };
  about?: string;
  tags?: string[];
  homepage?: string;
  repository?: string;
  documentation?: string;
  authorization?: {
    type: string;
    required?: boolean;
    recommended?: boolean;
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
  icon_dark?: string;
  description: string;
  manual_instructions: string;
  version?: string;
  developer?: string;
  category?: string;
  tags?: string[];
  about?: string;
  homepage?: string;
  repository?: string;
  documentation?: string;
  download_url?: string;
  platforms?: string[];
  installed: boolean;
  mcp_support?: {
    transports: string[];
    features: string[];
    status: "stable" | "beta" | "experimental";
  };
  features?: {
    name: string;
    description: string;
  }[];
  metadata?: {
    license?: string;
    pricing?: string;
  };
}

const getPlatformIcon = (platform: string) => {
  const p = platform.toLowerCase();
  if (p.includes('windows')) return <WindowRegular />;
  if (p.includes('macos')) return <LaptopRegular />;
  if (p.includes('linux')) return <DesktopRegular />;
  if (p.includes('ios')) return <PhoneRegular />;
  if (p.includes('android')) return <PhoneRegular />;
  if (p.includes('web')) return <GlobeRegular />;
  return <PhoneLaptopRegular />;
};

const isCurrentPlatform = (platform: string) => {
  const p = platform.toLowerCase();
  const ua = navigator.userAgent.toLowerCase();
  if (p.includes('windows') && ua.includes('win')) return true;
  if (p.includes('macos') && ua.includes('mac')) return true;
  if (p.includes('linux') && ua.includes('linux')) return true;
  if (p.includes('ios') && (ua.includes('iphone') || ua.includes('ipad'))) return true;
  if (p.includes('android') && ua.includes('android')) return true;
  return false;
};

const CopyButton = ({ text }: { text: string }) => {
  const [copied, setCopied] = useState(false);
  
  const handleCopy = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button 
      className={`copy-code-btn ${copied ? 'copied' : ''}`}
      onClick={handleCopy}
      title="Copy to clipboard"
    >
      {copied ? <CheckmarkCircleRegular style={{ fontSize: '12px' }} /> : <CopyRegular style={{ fontSize: '12px' }} />}
      {copied ? 'Copied' : 'Copy'}
    </button>
  );
};

const markdownComponents = {
  pre: ({ children }: any) => {
    // Extract text content from children (which is usually a <code> element)
    const getCodeText = (node: any): string => {
      if (typeof node === 'string') return node;
      if (Array.isArray(node)) return node.map(getCodeText).join('');
      if (node?.props?.children) return getCodeText(node.props.children);
      return '';
    };
    
    const text = getCodeText(children);
    
    return (
      <div className="code-block-wrapper">
        <CopyButton text={text} />
        <pre>{children}</pre>
      </div>
    );
  }
};

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
  const [selectedClient, setSelectedClient] = useState<ClientDefinition | null>(null);
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
  const [activeTab, setActiveTab] = useState<"active" | "catalog" | "clients" | "logs">("active");
  const [searchQuery, setSearchQuery] = useState("");
  const [toolFilter, setToolFilter] = useState<"all" | "official" | "community" | "custom">("all");
  const [newProfile, setNewProfile] = useState({ id: "" });
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
  const [optionalAuthExpanded, setOptionalAuthExpanded] = useState(false);

  const [logSearchQuery, setLogSearchQuery] = useState("");
  const [logLevelFilter, setLogLevelFilter] = useState<"ALL" | "INFO" | "WARN" | "ERROR">("ALL");

  // Load saved tool params on mount
  useEffect(() => {
    const loadSavedParams = async () => {
      try {
        const res = await fetch(`http://127.0.0.1:${appSettings.control_port}/api/tool-params`);
        if (res.ok) {
          const data = await res.json();
          setSavedToolParams(data || {});
        }
      } catch (err) {
        console.log("Could not load saved tool params:", err);
      }
    };
    loadSavedParams();

    // Fetch initial logs
    const loadLogs = async () => {
      try {
        const res = await fetch(`http://127.0.0.1:${appSettings.control_port}/api/logs`);
        if (res.ok) {
          const data = await res.json();
          if (data.logs) {
            setLogs(data.logs);
          }
        }
      } catch (err) {
        console.log("Could not load logs:", err);
      }
    };
    loadLogs();

    // Subscribe to real-time logs
    const eventSource = new EventSource(`http://127.0.0.1:${appSettings.control_port}/api/logs/stream`);
    
    eventSource.addEventListener('log', (event) => {
      try {
        const log = JSON.parse(event.data);
        setLogs(prev => [log, ...prev].slice(0, 1000));
      } catch (err) {
        console.error("Failed to parse log from SSE:", err);
      }
    });

    eventSource.onerror = (err) => {
      console.error("SSE connection error:", err);
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [appSettings.control_port]);

  // Reset optional auth accordion when tool changes
  useEffect(() => {
    setOptionalAuthExpanded(false);
  }, [selectedTool?.name]);

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

  const getThemedIcon = (iconPath: string | undefined) => {
    if (!iconPath) return iconPath;
    
    // Check for _light or _dark suffix before the extension
    const match = iconPath.match(/(.*)_(light|dark)\.([a-zA-Z0-9]+)$/);
    if (!match) return iconPath;
    
    const [_, base, currentTheme, ext] = match;
    const targetTheme = theme === 'dark' ? 'dark' : 'light';
    
    if (currentTheme === targetTheme) return iconPath;
    
    return `${base}_${targetTheme}.${ext}`;
  };

  // Save tool params when modified
  const saveToolParams = async (functionName: string, params: Record<string, any>) => {
    try {
      await fetch(`http://127.0.0.1:${appSettings.control_port}/api/tool-params`, {
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

  const formatLogTimestamp = (ts: string) => {
    try {
      const date = new Date(ts);
      if (isNaN(date.getTime())) return ts;
      return date.toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return ts;
    }
  };

  const getLogLevelStyle = (level: string): React.CSSProperties => {
    const base: React.CSSProperties = {
      padding: '1px 6px',
      borderRadius: '4px',
      fontSize: '10px',
      fontWeight: 700,
      textTransform: 'uppercase',
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      minWidth: '45px',
      marginRight: '8px'
    };

    switch (level) {
      case 'ERROR':
        return { ...base, background: 'rgba(255, 77, 77, 0.15)', color: '#ff4d4d', border: '1px solid rgba(255, 77, 77, 0.2)' };
      case 'WARN':
      case 'WARNING':
        return { ...base, background: 'rgba(255, 204, 0, 0.15)', color: '#ffcc00', border: '1px solid rgba(255, 204, 0, 0.2)' };
      case 'SUCCESS':
        return { ...base, background: 'rgba(0, 200, 83, 0.15)', color: '#00c853', border: '1px solid rgba(0, 200, 83, 0.2)' };
      case 'DEBUG':
        return { ...base, background: 'rgba(156, 39, 176, 0.15)', color: '#9c27b0', border: '1px solid rgba(156, 39, 176, 0.2)' };
      default: // INFO
        return { ...base, background: 'rgba(0, 109, 91, 0.15)', color: 'var(--accent-primary)', border: '1px solid var(--border-subtle)' };
    }
  };

  const filteredLogs = logs
    .filter(log => {
      if (logLevelFilter !== "ALL" && log.level !== logLevelFilter) return false;
      if (logSearchQuery && !log.message.toLowerCase().includes(logSearchQuery.toLowerCase())) return false;
      return true;
    });

  const clearLogs = async () => {
    try {
      const res = await fetch(`http://127.0.0.1:${appSettings.control_port}/api/logs`, {
        method: "DELETE"
      });
      if (res.ok) {
        setLogs([]);
        addLog("Logs cleared.", "INFO");
      }
    } catch (err) {
      console.log("Could not clear logs:", err);
    }
  };

  const revealLogs = async () => {
    try {
      await fetch(`http://127.0.0.1:${appSettings.control_port}/api/logs/reveal`, {
        method: "POST"
      });
    } catch (err) {
      console.log("Could not reveal logs:", err);
    }
  };

  const CONTROL_API = `http://127.0.0.1:${appSettings.control_port}/api`;

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
        setAppSettings(data.settings);
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
    
    // Soft reset UI state
    setActiveTab("active");
    setSelectedTool(null);
    setSelectedClient(null);
    setSearchQuery("");
    setShowProfileModal(false);
    addLog(`Switched to profile: ${profileId}`, "INFO");

    // Save the last selected profile to settings
    try {
      const newSettings = { ...appSettings, last_profile_id: profileId };
      await fetch(`${CONTROL_API}/settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(newSettings),
      });
      setAppSettings(newSettings);
      
      // Refresh data for the new profile
      fetchProfiles();
      fetchAllTools();
      fetchClients();
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
    // Only update local state if SSE is not active or for immediate feedback
    // But since SSE will push it back, we can just send it to the backend
    fetch(`http://127.0.0.1:${appSettings.control_port}/api/logs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ level, message })
    }).catch(err => {
      console.error("Failed to send log to backend:", err);
      // Fallback to local state if backend is unreachable
      setLogs(prev => [{ timestamp: new Date().toLocaleTimeString(), level, message }, ...prev].slice(0, 1000));
    });
  };

  // Separate builtin/primordial tools from installable tools
  const builtinTools = allTools.filter(t => t.source === "builtin");
  
  const filteredTools = allTools
    .filter(t => {
      // Exclude builtin tools from the catalog - they're always available
      if (t.source === "builtin") return false;
      
      const matchesSearch = 
        (t.name || "").toLowerCase().includes(searchQuery.toLowerCase()) || 
        (t.description || "").toLowerCase().includes(searchQuery.toLowerCase()) ||
        (t.title || "").toLowerCase().includes(searchQuery.toLowerCase());

      const matchesFilter = 
        toolFilter === "all" || 
        (toolFilter === "official" && t.source === "official") ||
        (toolFilter === "community" && t.source === "community") ||
        (toolFilter === "custom" && (t.source !== "official" && t.source !== "community"));

      return matchesSearch && matchesFilter;
    })
    .sort((a, b) => {
      const aName = a.title || a.name || "";
      const bName = b.title || b.name || "";
      
      // Official tools first
      if (a.source === 'official' && b.source !== 'official') return -1;
      if (a.source !== 'official' && b.source === 'official') return 1;
      
      return aName.localeCompare(bName);
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

  const deleteTool = async (name: string) => {
    if (!confirm(`Are you sure you want to delete custom tool "${name}"?`)) return;
    try {
      const res = await fetch(`${CONTROL_API}/tools?name=${name}`, { method: "DELETE" });
      if (res.ok) {
        addLog(`Deleted custom tool: ${name}`, "INFO");
        fetchAllTools(); // Refresh tool list
      } else {
        addLog(`Failed to delete custom tool: ${name}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error deleting custom tool: ${err}`, "ERROR");
    }
  };

  const updateProfileTools = async (profileId: string, tools: string[]) => {
    const profile = profiles.find(p => p.id === profileId);
    if (!profile) return;

    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          ...profile, 
          allow_tools: tools,
          disabled_system_tools: profile.disabled_system_tools || []
        }),
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

  const toggleSystemTool = async (toolName: string) => {
    if (!selectedProfile) return;

    const currentDisabled = selectedProfile.disabled_system_tools || [];
    const isCurrentlyDisabled = currentDisabled.includes(toolName);
    
    const newDisabled = isCurrentlyDisabled 
      ? currentDisabled.filter(t => t !== toolName)  // Enable: remove from disabled list
      : [...currentDisabled, toolName];               // Disable: add to disabled list

    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...selectedProfile, disabled_system_tools: newDisabled }),
      });
      if (res.ok) {
        const action = isCurrentlyDisabled ? "enabled" : "disabled";
        addLog(`System tool ${toolName} ${action}`, "INFO");
        fetchProfiles();
      } else {
        addLog(`Failed to toggle system tool: ${toolName}`, "ERROR");
      }
    } catch (err) {
      addLog(`Error toggling system tool: ${err}`, "ERROR");
    }
  };

  const updateProfileEnv = async (profileId: string, env: Record<string, string>) => {
    const profile = profiles.find(p => p.id === profileId);
    if (!profile) return;

    try {
      const res = await fetch(`${CONTROL_API}/profiles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          ...profile, 
          env: { ...(profile.env || {}), ...env },
          disabled_system_tools: profile.disabled_system_tools || []
        }),
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
      const url = `http://127.0.0.1:${appSettings.mcp_port}/profiles/${selectedProfile.id}/message`;
      
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
            <span className="app-name">MCP Scooter</span>
          </div>
          
          <div className="profile-controls-group">
            <div className="profile-selector-container" onClick={() => setShowProfileModal(true)}>
              <div className="profile-label-group">
                <span className="profile-label">Profile</span>
                <span className="profile-id-text">{selectedProfileId || 'Select Profile'}</span>
              </div>
              <ChevronDownRegular className="profile-chevron" />
            </div>

            <button className="add-profile-btn" onClick={() => setDrawer({ type: "add-profile" })} title="Add New Profile">
              <AddRegular />
            </button>
          </div>
        </div>

        {/* Main Grid */}
        <div className="main-grid catalog-full">
          {/* Detail View */}
          {selectedTool && (
            <section className="section-container detail-view">
              <header className="section-header">
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <button className="back-button" onClick={() => setSelectedTool(null)} title="Back to Catalog">
                    <ArrowLeftRegular />
                    <span>Back</span>
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
                  {(selectedTool.source === 'official' || selectedTool.source === 'community') && (
                    <button 
                      className="secondary" 
                      onClick={(e) => {
                        e.stopPropagation();
                        // Clone official/community tool to custom
                        const customTool = { ...selectedTool, source: 'custom', name: `${selectedTool.name}-custom`, title: `${selectedTool.title || selectedTool.name} (Custom)` };
                        setDrawer({ type: "add-custom-tool", data: JSON.stringify(customTool, null, 2) });
                      }}
                    >
                      Duplicate
                    </button>
                  )}
                  {selectedTool.source === 'custom' && (
                    <button 
                      className="secondary" 
                      onClick={(e) => {
                        e.stopPropagation();
                        setDrawer({ type: "add-custom-tool", data: JSON.stringify(selectedTool, null, 2) });
                      }}
                    >
                      Customize
                    </button>
                  )}
                  <button 
                    className="secondary"
                    onClick={(e) => { e.stopPropagation(); setDrawer({ type: "test-tool", data: selectedTool.name }); }}
                  >
                    Test
                  </button>
                  
                  {selectedTool.authorization && (
                    selectedTool.authorization.type === 'none' ? (
                      <div 
                        title="No Authentication Required"
                        className="auth-status-badge"
                      >
                        <CheckmarkRegular className="status-icon" />
                        <span>Auth Not Required</span>
                      </div>
                    ) : selectedTool.authorization.recommended ? (
                      <button 
                        onClick={(e) => { e.stopPropagation(); setDrawer({ type: "auth-config", data: selectedTool.name }); }}
                        title="Authentication is optional but recommended"
                        className="auth-btn recommended"
                      >
                        <KeyRegular className="auth-btn-icon" />
                        <span>Auth (Optional)</span>
                      </button>
                    ) : selectedTool.authorization.required === false ? (
                      <div 
                        title="No Authentication Required"
                        className="auth-status-badge"
                      >
                        <CheckmarkRegular className="status-icon" />
                        <span>Auth Not Required</span>
                      </div>
                    ) : (
                      <button 
                        onClick={(e) => { e.stopPropagation(); setDrawer({ type: "auth-config", data: selectedTool.name }); }}
                        title="Manage Authentication"
                        className="auth-btn"
                      >
                        <KeyRegular className="auth-btn-icon" />
                        <span>Auth</span>
                      </button>
                    )
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
                </div>
              </header>

              <div className="scroll-section detail-content" style={{ padding: '24px' }}>
                <div style={{ display: 'flex', gap: '32px', marginBottom: '32px' }}>
                  {selectedTool.icon ? (
                    <img 
                      src={getThemedIcon(selectedTool.icon)} 
                      alt={selectedTool.name} 
                      style={{ 
                        width: '80px', 
                        height: '80px', 
                        objectFit: 'contain',
                        backgroundColor: selectedTool.icon_background ? (theme === 'light' ? selectedTool.icon_background.light : selectedTool.icon_background.dark) : 'transparent',
                        borderRadius: '16px',
                        padding: '8px',
                        boxSizing: 'border-box'
                      }} 
                    />
                  ) : (
                    <img 
                      src={getThemedIcon("/registry-logos/mcp_fallback_light.svg")} 
                      alt={selectedTool.name} 
                      style={{ 
                        width: '80px', 
                        height: '80px', 
                        objectFit: 'contain',
                        borderRadius: '16px',
                        padding: '8px',
                        boxSizing: 'border-box',
                        background: 'var(--background-card)',
                        border: '1px solid var(--border-subtle)'
                      }} 
                    />
                  )}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', flex: 1 }}>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <span className="badge">{selectedTool.category || 'Utility'}</span>
                      {selectedTool.source === 'official' && (
                        <span className="badge" style={{ background: 'var(--accent-primary)', color: 'white', display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <BuildingRegular style={{ fontSize: '11px' }} />
                          Official
                        </span>
                      )}
                      {selectedTool.source === 'community' && (
                        <span className="badge" style={{ background: 'var(--text-secondary)', color: 'white', display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <PeopleRegular style={{ fontSize: '11px' }} />
                          Community
                        </span>
                      )}
                      {selectedTool.source === 'enterprise' && (
                        <span className="badge" style={{ background: 'var(--text-secondary)', color: 'white', display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <BoxRegular style={{ fontSize: '11px' }} />
                          Enterprise
                        </span>
                      )}
                      {selectedTool.source !== 'official' && selectedTool.source !== 'community' && selectedTool.source !== 'enterprise' && (
                        <span className="badge" style={{ background: 'var(--text-secondary)', color: 'white' }}>{selectedTool.source === 'local' ? 'Local' : 'Custom'}</span>
                      )}
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

                  {selectedTool.authorization && selectedTool.authorization.type !== 'none' && (selectedTool.authorization.required !== false || selectedTool.authorization.recommended === true) && selectedProfile && (
                    (() => {
                      const auth = selectedTool.authorization;
                      const isRecommended = auth.recommended === true;
                      const envVars = auth.type === 'custom' ? (auth.env_vars || []) : (auth.env_var ? [{ name: auth.env_var, display_name: auth.display_name || auth.env_var, description: auth.description, required: !isRecommended }] : []);
                      const missingVars = isRecommended 
                        ? envVars.filter(v => !selectedProfile.env?.[v.name])
                        : envVars.filter(v => v.required && !selectedProfile.env?.[v.name]);
                      const configuredVars = envVars.filter(v => selectedProfile.env?.[v.name]);

                      return (
                        <div style={{ marginBottom: '32px' }}>
                          {missingVars.length > 0 && (
                            isRecommended ? (
                              // Collapsible accordion for optional/recommended auth
                              <div style={{ 
                                fontSize: '12px', 
                                background: 'var(--background-card)', 
                                border: '1px solid var(--border-subtle)', 
                                borderRadius: '8px', 
                                color: 'var(--text-secondary)', 
                                marginBottom: configuredVars.length > 0 ? '12px' : '0',
                                overflow: 'hidden'
                              }}>
                                <button
                                  onClick={() => setOptionalAuthExpanded(!optionalAuthExpanded)}
                                  style={{
                                    width: '100%',
                                    padding: '16px 24px',
                                    background: 'transparent',
                                    border: 'none',
                                    cursor: 'pointer',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    color: 'var(--text-primary)',
                                    fontSize: '14px',
                                    fontWeight: 600
                                  }}
                                >
                                  <span style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                    <KeyRegular style={{ opacity: 0.6 }} />
                                    Optional Configuration
                                  </span>
                                  <ChevronDownRegular style={{ 
                                    transition: 'transform 0.2s ease',
                                    transform: optionalAuthExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                                    opacity: 0.6
                                  }} />
                                </button>
                                
                                {optionalAuthExpanded && (
                                  <div style={{ padding: '0 24px 24px 24px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                    <div style={{ fontSize: '13px', lineHeight: '1.4', color: 'var(--text-secondary)' }}>
                                      This tool works without authentication, but adding credentials can unlock better rate limits and features.
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
                                              style={{ flex: 1, padding: '10px', borderRadius: '6px', border: '1px solid var(--border-subtle)', background: 'var(--log-bg)', color: 'var(--text-primary)' }}
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

                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px', borderTop: '1px solid var(--border-subtle)', paddingTop: '12px' }}>
                                      {auth.help_url ? (
                                        <a 
                                          href={auth.help_url} 
                                          target="_blank" 
                                          rel="noopener noreferrer"
                                          style={{ color: 'var(--accent-primary)', textDecoration: 'none', fontSize: '12px', fontWeight: 500 }}
                                        >
                                          Need help getting started? ↗
                                        </a>
                                      ) : <div></div>}
                                      <button 
                                        style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', fontSize: '12px', cursor: 'pointer', opacity: 0.7, textDecoration: 'underline' }}
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
                              </div>
                            ) : (
                              // Non-collapsible required auth (original behavior)
                              <div style={{ 
                                fontSize: '12px', 
                                background: 'rgba(255, 204, 0, 0.1)', 
                                border: '1px solid rgba(255, 204, 0, 0.3)', 
                                padding: '24px', 
                                borderRadius: '8px', 
                                color: '#b38600', 
                                display: 'flex', 
                                flexDirection: 'column', 
                                gap: '16px', 
                                marginBottom: configuredVars.length > 0 ? '12px' : '0' 
                              }}>
                                <div style={{ fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px', fontSize: '14px' }}>
                                  <span>
                                    <WarningRegular style={{ verticalAlign: 'middle', marginRight: '4px', color: '#ffcc00' }} /> Configuration Required
                                  </span>
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
                            )
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
                          <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{selectedTool.about}</ReactMarkdown>
                        </div>
                      </div>
                    )}

                    <div className="detail-section">
                      <h3 style={{ fontSize: '13px', textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span>Capabilities</span>
                        {selectedTool.tools && selectedTool.tools.length > 0 && (
                          <span style={{ fontSize: '11px', opacity: 0.6, background: 'var(--background-card)', padding: '2px 8px', borderRadius: '10px', border: '1px solid var(--border-subtle)' }}>
                            {selectedTool.tools.length} {selectedTool.tools.length === 1 ? 'tool' : 'tools'}
                          </span>
                        )}
                      </h3>
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
                          {selectedTool.authorization.type === 'none' ? (
                            <div style={{ color: 'var(--text-secondary)' }}>
                              <CheckmarkRegular style={{ fontSize: '16px', color: 'var(--accent-primary)', verticalAlign: 'middle', marginRight: '8px' }} />
                              No authentication required for this MCP.
                            </div>
                          ) : selectedTool.authorization.recommended ? (
                            <>
                              <div style={{ fontWeight: 600, marginBottom: '4px', color: 'var(--text-primary)' }}>
                                {selectedTool.authorization.type === 'api_key' ? 'API Key (Optional)' : `${selectedTool.authorization.type} (Optional)`}
                              </div>
                              <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '8px', lineHeight: '1.4' }}>
                                Works without authentication, but adding credentials can improve rate limits.
                              </div>
                              {selectedTool.authorization.env_var && (
                                <div style={{ fontSize: '11px', opacity: 0.7, marginBottom: '8px' }}>
                                  Env: <code>{selectedTool.authorization.env_var}</code>
                                </div>
                              )}
                              {selectedTool.authorization.description && (
                                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.4' }}>{selectedTool.authorization.description}</div>
                              )}
                            </>
                          ) : selectedTool.authorization.required === false ? (
                            <div style={{ color: 'var(--text-secondary)' }}>
                              <CheckmarkRegular style={{ fontSize: '16px', color: 'var(--accent-primary)', verticalAlign: 'middle', marginRight: '8px' }} />
                              No authentication required for this MCP.
                            </div>
                          ) : (
                            <>
                              <div style={{ fontWeight: 600, marginBottom: '4px' }}>{selectedTool.authorization.type === 'api_key' ? 'API Key Required' : selectedTool.authorization.type}</div>
                              {selectedTool.authorization.env_var && (
                                <div style={{ fontSize: '11px', opacity: 0.7, marginBottom: '8px' }}>
                                  Env: <code>{selectedTool.authorization.env_var}</code>
                                </div>
                              )}
                              {selectedTool.authorization.description && (
                                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.4' }}>{selectedTool.authorization.description}</div>
                              )}
                            </>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </section>
          )}

          {selectedClient && (
            <section className="section-container detail-view">
              <header className="section-header">
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <button className="back-button" onClick={() => setSelectedClient(null)} title="Back to Clients">
                    <ArrowLeftRegular />
                    <span>Back</span>
                  </button>
                  <span style={{ fontSize: '18px', fontWeight: 700, textTransform: 'none', color: 'var(--text-primary)' }}>
                    {selectedClient.name}
                  </span>
                  {selectedClient.version && (
                    <span style={{ fontSize: '12px', color: 'var(--text-secondary)', fontWeight: 400 }}>v{selectedClient.version}</span>
                  )}
                </div>
                <div className="card-actions" style={{ display: 'flex', gap: '8px' }}>
                  {selectedClient.installed ? (
                    <button
                      className="primary"
                      onClick={async (e) => {
                        e.stopPropagation();
                        try {
                          const res = await fetch(`${CONTROL_API}/clients/sync`, {
                            method: "POST",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify({ target: selectedClient.id, profile: selectedProfileId })
                          });
                          if (res.ok) {
                            addLog(`Successfully configured ${selectedClient.name}.`, "INFO");
                            alert(`Successfully configured ${selectedClient.name}!`);
                          } else {
                            const err = await res.text();
                            addLog(`Failed to configure ${selectedClient.name}: ${err}`, "ERROR");
                            alert(`Failed to configure ${selectedClient.name}: ${err}`);
                          }
                        } catch (err: any) {
                          addLog(`Network error configuring ${selectedClient.name}: ${err.message}`, "ERROR");
                          alert(`Network error configuring ${selectedClient.name}`);
                        }
                      }}
                    >
                      Setup
                    </button>
                  ) : (
                    selectedClient.download_url ? (
                      <a
                        href={selectedClient.download_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="primary"
                        style={{ display: 'flex', alignItems: 'center', gap: '6px', textDecoration: 'none', padding: '8px 16px', borderRadius: '6px', fontSize: '13px', fontWeight: 600 }}
                      >
                        <ArrowDownloadRegular style={{ fontSize: '16px' }} />
                        Download
                      </a>
                    ) : (
                      <button className="primary" disabled>Not Installed</button>
                    )
                  )}
                </div>
              </header>

              <div className="scroll-section detail-content" style={{ padding: '24px' }}>
                {/* Hero Section */}
                <div style={{ display: 'flex', gap: '32px', marginBottom: '32px' }}>
                  {selectedClient.icon ? (
                    <img 
                      src={theme === 'dark' && selectedClient.icon_dark ? selectedClient.icon_dark : getThemedIcon(selectedClient.icon)} 
                      alt={selectedClient.name} 
                      style={{ 
                        width: '96px', 
                        height: '96px', 
                        objectFit: 'contain',
                        borderRadius: '20px',
                        padding: '12px',
                        boxSizing: 'border-box',
                        background: 'var(--background-card)',
                        border: '1px solid var(--border-subtle)'
                      }} 
                    />
                  ) : (
                    <div style={{ 
                      width: '96px', 
                      height: '96px', 
                      background: 'var(--border-subtle)', 
                      borderRadius: '20px', 
                      display: 'flex', 
                      alignItems: 'center', 
                      justifyContent: 'center' 
                    }}>
                      <PhoneLaptopRegular style={{ fontSize: '48px', opacity: 0.5 }} />
                    </div>
                  )}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', flex: 1 }}>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                      {selectedClient.category && <span className="badge">{selectedClient.category}</span>}
                      {selectedClient.mcp_support && (
                        <span className="badge" style={{ 
                          background: selectedClient.mcp_support.status === 'stable' ? 'rgba(0, 200, 83, 0.15)' : 
                                     selectedClient.mcp_support.status === 'beta' ? 'rgba(255, 193, 7, 0.15)' : 'rgba(156, 39, 176, 0.15)',
                          color: selectedClient.mcp_support.status === 'stable' ? '#00c853' : 
                                 selectedClient.mcp_support.status === 'beta' ? '#ffc107' : '#9c27b0'
                        }}>
                          MCP {selectedClient.mcp_support.status}
                        </span>
                      )}
                      {selectedClient.developer && (
                        <span style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                          <BuildingRegular style={{ fontSize: '14px' }} />
                          {selectedClient.developer}
                        </span>
                      )}
                    </div>
                    <p style={{ fontSize: '16px', lineHeight: '1.6', margin: 0, color: 'var(--text-primary)', fontWeight: 500 }}>
                      {selectedClient.description}
                    </p>
                    {selectedClient.tags && selectedClient.tags.length > 0 && (
                      <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                        {selectedClient.tags.map(tag => (
                          <span key={tag} style={{ 
                            fontSize: '11px', 
                            padding: '2px 8px', 
                            borderRadius: '10px', 
                            background: 'var(--card-hover)',
                            color: 'var(--text-secondary)'
                          }}>
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                {/* Two Column Layout */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 320px', gap: '32px' }}>
                  {/* Left Column - Main Content */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                    {/* About Section */}
                    {selectedClient.about && (
                      <div>
                        <div style={{ fontWeight: 700, marginBottom: '12px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7, display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <BookRegular style={{ fontSize: '14px' }} /> About
                        </div>
                        <div style={{ background: 'var(--background-card)', padding: '20px', borderRadius: '12px', border: '1px solid var(--border-subtle)' }}>
                          <div className="markdown-content" style={{ fontSize: '14px', lineHeight: '1.7', color: 'var(--text-primary)' }}>
                            <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{selectedClient.about}</ReactMarkdown>
                          </div>
                        </div>
                      </div>
                    )}

                    {/* Features Section */}
                    {selectedClient.features && selectedClient.features.length > 0 && (
                      <div>
                        <div style={{ fontWeight: 700, marginBottom: '12px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7, display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <RocketRegular style={{ fontSize: '14px' }} /> Key Features
                        </div>
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '12px' }}>
                          {selectedClient.features.map((feature, idx) => (
                            <div key={idx} style={{ 
                              background: 'var(--background-card)', 
                              padding: '16px', 
                              borderRadius: '10px', 
                              border: '1px solid var(--border-subtle)',
                              transition: 'border-color 0.2s'
                            }}>
                              <div style={{ fontWeight: 600, fontSize: '14px', marginBottom: '4px', color: 'var(--text-primary)' }}>
                                {feature.name}
                              </div>
                              <div style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.5' }}>
                                {feature.description}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Manual Configuration */}
                    <div>
                      <div style={{ fontWeight: 700, marginBottom: '12px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7 }}>Manual Configuration</div>
                      <div style={{ background: 'var(--log-bg)', padding: '20px', borderRadius: '12px', border: '1px solid var(--border-subtle)' }}>
                        <div className="markdown-content" style={{ fontSize: '13px', lineHeight: '1.6' }}>
                          <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{selectedClient.manual_instructions
                            .replace(/{profile}/g, selectedProfileId || 'work')
                            .replace(/6277/g, appSettings.mcp_port.toString())}</ReactMarkdown>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Right Column - Sidebar */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                    {/* Quick Info Card */}
                    <div style={{ background: 'var(--background-card)', padding: '20px', borderRadius: '12px', border: '1px solid var(--border-subtle)' }}>
                      <div style={{ fontWeight: 700, marginBottom: '16px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7 }}>Details</div>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                        {selectedClient.platforms && selectedClient.platforms.length > 0 && (
                          <div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Platforms</div>
                            <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                              {selectedClient.platforms.map(platform => {
                                const isCurrent = isCurrentPlatform(platform);
                                return (
                                  <span key={platform} style={{ 
                                    fontSize: '12px', 
                                    padding: '4px 10px', 
                                    borderRadius: '6px', 
                                    background: isCurrent ? 'var(--accent-primary)' : 'var(--card-hover)',
                                    color: isCurrent ? 'var(--accent-text)' : 'var(--text-primary)',
                                    fontWeight: 500,
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '6px'
                                  }}>
                                    {getPlatformIcon(platform)}
                                    {platform}
                                  </span>
                                );
                              })}
                            </div>
                          </div>
                        )}
                        {selectedClient.metadata?.pricing && (
                          <div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Pricing</div>
                            <div style={{ fontSize: '13px', fontWeight: 500 }}>{selectedClient.metadata.pricing}</div>
                          </div>
                        )}
                        {selectedClient.metadata?.license && (
                          <div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '4px' }}>License</div>
                            <div style={{ fontSize: '13px', fontWeight: 500 }}>{selectedClient.metadata.license}</div>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* MCP Support Card */}
                    {selectedClient.mcp_support && (
                      <div style={{ background: 'var(--background-card)', padding: '20px', borderRadius: '12px', border: '1px solid var(--border-subtle)' }}>
                        <div style={{ fontWeight: 700, marginBottom: '16px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7 }}>MCP Support</div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                          <div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Transports</div>
                            <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                              {selectedClient.mcp_support.transports.map(transport => (
                                <span key={transport} style={{ 
                                  fontSize: '11px', 
                                  padding: '3px 8px', 
                                  borderRadius: '4px', 
                                  background: 'var(--accent-primary)',
                                  color: 'white',
                                  fontWeight: 600,
                                  textTransform: 'uppercase'
                                }}>
                                  {transport}
                                </span>
                              ))}
                            </div>
                          </div>
                          <div>
                            <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '4px' }}>Features</div>
                            <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                              {selectedClient.mcp_support.features.map(feature => (
                                <span key={feature} style={{ 
                                  fontSize: '11px', 
                                  padding: '3px 8px', 
                                  borderRadius: '4px', 
                                  background: 'var(--card-hover)',
                                  fontWeight: 500
                                }}>
                                  {feature}
                                </span>
                              ))}
                            </div>
                          </div>
                        </div>
                      </div>
                    )}

                    {/* Links Card */}
                    {(selectedClient.homepage || selectedClient.repository || selectedClient.documentation) && (
                      <div style={{ background: 'var(--background-card)', padding: '20px', borderRadius: '12px', border: '1px solid var(--border-subtle)' }}>
                        <div style={{ fontWeight: 700, marginBottom: '16px', textTransform: 'uppercase', fontSize: '12px', opacity: 0.7 }}>Links</div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                          {selectedClient.homepage && (
                            <a 
                              href={selectedClient.homepage} 
                              target="_blank" 
                              rel="noopener noreferrer"
                              style={{ 
                                display: 'flex', 
                                alignItems: 'center', 
                                gap: '8px', 
                                fontSize: '13px', 
                                color: 'var(--accent-primary)',
                                textDecoration: 'none',
                                padding: '8px 12px',
                                borderRadius: '6px',
                                background: 'var(--card-hover)',
                                transition: 'background 0.2s'
                              }}
                            >
                              <HomeRegular style={{ fontSize: '16px' }} />
                              Homepage
                              <OpenRegular style={{ fontSize: '12px', marginLeft: 'auto', opacity: 0.6 }} />
                            </a>
                          )}
                          {selectedClient.repository && (
                            <a 
                              href={selectedClient.repository} 
                              target="_blank" 
                              rel="noopener noreferrer"
                              style={{ 
                                display: 'flex', 
                                alignItems: 'center', 
                                gap: '8px', 
                                fontSize: '13px', 
                                color: 'var(--accent-primary)',
                                textDecoration: 'none',
                                padding: '8px 12px',
                                borderRadius: '6px',
                                background: 'var(--card-hover)',
                                transition: 'background 0.2s'
                              }}
                            >
                              <BoxRegular style={{ fontSize: '16px' }} />
                              Repository
                              <OpenRegular style={{ fontSize: '12px', marginLeft: 'auto', opacity: 0.6 }} />
                            </a>
                          )}
                          {selectedClient.documentation && (
                            <a 
                              href={selectedClient.documentation} 
                              target="_blank" 
                              rel="noopener noreferrer"
                              style={{ 
                                display: 'flex', 
                                alignItems: 'center', 
                                gap: '8px', 
                                fontSize: '13px', 
                                color: 'var(--accent-primary)',
                                textDecoration: 'none',
                                padding: '8px 12px',
                                borderRadius: '6px',
                                background: 'var(--card-hover)',
                                transition: 'background 0.2s'
                              }}
                            >
                              <BookRegular style={{ fontSize: '16px' }} />
                              Documentation
                              <OpenRegular style={{ fontSize: '12px', marginLeft: 'auto', opacity: 0.6 }} />
                            </a>
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
          <section className="section-container" style={{ display: (selectedTool || selectedClient) ? 'none' : 'flex' }}>
            <header className="section-header">
              <div style={{ display: 'flex', gap: '12px' }}>
                <span 
                  className={`tab-link ${activeTab === 'active' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('active');
                    setSelectedTool(null);
                    setSelectedClient(null);
                  }}
                >
                  <CheckmarkCircleRegular /> Enabled Tools
                </span>
                <span 
                  className={`tab-link ${activeTab === 'catalog' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('catalog');
                    setSearchQuery(""); // Clear search when switching
                    setSelectedTool(null);
                    setSelectedClient(null);
                  }}
                >
                  <SearchRegular /> Tool Discovery
                </span>
                <span 
                  className={`tab-link ${activeTab === 'clients' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('clients');
                    setSearchQuery(""); 
                    setSelectedTool(null);
                    setSelectedClient(null);
                  }}
                >
                  <WindowRegular /> Apps
                </span>
                <span 
                  className={`tab-link ${activeTab === 'logs' ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab('logs');
                    setSearchQuery(""); 
                    setSelectedTool(null);
                    setSelectedClient(null);
                  }}
                >
                  <BookRegular /> Logs
                </span>
              </div>
              <span className="badge">
                {activeTab === 'active' 
                  ? `${(selectedProfile?.allow_tools?.length || 0) + builtinTools.filter(t => !selectedProfile?.disabled_system_tools?.includes(t.name)).length} Enabled` 
                  : activeTab === 'catalog'
                  ? `${(filteredTools || []).length} Available`
                  : activeTab === 'clients'
                  ? `${(allClients || []).length} Apps`
                  : `${(logs || []).length} Entries`}
              </span>
            </header>

            {activeTab === 'catalog' && (
              <div className="search-container">
                <div className="search-input-wrapper">
                  <span className="search-icon"><SearchRegular /></span>
                  <input 
                    type="text" 
                    className="search-input" 
                    placeholder="Search tools by name or description..." 
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                  />
                </div>
                
                <div className="filter-group">
                  <button 
                    className={`filter-btn ${toolFilter === 'all' ? 'active' : ''}`}
                    onClick={() => setToolFilter('all')}
                  >
                    All
                  </button>
                  <button 
                    className={`filter-btn ${toolFilter === 'official' ? 'active' : ''}`}
                    onClick={() => setToolFilter('official')}
                  >
                    <BuildingRegular style={{ fontSize: '14px' }} /> Official
                  </button>
                  <button 
                    className={`filter-btn ${toolFilter === 'community' ? 'active' : ''}`}
                    onClick={() => setToolFilter('community')}
                  >
                    <PeopleRegular style={{ fontSize: '14px' }} /> Community
                  </button>
                  <button 
                    className={`filter-btn ${toolFilter === 'custom' ? 'active' : ''}`}
                    onClick={() => setToolFilter('custom')}
                  >
                    <BoxRegular style={{ fontSize: '14px' }} /> Custom
                  </button>
                </div>

                <button className="add-tool-btn" onClick={() => setDrawer({ type: "add-custom-tool" })}>
                  <AddRegular style={{ fontSize: '18px' }} /> Bring Your Own Tool
                </button>
              </div>
            )}

            <div className="scroll-section" style={{ display: activeTab === 'logs' ? 'flex' : 'block', flexDirection: 'column' }}>
              <div 
                className={activeTab === 'logs' ? "" : (activeTab !== 'active' ? "card-grid grid-layout" : "card-grid")}
                style={activeTab === 'logs' ? { flex: 1, display: 'flex', flexDirection: 'column' } : {}}
              >
                {activeTab === 'active' && (
                  <>
                    {/* System Tools Section - Toggleable */}
                    {builtinTools.length > 0 && (
                      <div className="category-section" style={{ marginBottom: '16px' }}>
                        <div 
                          className="category-title" 
                          style={{ 
                            display: 'flex', 
                            alignItems: 'center', 
                            gap: '8px',
                            color: 'var(--text-secondary)',
                            fontSize: '11px',
                            textTransform: 'uppercase',
                            letterSpacing: '0.5px',
                            marginBottom: '8px'
                          }}
                        >
                          <span style={{ 
                            width: '16px', 
                            height: '16px', 
                            borderRadius: '4px', 
                            background: 'var(--accent-primary)', 
                            display: 'flex', 
                            alignItems: 'center', 
                            justifyContent: 'center',
                            fontSize: '10px',
                            color: 'white'
                          }}>⚙</span>
                          System Tools
                          <span style={{ 
                            fontWeight: 'normal', 
                            opacity: 0.7, 
                            textTransform: 'none',
                            fontSize: '10px'
                          }}>
                            (click to toggle)
                          </span>
                        </div>
                        <div style={{ 
                          display: 'flex', 
                          flexWrap: 'wrap', 
                          gap: '8px',
                          padding: '8px 12px',
                          background: 'var(--background-subtle)',
                          borderRadius: '8px',
                          border: '1px solid var(--border-subtle)'
                        }}>
                          {builtinTools.map(tool => {
                            const isDisabled = selectedProfile?.disabled_system_tools?.includes(tool.name);
                            return (
                              <button 
                                key={tool.name}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  toggleSystemTool(tool.name);
                                }}
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: '6px',
                                  padding: '4px 10px',
                                  background: isDisabled ? 'transparent' : 'var(--background-card)',
                                  borderRadius: '6px',
                                  fontSize: '12px',
                                  color: isDisabled ? 'var(--text-secondary)' : 'var(--text-primary)',
                                  border: isDisabled ? '1px dashed var(--border-subtle)' : '1px solid var(--border-subtle)',
                                  cursor: 'pointer',
                                  opacity: isDisabled ? 0.5 : 1,
                                  transition: 'all 0.15s ease',
                                  margin: 0,
                                  outline: 'none',
                                  fontFamily: 'inherit'
                                }}
                                title={`${tool.description}\n\nClick to ${isDisabled ? 'enable' : 'disable'}`}
                              >
                                {isDisabled ? (
                                  <EyeOffRegular style={{ fontSize: '12px', color: 'var(--text-secondary)' }} />
                                ) : (
                                  <CheckmarkRegular style={{ fontSize: '12px', color: 'var(--accent-primary)' }} />
                                )}
                                <span style={{ 
                                  fontFamily: 'monospace', 
                                  fontSize: '11px',
                                  textDecoration: isDisabled ? 'line-through' : 'none'
                                }}>
                                  {tool.name}
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    )}
                    
                    {/* User's Activated Tools */}
                    {selectedProfile?.allow_tools?.length > 0 && (
                      <div 
                        className="category-title" 
                        style={{ 
                          display: 'flex',
                          alignItems: 'center',
                          gap: '8px',
                          color: 'var(--text-secondary)',
                          fontSize: '11px',
                          textTransform: 'uppercase',
                          letterSpacing: '0.5px',
                          marginBottom: '8px'
                        }}
                      >
                        <span style={{ 
                          width: '16px', 
                          height: '16px', 
                          borderRadius: '4px', 
                          background: 'var(--accent-success)', 
                          display: 'flex', 
                          alignItems: 'center', 
                          justifyContent: 'center',
                          fontSize: '10px',
                          color: 'white'
                        }}>+</span>
                        Additional Tools
                      </div>
                    )}
                    
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
                              <img 
                                src={getThemedIcon(tool.icon)} 
                                alt={toolName} 
                                style={{ 
                                  width: '40px', 
                                  height: '40px', 
                                  objectFit: 'contain',
                                  backgroundColor: tool.icon_background ? (theme === 'light' ? tool.icon_background.light : tool.icon_background.dark) : 'transparent',
                                  borderRadius: '8px',
                                  padding: '4px',
                                  boxSizing: 'border-box'
                                }} 
                              />
                            ) : (
                              <img 
                                src={getThemedIcon("/registry-logos/mcp_fallback_light.svg")} 
                                alt={toolName} 
                                style={{ 
                                  width: '40px', 
                                  height: '40px', 
                                  objectFit: 'contain',
                                  borderRadius: '8px',
                                  padding: '4px',
                                  boxSizing: 'border-box',
                                  background: 'var(--background-card)',
                                  border: '1px solid var(--border-subtle)'
                                }} 
                              />
                            )}
                            <div className="card-info">
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <span className="card-title">{tool?.title || toolName}</span>
                                {tool?.tools && tool.tools.length > 0 && (
                                  <span style={{ fontSize: '10px', opacity: 0.6, background: 'var(--background-card)', padding: '1px 6px', borderRadius: '8px', border: '1px solid var(--border-subtle)' }}>
                                    {tool.tools.length} {tool.tools.length === 1 ? 'tool' : 'tools'}
                                  </span>
                                )}
                              </div>
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
                      <div style={{ 
                        textAlign: "center", 
                        padding: "40px 20px",
                        background: 'var(--background-subtle)',
                        borderRadius: '12px',
                        border: '1px dashed var(--border-subtle)'
                      }}>
                        <div style={{ fontSize: '14px', color: 'var(--text-secondary)', marginBottom: '12px' }}>
                          No additional tools enabled for this profile.
                        </div>
                        <button 
                          className="primary"
                          onClick={() => setActiveTab('catalog')}
                          style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                        >
                          <SearchRegular style={{ fontSize: '14px' }} />
                          Go to Tool Discovery
                        </button>
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
                                    <img 
                                      src={getThemedIcon(tool.icon)} 
                                      alt={tool.name} 
                                      style={{ 
                                        width: '40px', 
                                        height: '40px', 
                                        objectFit: 'contain',
                                        backgroundColor: tool.icon_background ? (theme === 'light' ? tool.icon_background.light : tool.icon_background.dark) : 'transparent',
                                        borderRadius: '8px',
                                        padding: '4px',
                                        boxSizing: 'border-box'
                                      }} 
                                    />
                                  ) : (
                                    <img 
                                      src={getThemedIcon("/registry-logos/mcp_fallback_light.svg")} 
                                      alt={tool.name} 
                                      style={{ 
                                        width: '40px', 
                                        height: '40px', 
                                        objectFit: 'contain',
                                        borderRadius: '8px',
                                        padding: '4px',
                                        boxSizing: 'border-box',
                                        background: 'var(--background-card)',
                                        border: '1px solid var(--border-subtle)'
                                      }} 
                                    />
                                  )}
                                  <div className="card-info">
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                      <span className="card-title">{tool.title || tool.name}</span>
                                      {tool.tools && tool.tools.length > 0 && (
                                        <span style={{ fontSize: '10px', opacity: 0.6, background: 'var(--background-card)', padding: '1px 6px', borderRadius: '8px', border: '1px solid var(--border-subtle)' }}>
                                          {tool.tools.length} {tool.tools.length === 1 ? 'tool' : 'tools'}
                                        </span>
                                      )}
                                      {tool.source === 'official' && (
                                        <span style={{ 
                                          fontSize: '9px', 
                                          padding: '2px 6px', 
                                          background: 'rgba(0, 120, 212, 0.1)', 
                                          color: 'var(--accent-primary)', 
                                          borderRadius: '10px', 
                                          fontWeight: 700,
                                          textTransform: 'uppercase',
                                          letterSpacing: '0.5px',
                                          border: '1px solid rgba(0, 120, 212, 0.2)',
                                          display: 'flex',
                                          alignItems: 'center',
                                          gap: '4px'
                                        }}>
                                          <BuildingRegular style={{ fontSize: '11px' }} />
                                          Official
                                        </span>
                                      )}
                                      {tool.source === 'community' && (
                                        <span style={{ 
                                          fontSize: '9px', 
                                          padding: '2px 6px', 
                                          background: 'rgba(128, 128, 128, 0.1)', 
                                          color: 'var(--text-secondary)', 
                                          borderRadius: '10px', 
                                          fontWeight: 700,
                                          textTransform: 'uppercase',
                                          letterSpacing: '0.5px',
                                          border: '1px solid rgba(128, 128, 128, 0.2)',
                                          display: 'flex',
                                          alignItems: 'center',
                                          gap: '4px'
                                        }}>
                                          <PeopleRegular style={{ fontSize: '11px' }} />
                                          Community
                                        </span>
                                      )}
                                      {tool.source === 'enterprise' && (
                                        <span style={{ 
                                          fontSize: '9px', 
                                          padding: '2px 6px', 
                                          background: 'rgba(128, 128, 128, 0.1)', 
                                          color: 'var(--text-secondary)', 
                                          borderRadius: '10px', 
                                          fontWeight: 700,
                                          textTransform: 'uppercase',
                                          letterSpacing: '0.5px',
                                          border: '1px solid rgba(128, 128, 128, 0.2)',
                                          display: 'flex',
                                          alignItems: 'center',
                                          gap: '4px'
                                        }}>
                                          <BoxRegular style={{ fontSize: '11px' }} />
                                          Enterprise
                                        </span>
                                      )}
                                      {tool.source !== 'official' && tool.source !== 'community' && tool.source !== 'enterprise' && (
                                        <span style={{ 
                                          fontSize: '9px', 
                                          padding: '2px 6px', 
                                          background: 'rgba(128, 128, 128, 0.1)', 
                                          color: 'var(--text-secondary)', 
                                          borderRadius: '10px', 
                                          fontWeight: 700,
                                          textTransform: 'uppercase',
                                          letterSpacing: '0.5px',
                                          border: '1px solid rgba(128, 128, 128, 0.2)'
                                        }}>{tool.source === 'local' ? 'Local' : 'Custom'}</span>
                                      )}
                                    </div>
                                    <span className="card-subtitle">{tool.description}</span>
                                  </div>
                                </div>
                                <div className="card-actions">
                                  {tool.source !== 'official' && tool.source !== 'community' && (
                                    <button 
                                      className="secondary" 
                                      style={{ marginRight: '4px', borderColor: '#ff4d4d', color: '#ff4d4d' }}
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        deleteTool(tool.name);
                                      }}
                                    >
                                      Delete
                                    </button>
                                  )}
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
                  </>
                )}

                {activeTab === 'clients' && (
                  <>
                    {allClients.map(client => (
                      <div 
                        key={client.id} 
                        className="compact-card clickable" 
                        onClick={() => setSelectedClient(client)}
                        style={{ flexDirection: 'column', alignItems: 'stretch', gap: '12px' }}
                      >
                        <div style={{ display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                          <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', flex: 1 }}>
                            {client.icon ? (
                              <img 
                                src={theme === 'dark' && client.icon_dark ? client.icon_dark : getThemedIcon(client.icon)} 
                                alt={client.name} 
                                style={{ 
                                  width: '40px', 
                                  height: '40px', 
                                  objectFit: 'contain',
                                  borderRadius: '8px',
                                  padding: '4px',
                                  boxSizing: 'border-box',
                                  background: 'var(--background-card)',
                                  border: '1px solid var(--border-subtle)'
                                }} 
                              />
                            ) : (
                              <div style={{ width: '40px', height: '40px', background: 'var(--border-subtle)', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                <PhoneLaptopRegular style={{ fontSize: '20px', opacity: 0.5 }} />
                              </div>
                            )}
                            <div className="card-info" style={{ flex: 1 }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                                <span className="card-title" style={{ fontSize: '15px' }}>{client.name}</span>
                                {client.developer && (
                                  <span style={{ fontSize: '11px', color: 'var(--text-secondary)', opacity: 0.8 }}>by {client.developer}</span>
                                )}
                              </div>
                              <span className="card-subtitle" style={{ fontSize: '12px', lineHeight: '1.4' }}>{client.description}</span>
                              <div style={{ display: 'flex', gap: '6px', marginTop: '8px', flexWrap: 'wrap', alignItems: 'center' }}>
                                {client.category && (
                                  <span style={{ 
                                    fontSize: '10px', 
                                    padding: '2px 8px', 
                                    borderRadius: '10px', 
                                    background: 'var(--card-hover)',
                                    color: 'var(--text-secondary)',
                                    fontWeight: 500
                                  }}>
                                    {client.category}
                                  </span>
                                )}
                                {client.mcp_support && (
                                  <span style={{ 
                                    fontSize: '10px', 
                                    padding: '2px 8px', 
                                    borderRadius: '10px', 
                                    background: client.mcp_support.status === 'stable' ? 'rgba(0, 200, 83, 0.15)' : 
                                               client.mcp_support.status === 'beta' ? 'rgba(255, 193, 7, 0.15)' : 'rgba(156, 39, 176, 0.15)',
                                    color: client.mcp_support.status === 'stable' ? '#00c853' : 
                                           client.mcp_support.status === 'beta' ? '#ffc107' : '#9c27b0',
                                    fontWeight: 600
                                  }}>
                                    MCP {client.mcp_support.status}
                                  </span>
                                )}
                                {client.platforms && client.platforms.length > 0 && (
                                  <span style={{ 
                                    fontSize: '10px', 
                                    color: 'var(--text-secondary)',
                                    opacity: 0.7
                                  }}>
                                    {client.platforms.slice(0, 3).join(' · ')}
                                  </span>
                                )}
                              </div>
                            </div>
                          </div>
                          {client.installed ? (
                            <button
                              className="primary"
                              style={{ flexShrink: 0, marginLeft: '12px' }}
                              onClick={async (e) => {
                                e.stopPropagation();
                                try {
                                  const res = await fetch(`${CONTROL_API}/clients/sync`, {
                                    method: "POST",
                                    headers: { "Content-Type": "application/json" },
                                    body: JSON.stringify({ target: client.id, profile: selectedProfileId })
                                  });
                                  if (res.ok) {
                                    addLog(`Successfully configured ${client.name}.`, "INFO");
                                    alert(`Successfully configured ${client.name}!`);
                                  } else {
                                    const err = await res.text();
                                    addLog(`Failed to configure ${client.name}: ${err}`, "ERROR");
                                    alert(`Failed to configure ${client.name}: ${err}`);
                                  }
                                } catch (err: any) {
                                  addLog(`Network error configuring ${client.name}: ${err.message}`, "ERROR");
                                  alert(`Network error configuring ${client.name}`);
                                }
                              }}
                            >
                              Setup
                            </button>
                          ) : (
                            client.download_url ? (
                              <a
                                href={client.download_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="primary"
                                style={{ flexShrink: 0, marginLeft: '12px', display: 'flex', alignItems: 'center', gap: '6px', textDecoration: 'none', padding: '6px 12px', borderRadius: '4px', fontSize: '12px', fontWeight: 600 }}
                              >
                                Download
                              </a>
                            ) : (
                              <button className="primary" disabled style={{ flexShrink: 0, marginLeft: '12px' }}>Download</button>
                            )
                          )}
                        </div>
                      </div>
                    ))}
                  </>
                )}
                {activeTab === 'logs' && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', flex: 1 }}>
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <div className="search-input-wrapper" style={{ flex: 1 }}>
                        <SearchRegular className="search-icon" />
                        <input 
                          type="text" 
                          className="search-input" 
                          placeholder="Search logs..." 
                          value={logSearchQuery}
                          onChange={(e) => setLogSearchQuery(e.target.value)}
                        />
                      </div>
                      <select 
                        className="filter-btn" 
                        value={logLevelFilter}
                        onChange={(e) => setLogLevelFilter(e.target.value as any)}
                      >
                        <option value="ALL">All Levels</option>
                        <option value="INFO">Info</option>
                        <option value="WARN">Warning</option>
                        <option value="ERROR">Error</option>
                      </select>
                      <button className="secondary" title="Clear Logs" onClick={clearLogs} style={{ height: '36px', width: '36px', padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <DeleteRegular />
                      </button>
                      <button className="secondary" title="Reveal Log File" onClick={revealLogs} style={{ height: '36px', width: '36px', padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <FolderOpenRegular />
                      </button>
                    </div>
                    
                    <div className="log-stream" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
                      {filteredLogs.map((log, i) => (
                        <div key={i} style={{ 
                          fontFamily: "Google Sans Code, JetBrains Mono, monospace", 
                          fontSize: "11px", 
                          marginBottom: "4px", 
                          borderBottom: '1px solid var(--border-subtle)', 
                          paddingBottom: '6px',
                          paddingTop: '2px',
                          display: 'flex',
                          alignItems: 'center'
                        }}>
                          <span style={{ opacity: 0.4, marginRight: '12px', whiteSpace: 'nowrap' }}>
                            {formatLogTimestamp(log.timestamp)}
                          </span>
                          <span style={getLogLevelStyle(log.level)}>{log.level}</span>
                          <span style={{ color: 'var(--text-primary)', lineHeight: '1.4' }}>{log.message}</span>
                        </div>
                      ))}
                      {filteredLogs.length === 0 && (
                        <div style={{ textAlign: 'center', opacity: 0.5, padding: '20px' }}>No logs found!</div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </section>

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
                    {tool?.authorization && tool.authorization.type !== 'none' && selectedProfile && (
                      (() => {
                        const auth = tool.authorization;
                        const isRecommended = auth.recommended === true;
                        const isOptional = auth.required === false || auth.recommended === true;
                        const envVars = auth.type === 'custom' ? (auth.env_vars || []) : (auth.env_var ? [{ name: auth.env_var, display_name: auth.display_name || auth.env_var, required: !isOptional }] : []);
                        const missingVars = isOptional 
                          ? envVars.filter(v => !selectedProfile.env?.[v.name])
                          : envVars.filter(v => v.required && !selectedProfile.env?.[v.name]);
                        const configuredVars = envVars.filter(v => selectedProfile.env?.[v.name]);
                        
                        // Skip showing anything if auth is optional and not recommended
                        if (isOptional && !isRecommended && missingVars.length > 0) return null;
                        
                        return (
                          <div style={{ marginBottom: '12px' }}>
                            {missingVars.length > 0 && (
                              <div style={{ 
                                fontSize: '11px', 
                                color: isRecommended ? 'var(--text-secondary)' : '#b38600', 
                                marginBottom: '8px', 
                                padding: '12px', 
                                background: isRecommended ? 'var(--background-card)' : 'rgba(255, 204, 0, 0.05)', 
                                borderRadius: '6px', 
                                border: isRecommended ? '1px solid var(--border-subtle)' : '1px solid rgba(255, 204, 0, 0.2)', 
                                display: 'flex', 
                                flexDirection: 'column', 
                                gap: '8px' 
                              }}>
                                <div style={{ fontWeight: 600 }}>
                                  {isRecommended ? (
                                    <><KeyRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> Optional Setup</>
                                  ) : (
                                    <><WarningRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> Missing Setup</>
                                  )}
                                </div>
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
                                  style={{ background: 'transparent', border: 'none', color: isRecommended ? 'var(--text-secondary)' : '#b38600', fontSize: '11px', cursor: 'pointer', opacity: 0.7, textDecoration: 'underline', alignSelf: 'flex-end', marginTop: '4px' }}
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
                    <JsonEditor
                      content={toolInput ? { text: toolInput } : { text: "{}" }}
                      onChange={(content: any) => {
                        if (content.text !== undefined) {
                          setToolInput(content.text);
                        } else if (content.json !== undefined) {
                          setToolInput(JSON.stringify(content.json, null, 2));
                        }
                      }}
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
                      dark={theme === 'dark'}
                      height="240px"
                      mode="text"
                    />
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
                <div style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '8px' }}>
                  Paste an MCP Tool Definition (JSON) to register it locally.
                </div>
                <div className="form-field">
                  <label>Tool Definition (JSON)</label>
                  <JsonEditor
                    content={drawer.data ? { text: drawer.data } : { json: {
                      name: "my-tool",
                      title: "My Custom Tool",
                      description: "A description of my tool",
                      category: "custom",
                      source: "custom",
                      runtime: {
                        transport: "stdio",
                        command: "npx",
                        args: ["@modelcontextprotocol/server-everything"]
                      }
                    }}}
                    onChange={(content: any) => {
                      if (content.text !== undefined) {
                        setDrawer({ ...drawer, data: content.text });
                      } else if (content.json !== undefined) {
                        setDrawer({ ...drawer, data: JSON.stringify(content.json, null, 2) });
                      }
                    }}
                    dark={theme === 'dark'}
                    height="400px"
                  />
                </div>
                <button 
                  className="primary" 
                  onClick={async () => {
                    try {
                      const td = JSON.parse(drawer.data || "");
                      const res = await fetch(`${CONTROL_API}/tools`, {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify(td),
                      });
                      if (res.ok) {
                        addLog(`Registered tool: ${td.name}`, "INFO");
                        fetchAllTools();
                        setDrawer(null);
                      } else {
                        const err = await res.text();
                        alert(`Failed to register tool: ${err}`);
                      }
                    } catch (err: any) {
                      alert(`Invalid JSON or Network Error: ${err.message}`);
                    }
                  }}
                >
                  Register Tool
                </button>
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
