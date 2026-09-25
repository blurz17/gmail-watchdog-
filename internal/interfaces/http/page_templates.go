package httpapi

// pageLayoutHTML is the shared layout wrapper for all dashboard pages.
const pageLayoutHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Gmail Watchdog — {{PAGE_TITLE}}</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Inter',sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh;display:flex}

/* Sidebar */
.sidebar{width:220px;background:#1e293b;border-right:1px solid #334155;padding:1.25rem 0;display:flex;flex-direction:column;position:fixed;top:0;left:0;height:100vh;z-index:10}
.sidebar-brand{padding:0 1.25rem 1.25rem;border-bottom:1px solid #334155;display:flex;align-items:center;gap:.5rem;font-weight:700;font-size:1rem}
.sidebar-brand span{font-size:1.4rem}
.sidebar-nav{padding:1rem 0;flex:1}
.sidebar-nav a{display:flex;align-items:center;gap:.6rem;padding:.6rem 1.25rem;color:#94a3b8;text-decoration:none;font-size:.85rem;transition:all .15s;border-left:3px solid transparent}
.sidebar-nav a:hover{color:#e2e8f0;background:#0f172a40}
.sidebar-nav a.active{color:#3b82f6;background:#3b82f620;border-left-color:#3b82f6;font-weight:600}

/* Main */
.main{margin-left:220px;flex:1;padding:2rem;min-height:100vh}
.page-header{margin-bottom:1.5rem}
.page-header h1{font-size:1.4rem;font-weight:700}
.page-header p{color:#64748b;font-size:.85rem;margin-top:.25rem}

/* Stat Cards */
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:1rem;margin-bottom:2rem}
.stat{background:#1e293b;border:1px solid #334155;border-radius:.75rem;padding:1.25rem}
.stat-label{font-size:.75rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;margin-bottom:.35rem}
.stat-value{font-size:1.75rem;font-weight:700}
.stat-value.green{color:#34d399}.stat-value.red{color:#f87171}.stat-value.blue{color:#60a5fa}.stat-value.yellow{color:#fbbf24}

/* Cards */
.card{background:#1e293b;border:1px solid #334155;border-radius:.75rem;overflow:hidden;margin-bottom:1.5rem}
.card-header{padding:1rem 1.25rem;border-bottom:1px solid #334155;font-weight:600;font-size:.95rem}
.card-body{padding:1.25rem}

/* Tables */
table{width:100%;border-collapse:collapse}
th{text-align:left;font-size:.7rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;padding:.6rem 1rem;border-bottom:1px solid #334155}
td{padding:.6rem 1rem;border-bottom:1px solid #1e293b40;font-size:.85rem;vertical-align:middle}
tr:last-child td{border-bottom:none}
tr:hover{background:#0f172a40}

/* Badges */
.badge{display:inline-block;padding:.15rem .5rem;border-radius:.25rem;font-size:.7rem;font-weight:600}
.badge-green{background:#065f4640;color:#34d399}
.badge-red{background:#7f1d1d40;color:#f87171}
.badge-yellow{background:#78350f40;color:#fbbf24}
.badge-blue{background:#1e3a5f40;color:#60a5fa}

/* Buttons */
.btn{padding:.45rem .85rem;border-radius:.4rem;border:none;cursor:pointer;font-size:.8rem;font-weight:600;transition:all .2s;text-decoration:none;display:inline-flex;align-items:center;gap:.3rem}
.btn-primary{background:#3b82f6;color:#fff}.btn-primary:hover{background:#2563eb}
.btn-sm{font-size:.75rem;padding:.3rem .6rem}
.btn-ghost{background:transparent;color:#94a3b8;border:1px solid #334155}.btn-ghost:hover{color:#e2e8f0;border-color:#64748b}

/* Forms */
.filter-bar{display:flex;gap:.5rem;margin-bottom:1.5rem;flex-wrap:wrap;align-items:center}
.filter-bar input,.filter-bar select{padding:.5rem .75rem;border-radius:.4rem;border:1px solid #334155;background:#0f172a;color:#e2e8f0;font-size:.85rem;outline:none}
.filter-bar input:focus,.filter-bar select:focus{border-color:#3b82f6}

/* Charts */
.chart-container{position:relative;height:220px;margin:1rem 0}
.bar-chart{display:flex;align-items:flex-end;gap:4px;height:180px;padding-top:20px}
.bar{flex:1;background:linear-gradient(180deg,#3b82f6,#1d4ed8);border-radius:3px 3px 0 0;position:relative;min-width:8px;transition:all .3s}
.bar:hover{background:linear-gradient(180deg,#60a5fa,#3b82f6)}
.bar:hover::after{content:attr(data-count);position:absolute;top:-22px;left:50%;transform:translateX(-50%);background:#1e293b;border:1px solid #334155;padding:2px 6px;border-radius:4px;font-size:.7rem;white-space:nowrap}
.chart-labels{display:flex;gap:4px;margin-top:4px}
.chart-labels span{flex:1;text-align:center;font-size:.6rem;color:#64748b;min-width:8px;overflow:hidden}

/* Hour grid */
.hour-grid{display:grid;grid-template-columns:repeat(12,1fr);gap:4px}
.hour-cell{aspect-ratio:1;border-radius:4px;display:flex;align-items:center;justify-content:center;font-size:.65rem;font-weight:600;position:relative}
.hour-cell::after{content:attr(data-label);position:absolute;bottom:-14px;font-size:.55rem;color:#64748b}

/* Provider bars */
.provider-bars{display:flex;flex-direction:column;gap:.5rem}
.provider-bar{display:flex;align-items:center;gap:.75rem}
.provider-bar-name{width:100px;font-size:.8rem;text-align:right;flex-shrink:0}
.provider-bar-track{flex:1;height:24px;background:#0f172a;border-radius:4px;overflow:hidden}
.provider-bar-fill{height:100%;background:linear-gradient(90deg,#3b82f6,#8b5cf6);border-radius:4px;display:flex;align-items:center;padding:0 8px;font-size:.7rem;font-weight:600;min-width:30px;transition:width .5s ease}
.provider-bar-count{font-size:.8rem;color:#64748b;width:40px;flex-shrink:0}

/* Timeline */
.timeline{display:flex;flex-direction:column;gap:0}
.timeline-item{display:flex;gap:1rem;padding:.75rem 0;border-bottom:1px solid #1e293b}
.timeline-item:last-child{border-bottom:none}
.timeline-dot{width:8px;height:8px;border-radius:50%;margin-top:6px;flex-shrink:0}
.timeline-content{flex:1}
.timeline-content .type{font-size:.75rem;font-weight:600;text-transform:uppercase;letter-spacing:.03em}
.timeline-content .details{font-size:.85rem;margin-top:.15rem}
.timeline-content .time{font-size:.7rem;color:#64748b;margin-top:.25rem}

/* Pagination */
.pagination{display:flex;align-items:center;justify-content:center;gap:.5rem;margin-top:1.5rem}
.pagination a,.pagination span{padding:.4rem .8rem;border-radius:.4rem;font-size:.8rem;text-decoration:none;color:#94a3b8;border:1px solid #334155}
.pagination a:hover{color:#e2e8f0;border-color:#64748b}
.pagination .current{background:#3b82f6;color:#fff;border-color:#3b82f6}

/* Empty */
.empty{text-align:center;padding:3rem;color:#64748b;font-size:.9rem}
.empty-icon{font-size:2.5rem;margin-bottom:.75rem}

/* Link */
a.email-link{color:#60a5fa;text-decoration:none}
a.email-link:hover{text-decoration:underline}

/* Responsive */
@media(max-width:768px){
  .sidebar{display:none}
  .main{margin-left:0}
  .stats{grid-template-columns:repeat(2,1fr)}
  .hour-grid{grid-template-columns:repeat(6,1fr)}
}
</style>
</head>
<body>
<div class="sidebar">
  <div class="sidebar-brand"><span>🐕</span> Gmail Watchdog</div>
  <nav class="sidebar-nav">
    <a href="/dashboard">📧 Accounts & Providers</a>
    <a href="/dashboard/health">🏥 System Health</a>
    <a href="/dashboard/analytics">📊 Analytics</a>
    <a href="/dashboard/activity">📜 Activity Feed</a>
    <a href="/dashboard/emails">📧 Email Browser</a>
    <a href="/dashboard/docs">📚 Documentation</a>
  </nav>
</div>
<div class="main">
{{PAGE_BODY}}
</div>
</body>
</html>`

// ─── System Health Page ───────────────────────────────────────

const healthPageHTML = `
<div class="page-header">
  <h1>🏥 System Health</h1>
  <p>Overall system status and component health</p>
</div>

<div class="stats">
  <div class="stat">
    <div class="stat-label">Database</div>
    <div class="stat-value {{if .DBHealthy}}green{{else}}red{{end}}">{{if .DBHealthy}}Healthy{{else}}Down{{end}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">DB Latency</div>
    <div class="stat-value blue">{{.DBLatencyMs}}ms</div>
  </div>
  <div class="stat">
    <div class="stat-label">Uptime</div>
    <div class="stat-value green">{{.Uptime}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Total Emails</div>
    <div class="stat-value blue">{{.TotalEmails}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Delivered</div>
    <div class="stat-value green">{{.DeliveredNotifs}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Pending</div>
    <div class="stat-value {{if gt .PendingNotifs 0}}yellow{{else}}green{{end}}">{{.PendingNotifs}}</div>
  </div>
</div>

<div class="card">
  <div class="card-header">Gmail Account Status</div>
  {{if .Accounts}}
  <table>
    <thead><tr><th>Status</th><th>Email</th><th>Last Sync</th></tr></thead>
    <tbody>
    {{range .Accounts}}
    <tr>
      <td>{{statusEmoji .Status}}</td>
      <td>{{.Email}}</td>
      <td>{{timeAgo .LastSyncAt}}</td>
    </tr>
    {{end}}
    </tbody>
  </table>
  {{else}}
  <div class="empty"><div class="empty-icon">📧</div>No accounts connected.</div>
  {{end}}
</div>`

// ─── Analytics Page ───────────────────────────────────────────

const analyticsPageHTML = `
<div class="page-header">
  <h1>📊 Analytics & Insights</h1>
  <p>Email monitoring trends over the last 30 days</p>
</div>

<div class="stats">
  <div class="stat">
    <div class="stat-label">Total Emails</div>
    <div class="stat-value blue">{{.TotalEmails}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Notifications Sent</div>
    <div class="stat-value green">{{.DeliveredNotifs}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Providers</div>
    <div class="stat-value yellow">{{.ProviderCount}}</div>
  </div>
  <div class="stat">
    <div class="stat-label">Accounts</div>
    <div class="stat-value blue">{{.AccountCount}}</div>
  </div>
</div>

<div class="card">
  <div class="card-header">📈 Emails per Day (Last 30 Days)</div>
  <div class="card-body">
    <div class="bar-chart" id="dailyChart"></div>
    <div class="chart-labels" id="dailyLabels"></div>
  </div>
</div>

<div style="display:grid;grid-template-columns:1fr 1fr;gap:1.5rem">
  <div class="card">
    <div class="card-header">🏢 By Provider</div>
    <div class="card-body">
      <div class="provider-bars" id="providerChart"></div>
    </div>
  </div>
  <div class="card">
    <div class="card-header">🕐 Peak Hours (UTC)</div>
    <div class="card-body">
      <div class="hour-grid" id="hourChart"></div>
    </div>
  </div>
</div>

<script>
(function(){
  const daily = {{.DailyJSON}};
  const providers = {{.ProviderJSON}};
  const hours = {{.HourJSON}};

  // Daily bar chart
  const chartEl = document.getElementById('dailyChart');
  const labelsEl = document.getElementById('dailyLabels');
  if(daily && daily.length > 0) {
    const max = Math.max(...daily.map(d => d.Count));
    daily.forEach(d => {
      const pct = max > 0 ? (d.Count / max * 100) : 0;
      const bar = document.createElement('div');
      bar.className = 'bar';
      bar.style.height = Math.max(pct, 4) + '%';
      bar.setAttribute('data-count', d.Count + ' emails');
      chartEl.appendChild(bar);
      const lbl = document.createElement('span');
      lbl.textContent = new Date(d.Date).getDate();
      labelsEl.appendChild(lbl);
    });
  } else {
    chartEl.innerHTML = '<div class="empty" style="width:100%"><div class="empty-icon">📊</div>No data yet</div>';
  }

  // Provider bars
  const provEl = document.getElementById('providerChart');
  if(providers && providers.length > 0) {
    const maxP = Math.max(...providers.map(p => p.Count));
    providers.forEach(p => {
      const pct = maxP > 0 ? (p.Count / maxP * 100) : 0;
      const row = document.createElement('div');
      row.className = 'provider-bar';
      row.innerHTML = '<div class="provider-bar-name">' + p.ProviderName + '</div>' +
        '<div class="provider-bar-track"><div class="provider-bar-fill" style="width:' + Math.max(pct,5) + '%">' + p.Count + '</div></div>';
      provEl.appendChild(row);
    });
  } else {
    provEl.innerHTML = '<div class="empty"><div class="empty-icon">🏢</div>No data yet</div>';
  }

  // Hour heatmap
  const hourEl = document.getElementById('hourChart');
  const hourMap = {};
  if(hours) hours.forEach(h => { hourMap[h.Hour] = h.Count; });
  const maxH = Math.max(...Object.values(hourMap), 1);
  for(let i = 0; i < 24; i++) {
    const cnt = hourMap[i] || 0;
    const intensity = maxH > 0 ? cnt / maxH : 0;
    const cell = document.createElement('div');
    cell.className = 'hour-cell';
    const r = Math.round(15 + intensity * 44);
    const g = Math.round(23 + intensity * 55);
    const b = Math.round(42 + intensity * 204);
    cell.style.background = 'rgb(' + r + ',' + g + ',' + b + ')';
    cell.textContent = cnt > 0 ? cnt : '';
    cell.setAttribute('data-label', i + ':00');
    hourEl.appendChild(cell);
  }
})();
</script>`

// ─── Activity Feed Page ───────────────────────────────────────

const activityPageHTML = `
<div class="page-header">
  <h1>📜 Activity Feed</h1>
  <p>Recent system events and operations</p>
</div>

<div class="card">
  {{if .Events}}
  <div class="card-body">
    <div class="timeline">
      {{range .Events}}
      <div class="timeline-item">
        <div class="timeline-dot" style="background:{{severityColor .Severity}}"></div>
        <div class="timeline-content">
          <div class="type" style="color:{{severityColor .Severity}}">{{.Type}}</div>
          <div class="details">{{.Details}}</div>
          <div class="time">{{fmtTime .CreatedAt}}</div>
        </div>
      </div>
      {{end}}
    </div>
  </div>
  {{else}}
  <div class="empty">
    <div class="empty-icon">📜</div>
    No system events recorded yet. Events will appear here as the service runs.
  </div>
  {{end}}
</div>`

// ─── Email Browser Page ───────────────────────────────────────

const emailBrowserPageHTML = `
<div class="page-header">
  <h1>📧 Email Browser</h1>
  <p>{{.Total}} emails captured</p>
</div>

<form class="filter-bar" method="get" action="/dashboard/emails">
  <input type="text" name="search" placeholder="Search subject or sender..." value="{{.Search}}" style="flex:1;min-width:200px">
  <select name="provider">
    <option value="">All Providers</option>
    {{range .Providers}}
    <option value="{{.ID}}" {{if eq $.ProviderID (.ID.String)}}selected{{end}}>{{.Name}}</option>
    {{end}}
  </select>
  <select name="account">
    <option value="">All Accounts</option>
    {{range .Accounts}}
    <option value="{{.ID}}" {{if eq $.AccountID (.ID.String)}}selected{{end}}>{{.Email}}</option>
    {{end}}
  </select>
  <button class="btn btn-primary" type="submit">Filter</button>
</form>

<div class="card">
  {{if .Emails}}
  <table>
    <thead><tr><th>Provider</th><th>From</th><th>Subject</th><th>Account</th><th>Received</th><th></th></tr></thead>
    <tbody>
    {{range .Emails}}
    <tr>
      <td><span class="badge badge-blue">{{.ProviderName}}</span></td>
      <td>{{truncate .From 30}}</td>
      <td>{{truncate .Subject 50}}</td>
      <td style="color:#64748b;font-size:.8rem">{{.AccountEmail}}</td>
      <td style="font-size:.8rem">{{fmtTime .ReceivedAt}}</td>
      <td><a class="email-link" href="{{gmailURL .GmailMessageID}}" target="_blank">Open ↗</a></td>
    </tr>
    {{end}}
    </tbody>
  </table>
  {{else}}
  <div class="empty">
    <div class="empty-icon">📧</div>
    No emails found matching your criteria.
  </div>
  {{end}}
</div>

{{if gt .TotalPages 1}}
<div class="pagination">
  {{if .HasPrev}}<a href="/dashboard/emails?page={{sub .Page 1}}&search={{.Search}}&provider={{.ProviderID}}&account={{.AccountID}}">← Prev</a>{{end}}
  <span class="current">{{.Page}} / {{.TotalPages}}</span>
  {{if .HasNext}}<a href="/dashboard/emails?page={{add .Page 1}}&search={{.Search}}&provider={{.ProviderID}}&account={{.AccountID}}">Next →</a>{{end}}
</div>
{{end}}`

// ─── Documentation Page ───────────────────────────────────────

const docsPageHTML = `
<style>
@import url('https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700&display=swap');
.docs-content { line-height: 1.6; font-size: 0.95rem; color: #cbd5e1; }
.docs-content h2 { font-size: 1.25rem; color: #f8fafc; margin: 1.5rem 0 0.75rem; border-bottom: 1px solid #334155; padding-bottom: 0.5rem; }
.docs-content h3 { font-size: 1.1rem; color: #e2e8f0; margin: 1.25rem 0 0.5rem; }
.docs-content p { margin-bottom: 1rem; }
.docs-content ul, .docs-content ol { margin: 0.5rem 0 1rem 1.5rem; }
.docs-content li { margin-bottom: 0.25rem; }
.docs-content code { background: #0f172a; padding: 0.2rem 0.4rem; border-radius: 0.25rem; font-family: monospace; font-size: 0.85rem; color: #60a5fa; border: 1px solid #334155; }
.docs-content pre { background: #0f172a; padding: 1rem; border-radius: 0.5rem; overflow-x: auto; border: 1px solid #334155; margin-bottom: 1rem; }
.docs-content pre code { background: transparent; padding: 0; border: none; color: #e2e8f0; }
.note-box { background: #1e3a8a30; border-left: 4px solid #3b82f6; padding: 1rem; margin: 1.5rem 0; border-radius: 0 0.5rem 0.5rem 0; }
.note-box strong { color: #60a5fa; }
.lang-toggle { display: flex; gap: 0.5rem; margin-bottom: 1rem; justify-content: flex-end; }
.lang-btn { padding: 0.4rem 0.8rem; background: #1e293b; border: 1px solid #334155; color: #94a3b8; border-radius: 0.4rem; cursor: pointer; font-size: 0.85rem; font-weight:600; }
.lang-btn.active { background: #3b82f6; color: white; border-color: #3b82f6; }
.ar-text { direction: rtl; text-align: right; font-family: 'Cairo', sans-serif; display: none; }
.ar-text ol, .ar-text ul { margin: 0.5rem 1.5rem 1rem 0; }
</style>

<div class="page-header" style="display:flex; justify-content:space-between; align-items:center;">
  <div>
    <h1 class="en-text">📚 Documentation</h1>
    <h1 class="ar-text">📚 التوثيق والتعليمات</h1>
    <p class="en-text">System overview and detailed setup instructions</p>
    <p class="ar-text">نظرة عامة على النظام وتعليمات الإعداد المفصلة</p>
  </div>
  <div class="lang-toggle">
    <button class="lang-btn active" onclick="setLang('en')">English</button>
    <button class="lang-btn" onclick="setLang('ar')" style="font-family:'Cairo',sans-serif;">العربية</button>
  </div>
</div>

<div class="card">
  <div class="card-body docs-content">
    
    <!-- ENGLISH CONTENT -->
    <div class="en-text">
      <h2>1. What is Gmail Watchdog?</h2>
      <p>Gmail Watchdog is a self-hosted monitoring tool designed to instantly alert you via Telegram when specific emails arrive in your Gmail inbox. It uses the official Gmail API to read metadata without compromising your password.</p>
      
      <div class="note-box">
        <strong>The Problem:</strong> When signing up for various web services, we often use alias emails or filters to keep our primary inbox clean. However, this means missing critical notifications (like password resets, server alerts, or billing failures).
        <br><br>
        <strong>The Solution:</strong> Gmail Watchdog connects securely via OAuth, watches for specific senders (like <code>noreply@github.com</code> or <code>@stripe.com</code>), and forwards those alerts directly to your Telegram, bypassing your noisy inbox.
      </div>

      <h2>2. Configuration (.env)</h2>
      <p>The system requires a <code>.env</code> file at the root of the project to operate. Here are the required variables:</p>
      <pre><code># Web Server & Dashboard
HTTP_PORT=8090
HTTP_API_KEY=your_secure_password_for_dashboard

# PostgreSQL Database (SSL mode disable is usually required for local dev)
DB_DSN=postgres://user:password@host:5432/dbname?sslmode=disable

# Google Cloud OAuth Credentials
GMAIL_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-google-client-secret

# Telegram Bot Setup
TELEGRAM_BOT_TOKEN=123456789:YOUR_TELEGRAM_BOT_TOKEN
TELEGRAM_CHAT_ID=123456789</code></pre>

      <h2>3. Setting up Google Cloud (OAuth API)</h2>
      <p>To allow the system to read your Gmail, you must create an OAuth application in Google Cloud. Follow these exact steps:</p>
      <ol>
        <li>Go to the <a href="https://console.cloud.google.com/" target="_blank" class="email-link">Google Cloud Console</a> and sign in.</li>
        <li>Click the dropdown at the top left and select <strong>New Project</strong>. Name it "Gmail Watchdog" and click Create.</li>
        <li>In the search bar, type "Gmail API", click on it, and hit <strong>Enable</strong>.</li>
        <li>Go to the left menu: <strong>APIs & Services &gt; OAuth consent screen</strong>.</li>
        <li>Select <strong>External</strong> user type and click Create.</li>
        <li>Fill in the mandatory fields (App name: "Watchdog", User support email, and Developer contact email). Click Save and Continue.</li>
        <li>On the <strong>Scopes</strong> page, click "Add or Remove Scopes". Search for and add: <code>https://www.googleapis.com/auth/gmail.readonly</code>. Save and Continue.</li>
        <li>On the <strong>Test Users</strong> page, click "Add Users" and type in your exact Gmail address. If you skip this, OAuth will fail! Save and Continue.</li>
        <li>Now go to <strong>Credentials</strong> (left menu) &gt; <strong>Create Credentials</strong> &gt; <strong>OAuth client ID</strong>.</li>
        <li>Application type: <strong>Web application</strong>.</li>
        <li>Under <strong>Authorized redirect URIs</strong>, click "Add URI". If running locally, enter <code>http://localhost:8090/dashboard/oauth/callback</code>. If deployed (like Heroku), enter your full URL, e.g., <code>https://your-app.herokuapp.com/dashboard/oauth/callback</code>.</li>
        <li>Click Create. Copy the <strong>Client ID</strong> and <strong>Client Secret</strong> into your <code>.env</code> file.</li>
      </ol>

      <h2>4. Setting up Telegram Bot</h2>
      <p>The system needs a bot to send you the alerts.</p>
      <ol>
        <li>Open Telegram and search for <strong>@BotFather</strong> (the official verified bot).</li>
        <li>Send the message <code>/newbot</code> and follow the prompts to give your bot a name and a username.</li>
        <li>BotFather will reply with an <strong>HTTP API Token</strong>. Copy this exactly into <code>TELEGRAM_BOT_TOKEN</code>.</li>
        <li><strong>CRITICAL:</strong> You must start a conversation with your new bot. Search for your bot's username in Telegram and click <strong>Start</strong>.</li>
        <li>To get your Chat ID, search for <strong>@userinfobot</strong> or <strong>@getmyid_bot</strong> in Telegram and click Start. It will reply with your ID (a string of numbers like <code>123456789</code>).</li>
        <li>Put that number into <code>TELEGRAM_CHAT_ID</code>.</li>
      </ol>

      <h2>5. Setting up PostgreSQL</h2>
      <p>The system requires a Postgres database to store rules, logs, and account tokens.</p>
      <ul>
        <li><strong>Local Development:</strong> You can run Postgres locally via Docker: <br><code>docker run --name watchdog-db -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres</code><br>Your DSN will be: <code>postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable</code></li>
        <li><strong>Production (Heroku):</strong> Go to the Resources tab in your Heroku dashboard, search for "Heroku Postgres", and attach the free/basic tier. Heroku will automatically inject a <code>DATABASE_URL</code> environment variable, which the app will use automatically if <code>DB_DSN</code> is not set.</li>
      </ul>
    </div>

    <!-- ARABIC CONTENT -->
    <div class="ar-text">
      <h2>١. ما هو Gmail Watchdog؟</h2>
      <p>أداة مراقبة ذاتية الاستضافة مصممة لتنبيهك فوراً عبر تيليجرام عند وصول رسائل بريد إلكتروني محددة إلى صندوق بريد جي ميل الخاص بك. يستخدم النظام واجهة برمجة تطبيقات جي ميل الرسمية لقراءة البيانات الوصفية دون المساس بكلمة مرورك.</p>
      
      <div class="note-box" style="border-left:none; border-right:4px solid #3b82f6; border-radius:0.5rem 0 0 0.5rem;">
        <strong>المشكلة:</strong> عند التسجيل في خدمات الويب المختلفة، غالباً ما نستخدم أسماء مستعارة أو فلاتر للحفاظ على نظافة صندوق البريد الرئيسي. لكن هذا يعني تفويت الإشعارات الحرجة (مثل إعادة تعيين كلمة المرور، تنبيهات الخوادم، أو فشل الدفع).
        <br><br>
        <strong>الحل:</strong> يتصل النظام بشكل آمن عبر OAuth، ويراقب مرسلين محددين (مثل <code>noreply@github.com</code>)، ثم يقوم بتوجيه هذه التنبيهات مباشرة إلى حسابك في تيليجرام، متجاوزاً صندوق البريد المزدحم.
      </div>

      <h2>٢. الإعدادات (ملف .env)</h2>
      <p>يتطلب النظام ملف <code>.env</code> في المجلد الرئيسي للمشروع. هذه هي المتغيرات المطلوبة:</p>
      <pre style="direction:ltr; text-align:left;"><code># خادم الويب ولوحة التحكم
HTTP_PORT=8090
HTTP_API_KEY=كلمة_مرور_قوية_للدخول_للوحة_التحكم

# قاعدة بيانات PostgreSQL
DB_DSN=postgres://user:password@host:5432/dbname?sslmode=disable

# بيانات اعتماد Google Cloud OAuth
GMAIL_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-google-client-secret

# إعدادات بوت تيليجرام
TELEGRAM_BOT_TOKEN=123456789:YOUR_TELEGRAM_BOT_TOKEN
TELEGRAM_CHAT_ID=123456789</code></pre>

      <h2>٣. إعداد Google Cloud (OAuth)</h2>
      <p>للسماح للنظام بقراءة بريدك، يجب إنشاء تطبيق OAuth في Google Cloud. اتبع هذه الخطوات بدقة:</p>
      <ol>
        <li>اذهب إلى <a href="https://console.cloud.google.com/" target="_blank" class="email-link">منصة Google Cloud</a> وقم بتسجيل الدخول.</li>
        <li>انقر على القائمة المنسدلة في الأعلى واختر <strong>New Project</strong>. سمه "Gmail Watchdog" واضغط Create.</li>
        <li>في شريط البحث، اكتب "Gmail API"، انقر عليها، ثم اضغط <strong>Enable</strong> لتفعيلها.</li>
        <li>من القائمة الجانبية، اذهب إلى <strong>APIs & Services &gt; OAuth consent screen</strong>.</li>
        <li>اختر نوع المستخدم <strong>External</strong> واضغط Create.</li>
        <li>املأ الحقول الإجبارية (اسم التطبيق، والبريد الإلكتروني للدعم). اضغط Save and Continue.</li>
        <li>في صفحة <strong>Scopes</strong>، انقر على "Add or Remove Scopes". ابحث عن <code>https://www.googleapis.com/auth/gmail.readonly</code> وأضفه.</li>
        <li>في صفحة <strong>Test Users</strong>، اضغط "Add Users" واكتب بريدك الإلكتروني (جي ميل) <strong>بدقة</strong>. إذا تخطيت هذه الخطوة سيفشل تسجيل الدخول!</li>
        <li>الآن اذهب إلى <strong>Credentials</strong> (القائمة الجانبية) &gt; <strong>Create Credentials</strong> &gt; <strong>OAuth client ID</strong>.</li>
        <li>نوع التطبيق (Application type): <strong>Web application</strong>.</li>
        <li>تحت <strong>Authorized redirect URIs</strong>، أضف الرابط التالي. إذا كنت تعمل محلياً: <code>http://localhost:8090/dashboard/oauth/callback</code>. إذا كان المشروع مرفوعاً (مثل Heroku)، ضع الرابط الكامل مثل <code>https://your-app.herokuapp.com/dashboard/oauth/callback</code>.</li>
        <li>اضغط Create. انسخ <strong>Client ID</strong> و <strong>Client Secret</strong> إلى ملف <code>.env</code>.</li>
      </ol>

      <h2>٤. إعداد بوت تيليجرام</h2>
      <p>يحتاج النظام إلى بوت لإرسال التنبيهات إليك.</p>
      <ol>
        <li>افتح تطبيق تيليجرام وابحث عن <strong>@BotFather</strong> (البوت الرسمي الموثق).</li>
        <li>أرسل رسالة <code>/newbot</code> واتبع التعليمات لاختيار اسم واسم مستخدم للبوت.</li>
        <li>سيرد عليك BotFather بـ <strong>HTTP API Token</strong>. انسخه بدقة إلى <code>TELEGRAM_BOT_TOKEN</code>.</li>
        <li><strong>هام جداً:</strong> يجب أن تبدأ محادثة مع البوت الخاص بك. ابحث عن اسم مستخدم البوت في تيليجرام واضغط <strong>Start</strong>.</li>
        <li>للحصول على معرف المحادثة (Chat ID) الخاص بك، ابحث عن <strong>@userinfobot</strong> في تيليجرام واضغط Start. سيرد عليك برقمك (مثل <code>123456789</code>).</li>
        <li>ضع هذا الرقم في <code>TELEGRAM_CHAT_ID</code>.</li>
      </ol>

      <h2>٥. إعداد PostgreSQL</h2>
      <p>يحتاج النظام إلى قاعدة بيانات لتخزين القواعد والسجلات ورموز الوصول.</p>
      <ul>
        <li><strong>محلياً للتشغيل والتطوير:</strong> يمكنك تشغيل Postgres عبر Docker باستخدام الأمر:<br><code style="direction:ltr;display:inline-block;">docker run --name watchdog-db -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres</code><br>سيكون الرابط: <code style="direction:ltr;display:inline-block;">postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable</code></li>
        <li><strong>في الإنتاج (Heroku):</strong> اذهب إلى قسم Resources، ابحث عن "Heroku Postgres" وأضف الباقة المجانية/الأساسية. سيقوم Heroku تلقائياً بإضافة متغير <code>DATABASE_URL</code> والذي سيستخدمه النظام تلقائياً.</li>
      </ul>
    </div>
  </div>
</div>

<script>
function setLang(lang) {
  const enEls = document.querySelectorAll('.en-text');
  const arEls = document.querySelectorAll('.ar-text');
  const btns = document.querySelectorAll('.lang-btn');
  
  btns.forEach(b => b.classList.remove('active'));
  
  if(lang === 'ar') {
    enEls.forEach(el => el.style.display = 'none');
    arEls.forEach(el => el.style.display = 'block');
    btns[1].classList.add('active');
  } else {
    arEls.forEach(el => el.style.display = 'none');
    enEls.forEach(el => el.style.display = 'block');
    btns[0].classList.add('active');
  }
}
</script>
`
