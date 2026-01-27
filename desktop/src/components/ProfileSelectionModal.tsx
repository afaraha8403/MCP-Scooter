import { useState } from 'react';
import { revealItemInDir } from '@tauri-apps/plugin-opener';
import { 
  DismissRegular, 
  SearchRegular, 
  SettingsRegular, 
  AddRegular,
  DeleteRegular,
  PersonRegular,
  CheckmarkCircleRegular
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

        <div className="search-container" style={{ marginTop: '16px', marginBottom: '16px' }}>
          <div className="search-input-wrapper">
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
        </div>

        <div className="scroll-section">
          <div className="card-grid" style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {filteredProfiles.map(p => (
              <div 
                key={p.id} 
                className={`profile-card ${selectedProfileId === p.id ? 'active' : ''}`}
                onClick={() => {
                  onSelectProfile(p.id);
                  onClose();
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', flex: 1 }}>
                  <div className="profile-card-icon">
                    <PersonRegular />
                  </div>
                  <div className="profile-card-info">
                    <span className="profile-card-title">{p.id}</span>
                    <span className="profile-card-subtitle">
                      {selectedProfileId === p.id ? 'Currently Active' : 'Available Profile'}
                    </span>
                  </div>
                </div>
                
                <div className="profile-card-actions">
                  {selectedProfileId === p.id && (
                    <CheckmarkCircleRegular style={{ color: 'var(--accent-primary)', fontSize: '20px' }} />
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
                  >
                    <DeleteRegular />
                  </button>
                </div>
              </div>
            ))}
            
            {filteredProfiles.length === 0 && (
              <div style={{ textAlign: 'center', padding: '40px 20px', opacity: 0.6, fontStyle: 'italic' }}>
                No profiles found matching "{search}"
              </div>
            )}
          </div>
        </div>

        <div style={{ marginTop: 'auto', paddingTop: '20px', borderTop: '1px solid var(--border-subtle)' }}>
          <button className="primary" style={{ width: '100%', height: '44px', marginBottom: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '14px', fontWeight: 700 }} onClick={() => {
            onClose();
            onCreateProfile();
          }}>
            <AddRegular style={{ fontSize: '20px' }} /> Create New Profile
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
                fontSize: '13px', 
                color: 'var(--accent-primary)', 
                textDecoration: 'none',
                opacity: 0.8,
                display: 'inline-flex',
                alignItems: 'center',
                gap: '6px',
                padding: '8px 16px',
                borderRadius: '8px',
                background: 'var(--log-bg)'
              }}
              onMouseOver={(e) => {
                e.currentTarget.style.opacity = '1';
                e.currentTarget.style.background = 'var(--card-hover)';
              }}
              onMouseOut={(e) => {
                e.currentTarget.style.opacity = '0.8';
                e.currentTarget.style.background = 'var(--log-bg)';
              }}
            >
              <SettingsRegular style={{ fontSize: '16px' }} /> Configure Profiles
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
