import React, { useState } from 'react';

interface Profile {
  id: string;
}

interface ProfileSelectionModalProps {
  isOpen: boolean;
  onClose: () => void;
  profiles: Profile[];
  selectedProfileId: string;
  onSelectProfile: (id: string) => void;
  onCreateProfile: () => void;
}

export function ProfileSelectionModal({
  isOpen,
  onClose,
  profiles,
  selectedProfileId,
  onSelectProfile,
  onCreateProfile
}: ProfileSelectionModalProps) {
  const [search, setSearch] = useState('');

  if (!isOpen) return null;

  const filteredProfiles = profiles.filter(p => 
    p.id.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="drawer-overlay" onClick={onClose}>
      <div className="drawer-content profile-selection-modal" onClick={e => e.stopPropagation()} style={{ width: '400px' }}>
        <div className="drawer-header">
          <span className="drawer-title">Switch Profile</span>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </div>

        <div className="search-container" style={{ marginTop: '16px' }}>
          <span className="search-icon">🔍</span>
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
                {selectedProfileId === p.id && (
                  <span style={{ fontSize: '10px', color: 'var(--accent-primary)', fontWeight: 600 }}>ACTIVE</span>
                )}
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
          <button className="primary" style={{ width: '100%', padding: '12px' }} onClick={() => {
            onClose();
            onCreateProfile();
          }}>
            + Create New Profile
          </button>
        </div>
      </div>
    </div>
  );
}
