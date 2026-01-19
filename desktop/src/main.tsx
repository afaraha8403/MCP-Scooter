import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

// Debug: Log when main.tsx loads
console.log("[DEBUG] main.tsx loaded");
if ((window as any).splashLog) {
  (window as any).splashLog("React mounting...", "active");
}

// Error boundary for debugging
class ErrorBoundary extends React.Component<{children: React.ReactNode}, {hasError: boolean, error: Error | null}> {
  constructor(props: {children: React.ReactNode}) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("[DEBUG] React Error:", error, errorInfo);
    if ((window as any).splashLog) {
      (window as any).splashLog(`ERROR: ${error.message}`, "active");
    }
  }

  render() {
    if (this.state.hasError) {
      const errorTitle = encodeURIComponent(`[BUG] ${this.state.error?.message || 'Unknown Error'}`);
      const errorBody = encodeURIComponent(`## Describe the Bug\n\nAn application error occurred in the frontend.\n\n## Actual Behavior\n\n\`\`\`\n${this.state.error?.message}\n\n${this.state.error?.stack}\n\`\`\`\n\n## Environment\n\n- **OS:** ${navigator.platform}\n- **User Agent:** ${navigator.userAgent}`);
      const githubUrl = `https://github.com/afaraha8403/MCP-Scooter/issues/new?template=bug_report.md&title=${errorTitle}&body=${errorBody}`;

      return (
        <div style={{ 
          padding: '40px', 
          background: '#1a1a2e', 
          color: '#ff6b6b', 
          fontFamily: 'monospace',
          minHeight: '100vh'
        }}>
          <h2>⚠️ Application Error</h2>
          <pre style={{ 
            background: '#0d0d1a', 
            padding: '20px', 
            borderRadius: '8px',
            overflow: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word'
          }}>
            {this.state.error?.message}
            {'\n\n'}
            {this.state.error?.stack}
          </pre>
          <div style={{ display: 'flex', gap: '12px', marginTop: '20px' }}>
            <button 
              onClick={() => window.location.reload()}
              style={{
                padding: '10px 20px',
                background: '#00d4aa',
                border: 'none',
                borderRadius: '4px',
                color: '#000',
                fontWeight: 'bold',
                cursor: 'pointer'
              }}
            >
              Reload App
            </button>
            <a 
              href={githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              style={{
                padding: '10px 20px',
                background: '#ff4d4d',
                border: 'none',
                borderRadius: '4px',
                color: '#fff',
                fontWeight: 'bold',
                textDecoration: 'none',
                cursor: 'pointer',
                display: 'inline-block'
              }}
            >
              Report Issue
            </a>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

try {
  const root = document.getElementById("root");
  if (!root) {
    throw new Error("Root element not found");
  }
  
  console.log("[DEBUG] Creating React root...");
  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </React.StrictMode>,
  );
  console.log("[DEBUG] React render called");
} catch (err) {
  console.error("[DEBUG] Fatal error during React initialization:", err);
  const errorTitle = encodeURIComponent(`[BUG] Fatal Initialization Error`);
  const errorBody = encodeURIComponent(`## Describe the Bug\n\nFatal error during React initialization.\n\n## Actual Behavior\n\n\`\`\`\n${err}\n\`\`\`\n\n## Environment\n\n- **OS:** ${navigator.platform}\n- **User Agent:** ${navigator.userAgent}`);
  const githubUrl = `https://github.com/afaraha8403/MCP-Scooter/issues/new?template=bug_report.md&title=${errorTitle}&body=${errorBody}`;

  document.body.innerHTML = `
    <div style="padding: 40px; background: #1a1a2e; color: #ff6b6b; font-family: monospace; min-height: 100vh;">
      <h2>⚠️ Fatal Initialization Error</h2>
      <pre style="background: #0d0d1a; padding: 20px; border-radius: 8px; color: #ff6b6b; margin-bottom: 20px;">${err}</pre>
      <div style="display: flex; gap: 12px;">
        <button onclick="window.location.reload()" style="padding: 10px 20px; background: #00d4aa; border: none; border-radius: 4px; color: #000; font-weight: bold; cursor: pointer;">Reload App</button>
        <a href="${githubUrl}" target="_blank" rel="noopener noreferrer" style="padding: 10px 20px; background: #ff4d4d; border: none; border-radius: 4px; color: #fff; font-weight: bold; text-decoration: none; cursor: pointer; display: inline-block;">Report Issue</a>
      </div>
    </div>
  `;
}
