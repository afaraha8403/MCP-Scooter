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
          <button 
            onClick={() => window.location.reload()}
            style={{
              marginTop: '20px',
              padding: '10px 20px',
              background: '#00d4aa',
              border: 'none',
              borderRadius: '4px',
              color: '#000',
              cursor: 'pointer'
            }}
          >
            Reload App
          </button>
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
  document.body.innerHTML = `
    <div style="padding: 40px; background: #1a1a2e; color: #ff6b6b; font-family: monospace; min-height: 100vh;">
      <h2>⚠️ Fatal Initialization Error</h2>
      <pre style="background: #0d0d1a; padding: 20px; border-radius: 8px;">${err}</pre>
    </div>
  `;
}
