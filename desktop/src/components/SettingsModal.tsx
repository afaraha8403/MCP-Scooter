import { useState } from 'react';
import { revealItemInDir } from '@tauri-apps/plugin-opener';
import { EyeRegular, EyeOffRegular, SettingsRegular, DismissRegular } from "@fluentui/react-icons";

interface Settings {
  control_port: number;
  mcp_port: number;
  enable_beta: boolean;
  gateway_api_key: string;
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
