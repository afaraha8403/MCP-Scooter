import React, { useState } from 'react';

interface Profile {
  id: string;
}

interface Settings {
  control_port: number;
  mcp_port: number;
  enable_beta: boolean;
  gateway_api_key: string;
}

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  profiles: Profile[];
  settings: Settings;
  onUpdateSettings: (s: Settings) => void;
  onDeleteProfile: (id: string) => void;
  onReset: () => void;
}

export function SettingsModal({ isOpen, onClose, profiles, settings, onUpdateSettings, onDeleteProfile, onReset }: SettingsModalProps) {
  if (!isOpen) return null;

  const handlePortChange = (key: keyof Settings, value: string) => {
    onUpdateSettings({ ...settings, [key]: parseInt(value) || 0 });
  };

  const handleResetClick = () => {
    if (confirm("WARNING: This will delete all profiles and reset the application. Are you sure?")) {
      onReset();
    }
  };

  return (
    <div className="drawer-overlay" onClick={onClose}>
      <div className="drawer-content settings-modal" onClick={e => e.stopPropagation()}>
        <div className="drawer-header">
          <span className="drawer-title">Global Settings</span>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </div>

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
            <span className="input-hint">Used for tools and integrations</span>
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
          
          <div className="port-list" style={{ marginTop: '16px' }}>
            <label style={{ fontSize: '11px', fontWeight: 600, color: 'var(--text-secondary)', textTransform: 'uppercase' }}>Active Profiles (Shared Port :{settings.mcp_port})</label>
            <div className="port-table">
              {profiles.map(p => (
                <div key={p.id} className="port-row" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <span className="profile-name">{p.id}</span>
                    <span className="port-number" style={{ fontSize: '10px', opacity: 0.6 }}>Active</span>
                  </div>
                  <button 
                    className="icon-btn" 
                    style={{ color: '#ff4d4d', opacity: 0.8 }} 
                    onClick={() => {
                      if (confirm(`Delete profile "${p.id}"?`)) {
                        onDeleteProfile(p.id);
                      }
                    }}
                    title="Delete Profile"
                  >
                    ✕
                  </button>
                </div>
              ))}
              {profiles.length === 0 && (
                 <div className="port-row" style={{ color: 'var(--text-secondary)', fontStyle: 'italic' }}>
                   No active profiles
                 </div>
              )}
            </div>
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
                {showApiKey ? "👁️" : "🙈"}
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
    </div>
  );
}
