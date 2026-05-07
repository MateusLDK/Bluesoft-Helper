package main

var htmlUI = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<title>Bluesoft Uploader</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=Inter:wght@400;500;600;700&display=swap');

  :root {
    --bg:        #0e1117;
    --surface:   #161b25;
    --elevated:  #1c222e;
    --border:    #252b3a;
    --border2:   #303749;
    --text:      #d8deec;
    --text-str:  #f0f3fb;
    --muted:     #707a99;
    --muted-2:   #8b94b1;
    --accent:    #5b9eff;
    --accent2:   #8b76ff;
    --ok:        #4ddb9c;
    --warn:      #f0c264;
    --err:       #ff6b6b;
    --radius:    8px;
    --radius-lg: 10px;
    --shadow:    0 1px 0 rgba(255,255,255,.03) inset, 0 4px 14px rgba(0,0,0,.18);
  }

  * { box-sizing: border-box; margin: 0; padding: 0; }

  body {
    font-family: 'Inter', system-ui, sans-serif;
    background:
      radial-gradient(1200px 600px at 90% -10%, rgba(91,158,255,.06), transparent 60%),
      radial-gradient(900px 500px at -10% 110%, rgba(139,118,255,.05), transparent 60%),
      var(--bg);
    color: var(--text);
    height: 100vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    -webkit-font-smoothing: antialiased;
  }

  /* ── Header ── */
  header {
    padding: 16px 26px;
    border-bottom: 1px solid var(--border);
    background: rgba(22,27,37,.75);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    gap: 14px;
    flex-shrink: 0;
  }

  .logo {
    width: 34px; height: 34px;
    background: linear-gradient(135deg, var(--accent), var(--accent2));
    border-radius: 9px;
    display: flex; align-items: center; justify-content: center;
    color: #fff; font-size: 16px; font-weight: 700;
    box-shadow: 0 4px 14px rgba(91,158,255,.25);
  }

  .brand h1 { font-size: 15px; font-weight: 600; color: var(--text-str); letter-spacing: -.2px; }
  .brand .tag { font-size: 11px; color: var(--muted); margin-top: 1px; }
  header .spacer { flex: 1; }
  header .conn-pill {
    display: inline-flex; align-items: center; gap: 6px;
    font-size: 11px; color: var(--muted-2);
    padding: 5px 10px; border: 1px solid var(--border2); border-radius: 99px;
    font-family: 'IBM Plex Mono', monospace;
  }
  header .conn-pill::before {
    content: ''; width: 6px; height: 6px; border-radius: 50%;
    background: var(--ok); box-shadow: 0 0 8px var(--ok);
  }

  main {
    display: grid;
    grid-template-columns: 420px 1fr;
    flex: 1;
    overflow: hidden;
    transition: grid-template-columns .35s ease;
  }
  main.running { grid-template-columns: 280px 1fr; }

  /* ── Painel esquerdo ── */
  .left {
    border-right: 1px solid var(--border);
    overflow-y: auto;
    padding: 22px 20px 18px;
    display: flex;
    flex-direction: column;
    gap: 22px;
  }
  .left::-webkit-scrollbar { width: 6px; }
  .left::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 99px; }

  .section-head {
    display: flex; align-items: baseline; gap: 8px;
    margin-bottom: 10px;
  }
  .section-head h2 {
    font-size: 12px; font-weight: 600;
    letter-spacing: .8px; text-transform: uppercase;
    color: var(--muted-2);
  }
  .section-head .hint {
    font-size: 11px; color: var(--muted); font-weight: 400;
  }

  .field { display: flex; flex-direction: column; gap: 5px; margin-bottom: 10px; }
  .field:last-child { margin-bottom: 0; }
  label.field-label {
    font-size: 11px; color: var(--muted-2); font-weight: 500;
    display: flex; align-items: center; gap: 6px;
  }

  input[type=text], input[type=password] {
    background: var(--bg);
    border: 1px solid var(--border2);
    border-radius: var(--radius);
    color: var(--text);
    font-family: 'IBM Plex Mono', monospace;
    font-size: 13px;
    padding: 9px 11px;
    outline: none;
    width: 100%;
    transition: border-color .15s, box-shadow .15s, background .15s;
  }
  input::placeholder { color: var(--muted); opacity: .6; }
  input:hover { border-color: var(--border2); background: rgba(255,255,255,.01); }
  input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px rgba(91,158,255,.12);
  }

  /* ── Operações ── */
  .ops-grid {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .op-check {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid var(--border2);
    border-radius: var(--radius);
    cursor: pointer;
    transition: all .18s ease;
    user-select: none;
    background: var(--elevated);
  }
  .op-check:hover { border-color: rgba(91,158,255,.5); background: rgba(91,158,255,.04); transform: translateY(-1px); }
  .op-check.checked {
    border-color: var(--accent);
    background: linear-gradient(180deg, rgba(91,158,255,.10), rgba(91,158,255,.04));
    box-shadow: 0 0 0 1px rgba(91,158,255,.18);
  }
  .op-check input[type=checkbox] { display: none; }

  .op-check .checkmark {
    width: 18px; height: 18px;
    border: 1.5px solid var(--border2);
    border-radius: 5px;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    transition: all .15s ease;
    color: #fff; font-size: 11px; font-weight: 700;
  }
  .op-check.checked .checkmark {
    background: var(--accent);
    border-color: var(--accent);
    box-shadow: 0 0 0 3px rgba(91,158,255,.18);
  }

  .op-check .op-info { flex: 1; min-width: 0; }
  .op-check .op-name { font-size: 13px; font-weight: 500; color: var(--text-str); }
  .op-check .op-desc { font-size: 11px; color: var(--muted); margin-top: 2px; }

  /* ── Sub-opção UN/CX ── */
  .sub-radio {
    display: flex;
    gap: 6px;
    margin-top: -2px;
    padding: 8px 14px 8px 44px;
    border-left: 2px solid rgba(91,158,255,.25);
    margin-left: 14px;
    animation: slideDown .2s ease;
  }
  .sub-radio.hidden { display: none; }
  .sub-radio .label-row {
    font-size: 11px; color: var(--muted-2);
    margin-right: 4px; align-self: center;
  }
  .sub-radio label {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 46px;
    padding: 5px 12px;
    border: 1px solid var(--border2);
    border-radius: 99px;
    cursor: pointer;
    font-size: 11px;
    font-family: 'IBM Plex Mono', monospace;
    font-weight: 500;
    color: var(--muted-2);
    transition: all .15s ease;
    user-select: none;
  }
  .sub-radio label:hover { border-color: var(--accent); color: var(--text); }
  .sub-radio input[type=radio] { display: none; }
  .sub-radio label.active {
    border-color: var(--accent);
    background: rgba(91,158,255,.12);
    color: var(--text-str);
  }

  @keyframes slideDown {
    from { opacity: 0; transform: translateY(-4px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* ── File drop ── */
  .file-drop {
    border: 1.5px dashed var(--border2);
    border-radius: var(--radius);
    padding: 22px 16px;
    text-align: center;
    cursor: pointer;
    position: relative;
    transition: all .18s ease;
    background: var(--elevated);
  }
  .file-drop:hover {
    border-color: var(--accent);
    background: rgba(91,158,255,.04);
    transform: translateY(-1px);
  }
  .file-drop.has-file {
    border-style: solid;
    border-color: var(--ok);
    background: rgba(77,219,156,.05);
  }
  .file-drop input[type=file] { position: absolute; inset: 0; opacity: 0; cursor: pointer; width: 100%; height: 100%; }
  .file-drop .drop-icon {
    font-size: 26px; margin-bottom: 6px;
    transition: transform .2s ease;
  }
  .file-drop:hover .drop-icon { transform: translateY(-2px); }
  .file-drop .drop-hint { font-size: 12px; color: var(--muted-2); }
  .file-drop .drop-hint strong { color: var(--text); font-weight: 600; }
  .file-drop .drop-name {
    font-size: 11px; color: var(--ok); margin-top: 6px;
    font-family: 'IBM Plex Mono', monospace; word-break: break-all;
    font-weight: 500;
  }
  .file-drop .drop-sub { font-size: 10px; color: var(--muted); margin-top: 4px; }

  /* ── Botões ── */
  .btn-row { display: flex; gap: 8px; margin-top: auto; padding-top: 4px; }
  .btn-send {
    background: linear-gradient(135deg, var(--accent), var(--accent2));
    border: none; border-radius: var(--radius);
    color: #fff;
    font-family: 'Inter', sans-serif;
    font-size: 13px; font-weight: 600;
    padding: 12px; cursor: pointer; flex: 1;
    transition: all .15s ease;
    letter-spacing: .2px;
    box-shadow: 0 4px 14px rgba(91,158,255,.22);
  }
  .btn-send:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 20px rgba(91,158,255,.32);
  }
  .btn-send:active:not(:disabled) { transform: translateY(0); }
  .btn-send:disabled { opacity: .35; cursor: not-allowed; box-shadow: none; }

  .btn-stop {
    background: transparent;
    border: 1px solid var(--err);
    border-radius: var(--radius);
    color: var(--err);
    font-family: 'Inter', sans-serif;
    font-size: 13px; font-weight: 600;
    padding: 12px 16px; cursor: pointer;
    transition: all .15s ease;
  }
  .btn-stop:hover:not(:disabled) {
    background: rgba(255,107,107,.10);
    transform: translateY(-1px);
  }
  .btn-stop:disabled { opacity: .25; cursor: not-allowed; border-color: var(--border2); color: var(--muted); }

  /* Em execução: stop fica sólido em vermelho pra chamar atenção */
  main.running .btn-stop:not(:disabled) {
    background: var(--err);
    color: #fff;
    box-shadow: 0 4px 14px rgba(255,107,107,.28);
  }
  main.running .btn-stop:not(:disabled):hover {
    background: #ff5252;
    box-shadow: 0 6px 20px rgba(255,107,107,.42);
  }

  /* ── Painel direito ── */
  .right { display: flex; flex-direction: column; overflow: hidden; }

  .log-header {
    padding: 12px 22px;
    border-bottom: 1px solid var(--border);
    background: rgba(22,27,37,.6);
    display: flex; align-items: center; gap: 12px;
    flex-shrink: 0;
    min-height: 50px;
  }

  .log-title {
    font-size: 11px; font-weight: 600;
    letter-spacing: .8px; text-transform: uppercase;
    color: var(--muted-2);
  }

  .progress-wrap {
    flex: 1; height: 4px;
    background: var(--border);
    border-radius: 99px;
    overflow: hidden;
    position: relative;
  }
  .progress-fill {
    height: 100%; width: 0%;
    background: linear-gradient(90deg, var(--accent), var(--accent2));
    transition: width .3s ease;
    border-radius: 99px;
  }

  .stat {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 11px; color: var(--muted-2);
    white-space: nowrap; min-width: 50px; text-align: right;
  }

  .badge {
    display: inline-flex; align-items: center; gap: 4px;
    font-size: 11px; font-weight: 600;
    padding: 3px 9px; border-radius: 99px;
    font-family: 'IBM Plex Mono', monospace;
  }
  .badge.ok  { background: rgba(77,219,156,.12); color: var(--ok); }
  .badge.err { background: rgba(255,107,107,.12); color: var(--err); }
  .badge.warn { background: rgba(240,194,100,.12); color: var(--warn); }

  .log-action {
    display: inline-flex; align-items: center; gap: 6px;
    font-size: 11px; font-weight: 500;
    padding: 5px 11px; border-radius: 99px;
    background: transparent;
    border: 1px solid var(--border2);
    color: var(--muted-2);
    font-family: 'Inter', sans-serif;
    cursor: pointer;
    transition: all .15s ease;
  }
  .log-action:hover {
    border-color: var(--accent);
    color: var(--text-str);
    background: rgba(91,158,255,.06);
  }
  .log-action svg { width: 13px; height: 13px; }

  .log-body {
    flex: 1; overflow-y: auto;
    padding: 16px 22px;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 12.5px; line-height: 1.85;
  }
  .log-body::-webkit-scrollbar { width: 6px; }
  .log-body::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 99px; }

  .ll {
    display: block;
    animation: fadeInLog .25s ease;
    padding: 1px 0;
  }
  .ll .ts { color: var(--muted); margin-right: 10px; opacity: .8; }
  .ll.ok    { color: var(--ok); }
  .ll.erro  { color: var(--err); }
  .ll.info  { color: var(--muted-2); }
  .ll.aviso { color: var(--warn); }

  @keyframes fadeInLog {
    from { opacity: 0; transform: translateX(-4px); }
    to { opacity: 1; transform: translateX(0); }
  }

  /* ── Empty state (onboarding) ── */
  .empty {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; padding: 40px 20px;
    height: 100%; max-width: 420px; margin: 0 auto;
    text-align: center;
  }
  .empty .empty-icon {
    width: 56px; height: 56px;
    border-radius: 14px;
    background: linear-gradient(135deg, rgba(91,158,255,.18), rgba(139,118,255,.12));
    display: flex; align-items: center; justify-content: center;
    font-size: 24px; margin-bottom: 16px;
    border: 1px solid rgba(91,158,255,.2);
  }
  .empty h3 {
    font-family: 'Inter', sans-serif; font-size: 15px;
    color: var(--text-str); font-weight: 600; margin-bottom: 6px;
  }
  .empty p {
    font-family: 'Inter', sans-serif; font-size: 12.5px;
    color: var(--muted-2); margin-bottom: 22px; line-height: 1.55;
  }
  .empty .steps {
    display: flex; flex-direction: column; gap: 8px;
    width: 100%;
  }
  .empty .step {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 14px;
    background: var(--elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-family: 'Inter', sans-serif;
    font-size: 12.5px; color: var(--text);
    text-align: left;
  }
  .empty .step .num {
    width: 22px; height: 22px;
    border-radius: 50%;
    background: rgba(91,158,255,.15);
    color: var(--accent); font-weight: 600;
    display: flex; align-items: center; justify-content: center;
    font-size: 11px; flex-shrink: 0;
    font-family: 'IBM Plex Mono', monospace;
  }

  hr.sep { border: none; border-top: 1px solid var(--border); margin: 0; }
</style>
</head>
<body>

<header>
  <div class="logo">⬆</div>
  <div class="brand">
    <h1>Bluesoft Uploader</h1>
    <div class="tag">Importação de produtos via planilha</div>
  </div>
  <div class="spacer"></div>
  <span class="conn-pill" id="connPill">pronto</span>
</header>

<main>
  <div class="left">

    <div>
      <div class="section-head">
        <h2>Credenciais</h2>
        <span class="hint">acesso à API</span>
      </div>
      <div class="field">
        <label class="field-label">Tenant</label>
        <input type="text" id="tenant" placeholder="ex.: minipreco">
      </div>
      <div class="field">
        <label class="field-label">Client ID</label>
        <input type="text" id="clientId" placeholder="seu client id">
      </div>
      <div class="field">
        <label class="field-label">Client Secret</label>
        <input type="password" id="clientSecret" placeholder="••••••••••••">
      </div>
    </div>

    <hr class="sep">

    <div>
      <div class="section-head">
        <h2>Operações</h2>
        <span class="hint">selecione uma ou mais</span>
      </div>
      <div class="ops-grid">

        <label class="op-check" id="ncm-label" onclick="toggleOp('alterarNCM', this)">
          <input type="checkbox" id="alterarNCM">
          <div class="checkmark" id="ncm-mark"></div>
          <div class="op-info">
            <div class="op-name">Alterar NCM</div>
            <div class="op-desc">Atualiza o NCM no cadastro do produto</div>
          </div>
        </label>

        <label class="op-check" id="lc-label" onclick="toggleOp('linhaCompra', this)">
          <input type="checkbox" id="linhaCompra">
          <div class="checkmark" id="lc-mark"></div>
          <div class="op-info">
            <div class="op-name">Linha de Compra</div>
            <div class="op-desc">Cadastra produto nas lojas selecionadas</div>
          </div>
        </label>

        <label class="op-check" id="lccd-label" onclick="toggleOp('linhaCompraCD', this)">
          <input type="checkbox" id="linhaCompraCD">
          <div class="checkmark" id="lccd-mark"></div>
          <div class="op-info">
            <div class="op-name">Linha de Compra CD</div>
            <div class="op-desc">Agrupa lojas por centro de distribuição</div>
          </div>
        </label>

        <div class="sub-radio hidden" id="cdTipoWrap" onclick="event.stopPropagation()">
          <span class="label-row">unidade:</span>
          <label class="active" id="cdTipoUNLabel">
            <input type="radio" name="cdTipo" value="UN" checked onchange="onCdTipo(this)">
            UN
          </label>
          <label id="cdTipoCXLabel">
            <input type="radio" name="cdTipo" value="CX" onchange="onCdTipo(this)">
            CX
          </label>
        </div>

        <label class="op-check" id="ll-label" onclick="toggleOp('linhaLoja', this)">
          <input type="checkbox" id="linhaLoja">
          <div class="checkmark" id="ll-mark"></div>
          <div class="op-info">
            <div class="op-name">Sortimento / Linha de Loja</div>
            <div class="op-desc">Define parâmetros de sortimento por loja</div>
          </div>
        </label>

      </div>
    </div>

    <hr class="sep">

    <div>
      <div class="section-head">
        <h2>Planilha</h2>
        <span class="hint">.xlsx do TOYNG</span>
      </div>
      <div class="file-drop" id="fileDrop">
        <input type="file" id="fileInput" accept=".xlsx" onchange="onFile(this)">
        <div class="drop-icon">📂</div>
        <div class="drop-hint">Clique ou arraste o <strong>.xlsx</strong> aqui</div>
        <div class="drop-sub">aba "Preencher", cabeçalhos na linha 8</div>
        <div class="drop-name" id="fileName"></div>
      </div>
    </div>

    <div class="btn-row">
      <button class="btn-send" id="btnEnviar" onclick="enviar()">▶ Enviar para API</button>
      <button class="btn-stop" id="btnParar" onclick="parar()" disabled>⏹ Parar</button>
    </div>

  </div>

  <div class="right">
    <div class="log-header">
      <span class="log-title">Log</span>
      <div class="progress-wrap"><div class="progress-fill" id="prog"></div></div>
      <span class="stat" id="stat">—</span>
      <span class="badge ok"  id="bOk"  style="display:none">0 ok</span>
      <span class="badge err" id="bErr" style="display:none">0 erros</span>
      <button class="log-action" onclick="abrirLogs()" title="Abrir pasta com logs detalhados de cada execução">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        Abrir logs
      </button>
    </div>
    <div class="log-body" id="logBody">
      <div class="empty">
        <div class="empty-icon">✨</div>
        <h3>Tudo pronto pra começar</h3>
        <p>Preencha as credenciais, escolha as operações, selecione a planilha e clique em <strong>Enviar</strong>.</p>
        <div class="steps">
          <div class="step"><div class="num">1</div>Preencher tenant, client ID e secret</div>
          <div class="step"><div class="num">2</div>Marcar pelo menos uma operação</div>
          <div class="step"><div class="num">3</div>Carregar a planilha .xlsx</div>
          <div class="step"><div class="num">4</div>Clicar em Enviar e acompanhar aqui</div>
        </div>
      </div>
    </div>
  </div>
</main>

<script>
// ── Toggle checkbox visual ──
function toggleOp(id, label) {
  const cb = document.getElementById(id)
  cb.checked = !cb.checked
  label.classList.toggle('checked', cb.checked)
  const mark = label.querySelector('.checkmark')
  mark.textContent = cb.checked ? '✓' : ''
  if (id === 'linhaCompraCD') {
    document.getElementById('cdTipoWrap').classList.toggle('hidden', !cb.checked)
  }
}

function onCdTipo(input) {
  document.getElementById('cdTipoUNLabel').classList.toggle('active', input.value === 'UN')
  document.getElementById('cdTipoCXLabel').classList.toggle('active', input.value === 'CX')
}

function parar() {
  const btn = document.getElementById('btnParar')
  btn.disabled = true
  fetch('/api/cancel', { method: 'POST' }).catch(() => {})
}

function abrirLogs() {
  fetch('/api/abrir-logs').catch(() => alert('Não foi possível abrir a pasta de logs.'))
}

// ── Arquivo ──
function onFile(input) {
  const f = input.files[0]
  if (!f) return
  setFile(f)
}

function setFile(f) {
  window._file = f
  document.getElementById('fileName').textContent = f.name
  document.getElementById('fileDrop').classList.add('has-file')
}

const drop = document.getElementById('fileDrop')
drop.addEventListener('dragover', e => { e.preventDefault(); drop.style.borderColor = 'var(--accent)' })
drop.addEventListener('dragleave', () => { drop.style.borderColor = '' })
drop.addEventListener('drop', e => {
  e.preventDefault()
  drop.style.borderColor = ''
  const f = e.dataTransfer.files[0]
  if (f && f.name.endsWith('.xlsx')) setFile(f)
})

// ── Log ──
function addLog(tipo, msg) {
  const body = document.getElementById('logBody')
  const empty = body.querySelector('.empty')
  if (empty) empty.remove()
  const ts = new Date().toLocaleTimeString('pt-BR', {hour:'2-digit',minute:'2-digit',second:'2-digit'})
  const line = document.createElement('span')
  line.className = 'll ' + tipo
  const tsSpan = document.createElement('span')
  tsSpan.className = 'ts'
  tsSpan.textContent = ts
  line.appendChild(tsSpan)
  line.appendChild(document.createTextNode(msg))
  body.appendChild(line)
  body.scrollTop = body.scrollHeight
}

// ── Enviar ──
async function enviar() {
  const tenant       = document.getElementById('tenant').value.trim()
  const clientId     = document.getElementById('clientId').value.trim()
  const clientSecret = document.getElementById('clientSecret').value.trim()
  const ncm          = document.getElementById('alterarNCM').checked
  const lc           = document.getElementById('linhaCompra').checked
  const lccd         = document.getElementById('linhaCompraCD').checked
  const ll           = document.getElementById('linhaLoja').checked

  if (!tenant || !clientId || !clientSecret) { alert('Preencha as credenciais.'); return }
  if (!ncm && !lc && !lccd && !ll) { alert('Selecione pelo menos uma operação.'); return }
  if (!window._file) { alert('Selecione um arquivo .xlsx.'); return }

  document.getElementById('logBody').innerHTML = ''
  document.getElementById('prog').style.width = '0%'
  document.getElementById('stat').textContent = '—'
  document.getElementById('bOk').style.display = 'none'
  document.getElementById('bErr').style.display = 'none'
  document.getElementById('btnEnviar').disabled = true
  document.getElementById('btnParar').disabled = false
  document.querySelector('main').classList.add('running')

  const cdTipo = document.querySelector('input[name="cdTipo"]:checked')?.value || 'UN'

  const form = new FormData()
  form.append('file', window._file)
  form.append('tenant', tenant)
  form.append('clientId', clientId)
  form.append('clientSecret', clientSecret)
  form.append('alterarNCM',    ncm  ? 'true' : 'false')
  form.append('linhaCompra',   lc   ? 'true' : 'false')
  form.append('linhaCompraCD', lccd ? 'true' : 'false')
  form.append('linhaLoja',     ll   ? 'true' : 'false')
  form.append('cdTipo',        cdTipo)

  const resp = await fetch('/api/upload', { method: 'POST', body: form })
  const reader = resp.body.getReader()
  const dec = new TextDecoder()
  let buf = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += dec.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop()

    for (const part of parts) {
      let evtName = 'message', data = ''
      for (const line of part.split('\n')) {
        if (line.startsWith('event: ')) evtName = line.slice(7)
        if (line.startsWith('data: '))  data    = line.slice(6)
      }
      if (!data) continue

      if (evtName === 'progress') {
        const p = JSON.parse(data)
        const pct = Math.round(p.atual / p.total * 100)
        document.getElementById('prog').style.width = pct + '%'
        document.getElementById('stat').textContent = p.atual + '/' + p.total
        continue
      }

      const e = JSON.parse(data)
      if (e.tipo === 'fim') {
        document.getElementById('btnEnviar').disabled = false
        document.getElementById('btnParar').disabled = true
        document.querySelector('main').classList.remove('running')
        if (e.mensagem) {
          const r = JSON.parse(e.mensagem)
          const bOk  = document.getElementById('bOk')
          const bErr = document.getElementById('bErr')
          bOk.textContent  = r.ok    + ' ok';    bOk.style.display  = 'inline-flex'
          bErr.textContent = r.erros + ' erros'; bErr.style.display = 'inline-flex'
        }
      } else {
        addLog(e.tipo, e.mensagem)
      }
    }
  }
  document.getElementById('btnEnviar').disabled = false
  document.getElementById('btnParar').disabled = true
  document.querySelector('main').classList.remove('running')
}

// Pré-preenche do .env
fetch('/api/env').then(r => r.json()).then(d => {
  if (d.tenant)   document.getElementById('tenant').value   = d.tenant
  if (d.clientId) document.getElementById('clientId').value = d.clientId
}).catch(() => {})
</script>
</body>
</html>`
