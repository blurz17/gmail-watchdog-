package httpapi

const loginHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Gmail Watchdog — Login</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'Inter',sans-serif;background:#0f172a;color:#e2e8f0;display:flex;justify-content:center;align-items:center;min-height:100vh}
.card{background:#1e293b;padding:2.5rem;border-radius:1rem;text-align:center;max-width:400px;width:90%;box-shadow:0 25px 50px -12px rgba(0,0,0,.5)}
h2{margin-bottom:.5rem;font-size:1.5rem}
p{color:#94a3b8;margin-bottom:1.5rem;font-size:.9rem}
input{padding:.75rem 1rem;border-radius:.5rem;border:1px solid #334155;background:#0f172a;color:#e2e8f0;width:100%;font-size:1rem;outline:none;transition:border-color .2s}
input:focus{border-color:#3b82f6}
button{padding:.75rem 1.5rem;border-radius:.5rem;border:none;background:#3b82f6;color:#fff;cursor:pointer;font-size:1rem;width:100%;margin-top:.75rem;font-weight:600;transition:background .2s}
button:hover{background:#2563eb}
.logo{font-size:2.5rem;margin-bottom:1rem}
</style></head><body>
<div class="card">
<div class="logo">🐕</div>
<h2>Gmail Watchdog</h2>
<p>Enter your API key to access the dashboard.</p>
<form method="get" action="/dashboard">
<input type="password" name="key" placeholder="API Key" required autocomplete="off">
<button type="submit">Sign In</button>
</form>
</div></body></html>`

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Gmail Watchdog — Dashboard</title>
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

/* Flash */
.flash{padding:.75rem 1rem;border-radius:.5rem;margin-bottom:1.5rem;font-size:.9rem;animation:fadeIn .3s}
.flash.success{background:#065f4620;border:1px solid #065f46;color:#34d399}
.flash.error{background:#7f1d1d20;border:1px solid #7f1d1d;color:#f87171}
@keyframes fadeIn{from{opacity:0;transform:translateY(-10px)}to{opacity:1;transform:translateY(0)}}

/* Section */
.section{margin-bottom:2rem}
.section-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:1rem}
.section-header h2{font-size:1.1rem;font-weight:600}

/* Cards */
.card{background:#1e293b;border:1px solid #334155;border-radius:.75rem;overflow:hidden}
.card-body{padding:1.25rem}

/* Tables */
table{width:100%;border-collapse:collapse}
th{text-align:left;font-size:.75rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;padding:.75rem 1rem;border-bottom:1px solid #334155}
td{padding:.75rem 1rem;border-bottom:1px solid #1e293b;font-size:.9rem;vertical-align:middle}
tr:last-child td{border-bottom:none}
tr:hover{background:#1e293b80}

/* Forms */
.form-row{display:flex;gap:.5rem;margin-top:1rem}
.form-row input{flex:1;padding:.6rem .75rem;border-radius:.5rem;border:1px solid #334155;background:#0f172a;color:#e2e8f0;font-size:.9rem;outline:none}
.form-row input:focus{border-color:#3b82f6}

/* Buttons */
.btn{padding:.5rem 1rem;border-radius:.5rem;border:none;cursor:pointer;font-size:.85rem;font-weight:600;transition:all .2s;text-decoration:none;display:inline-flex;align-items:center;gap:.35rem}
.btn-primary{background:#3b82f6;color:#fff}.btn-primary:hover{background:#2563eb}
.btn-success{background:#059669;color:#fff}.btn-success:hover{background:#047857}
.btn-danger{background:transparent;color:#f87171;border:1px solid #7f1d1d;font-size:.75rem;padding:.3rem .6rem}
.btn-danger:hover{background:#7f1d1d40}
.btn-sm{font-size:.75rem;padding:.3rem .6rem}

/* Rule tags */
.rule-tag{display:inline-flex;align-items:center;gap:.35rem;background:#0f172a;border:1px solid #334155;padding:.25rem .6rem;border-radius:2rem;font-size:.8rem;margin:.2rem}
.rule-tag form{display:inline}
.rule-tag button{background:none;border:none;color:#64748b;cursor:pointer;font-size:.7rem;padding:0}
.rule-tag button:hover{color:#f87171}

/* Empty state */
.empty{text-align:center;padding:2rem;color:#64748b;font-size:.9rem}

/* Add rule inline */
.add-rule-form{display:inline-flex;gap:.35rem;margin-top:.5rem}
.add-rule-form input{padding:.3rem .5rem;border-radius:.35rem;border:1px solid #334155;background:#0f172a;color:#e2e8f0;font-size:.8rem;width:180px}
.add-rule-form button{font-size:.75rem;padding:.3rem .6rem}

/* Responsive */
@media(max-width:768px){
  .sidebar{display:none}
  .main{margin-left:0}
  .form-row{flex-direction:column}
  .section-header{flex-direction:column;gap:.5rem;align-items:flex-start}
}
</style>
</head>
<body>

<div class="sidebar">
  <div class="sidebar-brand"><span>🐕</span> Gmail Watchdog</div>
  <nav class="sidebar-nav">
    <a href="/dashboard" class="active">📧 Accounts & Providers</a>
    <a href="/dashboard/health">🏥 System Health</a>
    <a href="/dashboard/analytics">📊 Analytics</a>
    <a href="/dashboard/activity">📜 Activity Feed</a>
    <a href="/dashboard/emails">📧 Email Browser</a>
  </nav>
</div>

<div class="main">

{{if .Flash}}
<div class="flash {{.FlashType}}">{{.Flash}}</div>
{{end}}

<!-- Gmail Accounts -->
<div class="section">
  <div class="section-header">
    <h2>📧 Gmail Accounts</h2>
  </div>
  <div class="card">
    {{if .Accounts}}
    <table>
      <thead><tr><th>Status</th><th>Email</th><th>Last Sync</th><th></th></tr></thead>
      <tbody>
      {{range .Accounts}}
      <tr>
        <td>{{statusEmoji .Status}}</td>
        <td>{{.Email}}</td>
        <td>{{timeAgo .LastSyncAt}}</td>
        <td>
          <form method="post" action="/dashboard/accounts/delete" style="display:inline" onsubmit="return confirm('Remove this account?')">
            <input type="hidden" name="id" value="{{.ID}}">
            <button class="btn btn-danger btn-sm">Remove</button>
          </form>
        </td>
      </tr>
      {{end}}
      </tbody>
    </table>
    {{else}}
    <div class="empty">No Gmail accounts connected yet.</div>
    {{end}}
    <div class="card-body">
      <form class="form-row" action="/dashboard/oauth/start" method="get">
        <input type="email" name="email" placeholder="account@gmail.com" required>
        <button class="btn btn-success" type="submit">+ Connect Gmail Account</button>
      </form>
    </div>
  </div>
</div>

<!-- Providers -->
<div class="section">
  <div class="section-header">
    <h2>🏢 Providers & Sender Rules</h2>
  </div>

  {{if .Providers}}
  {{range .Providers}}
  <div class="card" style="margin-bottom:.75rem">
    <div class="card-body">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:.5rem">
        <strong>{{.Provider.Name}}</strong>
        <form method="post" action="/dashboard/providers/delete" style="display:inline" onsubmit="return confirm('Delete provider {{.Provider.Name}} and all its rules?')">
          <input type="hidden" name="id" value="{{.Provider.ID}}">
          <button class="btn btn-danger btn-sm">Delete</button>
        </form>
      </div>

      <div>
        {{range .Rules}}
        <span class="rule-tag">
          {{ruleDisplay .}}
          <form method="post" action="/dashboard/rules/delete">
            <input type="hidden" name="id" value="{{.ID}}">
            <button title="Remove rule">✕</button>
          </form>
        </span>
        {{end}}
      </div>

      <form class="add-rule-form" method="post" action="/dashboard/rules">
        <input type="hidden" name="provider_id" value="{{.Provider.ID}}">
        <input type="text" name="value" placeholder="user@example.com or @domain.com" required>
        <button class="btn btn-primary btn-sm" type="submit">+ Add Rule</button>
      </form>
    </div>
  </div>
  {{end}}
  {{else}}
  <div class="card">
    <div class="empty">No providers configured yet. Add one below.</div>
  </div>
  {{end}}

  <div class="card" style="margin-top:.75rem">
    <div class="card-body">
      <form class="form-row" method="post" action="/dashboard/providers">
        <input type="text" name="name" placeholder="Provider name (e.g. Heroku)" required>
        <button class="btn btn-primary" type="submit">+ Add Provider</button>
      </form>
    </div>
  </div>
</div>

</div>
</body>
</html>`
