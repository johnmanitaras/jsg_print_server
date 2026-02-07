package api

const webUI = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>JetSetGo Print Server</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f5f5f5;color:#333;line-height:1.6}
a{color:#667eea;text-decoration:none}
a:hover{text-decoration:underline}

/* Header */
.hdr{background:linear-gradient(135deg,#667eea 0%,#764ba2 100%);color:#fff;padding:14px 20px;display:flex;align-items:center;justify-content:space-between;position:sticky;top:0;z-index:100}
.hdr h1{font-size:18px;font-weight:600}
.hdr-dot{width:10px;height:10px;border-radius:50%;display:inline-block;margin-left:8px}
.hdr-right{display:flex;align-items:center;font-size:13px;gap:6px}
.dot-green{background:#22c55e}.dot-red{background:#ef4444}.dot-yellow{background:#f59e0b}.dot-gray{background:#9ca3af}
.dot-pulse{animation:pulse 2s ease-in-out infinite}

/* Tab bar */
.tabs{display:flex;border-bottom:2px solid #e5e7eb;background:#fff;padding:0 16px;position:sticky;top:48px;z-index:99}
.tab{padding:12px 20px;cursor:pointer;font-size:14px;font-weight:500;color:#666;border-bottom:2px solid transparent;margin-bottom:-2px;transition:all .2s}
.tab:hover{color:#333}
.tab.active{color:#667eea;border-bottom-color:#667eea}
.tabs.hidden{display:none}

/* Content */
.content{max-width:900px;margin:0 auto;padding:20px}
.page{display:none;animation:fadeIn .3s ease}
.page.active{display:block}

/* Cards */
.card{background:#fff;border-radius:8px;padding:20px;margin-bottom:16px;box-shadow:0 1px 3px rgba(0,0,0,.1);transition:box-shadow .2s}
.card:hover{box-shadow:0 2px 8px rgba(0,0,0,.12)}
.card h2{font-size:16px;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid #eee}
.card h3{font-size:14px;margin-bottom:8px}

/* Buttons */
.btn{display:inline-flex;align-items:center;gap:6px;padding:8px 16px;border-radius:6px;border:none;cursor:pointer;font-size:14px;font-weight:500;transition:all .2s;line-height:1.4}
.btn:disabled{opacity:.5;cursor:not-allowed}
.btn-primary{background:#667eea;color:#fff}.btn-primary:hover:not(:disabled){background:#5a67d8}
.btn-secondary{background:#e5e7eb;color:#374151}.btn-secondary:hover:not(:disabled){background:#d1d5db}
.btn-danger{background:#fff;color:#ef4444;border:1px solid #ef4444}.btn-danger:hover:not(:disabled){background:#fef2f2}
.btn-sm{padding:5px 10px;font-size:12px}
.btn-row{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}

/* Forms */
.form-group{margin-bottom:14px}
.form-group label{display:block;font-size:13px;font-weight:500;margin-bottom:4px;color:#555}
.form-group input,.form-group select{width:100%;padding:8px 12px;border:1px solid #ddd;border-radius:6px;font-size:14px;transition:border .2s}
.form-group input:focus,.form-group select:focus{outline:none;border-color:#667eea;box-shadow:0 0 0 3px rgba(102,126,234,.15)}
.form-help{font-size:12px;color:#888;margin-top:3px}
.form-row{display:grid;grid-template-columns:1fr 1fr;gap:12px}

/* Status badges */
.badge{display:inline-block;padding:2px 10px;border-radius:20px;font-size:12px;font-weight:500}
.badge-green{background:#dcfce7;color:#166534}
.badge-red{background:#fee2e2;color:#991b1b}
.badge-yellow{background:#fef9c3;color:#854d0e}
.badge-blue{background:#dbeafe;color:#1e40af}
.badge-gray{background:#f3f4f6;color:#374151}

/* Status dots */
.sdot{width:10px;height:10px;border-radius:50%;display:inline-block;margin-right:6px;flex-shrink:0}

/* Hero card */
.hero{padding:24px;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:16px}
.hero-status{display:flex;align-items:center;gap:12px}
.hero-dot{width:40px;height:40px;border-radius:50%;flex-shrink:0}
.hero-text h3{font-size:18px;margin:0;border:none;padding:0}
.hero-text p{font-size:13px;color:#666;margin:0}
.hero-info{text-align:right;font-size:13px;color:#666}
.hero-info strong{color:#333;display:block}

/* Printer cards */
.p-card{background:#f9fafb;border-radius:6px;padding:14px;margin-bottom:10px;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;transition:background .2s}
.p-card.highlight{animation:highlight 2s ease}
.p-info{flex:1;min-width:200px}
.p-info h4{font-size:14px;margin-bottom:4px;display:flex;align-items:center;gap:6px}
.p-info p{font-size:12px;color:#666}
.p-badges{display:flex;gap:4px;margin:4px 0}
.p-actions{display:flex;gap:6px}

/* Jobs table */
.job-row{display:grid;grid-template-columns:120px 1fr 90px 100px 70px;gap:8px;padding:10px 0;border-bottom:1px solid #f0f0f0;font-size:13px;align-items:center;animation:slideIn .3s ease}
.job-row:last-child{border:none}
.job-id{font-family:'SF Mono','Cascadia Code','Courier New',monospace;font-size:12px;color:#666;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.job-hdr{font-weight:600;color:#555;font-size:12px;text-transform:uppercase;letter-spacing:.05em}

/* Logs */
.log-container{background:#1a1a2e;border-radius:8px;padding:16px;font-family:'SF Mono','Cascadia Code','Courier New',monospace;font-size:13px;max-height:500px;overflow-y:auto;color:#a0aec0}
.log-entry{padding:2px 0;white-space:pre-wrap;word-break:break-all}
.log-time{color:#667eea}
.log-info{color:#a0aec0}.log-warn{color:#f59e0b}.log-error{color:#ef4444}
.log-controls{display:flex;gap:8px;margin-bottom:12px;align-items:center;flex-wrap:wrap}
.filter-btn{padding:5px 12px;border-radius:4px;border:1px solid #ddd;background:#fff;cursor:pointer;font-size:12px;transition:all .2s}
.filter-btn.active{background:#667eea;color:#fff;border-color:#667eea}

/* Settings sections */
.settings-section{margin-bottom:8px}
.settings-section summary{cursor:pointer;padding:12px;font-weight:500;list-style:none;display:flex;align-items:center;gap:8px}
.settings-section summary::-webkit-details-marker{display:none}
.settings-section summary::before{content:"";display:inline-block;width:0;height:0;border-left:6px solid #666;border-top:4px solid transparent;border-bottom:4px solid transparent;transition:transform .2s}
.settings-section[open] summary::before{transform:rotate(90deg)}
.settings-body{padding:0 12px 12px}
.readonly{background:#f9fafb;color:#666;cursor:default}

/* Wizard */
.wizard{max-width:500px;margin:40px auto;text-align:center}
.wizard .card{padding:40px}
.steps{display:flex;align-items:center;justify-content:center;gap:0;margin-bottom:32px}
.step-circle{width:36px;height:36px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:14px;font-weight:600;border:2px solid #ddd;color:#999;background:#fff;transition:all .3s;flex-shrink:0}
.step-circle.active{border-color:#667eea;background:#667eea;color:#fff}
.step-circle.done{border-color:#22c55e;background:#22c55e;color:#fff}
.step-line{width:60px;height:2px;background:#ddd;transition:background .3s}
.step-line.done{background:#22c55e}
.wizard-icon{margin-bottom:20px}
.wizard h2{font-size:22px;margin-bottom:8px;border:none;padding:0}
.wizard p{color:#666;margin-bottom:20px;font-size:14px}
.wizard .form-group{text-align:left}
.code-input{font-family:'SF Mono','Cascadia Code','Courier New',monospace !important;font-size:2rem !important;letter-spacing:.5em !important;text-align:center;text-transform:uppercase}
.error-box{background:#fef2f2;border:1px solid #fecaca;border-radius:6px;padding:12px;color:#991b1b;font-size:13px;margin-top:12px;text-align:left}

/* Success animation */
.check-circle{width:80px;height:80px;border-radius:50%;background:#22c55e;margin:0 auto 20px;display:flex;align-items:center;justify-content:center}
.check-circle svg{width:40px;height:40px;stroke:#fff;stroke-width:3;fill:none;stroke-dasharray:50;stroke-dashoffset:50;animation:drawCheck .6s ease forwards .2s}

/* Toast */
.toast{position:fixed;top:60px;right:20px;padding:12px 20px;border-radius:6px;color:#fff;font-size:14px;z-index:200;animation:slideIn .3s ease;box-shadow:0 4px 12px rgba(0,0,0,.15)}
.toast-success{background:#22c55e}.toast-error{background:#ef4444}.toast-warn{background:#f59e0b}

/* Modal */
.modal-overlay{position:fixed;inset:0;background:rgba(0,0,0,.4);display:flex;align-items:center;justify-content:center;z-index:150;animation:fadeIn .2s}
.modal{background:#fff;border-radius:8px;padding:24px;max-width:400px;width:90%;box-shadow:0 8px 24px rgba(0,0,0,.2)}
.modal h3{margin-bottom:12px;font-size:16px}
.modal p{color:#666;font-size:14px;margin-bottom:16px}

/* Empty state */
.empty{text-align:center;padding:40px;color:#888}
.empty svg{margin-bottom:12px}
.empty p{margin-bottom:16px}

/* Spinner */
.spinner{width:16px;height:16px;border:2px solid rgba(255,255,255,.3);border-top-color:#fff;border-radius:50%;animation:spin .8s linear infinite;display:inline-block}
.spinner-dark{border-color:rgba(0,0,0,.1);border-top-color:#667eea}

/* Animations */
@keyframes fadeIn{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}
@keyframes slideIn{from{opacity:0;transform:translateY(-20px)}to{opacity:1;transform:translateY(0)}}
@keyframes pulse{0%,100%{transform:scale(1);opacity:1}50%{transform:scale(1.5);opacity:0}}
@keyframes spin{to{transform:rotate(360deg)}}
@keyframes drawCheck{to{stroke-dashoffset:0}}
@keyframes highlight{0%{background:#dbeafe}100%{background:#f9fafb}}

/* Responsive */
@media(max-width:640px){
 .content{padding:12px}
 .hero{flex-direction:column;align-items:flex-start}
 .hero-info{text-align:left}
 .form-row{grid-template-columns:1fr}
 .job-row{grid-template-columns:1fr 1fr;gap:4px}
 .job-row .job-id{grid-column:1/-1}
 .tabs{overflow-x:auto}
 .tab{padding:10px 14px;font-size:13px;white-space:nowrap}
}

/* Focus */
:focus-visible{outline:2px solid #667eea;outline-offset:2px}
</style>
</head>
<body>

<div class="hdr">
 <h1>JetSetGo Print Server</h1>
 <div class="hdr-right">
  <span id="hdr-status-text">Checking...</span>
  <span id="hdr-dot" class="hdr-dot dot-yellow"></span>
 </div>
</div>

<div class="tabs" id="tab-bar">
 <div class="tab active" data-page="dashboard" onclick="nav('dashboard')">Dashboard</div>
 <div class="tab" data-page="printers" onclick="nav('printers')">Printers</div>
 <div class="tab" data-page="settings" onclick="nav('settings')">Settings</div>
 <div class="tab" data-page="logs" onclick="nav('logs')">Logs</div>
</div>

<div class="content">
 <!-- Wizard -->
 <div class="page" id="page-wizard">
  <div class="wizard">
   <div class="steps" id="wizard-steps">
    <div class="step-circle active" id="ws1">1</div>
    <div class="step-line" id="wl1"></div>
    <div class="step-circle" id="ws2">2</div>
    <div class="step-line" id="wl2"></div>
    <div class="step-circle" id="ws3">3</div>
   </div>
   <div class="card">
    <!-- Step 1: Welcome -->
    <div id="wiz-step-1">
     <div class="wizard-icon">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#667eea" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="2" width="12" height="20" rx="1"/><path d="M6 18h12"/><path d="M6 14h12"/><circle cx="12" cy="21" r="0.5" fill="#667eea"/><path d="M10 6h4"/></svg>
     </div>
     <h2>Welcome to JetSetGo Print Server</h2>
     <p>Connect this server to your JetSetGo account to start printing receipts, tickets, and boarding passes.</p>
     <button class="btn btn-primary" onclick="wizardNext(2)" style="padding:12px 32px;font-size:16px">Get Started</button>
     <p style="font-size:12px;color:#999;margin-top:16px">You'll need a registration code from your JetSetGo admin dashboard</p>
    </div>
    <!-- Step 2: Register -->
    <div id="wiz-step-2" style="display:none">
     <h2>Connect to Cloud</h2>
     <p>Enter your organization details to register this print server.</p>
     <div class="form-group" style="text-align:left">
      <label for="reg-tenant">Tenant ID</label>
      <input type="text" id="reg-tenant" placeholder="e.g., tta" autocomplete="off">
      <div class="form-help">Your organization's tenant identifier, provided by JetSetGo.</div>
     </div>
     <div class="form-group" style="text-align:left">
      <label for="reg-code">Registration Code</label>
      <input type="text" id="reg-code" placeholder="Paste registration code here" autocomplete="off" style="font-family:monospace;font-size:14px">
      <div class="form-help">Enter the code shown in your JetSetGo admin dashboard.</div>
     </div>
     <div id="reg-error" class="error-box" style="display:none"></div>
     <button class="btn btn-primary" id="reg-btn" onclick="doRegister()" style="padding:12px 32px;font-size:16px;margin-top:8px">Register Server</button>
     <p style="margin-top:12px"><a href="javascript:void(0)" onclick="wizardNext(1)" style="color:#888;font-size:13px">Back</a></p>
    </div>
    <!-- Step 3: Success -->
    <div id="wiz-step-3" style="display:none">
     <div class="check-circle">
      <svg viewBox="0 0 24 24"><polyline points="4 12 9 17 20 6"/></svg>
     </div>
     <h2>Connected Successfully!</h2>
     <p>Your print server is now linked to JetSetGo. You can configure printers and start printing.</p>
     <div style="background:#f9fafb;border-radius:6px;padding:12px;margin:16px 0;font-size:13px;text-align:left">
      <div><strong>Server:</strong> <span id="reg-result-name">-</span></div>
      <div><strong>Tenant:</strong> <span id="reg-result-tenant">-</span></div>
     </div>
     <button class="btn btn-primary" onclick="goToDashboard()" style="padding:12px 32px">Go to Dashboard</button>
     <p id="redirect-text" style="font-size:12px;color:#999;margin-top:12px"></p>
    </div>
   </div>
  </div>
 </div>

 <!-- Dashboard -->
 <div class="page" id="page-dashboard">
  <div class="card hero" id="dash-hero">
   <div class="hero-status">
    <div class="hero-dot dot-yellow" id="dash-dot"></div>
    <div class="hero-text">
     <h3 id="dash-status-text">Checking connection...</h3>
     <p id="dash-status-sub">Please wait</p>
    </div>
   </div>
   <div class="hero-info" id="dash-info">
    <strong id="dash-server-name">-</strong>
    <span id="dash-tenant">-</span><br>
    <span id="dash-server-id" style="font-size:11px;font-family:monospace;color:#999" title="">-</span>
   </div>
  </div>

  <h3 style="font-size:14px;margin-bottom:8px;color:#555">Printers</h3>
  <div id="dash-printers"></div>

  <div class="card" style="margin-top:16px">
   <h2>Recent Print Jobs</h2>
   <div id="dash-jobs"></div>
  </div>

  <div class="btn-row" style="margin-top:4px">
   <button class="btn btn-secondary btn-sm" id="dash-test-btn" onclick="dashTestPrint()" style="display:none">Test Print</button>
   <button class="btn btn-secondary btn-sm" onclick="refreshDashboard()">Refresh Status</button>
  </div>
 </div>

 <!-- Printers -->
 <div class="page" id="page-printers">
  <div id="add-printer-toggle">
   <button class="btn btn-primary" onclick="toggleAddPrinter(true)">+ Add Printer</button>
   <button class="btn btn-secondary" onclick="scanPrinters()" id="scan-btn">Scan Network</button>
  </div>

  <div class="card" id="add-printer-form" style="display:none;margin-top:12px">
   <h2>Add Printer</h2>
   <div class="form-group">
    <label for="ap-name">Printer Name</label>
    <input type="text" id="ap-name" placeholder="e.g., Front Desk Receipt">
   </div>
   <div class="form-group">
    <label>Type</label>
    <div style="display:flex;gap:16px;margin-top:4px">
     <label style="display:flex;align-items:center;gap:4px;cursor:pointer;font-size:14px"><input type="radio" name="ap-type" value="network" checked onchange="togglePrinterType()"> Network</label>
     <label style="display:flex;align-items:center;gap:4px;cursor:pointer;font-size:14px"><input type="radio" name="ap-type" value="usb" onchange="togglePrinterType()"> USB</label>
    </div>
   </div>
   <div id="ap-network-fields">
    <div class="form-row">
     <div class="form-group"><label for="ap-address">IP Address</label><input type="text" id="ap-address" placeholder="192.168.1.100"></div>
     <div class="form-group"><label for="ap-port">Port</label><input type="number" id="ap-port" value="9100" placeholder="9100"></div>
    </div>
   </div>
   <div class="form-row">
    <div class="form-group">
     <label for="ap-width">Paper Width</label>
     <select id="ap-width"><option value="80">80mm</option><option value="58">58mm</option></select>
    </div>
    <div class="form-group">
     <label for="ap-id">Printer ID</label>
     <input type="text" id="ap-id" placeholder="auto-generated">
     <div class="form-help">Auto-generated from name, or set manually</div>
    </div>
   </div>
   <div class="btn-row">
    <button class="btn btn-primary" onclick="addPrinter()">Add Printer</button>
    <button class="btn btn-secondary" onclick="toggleAddPrinter(false)">Cancel</button>
   </div>
  </div>

  <div id="scan-results" style="display:none;margin-top:12px" class="card">
   <h2>Discovered Printers</h2>
   <div id="scan-list"></div>
  </div>

  <div id="printer-list" style="margin-top:12px"></div>
 </div>

 <!-- Settings -->
 <div class="page" id="page-settings">
  <details class="settings-section card" open>
   <summary>Cloud Connection</summary>
   <div class="settings-body">
    <div class="form-group"><label>Cloud Endpoint</label><input type="text" id="set-endpoint" class="readonly" readonly></div>
    <div class="form-group"><label>WebSocket Endpoint</label><input type="text" id="set-ws-endpoint" class="readonly" readonly></div>
    <div class="form-group">
     <label>Connection Method</label>
     <select id="set-use-ws" onchange="toggleWsSettings()">
      <option value="false">HTTP Polling (Recommended)</option>
      <option value="true">WebSocket (auto-fallback to polling)</option>
     </select>
    </div>
    <div id="set-poll-group" class="form-group" style="display:none">
     <label for="set-poll">Poll Interval (seconds)</label>
     <input type="number" id="set-poll" min="5" max="300" value="30">
    </div>
    <details style="margin-top:8px">
     <summary style="font-size:13px;color:#666;cursor:pointer">WebSocket Advanced Settings</summary>
     <div style="padding-top:8px">
      <div class="form-row">
       <div class="form-group"><label for="set-ws-delay">Reconnect Delay (s)</label><input type="number" id="set-ws-delay" min="1" max="60"></div>
       <div class="form-group"><label for="set-ws-max">Max Reconnect Delay (s)</label><input type="number" id="set-ws-max" min="5" max="300"></div>
      </div>
      <div class="form-group"><label for="set-ws-ping">Ping Interval (s)</label><input type="number" id="set-ws-ping" min="10" max="120"></div>
     </div>
    </details>
   </div>
  </details>

  <details class="settings-section card" open>
   <summary>Server Identity</summary>
   <div class="settings-body">
    <div class="form-row">
     <div class="form-group"><label for="set-name">Server Name</label><input type="text" id="set-name"></div>
     <div class="form-group"><label for="set-location">Location</label><input type="text" id="set-location" placeholder="Optional"></div>
    </div>
    <div class="form-group"><label for="set-server-id">Server ID</label><div style="display:flex;gap:6px"><input type="text" id="set-server-id" style="font-family:monospace;font-size:13px" placeholder="e.g., e81731a7-2bba-4f2e-..."><button class="btn btn-secondary btn-sm" onclick="copyText('set-server-id')">Copy</button></div><div class="form-help">Set during registration, or enter manually.</div></div>
    <div class="form-group"><label for="set-tenant">Tenant ID</label><input type="text" id="set-tenant" placeholder="e.g., tta"><div class="form-help">Your organization's tenant identifier.</div></div>
    <div class="form-group">
     <label for="set-apikey">API Key</label>
     <input type="text" id="set-apikey" style="font-family:monospace" placeholder="Enter API key">
     <div class="form-help">Authentication key for cloud connection. Set during registration, or enter manually.</div>
    </div>
   </div>
  </details>

  <details class="settings-section card">
   <summary>Local Server</summary>
   <div class="settings-body">
    <div class="form-row">
     <div class="form-group"><label for="set-port">HTTP Port</label><input type="number" id="set-port" min="1" max="65535"></div>
     <div class="form-group"><label for="set-host">Listen Address</label><input type="text" id="set-host" placeholder="0.0.0.0"></div>
    </div>
    <div class="form-group"><label>Config File</label><input type="text" id="set-config-path" class="readonly" readonly></div>
   </div>
  </details>

  <div class="btn-row" style="margin-top:16px">
   <button class="btn btn-primary" id="save-settings-btn" onclick="saveSettings()">Save Changes</button>
   <button class="btn btn-danger" onclick="confirmReRegister()">Re-Register Server</button>
  </div>
 </div>

 <!-- Logs -->
 <div class="page" id="page-logs">
  <div class="log-controls">
   <button class="filter-btn active" onclick="setLogFilter('all',this)">All</button>
   <button class="filter-btn" onclick="setLogFilter('info',this)">Info</button>
   <button class="filter-btn" onclick="setLogFilter('warn',this)">Warning</button>
   <button class="filter-btn" onclick="setLogFilter('error',this)">Error</button>
   <div style="flex:1"></div>
   <label style="font-size:12px;display:flex;align-items:center;gap:4px"><input type="checkbox" id="log-autoscroll" checked> Auto-scroll</label>
   <button class="btn btn-secondary btn-sm" onclick="downloadLogs()">Download</button>
  </div>
  <div class="log-container" id="log-viewer"></div>
 </div>
</div>

<!-- Modal container -->
<div id="modal-root"></div>
<!-- Toast container -->
<div id="toast-root"></div>

<script>
// ============ State ============
var currentPage = 'dashboard';
var logFilter = 'all';
var configData = null;
var statusData = null;
var pollTimer = null;
var logTimer = null;
var redirectTimer = null;

// ============ Routing ============
function nav(page) {
 if (page === currentPage) return;
 currentPage = page;
 window.location.hash = '/' + page;
 renderPage();
}

function renderPage() {
 var pages = document.querySelectorAll('.page');
 for (var i = 0; i < pages.length; i++) pages[i].classList.remove('active');
 var tabs = document.querySelectorAll('.tab');
 for (var i = 0; i < tabs.length; i++) tabs[i].classList.remove('active');

 var el = document.getElementById('page-' + currentPage);
 if (el) el.classList.add('active');
 var tab = document.querySelector('.tab[data-page="' + currentPage + '"]');
 if (tab) tab.classList.add('active');

 // Load data for page
 if (currentPage === 'dashboard') refreshDashboard();
 if (currentPage === 'printers') refreshPrinters();
 if (currentPage === 'settings') loadSettings();
 if (currentPage === 'logs') refreshLogs();

 // Manage timers
 clearInterval(pollTimer);
 clearInterval(logTimer);
 if (currentPage === 'dashboard') pollTimer = setInterval(refreshDashboard, 5000);
 if (currentPage === 'logs') logTimer = setInterval(refreshLogs, 3000);
}

function goToDashboard() {
 clearTimeout(redirectTimer);
 document.getElementById('tab-bar').classList.remove('hidden');
 nav('dashboard');
}

// ============ Init ============
function init() {
 // Check registration
 fetch('/api/config').then(function(r){return r.json()}).then(function(data) {
  configData = data;
  if (!data.is_registered) {
   showWizard();
  } else {
   var hash = window.location.hash.replace('#/', '');
   if (['dashboard','printers','settings','logs'].indexOf(hash) >= 0) {
    currentPage = hash;
   }
   renderPage();
   updateHeaderStatus();
  }
 }).catch(function() {
  currentPage = 'dashboard';
  renderPage();
 });
}

function showWizard() {
 document.getElementById('tab-bar').classList.add('hidden');
 currentPage = 'wizard';
 var pages = document.querySelectorAll('.page');
 for (var i = 0; i < pages.length; i++) pages[i].classList.remove('active');
 document.getElementById('page-wizard').classList.add('active');
 wizardNext(1);
}

// ============ Wizard ============
function wizardNext(step) {
 for (var i = 1; i <= 3; i++) {
  document.getElementById('wiz-step-' + i).style.display = (i === step) ? 'block' : 'none';
  var circle = document.getElementById('ws' + i);
  circle.className = 'step-circle' + (i < step ? ' done' : (i === step ? ' active' : ''));
  circle.innerHTML = i < step ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="3"><polyline points="4 12 9 17 20 6"/></svg>' : String(i);
 }
 document.getElementById('wl1').className = 'step-line' + (step > 1 ? ' done' : '');
 document.getElementById('wl2').className = 'step-line' + (step > 2 ? ' done' : '');
 if (step === 2) document.getElementById('reg-tenant').focus();
}

function doRegister() {
 var tenant = document.getElementById('reg-tenant').value.trim();
 var code = document.getElementById('reg-code').value.trim();
 var errBox = document.getElementById('reg-error');
 var btn = document.getElementById('reg-btn');

 if (!tenant || !code) {
  errBox.textContent = 'Please fill in both fields.';
  errBox.style.display = 'block';
  return;
 }

 errBox.style.display = 'none';
 btn.disabled = true;
 btn.innerHTML = '<span class="spinner"></span> Connecting...';

 fetch('/api/register', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({tenant: tenant, registration_code: code})
 }).then(function(r){return r.json()}).then(function(data) {
  btn.disabled = false;
  btn.textContent = 'Register Server';
  if (data.success) {
   document.getElementById('reg-result-name').textContent = data.server_name || data.server_id;
   document.getElementById('reg-result-tenant').textContent = tenant;
   wizardNext(3);
   startRedirectCountdown();
  } else {
   errBox.textContent = data.message || 'Registration failed.';
   errBox.style.display = 'block';
  }
 }).catch(function(err) {
  btn.disabled = false;
  btn.textContent = 'Register Server';
  errBox.textContent = 'Network error. Check your connection and try again.';
  errBox.style.display = 'block';
 });
}

function startRedirectCountdown() {
 var count = 3;
 var el = document.getElementById('redirect-text');
 el.textContent = 'Redirecting in ' + count + '...';
 redirectTimer = setInterval(function() {
  count--;
  if (count <= 0) {
   clearInterval(redirectTimer);
   goToDashboard();
  } else {
   el.textContent = 'Redirecting in ' + count + '...';
  }
 }, 1000);
}

// ============ Dashboard ============
function refreshDashboard() {
 fetch('/api/status').then(function(r){return r.json()}).then(function(data) {
  statusData = data;
  updateHeaderStatus();
  var dot = document.getElementById('dash-dot');
  var text = document.getElementById('dash-status-text');
  var sub = document.getElementById('dash-status-sub');

  if (data.cloud_connected) {
   dot.className = 'hero-dot dot-green dot-pulse';
   text.textContent = 'Connected to JetSetGo';
   var method = data.connection_method === 'websocket' ? 'WebSocket active' : 'Polling every ' + (configData && configData.cloud ? configData.cloud.poll_interval : '5s');
   sub.textContent = method;
  } else if (data.websocket && data.websocket.reconnecting) {
   dot.className = 'hero-dot dot-yellow dot-pulse';
   text.textContent = 'Reconnecting...';
   sub.textContent = data.websocket.last_error || 'Attempting to reconnect';
  } else if (!data.is_registered) {
   dot.className = 'hero-dot dot-gray';
   text.textContent = 'Not Registered';
   sub.textContent = 'Go to Settings to enter credentials, or use the registration wizard.';
  } else if (data.connection_method === 'none') {
   dot.className = 'hero-dot dot-yellow';
   text.textContent = 'Not Connected';
   sub.textContent = 'No cloud client running. Save Settings to start the connection.';
  } else {
   dot.className = 'hero-dot dot-red';
   text.textContent = 'Disconnected';
   var reason = '';
   if (data.websocket && data.websocket.last_error) reason = data.websocket.last_error;
   else reason = 'Attempting to connect to cloud...';
   sub.textContent = reason;
  }

  document.getElementById('dash-server-name').textContent = data.server_name || 'Print Server';
  document.getElementById('dash-tenant').textContent = data.tenant ? 'Tenant: ' + data.tenant : '';
  var sid = data.is_registered ? (configData && configData.cloud ? configData.cloud.server_id : '') : 'Not registered';
  var sidEl = document.getElementById('dash-server-id');
  sidEl.textContent = sid && sid.length > 16 ? sid.substring(0, 16) + '...' : sid;
  sidEl.title = sid;
 }).catch(function(){});

 // Printers summary
 fetch('/api/printers').then(function(r){return r.json()}).then(function(data) {
  var container = document.getElementById('dash-printers');
  var testBtn = document.getElementById('dash-test-btn');
  var printers = data.printers || [];
  if (printers.length === 0) {
   container.innerHTML = '<div class="card" style="text-align:center;padding:20px;color:#888;font-size:13px">No printers configured. <a href="javascript:nav(\'printers\')">Add one</a></div>';
   testBtn.style.display = 'none';
  } else {
   testBtn.style.display = '';
   var html = '<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(250px,1fr));gap:10px">';
   for (var i = 0; i < printers.length; i++) {
    var p = printers[i];
    var sc = p.status === 'online' ? 'dot-green' : (p.status === 'offline' ? 'dot-red' : 'dot-yellow');
    var tc = p.type === 'network' ? 'badge-blue' : 'badge-purple';
    html += '<div class="card" style="padding:14px;margin:0">';
    html += '<div style="display:flex;align-items:center;gap:6px;margin-bottom:4px"><span class="sdot ' + sc + '"></span><strong>' + esc(p.name) + '</strong></div>';
    html += '<div class="p-badges"><span class="badge badge-blue">' + esc(p.type) + '</span>';
    html += '<span class="badge badge-gray">' + (p.paper_width || 80) + 'mm</span></div>';
    html += '<button class="btn btn-secondary btn-sm" onclick="testPrint(\'' + esc(p.id) + '\')" style="margin-top:8px">Test Print</button>';
    html += '</div>';
   }
   html += '</div>';
   container.innerHTML = html;
  }
 }).catch(function(){});

 // Recent jobs
 fetch('/api/jobs').then(function(r){return r.json()}).then(function(data) {
  var container = document.getElementById('dash-jobs');
  var jobs = data.jobs || [];
  if (jobs.length === 0) {
   container.innerHTML = '<div class="empty"><svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#ccc" stroke-width="1.5"><rect x="6" y="2" width="12" height="20" rx="1"/><path d="M6 18h12"/><path d="M6 14h12"/></svg><p>No print jobs yet. Send a test print to verify your setup.</p></div>';
  } else {
   var html = '<div class="job-row job-hdr"><span>Job ID</span><span>Printer</span><span>Status</span><span>Time</span><span>Size</span></div>';
   var limit = Math.min(jobs.length, 20);
   for (var i = 0; i < limit; i++) {
    var j = jobs[i];
    var bc = j.status === 'completed' ? 'badge-green' : (j.status === 'failed' ? 'badge-red' : (j.status === 'printing' ? 'badge-blue' : 'badge-yellow'));
    html += '<div class="job-row">';
    html += '<span class="job-id" title="' + esc(j.id) + '">' + esc(j.id.length > 14 ? j.id.substring(0,14) + '..' : j.id) + '</span>';
    html += '<span>' + esc(j.printer_name || j.printer_id) + '</span>';
    html += '<span class="badge ' + bc + '">' + esc(j.status) + '</span>';
    html += '<span style="font-size:12px;color:#888">' + timeAgo(j.created_at) + '</span>';
    html += '<span style="font-size:12px;color:#888">' + formatBytes(j.data_size) + '</span>';
    html += '</div>';
   }
   container.innerHTML = html;
  }
 }).catch(function(){});
}

function dashTestPrint() {
 fetch('/api/printers').then(function(r){return r.json()}).then(function(data) {
  var printers = data.printers || [];
  if (printers.length > 0) testPrint(printers[0].id);
 });
}

// ============ Header Status ============
function updateHeaderStatus() {
 if (!statusData) return;
 var dot = document.getElementById('hdr-dot');
 var text = document.getElementById('hdr-status-text');
 if (statusData.cloud_connected) {
  dot.className = 'hdr-dot dot-green';
  text.textContent = 'Connected';
 } else if (statusData.websocket && statusData.websocket.reconnecting) {
  dot.className = 'hdr-dot dot-yellow dot-pulse';
  text.textContent = 'Reconnecting';
 } else if (!statusData.is_registered) {
  dot.className = 'hdr-dot dot-yellow';
  text.textContent = 'Not Registered';
 } else if (statusData.connection_method === 'none') {
  dot.className = 'hdr-dot dot-yellow';
  text.textContent = 'Not Started';
 } else {
  dot.className = 'hdr-dot dot-red';
  text.textContent = 'Disconnected';
 }
}

// ============ Printers Page ============
function refreshPrinters() {
 fetch('/api/printers').then(function(r){return r.json()}).then(function(data) {
  var container = document.getElementById('printer-list');
  var printers = data.printers || [];
  if (printers.length === 0) {
   container.innerHTML = '<div class="empty"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#ccc" stroke-width="1.5"><rect x="6" y="2" width="12" height="20" rx="1"/><path d="M6 18h12"/><path d="M6 14h12"/></svg><p>No printers configured</p><p style="font-size:13px">Add a printer manually or scan your network to discover available printers.</p></div>';
  } else {
   var html = '';
   for (var i = 0; i < printers.length; i++) {
    var p = printers[i];
    var sc = p.status === 'online' ? 'dot-green' : (p.status === 'offline' ? 'dot-red' : 'dot-yellow');
    var stext = p.status === 'online' ? 'Online' : (p.status === 'offline' ? 'Offline' : 'Unknown');
    html += '<div class="p-card" id="pc-' + esc(p.id) + '">';
    html += '<div class="p-info">';
    html += '<h4><span class="sdot ' + sc + '"></span>' + esc(p.name) + '</h4>';
    html += '<p>' + esc(p.type === 'network' ? p.address + ':' + p.port : 'USB') + ' &middot; ' + stext + '</p>';
    html += '<div class="p-badges"><span class="badge badge-blue">' + esc(p.type) + '</span>';
    html += '<span class="badge badge-gray">' + (p.paper_width || 80) + 'mm</span></div>';
    html += '</div>';
    html += '<div class="p-actions">';
    html += '<button class="btn btn-primary btn-sm" onclick="testPrint(\'' + esc(p.id) + '\')">Test Print</button>';
    html += '<button class="btn btn-secondary btn-sm" onclick="editPrinter(\'' + esc(p.id) + '\')">Edit</button>';
    html += '<button class="btn btn-danger btn-sm" onclick="confirmDeletePrinter(\'' + esc(p.id) + '\',\'' + esc(p.name) + '\')">Remove</button>';
    html += '</div></div>';
   }
   container.innerHTML = html;
  }
 }).catch(function(){});
}

function toggleAddPrinter(show) {
 document.getElementById('add-printer-form').style.display = show ? 'block' : 'none';
 if (show) document.getElementById('ap-name').focus();
}

function togglePrinterType() {
 var isNetwork = document.querySelector('input[name="ap-type"]:checked').value === 'network';
 document.getElementById('ap-network-fields').style.display = isNetwork ? 'block' : 'none';
}

function slugify(text) {
 return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

function addPrinter() {
 var name = document.getElementById('ap-name').value.trim();
 var type = document.querySelector('input[name="ap-type"]:checked').value;
 var address = document.getElementById('ap-address').value.trim();
 var port = parseInt(document.getElementById('ap-port').value) || 9100;
 var width = parseInt(document.getElementById('ap-width').value) || 80;
 var id = document.getElementById('ap-id').value.trim() || slugify(name);

 if (!name) { toast('Please enter a printer name', 'error'); return; }
 if (type === 'network' && !address) { toast('Please enter an IP address', 'error'); return; }

 var body = {id: id, name: name, type: type, paper_width: width};
 if (type === 'network') { body.address = address; body.port = port; }

 fetch('/api/printers', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify(body)
 }).then(function(r){return r.json()}).then(function(data) {
  if (data.success) {
   toast('Printer added successfully', 'success');
   toggleAddPrinter(false);
   // Clear form
   document.getElementById('ap-name').value = '';
   document.getElementById('ap-address').value = '';
   document.getElementById('ap-port').value = '9100';
   document.getElementById('ap-id').value = '';
   refreshPrinters();
  } else {
   toast(data.error || 'Failed to add printer', 'error');
  }
 }).catch(function(){ toast('Network error', 'error'); });
}

function scanPrinters() {
 var btn = document.getElementById('scan-btn');
 btn.disabled = true;
 btn.innerHTML = '<span class="spinner-dark" style="width:12px;height:12px;border-width:2px;display:inline-block;border:2px solid rgba(0,0,0,.1);border-top-color:#667eea;border-radius:50%;animation:spin .8s linear infinite"></span> Scanning...';

 fetch('/api/printers/discover', {method:'POST'}).then(function(r){return r.json()}).then(function(data) {
  btn.disabled = false;
  btn.textContent = 'Scan Network';
  var results = data.discovered || [];
  var container = document.getElementById('scan-results');
  var list = document.getElementById('scan-list');
  if (results.length === 0) {
   container.style.display = 'block';
   list.innerHTML = '<p style="color:#888;font-size:13px">No network printers found. Make sure printers are powered on and connected to the same network.</p>';
  } else {
   container.style.display = 'block';
   var html = '';
   for (var i = 0; i < results.length; i++) {
    var r = results[i];
    html += '<div class="p-card"><div class="p-info"><h4>' + esc(r.name) + '</h4><p>' + esc(r.address) + ':' + r.port + '</p></div>';
    html += '<button class="btn btn-primary btn-sm" onclick="prefillPrinter(\'' + esc(r.address) + '\',' + r.port + ',\'' + esc(r.name) + '\')">Add</button></div>';
   }
   list.innerHTML = html;
  }
 }).catch(function() {
  btn.disabled = false;
  btn.textContent = 'Scan Network';
  toast('Scan failed', 'error');
 });
}

function prefillPrinter(address, port, name) {
 toggleAddPrinter(true);
 document.getElementById('ap-name').value = name;
 document.getElementById('ap-address').value = address;
 document.getElementById('ap-port').value = port;
 document.getElementById('ap-id').value = slugify(name);
}

function testPrint(id) {
 toast('Sending test print...', 'info');
 fetch('/api/printers/' + encodeURIComponent(id) + '/test', {method:'POST'}).then(function(r){return r.json()}).then(function(data) {
  if (data.success) toast('Test print sent!', 'success');
  else toast('Test print failed: ' + (data.error || 'Unknown error'), 'error');
 }).catch(function(){ toast('Network error', 'error'); });
}

function editPrinter(id) {
 // Find current printer data
 fetch('/api/printers').then(function(r){return r.json()}).then(function(data) {
  var p = null;
  for (var i = 0; i < data.printers.length; i++) {
   if (data.printers[i].id === id) { p = data.printers[i]; break; }
  }
  if (!p) return;

  showModal(
   'Edit Printer: ' + p.name,
   '<div class="form-group"><label>Name</label><input type="text" id="edit-p-name" value="' + esc(p.name) + '"></div>' +
   (p.type === 'network' ? '<div class="form-row"><div class="form-group"><label>IP Address</label><input type="text" id="edit-p-address" value="' + esc(p.address) + '"></div><div class="form-group"><label>Port</label><input type="number" id="edit-p-port" value="' + p.port + '"></div></div>' : '') +
   '<div class="form-group"><label>Paper Width</label><select id="edit-p-width"><option value="80"' + (p.paper_width === 80 || !p.paper_width ? ' selected' : '') + '>80mm</option><option value="58"' + (p.paper_width === 58 ? ' selected' : '') + '>58mm</option></select></div>',
   function() {
    var body = {name: document.getElementById('edit-p-name').value.trim()};
    body.paper_width = parseInt(document.getElementById('edit-p-width').value);
    if (p.type === 'network') {
     body.address = document.getElementById('edit-p-address').value.trim();
     body.port = parseInt(document.getElementById('edit-p-port').value);
    }
    fetch('/api/printers/' + encodeURIComponent(id), {
     method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)
    }).then(function(r){return r.json()}).then(function(d) {
     closeModal();
     if (d.success) { toast('Printer updated', 'success'); refreshPrinters(); }
     else toast(d.error || 'Update failed', 'error');
    }).catch(function(){ toast('Network error', 'error'); });
   }
  );
 });
}

function confirmDeletePrinter(id, name) {
 showModal(
  'Remove Printer',
  '<p>Remove <strong>' + esc(name) + '</strong>? This won\'t delete the physical printer.</p>',
  function() {
   fetch('/api/printers/' + encodeURIComponent(id), {method:'DELETE'}).then(function(r){return r.json()}).then(function(d) {
    closeModal();
    if (d.success) { toast('Printer removed', 'success'); refreshPrinters(); }
    else toast(d.error || 'Remove failed', 'error');
   }).catch(function(){ toast('Network error', 'error'); });
  }
 );
}

// ============ Settings ============
function loadSettings() {
 fetch('/api/config').then(function(r){return r.json()}).then(function(data) {
  configData = data;
  var c = data.cloud || {};
  var s = data.server || {};
  document.getElementById('set-endpoint').value = c.endpoint || '';
  document.getElementById('set-ws-endpoint').value = c.ws_endpoint || '';
  document.getElementById('set-use-ws').value = c.use_websocket ? 'true' : 'false';
  document.getElementById('set-poll').value = parseDuration(c.poll_interval);
  document.getElementById('set-ws-delay').value = parseDuration(c.ws_reconnect_delay);
  document.getElementById('set-ws-max').value = parseDuration(c.ws_max_reconnect_delay);
  document.getElementById('set-ws-ping').value = parseDuration(c.ws_ping_interval);
  document.getElementById('set-name').value = c.server_name || '';
  document.getElementById('set-location').value = c.location || '';
  document.getElementById('set-server-id').value = c.server_id || '';
  document.getElementById('set-tenant').value = c.tenant || '';
  // Show prefix as placeholder hint, field stays empty unless user types a new key
  var akField = document.getElementById('set-apikey');
  akField.value = '';
  akField.placeholder = c.api_key_prefix ? c.api_key_prefix + ' (leave blank to keep current)' : 'Enter API key';
  document.getElementById('set-port').value = s.port || 8080;
  document.getElementById('set-host').value = s.host || '0.0.0.0';
  document.getElementById('set-config-path').value = data.config_path || '';
  toggleWsSettings();
 }).catch(function(){});
}

function toggleWsSettings() {
 var ws = document.getElementById('set-use-ws').value === 'true';
 document.getElementById('set-poll-group').style.display = ws ? 'none' : 'block';
}

function saveSettings() {
 var cloudCfg = {
  use_websocket: document.getElementById('set-use-ws').value === 'true',
  server_name: document.getElementById('set-name').value.trim(),
  location: document.getElementById('set-location').value.trim(),
  server_id: document.getElementById('set-server-id').value.trim(),
  tenant: document.getElementById('set-tenant').value.trim(),
  poll_interval: document.getElementById('set-poll').value + 's',
  ws_reconnect_delay: document.getElementById('set-ws-delay').value + 's',
  ws_max_reconnect_delay: document.getElementById('set-ws-max').value + 's',
  ws_ping_interval: document.getElementById('set-ws-ping').value + 's'
 };
 // Only send api_key if user entered a new one
 var newKey = document.getElementById('set-apikey').value.trim();
 if (newKey) cloudCfg.api_key = newKey;

 var body = {
  server: {
   port: parseInt(document.getElementById('set-port').value),
   host: document.getElementById('set-host').value.trim()
  },
  cloud: cloudCfg
 };

 fetch('/api/config', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)})
 .then(function(r){return r.json()}).then(function(data) {
  if (data.success) {
   toast('Settings saved.' + (data.restart_required ? ' Server address changed. Restart the server for this to take effect.' : ''), data.restart_required ? 'warn' : 'success');
   loadSettings(); // Refresh to show updated state
  } else {
   toast(data.error || 'Save failed', 'error');
  }
 }).catch(function(){ toast('Network error', 'error'); });
}

function confirmReRegister() {
 showModal(
  'Re-Register Server',
  '<p>This will disconnect from JetSetGo and require a new registration code. Are you sure?</p>',
  function() {
   fetch('/api/config', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({action:'re-register'})})
   .then(function(r){return r.json()}).then(function(data) {
    closeModal();
    if (data.success) {
     toast('Server unregistered. Redirecting to wizard...', 'success');
     setTimeout(function() {
      window.location.hash = '';
      showWizard();
     }, 1000);
    }
   }).catch(function(){ toast('Network error', 'error'); });
  }
 );
}

// ============ Logs ============
function refreshLogs() {
 var levelParam = logFilter === 'all' ? '' : '?level=' + logFilter;
 fetch('/api/logs' + levelParam).then(function(r){return r.json()}).then(function(data) {
  var viewer = document.getElementById('log-viewer');
  var logs = data.logs || [];
  var html = '';
  // Show newest first
  for (var i = logs.length - 1; i >= 0; i--) {
   var l = logs[i];
   var ts = l.timestamp ? new Date(l.timestamp).toLocaleString() : '';
   var lc = l.level === 'error' ? 'log-error' : (l.level === 'warn' ? 'log-warn' : 'log-info');
   html += '<div class="log-entry"><span class="log-time">[' + esc(ts) + ']</span> <span class="' + lc + '">' + esc((l.level || 'info').toUpperCase()) + '</span> ' + esc(l.message) + '</div>';
  }
  if (logs.length === 0) html = '<div style="color:#555">No log entries</div>';
  viewer.innerHTML = html;
  if (document.getElementById('log-autoscroll').checked) {
   viewer.scrollTop = 0;
  }
 }).catch(function(){});
}

function setLogFilter(filter, btn) {
 logFilter = filter;
 var btns = document.querySelectorAll('.filter-btn');
 for (var i = 0; i < btns.length; i++) btns[i].classList.remove('active');
 btn.classList.add('active');
 refreshLogs();
}

function downloadLogs() {
 var text = document.getElementById('log-viewer').innerText;
 var blob = new Blob([text], {type:'text/plain'});
 var a = document.createElement('a');
 a.href = URL.createObjectURL(blob);
 a.download = 'printserver-logs-' + new Date().toISOString().split('T')[0] + '.txt';
 a.click();
}

// ============ UI Helpers ============
function toast(msg, type) {
 var root = document.getElementById('toast-root');
 var el = document.createElement('div');
 el.className = 'toast toast-' + (type || 'success');
 el.textContent = msg;
 root.appendChild(el);
 setTimeout(function() { el.remove(); }, 4000);
}

function showModal(title, bodyHtml, onConfirm) {
 var root = document.getElementById('modal-root');
 root.innerHTML = '<div class="modal-overlay" onclick="if(event.target===this)closeModal()">' +
  '<div class="modal"><h3>' + title + '</h3>' +
  '<div>' + bodyHtml + '</div>' +
  '<div class="btn-row" style="margin-top:16px;justify-content:flex-end">' +
  '<button class="btn btn-secondary" onclick="closeModal()">Cancel</button>' +
  '<button class="btn btn-primary" id="modal-confirm">Confirm</button>' +
  '</div></div></div>';
 document.getElementById('modal-confirm').onclick = onConfirm;
}

function closeModal() {
 document.getElementById('modal-root').innerHTML = '';
}

function copyText(inputId) {
 var el = document.getElementById(inputId);
 if (navigator.clipboard) {
  navigator.clipboard.writeText(el.value).then(function() { toast('Copied!', 'success'); });
 } else {
  el.select();
  document.execCommand('copy');
  toast('Copied!', 'success');
 }
}

function esc(s) {
 if (!s) return '';
 var d = document.createElement('div');
 d.appendChild(document.createTextNode(String(s)));
 return d.innerHTML;
}

function timeAgo(ts) {
 if (!ts) return '-';
 var d = new Date(ts);
 var secs = Math.floor((Date.now() - d.getTime()) / 1000);
 if (secs < 5) return 'just now';
 if (secs < 60) return secs + 's ago';
 if (secs < 3600) return Math.floor(secs/60) + 'm ago';
 if (secs < 86400) return Math.floor(secs/3600) + 'h ago';
 return Math.floor(secs/86400) + 'd ago';
}

function formatBytes(b) {
 if (!b || b === 0) return '0 B';
 if (b < 1024) return b + ' B';
 if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
 return (b / 1048576).toFixed(1) + ' MB';
}

function parseDuration(s) {
 if (!s) return 0;
 // Parse Go duration like "30s", "1m0s", "500ms"
 var match = String(s).match(/(\d+)/);
 return match ? parseInt(match[1]) : 0;
}

// Auto-generate printer ID from name
document.getElementById('ap-name').addEventListener('input', function() {
 var idField = document.getElementById('ap-id');
 if (!idField.dataset.manual) {
  idField.value = slugify(this.value);
 }
});
document.getElementById('ap-id').addEventListener('input', function() {
 this.dataset.manual = this.value ? 'true' : '';
});

// Registration code - trim whitespace on paste
document.getElementById('reg-code').addEventListener('paste', function(e) {
 var self = this;
 setTimeout(function() { self.value = self.value.trim(); }, 0);
});

// Init
init();
</script>
</body>
</html>`
