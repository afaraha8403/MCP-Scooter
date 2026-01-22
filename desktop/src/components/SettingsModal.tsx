import { useState } from 'react';
import { revealItemInDir } from '@tauri-apps/plugin-opener';
import { EyeRegular, EyeOffRegular, SettingsRegular, DismissRegular } from "@fluentui/react-icons";

interface Settings {
  control_port: number;
  mcp_port: number;
  enable_beta: boolean;
  gateway_api_key: string;
  primary_ai_provider: string;
  primary_ai_model: string;
  fallback_ai_provider: string;
  fallback_ai_model: string;
}

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  settings: Settings;
  onUpdateSettings: (s: Settings) => void;
  onReset: () => void;
  settingsPath?: string;
}

export function SettingsModal({ isOpen, onClose, settings, onUpdateSettings, onReset, settingsPath }: SettingsModalProps) {
  const [showApiKey, setShowApiKey] = useState(false);

  if (!isOpen) return null;

  const handleOpenSettings = async () => {
    if (settingsPath) {
      try {
        await revealItemInDir(settingsPath);
      } catch (err) {
        console.error("Failed to open settings file location:", err);
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
          <span className="drawer-title">Global Settings</span>
          <button className="icon-btn" onClick={onClose}><DismissRegular /></button>
        </div>

        <div className="scroll-section">
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

            <div className="form-field" style={{ marginTop: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
              <input 
                type="checkbox" 
                id="enable_beta"
                checked={settings.enable_beta} 
                onChange={e => onUpdateSettings({ ...settings, enable_beta: e.target.checked })}
                style={{ width: 'auto' }}
              />
              <label htmlFor="enable_beta" style={{ marginBottom: 0 }}>Include Beta Releases</label>
            </div>
            <span className="input-hint">Early access to new features (may be unstable)</span>
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
                  value={""}
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
                  value={""}
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
        </div>

        {settingsPath && (
          <div style={{ marginTop: 'auto', padding: '16px', borderTop: '1px solid var(--border-subtle)', textAlign: 'center' }}>
            <a 
              href="#" 
              onClick={(e) => {
                e.preventDefault();
                handleOpenSettings();
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
              <SettingsRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> Manually configure settings.yaml
            </a>
          </div>
        )}
      </div>
    </div>
  );
}
