const statusDisplay = document.getElementById("status");
const statusValue = statusDisplay.querySelector('.status-value');
const powerLight = document.getElementById("power-light");
const networkLight = document.getElementById("network-light");

// State display elements
const stateIndicator = document.getElementById("state-indicator");
const indicatorDot = stateIndicator?.querySelector('.indicator-dot');
const indicatorText = stateIndicator?.querySelector('.indicator-text');
const visibilityValue = document.getElementById("visibility-value");
const actionValue = document.getElementById("action-value");
const updatedValue = document.getElementById("updated-value");

const setStatus = (msg, ok = true) => {
  if (statusValue) {
    statusValue.textContent = msg;
    statusValue.style.color = ok ? 'var(--accent-green)' : 'var(--primary-red)';
    
    // Flash the status display
    statusDisplay.style.animation = 'none';
    setTimeout(() => {
      statusDisplay.style.animation = 'statusFlash 0.3s ease';
    }, 10);
  }
  
  // Update status lights based on response
  if (ok) {
    powerLight.style.background = 'var(--accent-green)';
    powerLight.style.boxShadow = '0 0 8px var(--accent-green)';
  } else {
    powerLight.style.background = 'var(--primary-red)';
    powerLight.style.boxShadow = '0 0 8px var(--primary-red)';
  }
  
  console.log(msg);
};

// Fetch and display current state
async function fetchState() {
  try {
    const res = await fetch("/state");
    if (res.ok) {
      const state = await res.json();
      updateStateDisplay(state);
    }
  } catch (e) {
    console.error("Failed to fetch state:", e);
  }
}

// Update state display UI
function updateStateDisplay(state) {
  if (!state) return;
  
  // Update indicator
  if (indicatorDot && indicatorText) {
    if (state.visible) {
      indicatorDot.className = 'indicator-dot visible';
      indicatorText.textContent = 'VISIBLE';
    } else {
      indicatorDot.className = 'indicator-dot hidden';
      indicatorText.textContent = 'HIDDEN';
    }
  }
  
  // Update details
  if (visibilityValue) {
    visibilityValue.textContent = state.visible ? 'VISIBLE' : 'HIDDEN';
    visibilityValue.style.color = state.visible ? 'var(--accent-green)' : 'var(--primary-red)';
  }
  
  if (actionValue) {
    actionValue.textContent = state.lastAction || 'unknown';
  }
  
  if (updatedValue) {
    const date = new Date(state.lastUpdated);
    updatedValue.textContent = date.toLocaleString();
  }
}

async function call(path) {
  try {
    // Show loading state
    statusValue.textContent = 'Processing...';
    statusValue.style.color = 'var(--warning-yellow)';
    networkLight.style.background = 'var(--warning-yellow)';
    networkLight.style.boxShadow = '0 0 8px var(--warning-yellow)';
    
    const res = await fetch(path, { method: "GET" });
    const text = await res.text();
    
    // Reset network light
    networkLight.style.background = 'var(--accent-green)';
    networkLight.style.boxShadow = '0 0 8px var(--accent-green)';
    
    setStatus(`${res.status}: ${text}`, res.ok);
    
    // Refresh state after successful operation
    if (res.ok) {
      setTimeout(fetchState, 500);
    }
  } catch (e) {
    networkLight.style.background = 'var(--primary-red)';
    networkLight.style.boxShadow = '0 0 8px var(--primary-red)';
    setStatus(`Error: ${e.message}`, false);
  }
}

// Button events
// Note: previously we used touch handlers + synthetic clicks, which can double-fire on iOS.
// Rely on click (covers mouse, touch, and keyboard activation) + CSS :active for press feedback.
function setupButtonEvents(buttonId, apiPath) {
  const button = document.getElementById(buttonId);
  if (!button) return;
  
  button.addEventListener("click", () => call(apiPath));
}

setupButtonEvents("show", "/show");
setupButtonEvents("hide", "/hide");
setupButtonEvents("toggle-on", "/toggle?value=true");
setupButtonEvents("toggle-off", "/toggle?value=false");

// Add CSS animation for status flash
const style = document.createElement('style');
style.textContent = `
  @keyframes statusFlash {
    0% { background: rgba(0, 0, 0, 0.3); }
    50% { background: rgba(0, 184, 148, 0.2); }
    100% { background: rgba(0, 0, 0, 0.3); }
  }
`;
document.head.appendChild(style);

// Initialize: fetch current state on page load
document.addEventListener('DOMContentLoaded', () => {
  fetchState();
  
  // Auto-refresh state every 30 seconds
  setInterval(fetchState, 30000);
});
