import { useState, useEffect } from 'react';
import { revealItemInDir } from '@tauri-apps/plugin-opener';
import { EyeRegular, EyeOffRegular, SettingsRegular, DismissRegular, ArrowResetRegular } from "@fluentui/react-icons";

interface Settings {
  control_port: number;
  mcp_port: number;
  enable_beta: boolean;
  verbose_logging: boolean;
  gateway_api_key: string;
  // Tool lifecycle settings
  auto_cleanup_enabled: boolean;
  auto_cleanup_minutes: number;
  cleanup_on_session: boolean;
  max_active_servers: number;
  quota_policy: 'block' | 'evict';
  // AI routing configuration
  primary_ai_provider: string;
  primary_ai_model: string;
  fallback_ai_provider: string;
  fallback_ai_model: string;
}

interface Profile {
  id: string;
  remote_auth_mode: string;
  remote_server_url: string;
  env: Record<string, string>;
  allow_tools: string[];
  disabled_system_tools: string[];
}

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  settings: Settings;
  onUpdateSettings: (s: Settings) => void;
  onReset: () => void;
  settingsPath?: string;
  configPath?: string;
  profiles: Profile[];
  selectedProfileId: string;
  onUpdateProfile: (oldId: string, p: Profile) => void;
  initialTab?: 'global' | 'profile';
}

export function SettingsModal({ 
  isOpen, 
  onClose, 
  settings, 
  onUpdateSettings, 
  onReset, 
  settingsPath,
  configPath,
  profiles,
  selectedProfileId,
  onUpdateProfile,
  initialTab = 'global'
}: SettingsModalProps) {
  const [showApiKey, setShowApiKey] = useState(false);
  const [activeTab, setActiveTab] = useState<'global' | 'profile'>(initialTab);

  // Update activeTab when modal opens
  useEffect(() => {
    if (isOpen) {
      setActiveTab(initialTab);
    }
  }, [isOpen, initialTab]);

  if (!isOpen) return null;

  const selectedProfile = profiles.find(p => p.id === selectedProfileId);

  const handleOpenSettings = async () => {
    if (settingsPath) {
      try {
        await revealItemInDir(settingsPath);
      } catch (err) {
        console.error("Failed to open settings file location:", err);
      }
    }
  };

  const handleOpenConfig = async () => {
    if (configPath) {
      try {
        await revealItemInDir(configPath);
      } catch (err) {
        console.error("Failed to open config file location:", err);
      }
    }
  };

  const handlePortChange = (key: keyof Settings, value: string) => {
    onUpdateSettings({ ...settings, [key]: parseInt(value) || 0 });
  };

  const handleResetClick = () => {
    if (confirm("WARNING: This will delete all profiles and reset the application. Are you sure?")) {
      onReset();
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    alert("Copied to clipboard!");
  };

  const handleAIKeyChange = async (type: 'primary' | 'fallback', value: string) => {
    try {
      const endpoint = type === 'primary' 
        ? `http://localhost:${settings.control_port}/api/credentials/ai-primary`
        : `http://localhost:${settings.control_port}/api/credentials/ai-fallback`;
      
      const res = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value }),
      });
      
      if (!res.ok) {
        const err = await res.text();
        alert(`Failed to save ${type} AI key: ${err}`);
        return;
      }
      
      alert(`${type.charAt(0).toUpperCase() + type.slice(1)} AI key saved successfully!`);
    } catch (err) {
      console.error(`Failed to save ${type} AI key:`, err);
      alert(`Failed to save ${type} AI key. Check console for details.`);
    }
  };

  const handleAIKeyDelete = async (type: 'primary' | 'fallback') => {
    if (!confirm(`Are you sure you want to remove the ${type} AI key?`)) {
      return;
    }
    
    try {
      const endpoint = type === 'primary' 
        ? `http://localhost:${settings.control_port}/api/credentials/ai-primary`
        : `http://localhost:${settings.control_port}/api/credentials/ai-fallback`;
      
      const res = await fetch(endpoint, {
        method: "DELETE",
      });
      
      if (!res.ok) {
        const err = await res.text();
        alert(`Failed to remove ${type} AI key: ${err}`);
        return;
      }
      
      alert(`${type.charAt(0).toUpperCase() + type.slice(1)} AI key removed successfully!`);
    } catch (err) {
      console.error(`Failed to remove ${type} AI key:`, err);
      alert(`Failed to remove ${type} AI key. Check console for details.`);
    }
  };

  const handleRegenerateKey = async () => {
    try {
      const res = await fetch(`http://localhost:${settings.control_port}/api/settings/regenerate-key`, {
        method: "POST",
      });
      if (res.ok) {
        const data = await res.json();
        onUpdateSettings({ ...settings, gateway_api_key: data.gateway_api_key });
      }
    } catch (err) {
      console.error("Failed to regenerate API key:", err);
    }
  };

  return (
    <div className="drawer-overlay" onClick={onClose}>
      <div className="drawer-content settings-modal" onClick={e => e.stopPropagation()}>
        <div className="drawer-header">
          <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
            <span className="drawer-title">Settings</span>
            <div className="settings-tabs">
              <button 
                className={`tab-btn ${activeTab === 'global' ? 'active' : ''}`}
                onClick={() => setActiveTab('global')}
              >
                App
              </button>
              <button 
                className={`tab-btn ${activeTab === 'profile' ? 'active' : ''}`}
                onClick={() => setActiveTab('profile')}
              >
                Profile
              </button>
            </div>
          </div>
          <button className="icon-btn" onClick={onClose}><DismissRegular /></button>
        </div>

        <div className="scroll-section">
          {activeTab === 'global' ? (
            <>
              <div className="settings-section">
                <h3>Network Configuration</h3>
                <div className="form-field">
                  <label>Control Plane Port</label>
                  <input 
                    type="number" 
                    value={settings.control_port} 
                    onChange={e => handlePortChange('control_port', e.target.value)}
                  />
                  <span className="input-hint">Used for the management API</span>
                </div>

                <div className="form-field" style={{ marginTop: '12px' }}>
                  <label>MCP Gateway Port</label>
                  <input 
                    type="number" 
                    value={settings.mcp_port} 
                    onChange={e => handlePortChange('mcp_port', e.target.value)}
                  />
                  <span className="input-hint">Shared port for all profiles (path-based routing)</span>
                </div>

                <div className="settings-field" style={{ marginTop: '16px' }}>
                  <div 
                    className="toggle-switch-container" 
                    onClick={() => onUpdateSettings({ ...settings, enable_beta: !settings.enable_beta })}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className={`toggle-switch ${settings.enable_beta ? 'active' : ''}`} />
                    <span className="toggle-switch-label">Include Beta Releases</span>
                  </div>
                  <p className="settings-field-helper">Early access to new features (may be unstable)</p>
                </div>

                <div className="settings-field" style={{ marginTop: '16px' }}>
                  <div 
                    className="toggle-switch-container" 
                    onClick={() => onUpdateSettings({ ...settings, verbose_logging: !settings.verbose_logging })}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className={`toggle-switch ${settings.verbose_logging ? 'active' : ''}`} />
                    <span className="toggle-switch-label">Verbose Logging</span>
                  </div>
                  <p className="settings-field-helper">Enable detailed TRACE-level logs for debugging (may affect performance)</p>
                </div>
              </div>

              <div className="settings-section">
                <h3>Security</h3>
                <div className="form-field">
                  <label>Gateway API Key</label>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <input 
                      type={showApiKey ? "text" : "password"} 
                      value={settings.gateway_api_key || "No key configured"} 
                      readOnly
                      style={{ flex: 1, fontFamily: 'monospace', fontSize: '12px' }}
                    />
                    <button className="icon-btn" onClick={() => setShowApiKey(!showApiKey)} title={showApiKey ? "Hide" : "Show"}>
                      {showApiKey ? <EyeOffRegular style={{ fontSize: '16px' }} /> : <EyeRegular style={{ fontSize: '16px' }} />}
                    </button>
                  </div>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                    <button 
                      className="secondary-btn" 
                      style={{ flex: 1, padding: '6px' }}
                      onClick={() => copyToClipboard(settings.gateway_api_key)}
                      disabled={!settings.gateway_api_key}
                    >
                      Copy Key
                    </button>
                    <button 
                      className="secondary-btn" 
                      style={{ flex: 1, padding: '6px' }}
                      onClick={handleRegenerateKey}
                    >
                      {settings.gateway_api_key ? "Regenerate" : "Generate Key"}
                    </button>
                  </div>
                  {!settings.gateway_api_key && (
                    <span className="input-hint" style={{ color: '#ffcc00' }}>
                      ⚠️ Gateway is currently unsecured. Generate a key to require authentication.
                    </span>
                  )}
                  {settings.gateway_api_key && (
                    <span className="input-hint">
                      Required for IDEs to connect to the MCP Gateway.
                    </span>
                  )}
                </div>
              </div>

              <div className="settings-section">
                <div className="settings-section-header">
                  <h3>Tool Lifecycle</h3>
                  <button 
                    className="reset-defaults-btn"
                    onClick={() => onUpdateSettings({
                      ...settings,
                      auto_cleanup_enabled: true,
                      auto_cleanup_minutes: 10,
                      cleanup_on_session: false,
                      max_active_servers: 5,
                      quota_policy: 'evict'
                    })}
                    title="Reset tool lifecycle settings to recommended defaults"
                  >
                    <ArrowResetRegular /> Reset to Defaults
                  </button>
                </div>
                
                {/* Auto-cleanup Toggle */}
                <div className="settings-field">
                  <div 
                    className="toggle-switch-container" 
                    onClick={() => onUpdateSettings({ ...settings, auto_cleanup_enabled: !settings.auto_cleanup_enabled })}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className={`toggle-switch ${settings.auto_cleanup_enabled ? 'active' : ''}`} />
                    <span className="toggle-switch-label">Auto-cleanup inactive tools</span>
                  </div>
                  <p className="settings-field-helper">
                    Automatically unload tool servers after a period of inactivity to reduce context bloat and free up resources.
                  </p>
                </div>

                {/* Cleanup Timing - Horizontal Toggle */}
                {settings.auto_cleanup_enabled && (
                  <div className="settings-field">
                    <div className="settings-field-label">Cleanup Timing</div>
                    <div className="toggle-button-group" style={{ marginTop: '8px' }}>
                      <button 
                        className={`toggle-button ${settings.auto_cleanup_minutes === 30 ? 'active' : ''}`}
                        onClick={() => onUpdateSettings({ ...settings, auto_cleanup_minutes: 30 })}
                      >
                        🐢 Relaxed (30m)
                      </button>
                      <button 
                        className={`toggle-button ${settings.auto_cleanup_minutes === 10 ? 'active' : ''}`}
                        onClick={() => onUpdateSettings({ ...settings, auto_cleanup_minutes: 10 })}
                      >
                        ⚖️ Normal (10m)
                      </button>
                      <button 
                        className={`toggle-button ${settings.auto_cleanup_minutes === 5 ? 'active' : ''}`}
                        onClick={() => onUpdateSettings({ ...settings, auto_cleanup_minutes: 5 })}
                      >
                        ⚡ Aggressive (5m)
                      </button>
                      <button 
                        className={`toggle-button ${![30, 10, 5].includes(settings.auto_cleanup_minutes) ? 'active' : ''}`}
                        onClick={() => {
                          if ([30, 10, 5].includes(settings.auto_cleanup_minutes)) {
                            onUpdateSettings({ ...settings, auto_cleanup_minutes: 15 });
                          }
                        }}
                      >
                        🎛️ Custom
                      </button>
                      {![30, 10, 5].includes(settings.auto_cleanup_minutes) && (
                        <input 
                          type="number"
                          className="toggle-custom-input"
                          value={settings.auto_cleanup_minutes}
                          onChange={e => {
                            const val = parseInt(e.target.value) || 2;
                            onUpdateSettings({ ...settings, auto_cleanup_minutes: Math.max(2, val) });
                          }}
                          min="2"
                          placeholder="min"
                        />
                      )}
                    </div>
                    <p className="settings-field-helper">
                      <strong>Relaxed:</strong> Keep tools loaded longer for extended sessions. <strong>Normal:</strong> Balanced for typical use (recommended). <strong>Aggressive:</strong> Quick cleanup for minimal context. Minimum: 2 minutes.
                    </p>
                  </div>
                )}

                {/* Reset on Session Toggle */}
                <div className="settings-field">
                  <div 
                    className="toggle-switch-container" 
                    onClick={() => onUpdateSettings({ ...settings, cleanup_on_session: !settings.cleanup_on_session })}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className={`toggle-switch ${settings.cleanup_on_session ? 'active' : ''}`} />
                    <span className="toggle-switch-label">Reset tools on new conversation</span>
                  </div>
                  <p className="settings-field-helper">
                    When enabled, all active tool servers are deactivated when a new AI session starts. This ensures each conversation begins with a clean slate.
                  </p>
                </div>

                {/* Maximum Active Servers - Horizontal Toggle */}
                <div className="settings-field">
                  <div className="settings-field-label">Maximum Active Servers</div>
                  <div className="toggle-button-group" style={{ marginTop: '8px' }}>
                    <button 
                      className={`toggle-button ${settings.max_active_servers === 0 ? 'active' : ''}`}
                      onClick={() => onUpdateSettings({ ...settings, max_active_servers: 0 })}
                    >
                      ∞ Unlimited
                    </button>
                    <button 
                      className={`toggle-button ${settings.max_active_servers === 3 ? 'active' : ''}`}
                      onClick={() => onUpdateSettings({ ...settings, max_active_servers: 3 })}
                    >
                      3
                    </button>
                    <button 
                      className={`toggle-button ${settings.max_active_servers === 5 ? 'active' : ''}`}
                      onClick={() => onUpdateSettings({ ...settings, max_active_servers: 5 })}
                    >
                      5
                    </button>
                    <button 
                      className={`toggle-button ${settings.max_active_servers === 10 ? 'active' : ''}`}
                      onClick={() => onUpdateSettings({ ...settings, max_active_servers: 10 })}
                    >
                      10
                    </button>
                    <button 
                      className={`toggle-button ${![0, 3, 5, 10].includes(settings.max_active_servers) ? 'active' : ''}`}
                      onClick={() => {
                        if ([0, 3, 5, 10].includes(settings.max_active_servers)) {
                          onUpdateSettings({ ...settings, max_active_servers: 7 });
                        }
                      }}
                    >
                      Custom
                    </button>
                    {![0, 3, 5, 10].includes(settings.max_active_servers) && (
                      <input 
                        type="number"
                        className="toggle-custom-input"
                        value={settings.max_active_servers}
                        onChange={e => {
                          const val = parseInt(e.target.value) || 1;
                          onUpdateSettings({ ...settings, max_active_servers: Math.max(1, val) });
                        }}
                        min="1"
                        placeholder="max"
                      />
                    )}
                  </div>
                  <p className="settings-field-helper">
                    Limits concurrent tool servers to manage context size. <strong>5 servers</strong> is recommended for most use cases. Set to <strong>Unlimited</strong> if you need many tools simultaneously.
                  </p>
                </div>

                {/* Quota Policy */}
                {settings.max_active_servers > 0 && (
                  <div className="settings-field">
                    <div className="settings-field-label">When Limit is Reached</div>
                    <div className="toggle-button-group" style={{ marginTop: '8px' }}>
                      <button 
                        className={`toggle-button ${settings.quota_policy === 'evict' ? 'active' : ''}`}
                        onClick={() => onUpdateSettings({ ...settings, quota_policy: 'evict' })}
                      >
                        🔄 Auto-evict oldest
                      </button>
                      <button 
                        className={`toggle-button ${settings.quota_policy === 'block' ? 'active' : ''}`}
                        onClick={() => onUpdateSettings({ ...settings, quota_policy: 'block' })}
                      >
                        🚫 Block new activations
                      </button>
                    </div>
                    <p className="settings-field-helper">
                      <strong>Auto-evict:</strong> Automatically removes the least recently used server to make room (recommended). <strong>Block:</strong> Prevents new tools from activating until you manually deactivate one.
                    </p>
                  </div>
                )}
              </div>

              <div className="settings-section">
                <h3>AI Routing & Dispatching</h3>
                <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '12px' }}>
                  Configure AI providers for automatic semantic tool routing. These keys enable intelligent tool selection when using scooter_execute.
                </p>
                
                <div className="form-field">
                  <label>Primary AI Provider</label>
                  <select 
                    value={settings.primary_ai_provider || ''} 
                    onChange={e => onUpdateSettings({ ...settings, primary_ai_provider: e.target.value })}
                    style={{ width: '100%', padding: '8px' }}
                  >
                    <option value="">Select provider</option>
                    <option value="gemini">Gemini</option>
                    <option value="openrouter">OpenRouter</option>
                  </select>
                  <span className="input-hint">Primary provider for semantic routing (Gemini or OpenRouter)</span>
                </div>

                <div className="form-field" style={{ marginTop: '12px' }}>
                  <label>Primary AI Model</label>
                  <input 
                    type="text" 
                    value={settings.primary_ai_model || ''} 
                    onChange={e => onUpdateSettings({ ...settings, primary_ai_model: e.target.value })}
                    placeholder="e.g., gemini-2.0-flash-exp or openai/gpt-4o-mini"
                  />
                  <span className="input-hint">Enter your preferred model name (no defaults configured)</span>
                </div>

                <div className="form-field" style={{ marginTop: '12px' }}>
                  <label>Primary AI Key</label>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <input 
                      type="password" 
                      value={""} // Never display existing key
                      onChange={e => handleAIKeyChange('primary', e.target.value)}
                      placeholder="Enter API key"
                      style={{ flex: 1, fontFamily: 'monospace', fontSize: '12px' }}
                    />
                  </div>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                    <button 
                      className="secondary-btn" 
                      style={{ flex: 1, padding: '6px' }}
                      onClick={() => handleAIKeyDelete('primary')}
                      disabled={!settings.primary_ai_provider}
                    >
                      Remove Key
                    </button>
                  </div>
                  <span className="input-hint">Stored securely in system keychain</span>
                </div>

                <div className="form-field" style={{ marginTop: '16px' }}>
                  <label>Fallback AI Provider</label>
                  <select 
                    value={settings.fallback_ai_provider || ''} 
                    onChange={e => onUpdateSettings({ ...settings, fallback_ai_provider: e.target.value })}
                    style={{ width: '100%', padding: '8px' }}
                  >
                    <option value="">Select provider</option>
                    <option value="gemini">Gemini</option>
                    <option value="openrouter">OpenRouter</option>
                  </select>
                  <span className="input-hint">Fallback provider if primary fails</span>
                </div>

                <div className="form-field" style={{ marginTop: '12px' }}>
                  <label>Fallback AI Model</label>
                  <input 
                    type="text" 
                    value={settings.fallback_ai_model || ''} 
                    onChange={e => onUpdateSettings({ ...settings, fallback_ai_model: e.target.value })}
                    placeholder="e.g., gemini-2.5-pro-exp or openai/gpt-4o"
                  />
                  <span className="input-hint">Enter your preferred model name</span>
                </div>

                <div className="form-field" style={{ marginTop: '12px' }}>
                  <label>Fallback AI Key</label>
                  <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                    <input 
                      type="password" 
                      value={""} // Never display existing key
                      onChange={e => handleAIKeyChange('fallback', e.target.value)}
                      placeholder="Enter API key"
                      style={{ flex: 1, fontFamily: 'monospace', fontSize: '12px' }}
                    />
                  </div>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
                    <button 
                      className="secondary-btn" 
                      style={{ flex: 1, padding: '6px' }}
                      onClick={() => handleAIKeyDelete('fallback')}
                      disabled={!settings.fallback_ai_provider}
                    >
                      Remove Key
                    </button>
                  </div>
                  <span className="input-hint">Stored securely in system keychain</span>
                </div>
              </div>

              <div className="settings-section danger-zone">
                <h3>Danger Zone</h3>
                <p className="danger-text">Irreversible actions</p>
                <button className="danger-btn" onClick={handleResetClick}>
                  Reset Entire Application
                </button>
              </div>
            </>
          ) : (
            <div className="settings-section">
              <h3>Profile: {selectedProfileId || 'None'}</h3>
              {!selectedProfile ? (
                <p style={{ color: 'var(--text-secondary)', fontSize: '14px' }}>
                  No profile selected. Please select a profile from the sidebar to configure its settings.
                </p>
              ) : (
                <>
                  <div className="form-field">
                    <label>Profile Name</label>
                    <input 
                      type="text" 
                      value={selectedProfile.id || ''} 
                      onChange={e => onUpdateProfile(selectedProfile.id, { ...selectedProfile, id: e.target.value })}
                      placeholder="e.g., work, personal"
                    />
                    <span className="input-hint">Unique identifier for this profile</span>
                  </div>

                  <div className="form-field" style={{ marginTop: '12px' }}>
                    <label>Remote Server URL</label>
                    <input 
                      type="text" 
                      value={selectedProfile.remote_server_url || ''} 
                      onChange={e => onUpdateProfile(selectedProfile.id, { ...selectedProfile, remote_server_url: e.target.value })}
                      placeholder="e.g., https://mcp.example.com"
                    />
                    <span className="input-hint">URL of the remote MCP server to proxy to</span>
                  </div>

                  <div className="form-field" style={{ marginTop: '12px' }}>
                    <label>Remote Auth Mode</label>
                    <select 
                      value={selectedProfile.remote_auth_mode || 'none'} 
                      onChange={e => onUpdateProfile(selectedProfile.id, { ...selectedProfile, remote_auth_mode: e.target.value })}
                      style={{ width: '100%', padding: '8px' }}
                    >
                      <option value="none">None</option>
                      <option value="oauth2">OAuth 2.0</option>
                    </select>
                    <span className="input-hint">Authentication method for the remote server</span>
                  </div>
                </>
              )}
            </div>
          )}
        </div>

        {(activeTab === 'global' ? settingsPath : configPath) && (
          <div style={{ marginTop: 'auto', padding: '16px', borderTop: '1px solid var(--border-subtle)', textAlign: 'center' }}>
            <a 
              href="#" 
              onClick={(e) => {
                e.preventDefault();
                if (activeTab === 'global') {
                  handleOpenSettings();
                } else {
                  handleOpenConfig();
                }
              }}
              style={{ 
                fontSize: '12px', 
                color: 'var(--accent-primary)', 
                textDecoration: 'none',
                opacity: 0.8,
                display: 'inline-block',
                padding: '4px 8px'
              }}
              onMouseOver={(e) => e.currentTarget.style.opacity = '1'}
              onMouseOut={(e) => e.currentTarget.style.opacity = '0.8'}
            >
              <SettingsRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> 
              {activeTab === 'global' ? "Manual Configure Settings" : "Manual Configure Profile"}
            </a>
          </div>
        )}
      </div>
    </div>
  );
}
