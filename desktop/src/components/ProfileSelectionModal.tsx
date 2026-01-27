import { useState } from 'react';
import { revealItemInDir } from '@tauri-apps/plugin-opener';
import { 
  DismissRegular, 
  SearchRegular, 
  SettingsRegular, 
  AddRegular,
  DeleteRegular
} from "@fluentui/react-icons";

interface Profile {
  id: string;
}

interface ProfileSelectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  profiles: Profile[];
  selectedProfileId: string;
  onSelectProfile: (id: string) => void;
  onDeleteProfile: (id: string) => void;
  onCreateProfile: () => void;
  onOpenSettings: () => void;
  configPath?: string;
}

export function ProfileSelectionModal({
  isOpen,
  onClose,
  profiles,
  selectedProfileId,
  onSelectProfile,
  onDeleteProfile,
  onCreateProfile,
  onOpenSettings,
  configPath
}: ProfileSelectionModalProps) {
  const [search, setSearch] = useState('');

  if (!isOpen) return null;

  const handleOpenConfig = async () => {
    if (configPath) {
      try {
        await revealItemInDir(configPath);
      } catch (err) {
        console.error("Failed to open config file location:", err);
      }
    }
  };

  const filteredProfiles = profiles.filter(p => 
    p.id.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="drawer-overlay" onClick={onClose}>
      <div className="drawer-content profile-selection-modal" onClick={e => e.stopPropagation()}>
        <div className="drawer-header">
          <span className="drawer-title">Switch Profile</span>
          <button className="icon-btn" onClick={onClose}><DismissRegular /></button>
        </div>

        <div className="search-container" style={{ marginTop: '16px' }}>
          <span className="search-icon"><SearchRegular /></span>
          <input 
            type="text" 
            className="search-input" 
            placeholder="Search profiles..." 
            value={search}
            onChange={e => setSearch(e.target.value)}
            autoFocus
          />
        </div>

        <div className="scroll-section" style={{ marginTop: '8px' }}>
          <div className="card-grid">
            {filteredProfiles.map(p => (
              <div 
                key={p.id} 
                className={`compact-card profile-card ${selectedProfileId === p.id ? 'active' : ''}`}
                onClick={() => {
                  onSelectProfile(p.id);
                  onClose();
                }}
                style={{ cursor: 'pointer', borderLeft: selectedProfileId === p.id ? '4px solid var(--accent-primary)' : '1px solid var(--border-subtle)' }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <div className={`status-dot ${selectedProfileId === p.id ? 'active' : ''}`}></div>
                  <div className="card-info">
                    <span className="card-title">{p.id}</span>
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  {selectedProfileId === p.id && (
                    <span style={{ fontSize: '10px', color: 'var(--accent-primary)', fontWeight: 600 }}>SELECTED</span>
                  )}
                  <button 
                    className="icon-btn delete-profile-btn" 
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm(`Delete profile "${p.id}"?`)) {
                        onDeleteProfile(p.id);
                      }
                    }}
                    title="Delete Profile"
                    style={{ opacity: 0.5 }}
                  >
                    <DeleteRegular />
                  </button>
                </div>
              </div>
            ))}
            
            {filteredProfiles.length === 0 && (
              <div style={{ textAlign: 'center', padding: '20px', opacity: 0.6, fontStyle: 'italic' }}>
                No profiles found
              </div>
            )}
          </div>
        </div>

        <div style={{ marginTop: 'auto', paddingTop: '16px', borderTop: '1px solid var(--border-subtle)' }}>
          <button className="primary" style={{ width: '100%', padding: '12px', marginBottom: '12px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px' }} onClick={() => {
            onClose();
            onCreateProfile();
          }}>
            <AddRegular /> Create New Profile
          </button>
          
          <div style={{ textAlign: 'center' }}>
            <a 
              href="#" 
              onClick={(e) => {
                e.preventDefault();
                onClose();
                onOpenSettings();
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
              <SettingsRegular style={{ verticalAlign: 'middle', marginRight: '4px' }} /> Configure Profiles
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
