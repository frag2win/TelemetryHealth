import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App.tsx';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<Props, State> {
  public override state: State = {
    hasError: false,
    error: null
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public override componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Uncaught error inside dashboard:', error, errorInfo);
  }

  public override render() {
    if (this.state.hasError) {
      return (
        <div
          style={{
            padding: '40px',
            background: 'var(--ink, #0B0F0E)',
            color: 'var(--paper, #E8ECE9)',
            height: '100vh',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center',
            fontFamily: 'var(--sans, sans-serif)',
            gap: '16px'
          }}
        >
          <h1 style={{ color: 'var(--red, #FF4D4D)', margin: 0, fontSize: '24px' }}>Critical UI System Error</h1>
          <p style={{ color: 'var(--muted, #7C8985)', fontFamily: 'var(--mono, monospace)', fontSize: '13px' }}>
            {this.state.error?.toString()}
          </p>
          <button
            style={{
              background: 'var(--panel-2, #161D1B)',
              border: '1px solid var(--phosphor, #5CE1A5)',
              color: 'var(--phosphor, #5CE1A5)',
              padding: '8px 16px',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '13px',
              fontFamily: 'var(--mono, monospace)'
            }}
            onClick={() => window.location.reload()}
          >
            Reload Dashboard
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </StrictMode>
);
