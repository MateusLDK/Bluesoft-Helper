package main

var htmlUI = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<title>Bluesoft Uploader</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
<style>
  :root {
    --bg:           #0a0d12;
    --bg-2:         #0f131a;
    --bg-3:         #161b24;
    --border:        #222936;
    --border-strong: #2c3441;
    --text:       #e6e9ef;
    --text-dim:   #9ba3b3;
    --text-faint: #5e6675;
    --accent:        #5ac8fa;
    --accent-bg:     rgba(90, 200, 250, 0.10);
    --accent-border: rgba(90, 200, 250, 0.35);
    --green:    #34d399;
    --green-bg: rgba(52, 211, 153, 0.12);
    --amber:    #fbbf24;
    --amber-bg: rgba(251, 191, 36, 0.12);
    --red:      #f87171;
    --red-bg:   rgba(248, 113, 113, 0.12);
    --purple:   #a78bfa;
    --font-ui:   'Inter', -apple-system, system-ui, sans-serif;
    --font-mono: 'JetBrains Mono', 'SF Mono', Menlo, monospace;
    --r-sm: 4px; --r-md: 6px; --r-lg: 8px; --r-xl: 10px; --r-2xl: 12px;
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }

  html, body {
    height: 100%;
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-ui);
    -webkit-font-smoothing: antialiased;
  }
  body { display: flex; flex-direction: column; overflow: hidden; }
  .mono { font-family: var(--font-mono); }

  ::-webkit-scrollbar { width: 8px; height: 8px; }
  ::-webkit-scrollbar-thumb { background: var(--border-strong); border-radius: 99px; }
  ::-webkit-scrollbar-track { background: transparent; }

  /* ── Header ───────────────────────────────────────────── */
  header {
    flex-shrink: 0;
    height: 56px;
    padding: 0 24px;
    display: flex;
    align-items: center;
    gap: 12px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-2);
  }
  .brand {
    display: flex; align-items: center; gap: 10px;
    font-weight: 600;
  }
  .brand .logo {
    width: 28px; height: 28px;
    border-radius: 7px;
    background: linear-gradient(135deg, var(--accent), var(--purple));
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-size: 13px;
  }
  .brand .name { font-size: 14px; }
  .brand .sub { font-size: 11px; color: var(--text-dim); margin-left: 6px; font-weight: 400; }

  .header-right { margin-left: auto; display: flex; gap: 8px; align-items: center; }

  .pill {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 5px 11px;
    border: 1px solid var(--border-strong);
    border-radius: 99px;
    font-size: 12px;
    color: var(--text-dim);
    background: var(--bg-3);
    cursor: default;
    transition: border-color .15s, color .15s, background .15s;
  }
  .pill .pill-label { color: var(--text-dim); }
  .pill .pill-value { color: var(--text); }
  .pill.tenant { cursor: pointer; }
  .pill.tenant:hover { border-color: var(--accent-border); }
  .pill.tenant .pill-value { font-family: var(--font-mono); font-weight: 500; }
  .pill .caret { color: var(--text-faint); margin-left: 2px; font-size: 10px; }

  .pill.status::before {
    content: ''; width: 7px; height: 7px; border-radius: 50%;
    background: var(--text-faint);
    box-shadow: 0 0 8px transparent;
    transition: background .15s, box-shadow .15s;
  }
  .pill.status.aguardando::before { background: var(--text-faint); }
  .pill.status.pronto::before    { background: var(--green); box-shadow: 0 0 8px var(--green); }
  .pill.status.enviando::before  { background: var(--accent); box-shadow: 0 0 8px var(--accent); animation: pulse 1.4s ease-in-out infinite; }
  .pill.status.concluido::before { background: var(--green); box-shadow: 0 0 8px var(--green); }
  .pill.status.erro::before      { background: var(--red); box-shadow: 0 0 8px var(--red); }
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: .4; }
  }

  /* ── Main / screens ───────────────────────────────────── */
  main {
    flex: 1;
    overflow: auto;
    position: relative;
  }
  .screen {
    display: none;
    min-height: 100%;
    padding: 32px 24px 40px;
  }
  .screen.active { display: block; }

  /* ── Botões base ──────────────────────────────────────── */
  .btn {
    display: inline-flex; align-items: center; justify-content: center; gap: 8px;
    padding: 10px 18px;
    border-radius: var(--r-lg);
    font-family: var(--font-ui);
    font-size: 13px; font-weight: 600;
    border: 1px solid transparent;
    cursor: pointer;
    transition: all .15s ease;
    white-space: nowrap;
  }
  .btn-primary {
    background: var(--accent);
    color: #0a0d12;
  }
  .btn-primary:hover:not(:disabled) { background: #7dd3fc; }
  .btn-primary:disabled { opacity: .35; cursor: not-allowed; }

  .btn-ghost {
    background: transparent;
    border-color: var(--border-strong);
    color: var(--text);
  }
  .btn-ghost:hover:not(:disabled) { border-color: var(--accent-border); color: var(--accent); }

  .btn-danger {
    background: var(--red);
    color: #0a0d12;
  }
  .btn-danger:hover:not(:disabled) { background: #fca5a5; }

  .btn-link {
    background: transparent;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    font-family: var(--font-ui);
    font-size: 12px;
    padding: 4px 8px;
    transition: color .15s;
  }
  .btn-link:hover { color: var(--accent); }

  /* ── Inputs ───────────────────────────────────────────── */
  .field { display: flex; flex-direction: column; gap: 6px; }
  .field-row {
    display: flex; align-items: baseline; justify-content: space-between; gap: 12px;
  }
  .field-label { font-size: 12px; font-weight: 500; color: var(--text); }
  .field-hint { font-size: 11px; color: var(--text-dim); }
  .input {
    width: 100%;
    background: var(--bg-3);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-lg);
    padding: 10px 12px;
    font-family: var(--font-mono);
    font-size: 13px;
    color: var(--text);
    transition: border-color .15s, box-shadow .15s;
  }
  .input::placeholder { color: var(--text-faint); }
  .input:focus {
    outline: none;
    border-color: var(--accent-border);
    box-shadow: 0 0 0 3px var(--accent-bg);
  }
  .input-group { position: relative; }
  .input-group .input { padding-right: 64px; }
  .input-group .toggle {
    position: absolute; right: 10px; top: 50%; transform: translateY(-50%);
    background: transparent; border: none;
    font-size: 11px; color: var(--text-dim);
    cursor: pointer; padding: 4px 6px;
  }
  .input-group .toggle:hover { color: var(--accent); }

  /* ── Cards ────────────────────────────────────────────── */
  .card {
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--r-2xl);
  }

  /* ─────────────────────────────────────────────────────────
     0. Setup
     ───────────────────────────────────────────────────────── */
  .setup-wrap {
    max-width: 640px;
    margin: 32px auto;
    display: flex; flex-direction: column; gap: 24px;
  }
  .badge {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 4px 10px;
    border-radius: 99px;
    font-size: 10px; font-weight: 600;
    letter-spacing: 1px; text-transform: uppercase;
    background: var(--accent-bg);
    color: var(--accent);
    border: 1px solid var(--accent-border);
    align-self: flex-start;
  }
  .setup-title h1 {
    font-size: 24px; font-weight: 600; letter-spacing: -.3px;
    margin-bottom: 8px;
  }
  .setup-title p {
    color: var(--text-dim); font-size: 13px; line-height: 1.6;
  }
  .setup-title p code {
    font-family: var(--font-mono); font-size: 12px;
    padding: 2px 6px; border-radius: 4px;
    background: var(--bg-3); color: var(--text);
  }
  .setup-form {
    padding: 24px;
    display: flex; flex-direction: column; gap: 18px;
  }
  .setup-actions {
    display: flex; align-items: center; justify-content: space-between;
    margin-top: 4px;
  }
  .setup-feedback {
    padding: 12px 16px;
    border-radius: var(--r-lg);
    font-size: 13px;
    display: none;
  }
  .setup-feedback.ok {
    display: block;
    background: var(--green-bg);
    color: var(--green);
    border: 1px solid rgba(52, 211, 153, 0.30);
  }
  .setup-feedback.err {
    display: block;
    background: var(--red-bg);
    color: var(--red);
    border: 1px solid rgba(248, 113, 113, 0.30);
  }

  /* ─────────────────────────────────────────────────────────
     1. Idle
     ───────────────────────────────────────────────────────── */
  .idle-wrap {
    max-width: 720px;
    margin: 32px auto;
    display: flex; flex-direction: column; align-items: center;
    gap: 28px;
    padding-top: 40px;
  }
  .idle-title {
    text-align: center;
  }
  .idle-title h1 {
    font-size: 24px; font-weight: 600; letter-spacing: -.3px;
    margin-bottom: 8px;
  }
  .idle-title p {
    color: var(--text-dim); font-size: 13px; line-height: 1.6;
    max-width: 520px;
  }
  .idle-title code {
    font-family: var(--font-mono); font-size: 12px;
    padding: 2px 6px; border-radius: 4px;
    background: var(--bg-3); color: var(--text);
  }
  .dropzone {
    width: 100%; max-width: 560px;
    padding: 56px 32px;
    border: 1.5px dashed var(--border-strong);
    border-radius: var(--r-2xl);
    background: var(--bg-2);
    text-align: center;
    cursor: pointer;
    position: relative;
    transition: all .18s ease;
  }
  .dropzone:hover, .dropzone.dragover {
    border-color: var(--accent);
    background: var(--accent-bg);
  }
  .dropzone input[type=file] {
    position: absolute; inset: 0; opacity: 0; cursor: pointer; width: 100%; height: 100%;
  }
  .dropzone .drop-icon {
    width: 44px; height: 44px;
    margin: 0 auto 16px;
    display: flex; align-items: center; justify-content: center;
    color: var(--accent);
  }
  .dropzone .drop-title {
    font-size: 15px; font-weight: 500; margin-bottom: 6px;
  }
  .dropzone .drop-hint {
    font-size: 12px; color: var(--text-dim);
    font-family: var(--font-mono);
  }
  .idle-links {
    display: flex; gap: 16px; align-items: center;
    color: var(--text-faint); font-size: 12px;
  }
  .idle-links .sep { color: var(--text-faint); }

  /* ─────────────────────────────────────────────────────────
     2. Ready
     ───────────────────────────────────────────────────────── */
  .ready-grid {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 28px;
    max-width: 1100px;
    margin: 0 auto;
  }
  .ready-section { display: flex; flex-direction: column; gap: 14px; }
  .ready-label {
    font-size: 10px; font-weight: 600; letter-spacing: 1.2px;
    text-transform: uppercase; color: var(--text-dim);
  }

  .file-card {
    padding: 14px;
    display: flex; align-items: center; gap: 12px;
  }
  .file-card .ext {
    width: 38px; height: 38px;
    border-radius: var(--r-md);
    background: var(--accent-bg);
    color: var(--accent);
    border: 1px solid var(--accent-border);
    display: flex; align-items: center; justify-content: center;
    font-size: 11px; font-weight: 700;
    font-family: var(--font-mono);
    flex-shrink: 0;
  }
  .file-card .file-info { flex: 1; min-width: 0; }
  .file-card .file-name {
    font-family: var(--font-mono); font-size: 12px; font-weight: 500;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .file-card .file-meta {
    margin-top: 4px; font-size: 11px; color: var(--text-dim);
    font-family: var(--font-mono);
  }
  .file-card .file-x {
    background: transparent; border: none; color: var(--text-faint);
    cursor: pointer; font-size: 18px; padding: 4px 8px;
  }
  .file-card .file-x:hover { color: var(--red); }

  .validation-list { display: flex; flex-direction: column; gap: 10px; }
  .validation-item {
    display: flex; align-items: flex-start; gap: 10px;
    font-size: 12.5px; color: var(--text);
  }
  .validation-item .v-icon {
    width: 18px; height: 18px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; margin-top: 1px;
    font-size: 10px; font-weight: 700;
  }
  .validation-item.ok .v-icon { background: var(--green-bg); color: var(--green); }
  .validation-item.warn .v-icon { background: var(--amber-bg); color: var(--amber); }
  .validation-item.warn { color: var(--amber); }

  .ops-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .op-card {
    padding: 16px;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--r-lg);
    cursor: pointer;
    user-select: none;
    transition: all .15s ease;
    display: flex; gap: 12px; align-items: flex-start;
  }
  .op-card:hover { border-color: var(--accent-border); }
  .op-card.checked {
    border-color: var(--accent);
    background: var(--accent-bg);
  }
  .op-card .checkmark {
    width: 18px; height: 18px;
    border: 1.5px solid var(--border-strong);
    border-radius: var(--r-sm);
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; margin-top: 1px;
    color: #0a0d12; font-size: 11px; font-weight: 700;
    transition: all .15s;
  }
  .op-card.checked .checkmark {
    background: var(--accent); border-color: var(--accent);
  }
  .op-card-info { flex: 1; }
  .op-card-name { font-size: 13px; font-weight: 500; margin-bottom: 4px; }
  .op-card-desc { font-size: 11.5px; color: var(--text-dim); line-height: 1.4; }
  .op-card-extra {
    margin-top: 10px;
    display: flex; gap: 6px;
    animation: slide-down .2s ease;
  }
  .op-card-extra.hidden { display: none; }
  .op-card-extra label {
    flex: 1;
    padding: 5px 0;
    text-align: center;
    border: 1px solid var(--border-strong);
    border-radius: var(--r-md);
    font-size: 11px; font-family: var(--font-mono);
    color: var(--text-dim);
    cursor: pointer;
    transition: all .15s;
  }
  .op-card-extra label:hover { border-color: var(--accent-border); color: var(--text); }
  .op-card-extra label.active {
    background: var(--accent-bg); border-color: var(--accent); color: var(--text);
  }
  .op-card-extra input[type=radio] { display: none; }
  @keyframes slide-down {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .ready-footer {
    margin-top: 20px;
    padding: 18px;
    display: flex; align-items: center; justify-content: space-between; gap: 16px;
  }
  .ready-footer .summary-text { display: flex; flex-direction: column; gap: 3px; }
  .ready-footer .summary-text .t { font-size: 13px; font-weight: 500; }
  .ready-footer .summary-text .s { font-size: 11.5px; color: var(--text-dim); font-family: var(--font-mono); }

  /* ─────────────────────────────────────────────────────────
     3. Sending
     ───────────────────────────────────────────────────────── */
  .sending-wrap {
    max-width: 1200px;
    margin: 0 auto;
    display: flex; flex-direction: column;
  }
  .sending-top {
    padding: 16px 0;
    display: flex; align-items: center; gap: 16px;
  }
  .sending-top .info { flex: 1; min-width: 0; }
  .sending-top .info .t { font-size: 14px; font-weight: 600; margin-bottom: 4px; }
  .sending-top .info .s {
    font-family: var(--font-mono); font-size: 11.5px;
    color: var(--text-dim);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .sending-top .info .s strong { color: var(--text); font-weight: 500; }
  .sending-top .ctrls { display: flex; gap: 8px; }

  .progress-wrap {
    height: 4px; background: var(--border);
    border-radius: 99px; overflow: hidden;
    margin-bottom: 14px;
  }
  .progress-fill {
    height: 100%; width: 0;
    background: linear-gradient(90deg, var(--accent), var(--purple));
    transition: width .25s ease;
    box-shadow: 0 0 12px rgba(90,200,250,.45);
  }

  .stats-row {
    display: flex; gap: 28px; align-items: baseline;
    font-size: 12.5px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }
  .stats-row .stat { display: flex; gap: 6px; align-items: baseline; }
  .stats-row .stat .label { color: var(--text-dim); font-size: 11px; }
  .stats-row .stat .val   { font-family: var(--font-mono); font-weight: 600; font-size: 14px; }
  .stats-row .stat.processadas .val { color: var(--text); }
  .stats-row .stat.success .val    { color: var(--green); }
  .stats-row .stat.success::before { content:''; width:7px; height:7px; border-radius:50%; background:var(--green); align-self: center; }
  .stats-row .stat.warn .val       { color: var(--amber); }
  .stats-row .stat.warn::before    { content:''; width:7px; height:7px; border-radius:50%; background:var(--amber); align-self: center; }
  .stats-row .stat.err .val        { color: var(--red); }
  .stats-row .stat.err::before     { content:''; width:7px; height:7px; border-radius:50%; background:var(--red); align-self: center; }
  .stats-row .stat.eta { color: var(--text-dim); margin-left: auto; font-family: var(--font-mono); font-size: 12px; }

  .filter-bar {
    display: flex; align-items: center; gap: 4px;
    margin-bottom: 14px;
    border-bottom: 1px solid var(--border);
    padding-bottom: 12px;
  }
  .filter-tab {
    background: transparent; border: 1px solid transparent;
    color: var(--text-dim);
    padding: 6px 12px;
    border-radius: var(--r-md);
    font-size: 12px; font-weight: 500;
    cursor: pointer;
    display: inline-flex; align-items: center; gap: 6px;
    transition: all .15s;
  }
  .filter-tab:hover { color: var(--text); background: var(--bg-3); }
  .filter-tab.active {
    background: var(--bg-3);
    border-color: var(--border-strong);
    color: var(--text);
  }
  .filter-tab .count {
    font-family: var(--font-mono); font-size: 11px;
    padding: 1px 6px; border-radius: 99px;
    background: var(--border); color: var(--text-dim);
    min-width: 20px; text-align: center;
  }
  .filter-tab.active .count { background: var(--border-strong); color: var(--text); }
  .filter-bar .right { margin-left: auto; }

  .log-list {
    flex: 1;
    border: 1px solid var(--border);
    border-radius: var(--r-lg);
    background: var(--bg-2);
    max-height: calc(100vh - 280px);
    overflow-y: auto;
  }
  .log-row {
    display: grid;
    grid-template-columns: 84px 16px 88px 1fr 88px;
    gap: 14px;
    align-items: start;
    padding: 9px 16px;
    font-family: var(--font-mono);
    font-size: 12px;
    border-bottom: 1px solid rgba(255,255,255,.02);
    animation: row-in .2s ease;
  }
  .log-row:last-child { border-bottom: none; }
  .log-row.ok    { background: transparent; }
  .log-row.aviso { background: rgba(251,191,36,.04); }
  .log-row.erro  { background: rgba(248,113,113,.05); }
  .log-row .ts   { color: var(--text-faint); }
  .log-row .dot  {
    width: 7px; height: 7px; border-radius: 50%;
    margin-top: 6px;
    background: var(--text-faint);
  }
  .log-row.ok .dot    { background: var(--green); }
  .log-row.aviso .dot { background: var(--amber); }
  .log-row.erro .dot  { background: var(--red); }
  .log-row .op {
    color: var(--text-dim); font-size: 10.5px;
    font-weight: 500; letter-spacing: .4px; text-transform: uppercase;
    padding-top: 1px;
  }
  .log-row .msg {
    color: var(--text);
    word-break: break-word;
    line-height: 1.5;
  }
  .log-row .linha-ref {
    color: var(--text-dim);
    font-size: 11px;
    text-align: right;
  }
  .log-row .linha-ref a {
    color: var(--text-dim); text-decoration: none;
    border: 1px solid var(--border-strong);
    padding: 2px 8px; border-radius: var(--r-md);
    transition: all .15s;
  }
  .log-row .linha-ref a:hover { color: var(--accent); border-color: var(--accent-border); }
  .log-row.info { color: var(--text-dim); }
  .log-row.info .msg { color: var(--text-dim); }

  @keyframes row-in {
    from { opacity: 0; transform: translateX(-4px); }
    to { opacity: 1; transform: translateX(0); }
  }

  /* ─────────────────────────────────────────────────────────
     4. Done
     ───────────────────────────────────────────────────────── */
  .done-wrap {
    max-width: 880px;
    margin: 0 auto;
    display: flex; flex-direction: column; gap: 20px;
  }
  .done-hero { padding: 24px; }
  .done-hero h2 {
    font-size: 22px; font-weight: 600; margin: 12px 0 6px;
    letter-spacing: -.2px;
  }
  .done-hero .meta {
    font-size: 12px; color: var(--text-dim);
    font-family: var(--font-mono);
  }
  .done-hero .meta strong { color: var(--text); font-weight: 500; }
  .done-hero .actions {
    margin-top: 18px;
    display: flex; gap: 10px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 12px;
  }
  .stats-grid .stat-cell {
    padding: 16px;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--r-lg);
  }
  .stats-grid .stat-cell .val {
    font-family: var(--font-mono); font-size: 28px; font-weight: 600;
    line-height: 1; margin-bottom: 6px;
  }
  .stats-grid .stat-cell .lbl {
    font-size: 11px; color: var(--text-dim); text-transform: uppercase;
    letter-spacing: .8px;
  }
  .stats-grid .stat-cell.s-ok .val { color: var(--green); }
  .stats-grid .stat-cell.s-warn .val { color: var(--amber); }
  .stats-grid .stat-cell.s-err .val { color: var(--red); }

  .stack-bar {
    height: 8px; border-radius: 99px; overflow: hidden;
    background: var(--bg-3); display: flex;
  }
  .stack-bar > div { height: 100%; }
  .stack-bar .b-ok { background: var(--green); }
  .stack-bar .b-warn { background: var(--amber); }
  .stack-bar .b-err { background: var(--red); }

  .errors-card { padding: 0; overflow: hidden; }
  .errors-card .head {
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
    display: flex; align-items: center; justify-content: space-between;
  }
  .errors-card .head .t { font-size: 13px; font-weight: 600; }
  .errors-card .head .s { font-size: 11px; color: var(--text-dim); font-family: var(--font-mono); }
  .errors-card .body {
    max-height: 360px; overflow-y: auto;
  }
  .err-item {
    padding: 10px 18px;
    border-bottom: 1px solid rgba(255,255,255,.03);
    display: grid;
    grid-template-columns: 60px 130px 1fr 90px;
    gap: 14px; align-items: center;
    font-size: 12px; font-family: var(--font-mono);
  }
  .err-item:last-child { border-bottom: none; }
  .err-item .linha { color: var(--text-dim); }
  .err-item .ean { color: var(--text); }
  .err-item .motivo { color: var(--red); }
  .err-item .op-tag {
    display: inline-flex; align-items: center; padding: 2px 8px;
    border-radius: 99px; font-size: 10px;
    background: var(--bg-3); color: var(--text-dim);
    text-transform: uppercase; letter-spacing: .4px;
  }

  .errors-card .footer {
    padding: 14px 18px;
    border-top: 1px solid var(--border);
    display: flex; align-items: center; justify-content: space-between;
  }
  .errors-card .footer .info { font-size: 12px; color: var(--text-dim); }

  .pt-keys-card { padding: 20px; margin-top: 16px; }
  .pt-keys-card .head { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
  .pt-keys-card .head .t { font-size: 13px; font-weight: 600; }
  .pt-keys-card .keys { display: flex; flex-wrap: wrap; gap: 10px; }
  .pt-keys-card .key-pill {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 8px 14px;
    border-radius: var(--r-lg);
    font-family: var(--font-mono); font-size: 15px; font-weight: 600;
    background: var(--accent-bg);
    border: 1px solid var(--accent-border);
    color: var(--accent);
  }
  .pt-keys-card .key-pill .key-icon { font-size: 12px; opacity: .7; }

  /* hidden util */
  .hidden { display: none !important; }
</style>
</head>
<body>

<header>
  <div class="brand">
    <div class="logo">↑</div>
    <div>
      <span class="name">Bluesoft Uploader</span>
      <span class="sub">via .xlsx</span>
    </div>
  </div>
  <div class="header-right">
    <span id="tenantPill" class="pill tenant hidden" onclick="goToSetup()">
      <span class="pill-label">tenant</span>
      <span class="pill-value" id="tenantValue">—</span>
      <span class="caret">▾</span>
    </span>
    <span id="statusPill" class="pill status aguardando">
      <span class="status-label">aguardando</span>
    </span>
  </div>
</header>

<main>

  <!-- ─── SETUP ─── -->
  <section class="screen" data-screen="setup">
    <div class="setup-wrap">
      <span class="badge"><span style="background:var(--accent);width:5px;height:5px;border-radius:50%;display:inline-block"></span> Configuração inicial</span>
      <div class="setup-title">
        <h1>Conectar ao Bluesoft ERP</h1>
        <p>Informe as credenciais da API uma vez. Elas ficam salvas em <code>.env</code> na pasta do executável e são reusadas nas próximas execuções.</p>
      </div>
      <div class="card setup-form">
        <div class="field">
          <div class="field-row">
            <label class="field-label" for="setupTenant">Tenant</label>
            <span class="field-hint">o subdomínio do cliente no Bluesoft (ex: minipreco)</span>
          </div>
          <input type="text" id="setupTenant" class="input" placeholder="minipreco" oninput="setupChanged()">
        </div>
        <div class="field">
          <div class="field-row">
            <label class="field-label" for="setupClientId">Client ID</label>
            <span class="field-hint">fornecido pela equipe Bluesoft</span>
          </div>
          <input type="text" id="setupClientId" class="input" placeholder="79648963c7fe9d13e454aae876b6…" oninput="setupChanged()">
        </div>
        <div class="field">
          <div class="field-row">
            <label class="field-label" for="setupSecret">Client Secret</label>
            <span class="field-hint">mantenha em segredo — armazenado localmente em .env</span>
          </div>
          <div class="input-group">
            <input type="password" id="setupSecret" class="input" placeholder="••••••••••••••••••••••••" oninput="setupChanged()">
            <button class="toggle" onclick="toggleSecret()">mostrar</button>
          </div>
        </div>
        <div class="setup-feedback" id="setupFeedback"></div>
        <div class="setup-actions">
          <button class="btn-link" onclick="window.open('https://erp.bluesoft.com.br', '_blank')">⓵ onde encontro essas credenciais?</button>
          <div style="display:flex;gap:8px;align-items:center">
            <button class="btn btn-ghost hidden" id="setupCancel" onclick="cancelSetup()">Cancelar</button>
            <button class="btn btn-primary" id="setupCta" disabled onclick="setupCtaClick()">Testar conexão</button>
          </div>
        </div>
      </div>
    </div>
  </section>

  <!-- ─── IDLE ─── -->
  <section class="screen" data-screen="idle">
    <div class="idle-wrap">
      <div class="idle-title">
        <h1>Importar planilha</h1>
        <p>Solte o arquivo <code>.xlsx</code> abaixo. Você escolhe quais operações executar no próximo passo.</p>
      </div>
      <div class="dropzone" id="dropzone">
        <input type="file" id="fileInput" accept=".xlsx" onchange="onFile(this.files[0])">
        <div class="drop-icon">
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="17 8 12 3 7 8"/>
            <line x1="12" y1="3" x2="12" y2="15"/>
          </svg>
        </div>
        <div class="drop-title">Arraste o .xlsx ou clique para selecionar</div>
        <div class="drop-hint">aba "Preencher" · cabeçalhos na linha 8</div>
      </div>
      <div class="idle-links">
        <button class="btn-link" onclick="abrirLogs()">📁 abrir pasta de logs</button>
        <span class="sep">·</span>
        <button class="btn-link" onclick="goToSetup()">configurar credenciais</button>
      </div>
    </div>
  </section>

  <!-- ─── READY ─── -->
  <section class="screen" data-screen="ready">
    <div class="ready-grid">
      <div class="ready-section">
        <div>
          <div class="ready-label" style="margin-bottom:10px;">Arquivo</div>
          <div class="card file-card">
            <div class="ext">XLSX</div>
            <div class="file-info">
              <div class="file-name" id="readyFilename">—</div>
              <div class="file-meta" id="readyFilemeta">— linhas · — KB</div>
            </div>
            <button class="file-x" onclick="resetUpload()" title="Remover arquivo">×</button>
          </div>
        </div>
        <div>
          <div class="ready-label" style="margin-bottom:10px;margin-top:6px;">Validação</div>
          <div class="validation-list" id="validationList"></div>
        </div>
      </div>

      <div class="ready-section">
        <div>
          <div class="ready-label" style="margin-bottom:10px;">Operações</div>
          <p style="font-size:12px;color:var(--text-dim);margin-bottom:16px;">Selecione o que executar nessa importação. A planilha contém os dados de todas as operações.</p>
          <div class="ops-grid">

            <div class="op-card" data-op="alterarNCM" onclick="toggleOp(this)">
              <div class="checkmark"></div>
              <div class="op-card-info">
                <div class="op-card-name">Alterar NCM</div>
                <div class="op-card-desc">Atualiza o NCM no cadastro do produto</div>
              </div>
            </div>

            <div class="op-card" data-op="linhaCompra" onclick="toggleOp(this)">
              <div class="checkmark"></div>
              <div class="op-card-info">
                <div class="op-card-name">Linha de Compra</div>
                <div class="op-card-desc">Cadastra produto nas lojas selecionadas</div>
              </div>
            </div>

            <div class="op-card" data-op="linhaLoja" onclick="toggleOp(this)">
              <div class="checkmark"></div>
              <div class="op-card-info">
                <div class="op-card-name">Sortimento / Linha de Loja</div>
                <div class="op-card-desc">Define parâmetros de sortimento por loja</div>
              </div>
            </div>

            <div class="op-card" data-op="linhaCompraCD" onclick="toggleOp(this)">
              <div class="checkmark"></div>
              <div class="op-card-info">
                <div class="op-card-name">Linha de Compra CD</div>
                <div class="op-card-desc">Agrupa lojas por centro de distribuição</div>
                <div class="op-card-extra hidden" id="cdTipoExtra" onclick="event.stopPropagation()">
                  <label class="active" id="cdTipoUNLabel">
                    <input type="radio" name="cdTipo" value="UN" checked onchange="onCdTipo(this)">
                    UN
                  </label>
                  <label id="cdTipoCXLabel">
                    <input type="radio" name="cdTipo" value="CX" onchange="onCdTipo(this)">
                    CX
                  </label>
                </div>
              </div>
            </div>

            <div class="op-card" data-op="preTransf" onclick="toggleOp(this)">
              <div class="checkmark"></div>
              <div class="op-card-info">
                <div class="op-card-name">Pré-transferência</div>
                <div class="op-card-desc">Cria pré-transferência multiloja a partir das quantidades por loja</div>
                <div class="op-card-extra hidden" id="preOrigemExtra" onclick="event.stopPropagation()">
                  <label style="flex:1;text-align:left;padding:0;border:none;background:transparent;color:var(--text-dim);font-size:11px;">loja origem:</label>
                  <input type="number" id="preOrigemInput" placeholder="ex: 5" min="1"
                    style="width:90px;background:var(--bg-3);border:1px solid var(--border-strong);border-radius:var(--r-md);padding:5px 8px;color:var(--text);font-family:var(--font-mono);font-size:11px;outline:none;"
                    oninput="refreshReadyCta()">
                </div>
              </div>
            </div>

          </div>
        </div>

        <div class="card ready-footer">
          <div class="summary-text">
            <span class="t" id="readyCtaTitle">Selecione uma operação</span>
            <span class="s" id="readyCtaSubtitle">— linhas processadas</span>
          </div>
          <button class="btn btn-primary" id="readyCta" disabled onclick="iniciarRun()">→ Enviar para API</button>
        </div>
      </div>
    </div>
  </section>

  <!-- ─── SENDING ─── -->
  <section class="screen" data-screen="sending">
    <div class="sending-wrap">
      <div class="sending-top">
        <div class="info">
          <div class="t">Enviando para API</div>
          <div class="s"><strong id="sendingFilename">—</strong> · operação: <strong id="sendingOp">—</strong></div>
        </div>
        <div class="ctrls">
          <button class="btn btn-danger" onclick="parar()" id="btnParar">■ Parar</button>
        </div>
      </div>

      <div class="progress-wrap"><div class="progress-fill" id="progFill"></div></div>

      <div class="stats-row">
        <div class="stat processadas"><span class="label">processadas</span><span class="val" id="statProc">0/0</span></div>
        <div class="stat success"><span class="val" id="statOk">0</span><span class="label">sucesso</span></div>
        <div class="stat warn"><span class="val" id="statWarn">0</span><span class="label">avisos</span></div>
        <div class="stat err"><span class="val" id="statErr">0</span><span class="label">erros</span></div>
        <div class="stat eta" id="statEta">—</div>
      </div>

      <div class="filter-bar">
        <button class="filter-tab active" data-filter="tudo" onclick="setFiltro('tudo')">Tudo <span class="count" id="cntTudo">0</span></button>
        <button class="filter-tab" data-filter="ok" onclick="setFiltro('ok')">Sucesso <span class="count" id="cntOk">0</span></button>
        <button class="filter-tab" data-filter="aviso" onclick="setFiltro('aviso')">Avisos <span class="count" id="cntAviso">0</span></button>
        <button class="filter-tab" data-filter="erro" onclick="setFiltro('erro')">Erros <span class="count" id="cntErro">0</span></button>
        <div class="right">
          <button class="btn-link" onclick="abrirLogs()">↓ exportar log</button>
        </div>
      </div>

      <div class="log-list" id="logList"></div>
    </div>
  </section>

  <!-- ─── DONE ─── -->
  <section class="screen" data-screen="done">
    <div class="done-wrap">
      <div class="card done-hero">
        <span class="badge" id="doneBadge"><span style="background:var(--green);width:5px;height:5px;border-radius:50%;display:inline-block"></span> Concluído</span>
        <h2 id="doneTitle">Importação concluída</h2>
        <div class="meta"><strong id="doneFilename">—</strong> · <span id="doneOps">—</span> · <span id="doneDuration">—</span></div>
        <div class="actions">
          <button class="btn btn-ghost" onclick="abrirLogs()">↗ relatório completo</button>
          <button class="btn btn-ghost" onclick="resetUpload()">↻ nova importação</button>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-cell"><div class="val mono" id="doneTotal">0</div><div class="lbl">total</div></div>
        <div class="stat-cell s-ok"><div class="val mono" id="doneOk">0</div><div class="lbl">sucesso</div></div>
        <div class="stat-cell s-warn"><div class="val mono" id="doneWarn">0</div><div class="lbl">avisos</div></div>
        <div class="stat-cell s-err"><div class="val mono" id="doneErr">0</div><div class="lbl">erros</div></div>
      </div>

      <div class="stack-bar" id="doneStack"></div>

      <div class="card errors-card hidden" id="errorsCard">
        <div class="head">
          <span class="t">Erros</span>
          <span class="s" id="errorsSub">0 itens</span>
        </div>
        <div class="body" id="errorsBody"></div>
        <div class="footer">
          <span class="info">Reenvia somente as linhas que falharam.</span>
          <button class="btn btn-primary" onclick="reenviarErros()">↻ Reenviar erros</button>
        </div>
      </div>

      <div class="card pt-keys-card hidden" id="ptKeysCard">
        <div class="head">
          <span class="t">🔑 Chaves de Pré-transferência</span>
        </div>
        <div class="keys" id="ptKeysList"></div>
      </div>

    </div>
  </section>

</main>

<script>
// ───────────────────────────────────────────────────────────
// State machine
// ───────────────────────────────────────────────────────────
const state = {
  current: null,
  configured: false,
  tenant: '',
  session: null,            // { id, filename, size, linhas, avisos }
  ops: { alterarNCM: false, linhaCompra: false, linhaLoja: false, linhaCompraCD: false },
  cdTipo: 'UN',
  run: null,                // { startedAt, totals, opsUsed, errors[] }
  logs: [],                 // todos eventos recebidos
  filtro: 'tudo',
}

function setScreen(name) {
  state.current = name
  document.querySelectorAll('.screen').forEach(el => {
    el.classList.toggle('active', el.dataset.screen === name)
  })
}

function setStatus(label, kind) {
  const pill = document.getElementById('statusPill')
  pill.classList.remove('aguardando','pronto','enviando','concluido','erro')
  pill.classList.add(kind)
  pill.querySelector('.status-label').textContent = label
}

function setTenantPill(t) {
  state.tenant = t
  const pill = document.getElementById('tenantPill')
  if (!t) { pill.classList.add('hidden'); return }
  pill.classList.remove('hidden')
  document.getElementById('tenantValue').textContent = t
}

// ───────────────────────────────────────────────────────────
// Boot
// ───────────────────────────────────────────────────────────
async function boot() {
  try {
    const r = await fetch('/api/setup')
    const d = await r.json()
    state.configured = !!d.configured
    setTenantPill(d.tenant || '')
    if (!state.configured) {
      goToSetup()
    } else {
      goToIdle()
    }
  } catch (e) {
    goToSetup()
  }
}

// ───────────────────────────────────────────────────────────
// Setup
// ───────────────────────────────────────────────────────────
function goToSetup() {
  setScreen('setup')
  setStatus('aguardando', 'aguardando')
  // prefill tenant se já configurado (edit mode)
  if (state.tenant) {
    document.getElementById('setupTenant').value = state.tenant
    document.getElementById('setupCancel').classList.remove('hidden')
  } else {
    document.getElementById('setupCancel').classList.add('hidden')
  }
  setupChanged()
  resetSetupFeedback()
}

function cancelSetup() {
  if (state.configured) goToIdle()
}

function setupChanged() {
  const t = document.getElementById('setupTenant').value.trim()
  const c = document.getElementById('setupClientId').value.trim()
  const s = document.getElementById('setupSecret').value.trim()
  const cta = document.getElementById('setupCta')
  cta.disabled = !(t && c && s)
  // resetar pra modo "testar" se mexer depois de testado
  if (cta.dataset.phase === 'success') {
    cta.textContent = 'Testar conexão'
    cta.dataset.phase = 'form'
    resetSetupFeedback()
  }
}

function resetSetupFeedback() {
  const fb = document.getElementById('setupFeedback')
  fb.className = 'setup-feedback'
  fb.textContent = ''
}

function setSetupFeedback(kind, msg) {
  const fb = document.getElementById('setupFeedback')
  fb.className = 'setup-feedback ' + kind
  fb.textContent = msg
}

function toggleSecret() {
  const inp = document.getElementById('setupSecret')
  const btn = inp.parentElement.querySelector('.toggle')
  if (inp.type === 'password') { inp.type = 'text'; btn.textContent = 'ocultar' }
  else { inp.type = 'password'; btn.textContent = 'mostrar' }
}

async function setupCtaClick() {
  const cta = document.getElementById('setupCta')
  if (cta.dataset.phase === 'success') {
    await setupSave()
    return
  }
  await setupTest()
}

async function setupTest() {
  const tenant = document.getElementById('setupTenant').value.trim()
  const clientId = document.getElementById('setupClientId').value.trim()
  const clientSecret = document.getElementById('setupSecret').value.trim()
  const cta = document.getElementById('setupCta')
  cta.disabled = true
  cta.textContent = 'Testando…'
  resetSetupFeedback()
  try {
    const r = await fetch('/api/setup/test', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({tenant, clientId, clientSecret})
    })
    const d = await r.json()
    if (!d.ok) {
      setSetupFeedback('err', '✗ ' + (d.error || 'falha na autenticação'))
      cta.disabled = false
      cta.textContent = 'Testar conexão'
      cta.dataset.phase = 'form'
      return
    }
    setSetupFeedback('ok', '✓ credenciais válidas — clique em Salvar para gravar o .env')
    cta.disabled = false
    cta.textContent = '✓ Salvar .env e continuar'
    cta.dataset.phase = 'success'
  } catch (e) {
    setSetupFeedback('err', '✗ erro de conexão: ' + e.message)
    cta.disabled = false
    cta.textContent = 'Tentar novamente'
    cta.dataset.phase = 'form'
  }
}

async function setupSave() {
  const tenant = document.getElementById('setupTenant').value.trim()
  const clientId = document.getElementById('setupClientId').value.trim()
  const clientSecret = document.getElementById('setupSecret').value.trim()
  const cta = document.getElementById('setupCta')
  cta.disabled = true
  cta.textContent = 'Salvando…'
  try {
    const r = await fetch('/api/setup/save', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({tenant, clientId, clientSecret})
    })
    if (!r.ok) throw new Error('falha ao gravar')
    state.configured = true
    setTenantPill(tenant)
    cta.textContent = 'Testar conexão'
    cta.dataset.phase = 'form'
    goToIdle()
  } catch (e) {
    setSetupFeedback('err', '✗ ' + e.message)
    cta.disabled = false
    cta.textContent = '✓ Salvar .env e continuar'
    cta.dataset.phase = 'success'
  }
}

// ───────────────────────────────────────────────────────────
// Idle
// ───────────────────────────────────────────────────────────
function goToIdle() {
  state.session = null
  state.ops = { alterarNCM: false, linhaCompra: false, linhaLoja: false, linhaCompraCD: false, preTransf: false }
  state.cdTipo = 'UN'
  setScreen('idle')
  setStatus('aguardando', 'aguardando')
  document.getElementById('fileInput').value = ''
}

// drag & drop handlers
const dz = document.getElementById('dropzone')
dz.addEventListener('dragover', e => { e.preventDefault(); dz.classList.add('dragover') })
dz.addEventListener('dragleave', () => dz.classList.remove('dragover'))
dz.addEventListener('drop', e => {
  e.preventDefault(); dz.classList.remove('dragover')
  const f = e.dataTransfer.files[0]
  if (f && f.name.toLowerCase().endsWith('.xlsx')) onFile(f)
})

async function onFile(file) {
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const r = await fetch('/api/upload', { method: 'POST', body: fd })
    const d = await r.json()
    if (!r.ok) throw new Error(d.error || 'falha na validação')
    state.session = d
    goToReady()
  } catch (e) {
    alert('Erro ao validar planilha: ' + e.message)
  }
}

function abrirLogs() {
  fetch('/api/abrir-logs').catch(() => alert('Não foi possível abrir a pasta de logs.'))
}

// ───────────────────────────────────────────────────────────
// Ready
// ───────────────────────────────────────────────────────────
function goToReady() {
  setScreen('ready')
  setStatus('pronto', 'pronto')
  const s = state.session
  document.getElementById('readyFilename').textContent = s.filename
  const formatoLabel = s.formato === 'pt' ? 'Pré-transferência' : 'Ficha de cadastro'
  document.getElementById('readyFilemeta').textContent =
    formatoLabel + ' · ' + s.linhas + ' linhas · ' + formatSize(s.size)

  // validação list
  const list = document.getElementById('validationList')
  list.innerHTML = ''
  if (s.formato === 'pt') {
    appendValidation(list, 'ok', 'modelo Pré-transferência detectado')
    appendValidation(list, 'ok', 'cabeçalhos na linha 1')
    appendValidation(list, 'ok', s.linhas + ' linhas válidas')
  } else {
    appendValidation(list, 'ok', 'aba "Preencher" encontrada')
    appendValidation(list, 'ok', 'cabeçalhos detectados na linha 8')
    appendValidation(list, 'ok', s.linhas + ' linhas válidas')
  }
  for (const a of (s.avisos || [])) appendValidation(list, 'warn', a)

  // reset ops + extras
  document.querySelectorAll('.op-card').forEach(c => {
    c.classList.remove('checked')
    c.classList.remove('disabled')
    c.style.opacity = ''
    c.style.pointerEvents = ''
    c.removeAttribute('title')
  })
  document.querySelectorAll('.op-card .checkmark').forEach(c => c.textContent = '')
  document.getElementById('cdTipoExtra').classList.add('hidden')
  document.getElementById('preOrigemExtra').classList.add('hidden')
  document.getElementById('preOrigemInput').value = ''
  state.ops = { alterarNCM: false, linhaCompra: false, linhaLoja: false, linhaCompraCD: false, preTransf: false }

  // No modelo PT, só Pré-transferência é permitida; outros cards ficam desabilitados.
  if (s.formato === 'pt') {
    document.querySelectorAll('.op-card').forEach(c => {
      if (c.dataset.op !== 'preTransf') {
        c.classList.add('disabled')
        c.style.opacity = '0.35'
        c.style.pointerEvents = 'none'
        c.setAttribute('title', 'disponível apenas na ficha de cadastro')
      }
    })
    // Marca PT por padrão.
    const ptCard = document.querySelector('.op-card[data-op="preTransf"]')
    state.ops.preTransf = true
    ptCard.classList.add('checked')
    ptCard.querySelector('.checkmark').textContent = '✓'
    // No modelo PT, origem vem da planilha — não mostra input.
  }
  refreshReadyCta()
}

function appendValidation(parent, kind, text) {
  const el = document.createElement('div')
  el.className = 'validation-item ' + kind
  el.innerHTML = '<span class="v-icon">' + (kind === 'ok' ? '✓' : '!') + '</span><span>' + escapeHtml(text) + '</span>'
  parent.appendChild(el)
}

function toggleOp(card) {
  const op = card.dataset.op
  state.ops[op] = !state.ops[op]
  card.classList.toggle('checked', state.ops[op])
  card.querySelector('.checkmark').textContent = state.ops[op] ? '✓' : ''
  if (op === 'linhaCompraCD') {
    document.getElementById('cdTipoExtra').classList.toggle('hidden', !state.ops[op])
  }
  // Mutex: Pré-transferência roda sozinha.
  if (state.ops[op]) {
    if (op === 'preTransf') {
      ['alterarNCM','linhaCompra','linhaLoja','linhaCompraCD'].forEach(o => {
        if (state.ops[o]) {
          state.ops[o] = false
          const c = document.querySelector('.op-card[data-op="'+o+'"]')
          c.classList.remove('checked')
          c.querySelector('.checkmark').textContent = ''
        }
      })
      document.getElementById('cdTipoExtra').classList.add('hidden')
    } else {
      if (state.ops.preTransf) {
        state.ops.preTransf = false
        const c = document.querySelector('.op-card[data-op="preTransf"]')
        c.classList.remove('checked')
        c.querySelector('.checkmark').textContent = ''
      }
    }
  }
  // Input de origem: só na ficha + PT marcada.
  const formato = state.session && state.session.formato
  const mostrarOrigem = state.ops.preTransf && formato === 'ficha'
  document.getElementById('preOrigemExtra').classList.toggle('hidden', !mostrarOrigem)
  refreshReadyCta()
}

function onCdTipo(input) {
  state.cdTipo = input.value
  document.getElementById('cdTipoUNLabel').classList.toggle('active', input.value === 'UN')
  document.getElementById('cdTipoCXLabel').classList.toggle('active', input.value === 'CX')
}

function refreshReadyCta() {
  const opsList = []
  if (state.ops.alterarNCM) opsList.push('Alterar NCM')
  if (state.ops.linhaCompra) opsList.push('Linha de Compra')
  if (state.ops.linhaLoja) opsList.push('Sortimento')
  if (state.ops.linhaCompraCD) opsList.push('Linha CD (' + state.cdTipo + ')')
  if (state.ops.preTransf) opsList.push('Pré-transferência')
  const cta = document.getElementById('readyCta')
  const t = document.getElementById('readyCtaTitle')
  const s = document.getElementById('readyCtaSubtitle')

  // Validação extra: ficha + PT exige loja origem preenchida.
  const formato = state.session && state.session.formato
  let bloqueado = false
  let bloqueioMsg = ''
  if (state.ops.preTransf && formato === 'ficha') {
    const inp = document.getElementById('preOrigemInput')
    const v = parseInt(inp.value, 10)
    if (!v || v <= 0) { bloqueado = true; bloqueioMsg = 'Informe a loja origem' }
  }

  if (opsList.length === 0) {
    cta.disabled = true
    t.textContent = 'Selecione uma operação'
    s.textContent = state.session.linhas + ' linhas a processar'
  } else if (bloqueado) {
    cta.disabled = true
    t.textContent = bloqueioMsg
    s.textContent = opsList.join(' · ')
  } else {
    cta.disabled = false
    t.textContent = 'Pronto pra enviar'
    s.textContent = opsList.length + ' operações · ' + state.session.linhas + ' linhas processadas'
  }
}

// ───────────────────────────────────────────────────────────
// Sending / SSE
// ───────────────────────────────────────────────────────────
let abortCtrl = null

async function iniciarRun() {
  await rodar({ retry: false, linhas: null })
}

async function reenviarErros() {
  const linhas = (state.run && state.run.errors)
    ? Array.from(new Set(state.run.errors.map(e => e.linha).filter(n => n > 0)))
    : []
  if (linhas.length === 0) return alert('Nenhum erro com linha rastreável para reenviar.')
  await rodar({ retry: true, linhas })
}

async function rodar({ retry, linhas }) {
  state.logs = []
  state.filtro = 'tudo'
  document.querySelectorAll('.filter-tab').forEach(t => t.classList.toggle('active', t.dataset.filter === 'tudo'))
  document.getElementById('logList').innerHTML = ''
  document.getElementById('progFill').style.width = '0%'
  document.getElementById('statProc').textContent = '0/' + (retry ? linhas.length : state.session.linhas)
  document.getElementById('statOk').textContent = '0'
  document.getElementById('statWarn').textContent = '0'
  document.getElementById('statErr').textContent = '0'
  document.getElementById('statEta').textContent = '—'
  ;['cntTudo','cntOk','cntAviso','cntErro'].forEach(id => document.getElementById(id).textContent = '0')

  const opsLabel = []
  if (state.ops.alterarNCM) opsLabel.push('NCM')
  if (state.ops.linhaCompra) opsLabel.push('Compra')
  if (state.ops.linhaLoja) opsLabel.push('Sortimento')
  if (state.ops.linhaCompraCD) opsLabel.push('CD(' + state.cdTipo + ')')
  if (state.ops.preTransf) opsLabel.push('Pré-transferência')
  document.getElementById('sendingFilename').textContent = state.session.filename
  document.getElementById('sendingOp').textContent = opsLabel.join(' · ')

  state.run = {
    startedAt: Date.now(),
    totals: { ok: 0, warn: 0, erro: 0, processadas: 0, total: retry ? linhas.length : state.session.linhas },
    errors: [],
    opsUsed: opsLabel.join(' · '),
    retry,
  }

  setScreen('sending')
  setStatus('enviando', 'enviando')

  const url = retry ? '/api/retry' : '/api/run'
  const lojaOrigem = parseInt(document.getElementById('preOrigemInput').value, 10) || 0
  const body = {
    id: state.session.id,
    alterarNCM: state.ops.alterarNCM,
    linhaCompra: state.ops.linhaCompra,
    linhaCompraCD: state.ops.linhaCompraCD,
    linhaLoja: state.ops.linhaLoja,
    preTransferencia: state.ops.preTransf,
    cdTipo: state.cdTipo,
    lojaOrigem,
  }
  if (retry) body.linhas = linhas

  abortCtrl = new AbortController()
  try {
    const resp = await fetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(body),
      signal: abortCtrl.signal,
    })
    const reader = resp.body.getReader()
    const dec = new TextDecoder()
    let buf = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buf += dec.decode(value, { stream: true })
      const parts = buf.split('\n\n')
      buf = parts.pop()
      for (const part of parts) processSSEPart(part)
    }
  } catch (e) {
    if (e.name !== 'AbortError') {
      addLogEntry({ tipo: 'erro', mensagem: 'erro no stream: ' + e.message })
    }
  }
}

function processSSEPart(part) {
  let evtName = 'message', data = ''
  for (const line of part.split('\n')) {
    if (line.startsWith('event: ')) evtName = line.slice(7)
    if (line.startsWith('data: '))  data    = line.slice(6)
  }
  if (!data) return
  if (evtName === 'progress') {
    const p = JSON.parse(data)
    state.run.totals.processadas = p.atual
    state.run.totals.total = p.total
    document.getElementById('statProc').textContent = p.atual + '/' + p.total
    document.getElementById('progFill').style.width = (p.atual / p.total * 100) + '%'
    refreshETA()
    return
  }
  const evt = JSON.parse(data)
  if (evt.tipo === 'fim') {
    finalizarRun(evt.mensagem)
    return
  }
  addLogEntry(evt)
}

function refreshETA() {
  const r = state.run
  if (!r || r.totals.processadas === 0) return
  const elapsed = (Date.now() - r.startedAt) / 1000
  const rate = r.totals.processadas / elapsed
  const restante = r.totals.total - r.totals.processadas
  const eta = restante / rate
  document.getElementById('statEta').textContent = restante > 0 ? '~' + Math.max(1, Math.round(eta)) + 's restantes' : ''
}

function addLogEntry(evt) {
  state.logs.push(evt)
  // contadores
  if (evt.tipo === 'ok') state.run.totals.ok++
  else if (evt.tipo === 'aviso') state.run.totals.warn++
  else if (evt.tipo === 'erro') {
    state.run.totals.erro++
    state.run.errors.push(evt)
  }
  document.getElementById('statOk').textContent = state.run.totals.ok
  document.getElementById('statWarn').textContent = state.run.totals.warn
  document.getElementById('statErr').textContent = state.run.totals.erro
  document.getElementById('cntTudo').textContent = state.logs.length
  document.getElementById('cntOk').textContent = state.run.totals.ok
  document.getElementById('cntAviso').textContent = state.run.totals.warn
  document.getElementById('cntErro').textContent = state.run.totals.erro
  // render se filtro permite
  if (entryMatchesFiltro(evt)) renderLogRow(evt)
}

function entryMatchesFiltro(evt) {
  if (state.filtro === 'tudo') return true
  return evt.tipo === state.filtro
}

function renderLogRow(evt) {
  const list = document.getElementById('logList')
  const row = document.createElement('div')
  row.className = 'log-row ' + (evt.tipo || 'info')
  const ts = new Date().toLocaleTimeString('pt-BR', {hour:'2-digit',minute:'2-digit',second:'2-digit'})
  const linhaCell = (evt.linha && evt.tipo === 'erro')
    ? '<span class="linha-ref"><a title="linha na planilha">linha ' + evt.linha + '</a></span>'
    : '<span class="linha-ref"></span>'
  row.innerHTML =
    '<span class="ts">' + ts + '</span>' +
    '<span class="dot"></span>' +
    '<span class="op">' + escapeHtml(evt.op || '') + '</span>' +
    '<span class="msg">' + escapeHtml(evt.mensagem || '') + '</span>' +
    linhaCell
  list.appendChild(row)
  list.scrollTop = list.scrollHeight
}

function setFiltro(f) {
  state.filtro = f
  document.querySelectorAll('.filter-tab').forEach(t => t.classList.toggle('active', t.dataset.filter === f))
  // re-render só os que batem
  const list = document.getElementById('logList')
  list.innerHTML = ''
  for (const e of state.logs) {
    if (entryMatchesFiltro(e)) renderLogRow(e)
  }
}

function parar() {
  fetch('/api/cancel', { method: 'POST' }).catch(() => {})
  document.getElementById('btnParar').disabled = true
}

function finalizarRun(resumoStr) {
  let resumo = { ok: 0, erros: 0, avisos: 0 }
  try { if (resumoStr) resumo = JSON.parse(resumoStr) } catch {}
  state.run.totals.ok = resumo.ok
  state.run.totals.erro = resumo.erros
  state.run.totals.warn = resumo.avisos
  state.run.preTransferenciaKeys = resumo.preTransferenciaKeys || []
  document.getElementById('btnParar').disabled = false
  goToDone()
}

// ───────────────────────────────────────────────────────────
// Done
// ───────────────────────────────────────────────────────────
function goToDone() {
  setScreen('done')
  setStatus('concluído', 'concluido')
  const r = state.run
  const total = (r.totals.ok|0) + (r.totals.warn|0) + (r.totals.erro|0)
  document.getElementById('doneFilename').textContent = state.session.filename
  document.getElementById('doneOps').textContent = r.opsUsed || '—'
  const dur = ((Date.now() - r.startedAt) / 1000)
  document.getElementById('doneDuration').textContent = dur < 60
    ? Math.round(dur) + 's'
    : Math.floor(dur/60) + 'm ' + Math.round(dur%60) + 's'
  document.getElementById('doneTotal').textContent = total
  document.getElementById('doneOk').textContent   = r.totals.ok
  document.getElementById('doneWarn').textContent = r.totals.warn
  document.getElementById('doneErr').textContent  = r.totals.erro
  // stack bar
  const stack = document.getElementById('doneStack')
  stack.innerHTML = ''
  if (total > 0) {
    const okPct   = r.totals.ok   / total * 100
    const warnPct = r.totals.warn / total * 100
    const errPct  = r.totals.erro / total * 100
    if (okPct   > 0) stack.insertAdjacentHTML('beforeend', '<div class="b-ok"   style="width:' + okPct + '%"></div>')
    if (warnPct > 0) stack.insertAdjacentHTML('beforeend', '<div class="b-warn" style="width:' + warnPct + '%"></div>')
    if (errPct  > 0) stack.insertAdjacentHTML('beforeend', '<div class="b-err"  style="width:' + errPct + '%"></div>')
  } else {
    stack.innerHTML = '<div style="width:100%;background:var(--bg-3)"></div>'
  }

  // pt keys card
  const ptKeysCard = document.getElementById('ptKeysCard')
  const ptKeysList = document.getElementById('ptKeysList')
  const ptKeys = state.run.preTransferenciaKeys || []
  if (ptKeys.length > 0) {
    ptKeysCard.classList.remove('hidden')
    ptKeysList.innerHTML = ptKeys.map(k =>
      '<span class="key-pill"><span class="key-icon">#</span>' + k + '</span>'
    ).join('')
  } else {
    ptKeysCard.classList.add('hidden')
  }

  // erros card
  const card = document.getElementById('errorsCard')
  const body = document.getElementById('errorsBody')
  body.innerHTML = ''
  const errs = (r.errors || []).filter(e => e.tipo === 'erro')
  if (errs.length === 0) {
    card.classList.add('hidden')
    return
  }
  card.classList.remove('hidden')
  document.getElementById('errorsSub').textContent = errs.length + ' itens'
  for (const e of errs) {
    const div = document.createElement('div')
    div.className = 'err-item'
    div.innerHTML =
      '<span class="linha">linha ' + (e.linha || '—') + '</span>' +
      '<span class="ean">' + escapeHtml(e.ean || '') + '</span>' +
      '<span class="motivo">' + escapeHtml(e.mensagem || '') + '</span>' +
      '<span><span class="op-tag">' + escapeHtml(e.op || '') + '</span></span>'
    body.appendChild(div)
  }
}

function resetUpload() {
  goToIdle()
}

// ───────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────
function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function formatSize(b) {
  if (!b) return '0 B'
  if (b < 1024) return b + ' B'
  if (b < 1024*1024) return (b/1024).toFixed(1) + ' KB'
  return (b/1024/1024).toFixed(1) + ' MB'
}

boot()
</script>
</body>
</html>`
