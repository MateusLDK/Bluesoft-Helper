import { toast } from './toast.js'
import { api } from './api.js'

// ───────────────────────────────────────────────────────────
// Constantes
// ───────────────────────────────────────────────────────────
const TIPO = { OK: 'ok', AVISO: 'aviso', ERRO: 'erro', INFO: 'info', FIM: 'fim' }
const MODO = { DADOS: 'dados', FOTOS: 'fotos' }

// Fonte única das operações: rótulo longo (tela ready) e curto (tela sending).
const OPS = [
  { key: 'alterarNCM',      label: 'Alterar NCM',       short: 'NCM' },
  { key: 'alterarSubgrupo', label: 'Alterar Subgrupo',  short: 'Subgrupo' },
  { key: 'linhaCompra',     label: 'Linha de Compra',   short: 'Compra' },
  { key: 'linhaLoja',       label: 'Sortimento',        short: 'Sortimento' },
  { key: 'linhaCompraCD',   label: 'Linha CD',          short: 'CD' },
  { key: 'preTransf',       label: 'Pré-transferência', short: 'Pré-transferência' },
]
const OP_KEYS = OPS.map(o => o.key)
const emptyOps = () => Object.fromEntries(OP_KEYS.map(k => [k, false]))
const selectedOps = () => OPS.filter(o => state.ops[o.key])
const opLabel = o => o.key === 'linhaCompraCD' ? o.label + ' (' + state.cdTipo + ')' : o.label
const opShort = o => o.key === 'linhaCompraCD' ? 'CD(' + state.cdTipo + ')' : o.short

// ───────────────────────────────────────────────────────────
// State machine
// ───────────────────────────────────────────────────────────
const state = {
  current: null,
  configured: false,
  tenant: '',
  session: null,            // { id, filename, size, linhas, avisos }
  ops: emptyOps(),
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
    const [setup, status] = await Promise.all([api('/api/setup'), api('/api/status')])
    if (status.needsRestart) {
      document.getElementById('restartOverlay').classList.remove('hidden')
      return
    }
    state.configured = !!setup.configured
    setTenantPill(setup.tenant || '')
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

// Regras de validação client-side. O teste de conexão continua sendo a verdade;
// isto só evita enviar credenciais obviamente malformadas.
const SETUP_RULES = {
  setupTenant: {
    ok: v => /^[a-z0-9-]{2,}$/.test(v),
    msg: 'use apenas letras minúsculas, números e hífen (mín. 2)',
  },
  setupClientId: {
    ok: v => v.length >= 6 && !/\s/.test(v),
    msg: 'sem espaços, mínimo 6 caracteres',
  },
  setupSecret: {
    ok: v => v.length >= 6 && !/\s/.test(v),
    msg: 'sem espaços, mínimo 6 caracteres',
  },
}

function setupChanged() {
  let allValid = true
  for (const [id, rule] of Object.entries(SETUP_RULES)) {
    const inp = document.getElementById(id)
    const errEl = document.getElementById('err-' + id)
    const v = inp.value.trim()
    if (v === '') {                 // vazio: invalida sem mostrar erro
      errEl.textContent = ''
      inp.classList.remove('invalid')
      allValid = false
    } else if (rule.ok(v)) {
      errEl.textContent = ''
      inp.classList.remove('invalid')
    } else {
      errEl.textContent = rule.msg
      inp.classList.add('invalid')
      allValid = false
    }
  }
  const cta = document.getElementById('setupCta')
  cta.disabled = !allValid
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
    const d = await api('/api/setup/test', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({tenant, clientId, clientSecret})
    })
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
    await api('/api/setup/save', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({tenant, clientId, clientSecret})
    })
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
const modoConfig = {
  dados: {
    accept: '.xlsx',
    title: 'Arraste o .xlsx ou clique para selecionar',
    hint: 'aba "Preencher" · cabeçalhos na linha 8',
  },
  fotos: {
    accept: '.zip,.rar',
    title: 'Arraste o .zip/.rar ou clique para selecionar',
    hint: 'arquivo com as fotos dos produtos',
  },
}

function onModoChange(modo) {
  state.idleModo = modo
  document.getElementById('radioLabelDados').classList.toggle('active', modo === MODO.DADOS)
  document.getElementById('radioLabelFotos').classList.toggle('active', modo === MODO.FOTOS)
  const cfg = modoConfig[modo]
  document.getElementById('fileInput').accept = cfg.accept
  document.getElementById('fileInput').value = ''
  document.getElementById('dropTitle').textContent = cfg.title
  document.getElementById('dropHint').textContent = cfg.hint
  const s = document.getElementById('fotosStatusIdle')
  s.style.display = 'none'
  s.textContent = ''
}

function goToIdle() {
  state.session = null
  state.ops = emptyOps()
  state.cdTipo = 'UN'
  state.idleModo = MODO.DADOS
  document.querySelector('input[name=idleModo][value=dados]').checked = true
  onModoChange(MODO.DADOS)
  setScreen('idle')
  setStatus('aguardando', 'aguardando')
}

async function onFileChange(file) {
  if (!file) return
  if (state.idleModo === MODO.FOTOS) {
    await uploadFotosInline(file)
  } else {
    await onFile(file)
  }
}

async function onFile(file) {
  if (!file) return
  const dz = document.getElementById('dropzone')
  dz.classList.add('busy')
  const fd = new FormData()
  fd.append('file', file)
  try {
    const d = await api('/api/upload', { method: 'POST', body: fd, timeout: 60000 })
    state.session = d
    goToReady()
  } catch (e) {
    toast('Erro ao validar planilha: ' + e.message, { kind: 'erro' })
  } finally {
    dz.classList.remove('busy')
  }
}

async function uploadFotosInline(file) {
  const ext = file.name.split('.').pop().toLowerCase()
  if (ext !== 'zip' && ext !== 'rar') {
    toast('Selecione um arquivo .zip ou .rar', { kind: 'aviso' })
    return
  }
  const statusEl = document.getElementById('fotosStatusIdle')
  statusEl.style.display = 'block'
  statusEl.replaceChildren()
  document.getElementById('fileInput').value = ''
  const overlay = document.getElementById('fotosOverlay')
  overlay.classList.remove('hidden')
  try {
    const fd = new FormData()
    fd.append('arquivo', file)
    const r = await fetch('/fotos/upload', { method: 'POST', body: fd })
    if (!r.ok) {
      const msg = await r.text()
      throw new Error(msg.trim() || 'Erro desconhecido')
    }
    const erros = (r.headers.get('X-Nao-Encontrados') || '').split(',').filter(Boolean)

    const blob = await r.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = 'importacao_fotos.csv'
    document.body.appendChild(a); a.click(); a.remove()
    URL.revokeObjectURL(url)

    const okLine = document.createElement('div')
    okLine.className = 'txt-ok'
    okLine.textContent = '✓ CSV baixado'
    statusEl.appendChild(okLine)
    if (erros.length > 0) {
      const head = document.createElement('div')
      head.className = 'txt-erro'
      head.textContent = '✗ ' + erros.length + ' não encontrada' + (erros.length !== 1 ? 's' : '') + ' no banco:'
      const box = document.createElement('div')
      box.className = 'fotos-erros-box'
      box.textContent = erros.join('\n')
      statusEl.append(head, box)
    }
  } catch (e) {
    const err = document.createElement('div')
    err.className = 'txt-erro'
    err.textContent = e.message
    statusEl.appendChild(err)
  } finally {
    overlay.classList.add('hidden')
  }
}

function abrirLogs() {
  fetch('/api/abrir-logs').catch(() => toast('Não foi possível abrir a pasta de logs.', { kind: 'erro' }))
}

// ───────────────────────────────────────────────────────────
// Árvore mercadológica (modal de download)
// ───────────────────────────────────────────────────────────
// EDITAR AQUI quando a lista de departamentos mudar.
const DEPARTAMENTOS = [
  'BANHEIRO',
  'BOMBONIERE',
  'BRICOLAGEM',
  'BRINQUEDOS',
  'CAMEBA',
  'COZINHA NOVA',
  'CUIDADOS PESSOAIS',
  'DECORACAO',
  'ELETRO E INFORMATICA',
  'FLORICULTURA',
  'LAZER',
  'LIMPEZA NOVA',
  'MODA E ACESSORIOS',
  'NATAL',
  'ORGANIZACAO',
  'PAPELARIA',
  'PET CARE',
  'POTE',
  'UTILIDADES NOVA'
]

let arvoreSelectPopulado = false

// ───────────────────────────────────────────────────────────
// Modais acessíveis: foco preso, ESC fecha, foco devolvido ao gatilho.
// ───────────────────────────────────────────────────────────
let modalLastFocus = null

function focusables(el) {
  return Array.from(el.querySelectorAll(
    'a[href], button:not([disabled]), select:not([disabled]), input:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
  )).filter(x => x.offsetParent !== null)
}

function openModal(el) {
  modalLastFocus = document.activeElement
  el.classList.remove('hidden')
  const f = focusables(el)
  if (f[0]) f[0].focus()
  el._trap = e => {
    if (e.key === 'Escape') { e.preventDefault(); closeModal(el); return }
    if (e.key !== 'Tab') return
    const items = focusables(el)
    if (items.length === 0) return
    const first = items[0], last = items[items.length - 1]
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus() }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus() }
  }
  document.addEventListener('keydown', el._trap)
}

function closeModal(el) {
  el.classList.add('hidden')
  if (el._trap) { document.removeEventListener('keydown', el._trap); el._trap = null }
  if (modalLastFocus && modalLastFocus.focus) modalLastFocus.focus()
}

// Define uma mensagem colorida (ok/erro) num elemento de status, sem innerHTML.
function setStatusMsg(el, kind, msg) {
  el.replaceChildren()
  if (!msg) return
  const span = document.createElement('span')
  span.className = kind === 'ok' ? 'txt-ok' : 'txt-erro'
  span.textContent = msg
  el.appendChild(span)
}

function abrirModalArvore() {
  const sel = document.getElementById('arvoreDepto')
  if (!arvoreSelectPopulado) {
    for (const d of DEPARTAMENTOS) sel.add(new Option(d, d))
    arvoreSelectPopulado = true
  }
  document.getElementById('arvoreStatus').replaceChildren()
  openModal(document.getElementById('arvoreOverlay'))
}

function fecharModalArvore() {
  closeModal(document.getElementById('arvoreOverlay'))
}

async function baixarArvore() {
  const dep = document.getElementById('arvoreDepto').value
  const cta = document.getElementById('arvoreCta')
  const statusEl = document.getElementById('arvoreStatus')
  if (!dep) { setStatusMsg(statusEl, 'erro', 'selecione um departamento'); return }
  cta.disabled = true
  cta.textContent = 'baixando…'
  statusEl.replaceChildren()
  try {
    const r = await api('/arvore/baixar?departamento=' + encodeURIComponent(dep), { raw: true, timeout: 90000 })
    const blob = await r.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = 'arvore_mercadologica.csv'
    document.body.appendChild(a); a.click(); a.remove()
    URL.revokeObjectURL(url)
    setStatusMsg(statusEl, 'ok', '✓ CSV baixado')
  } catch (e) {
    setStatusMsg(statusEl, 'erro', e.message)
  } finally {
    cta.disabled = false
    cta.textContent = '↓ Baixar CSV'
  }
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
    c.setAttribute('aria-checked', 'false')
    c.removeAttribute('aria-disabled')
    c.setAttribute('tabindex', '0')
  })
  document.querySelectorAll('.op-card .checkmark').forEach(c => c.textContent = '')
  document.getElementById('cdTipoExtra').classList.add('hidden')
  document.getElementById('preOrigemExtra').classList.add('hidden')
  document.getElementById('preOrigemInput').value = ''
  state.ops = emptyOps()

  // No modelo PT, só Pré-transferência é permitida; outros cards ficam desabilitados.
  if (s.formato === 'pt') {
    document.querySelectorAll('.op-card').forEach(c => {
      if (c.dataset.op !== 'preTransf') {
        c.classList.add('disabled')
        c.style.opacity = '0.35'
        c.style.pointerEvents = 'none'
        c.setAttribute('title', 'disponível apenas na ficha de cadastro')
        c.setAttribute('aria-disabled', 'true')
        c.setAttribute('tabindex', '-1')
      }
    })
    // Marca PT por padrão.
    const ptCard = document.querySelector('.op-card[data-op="preTransf"]')
    state.ops.preTransf = true
    ptCard.classList.add('checked')
    ptCard.setAttribute('aria-checked', 'true')
    ptCard.querySelector('.checkmark').textContent = '✓'
    // No modelo PT, origem vem da planilha — não mostra input.
  }
  refreshReadyCta()
}

function appendValidation(parent, kind, text) {
  const el = document.createElement('div')
  el.className = 'validation-item ' + kind
  const icon = document.createElement('span')
  icon.className = 'v-icon'
  icon.textContent = kind === 'ok' ? '✓' : '!'
  const span = document.createElement('span')
  span.textContent = text
  el.append(icon, span)
  parent.appendChild(el)
}

function toggleOp(card) {
  const op = card.dataset.op
  state.ops[op] = !state.ops[op]
  card.classList.toggle('checked', state.ops[op])
  card.setAttribute('aria-checked', String(state.ops[op]))
  card.querySelector('.checkmark').textContent = state.ops[op] ? '✓' : ''
  if (op === 'linhaCompraCD') {
    document.getElementById('cdTipoExtra').classList.toggle('hidden', !state.ops[op])
  }
  // Mutex: Pré-transferência roda sozinha.
  if (state.ops[op]) {
    if (op === 'preTransf') {
      OP_KEYS.filter(k => k !== 'preTransf').forEach(o => {
        if (state.ops[o]) {
          state.ops[o] = false
          const c = document.querySelector('.op-card[data-op="'+o+'"]')
          c.classList.remove('checked')
          c.setAttribute('aria-checked', 'false')
          c.querySelector('.checkmark').textContent = ''
        }
      })
      document.getElementById('cdTipoExtra').classList.add('hidden')
    } else {
      if (state.ops.preTransf) {
        state.ops.preTransf = false
        const c = document.querySelector('.op-card[data-op="preTransf"]')
        c.classList.remove('checked')
        c.setAttribute('aria-checked', 'false')
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
  const opsList = selectedOps().map(opLabel)
  const cta = document.getElementById('readyCta')
  const t = document.getElementById('readyCtaTitle')
  const s = document.getElementById('readyCtaSubtitle')

  // Validação extra: ficha + PT exige loja origem preenchida.
  const formato = state.session && state.session.formato
  let bloqueado = false
  let bloqueioMsg = ''
  if (state.ops.preTransf && formato === 'ficha') {
    const v = parseInt(document.getElementById('preOrigemInput').value, 10)
    if (!Number.isFinite(v) || v <= 0) { bloqueado = true; bloqueioMsg = 'Informe a loja origem' }
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

// Scheduler de log: o backend emite em rajadas (uma por loja em paralelo),
// então acumulamos e renderizamos uma vez por frame.
const MAX_LOG_ROWS = 1500
let logPending = []
let logFlushScheduled = false
let countersDirty = false

async function iniciarRun() {
  await rodar({ retry: false, linhas: null })
}

async function reenviarErros() {
  const linhas = (state.run && state.run.errors)
    ? Array.from(new Set(state.run.errors.map(e => e.linha).filter(n => n > 0)))
    : []
  if (linhas.length === 0) {
    toast('Nenhum erro com linha rastreável para reenviar.', { kind: 'aviso' })
    return
  }
  await rodar({ retry: true, linhas })
}

async function rodar({ retry, linhas }) {
  state.logs = []
  state.filtro = 'tudo'
  logPending = []
  document.querySelectorAll('.filter-tab').forEach(t => {
    const on = t.dataset.filter === 'tudo'
    t.classList.toggle('active', on)
    t.setAttribute('aria-pressed', String(on))
  })
  document.getElementById('logList').innerHTML = ''
  document.getElementById('progFill').style.width = '0%'
  document.getElementById('statProc').textContent = '0/' + (retry ? linhas.length : state.session.linhas)
  document.getElementById('statOk').textContent = '0'
  document.getElementById('statWarn').textContent = '0'
  document.getElementById('statErr').textContent = '0'
  document.getElementById('statEta').textContent = '—'
  ;['cntTudo','cntOk','cntAviso','cntErro'].forEach(id => document.getElementById(id).textContent = '0')

  const opsLabel = selectedOps().map(opShort)
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
    alterarSubgrupo: state.ops.alterarSubgrupo,
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
  for (let line of part.split('\n')) {
    if (line.endsWith('\r')) line = line.slice(0, -1)   // tolera CRLF
    if (line.startsWith('event: ')) evtName = line.slice(7)
    else if (line.startsWith('data: ')) data = line.slice(6)
  }
  if (!data) return
  let payload
  try {
    payload = JSON.parse(data)
  } catch {
    addLogEntry({ tipo: TIPO.AVISO, mensagem: 'evento ignorado (formato inválido)' })
    return
  }
  if (evtName === 'progress') {
    state.run.totals.processadas = payload.atual
    state.run.totals.total = payload.total
    document.getElementById('statProc').textContent = payload.atual + '/' + payload.total
    const pct = payload.total > 0 ? (payload.atual / payload.total * 100) : 0
    document.getElementById('progFill').style.width = pct + '%'
    refreshETA()
    return
  }
  if (payload.tipo === TIPO.FIM) {
    finalizarRun(payload.mensagem)
    return
  }
  addLogEntry(payload)
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
  if (evt.tipo === TIPO.OK) state.run.totals.ok++
  else if (evt.tipo === TIPO.AVISO) state.run.totals.warn++
  else if (evt.tipo === TIPO.ERRO) {
    state.run.totals.erro++
    state.run.errors.push(evt)
  }
  countersDirty = true
  if (entryMatchesFiltro(evt)) logPending.push(evt)
  scheduleLogFlush()
}

function entryMatchesFiltro(evt) {
  if (state.filtro === 'tudo') return true
  return evt.tipo === state.filtro
}

function scheduleLogFlush() {
  if (logFlushScheduled) return
  logFlushScheduled = true
  requestAnimationFrame(flushLog)
}

function flushLog() {
  logFlushScheduled = false
  const list = document.getElementById('logList')
  if (logPending.length) {
    // autoscroll só se o usuário já estava no fim
    const nearBottom = list.scrollHeight - list.scrollTop - list.clientHeight < 40
    const frag = document.createDocumentFragment()
    for (const evt of logPending) frag.appendChild(buildLogRow(evt))
    logPending = []
    list.appendChild(frag)
    trimLogRows(list)
    if (nearBottom) list.scrollTop = list.scrollHeight
  }
  if (countersDirty) {
    updateCounters()
    countersDirty = false
  }
}

function updateCounters() {
  document.getElementById('statOk').textContent = state.run.totals.ok
  document.getElementById('statWarn').textContent = state.run.totals.warn
  document.getElementById('statErr').textContent = state.run.totals.erro
  document.getElementById('cntTudo').textContent = state.logs.length
  document.getElementById('cntOk').textContent = state.run.totals.ok
  document.getElementById('cntAviso').textContent = state.run.totals.warn
  document.getElementById('cntErro').textContent = state.run.totals.erro
}

// Limita o nº de linhas no DOM; o log completo fica no arquivo em disco.
function trimLogRows(list) {
  const rows = list.querySelectorAll('.log-row')
  const excess = rows.length - MAX_LOG_ROWS
  if (excess <= 0) return
  for (let i = 0; i < excess; i++) rows[i].remove()
  if (!list.querySelector('.log-trim-note')) {
    const note = document.createElement('div')
    note.className = 'log-trim-note'
    note.textContent = 'Mostrando as últimas ' + MAX_LOG_ROWS + ' linhas — o log completo está no arquivo.'
    list.insertBefore(note, list.firstChild)
  }
}

function buildLogRow(evt) {
  const row = document.createElement('div')
  row.className = 'log-row ' + (evt.tipo || 'info')
  const ts = new Date().toLocaleTimeString('pt-BR', {hour:'2-digit',minute:'2-digit',second:'2-digit'})

  const tsEl = document.createElement('span'); tsEl.className = 'ts'; tsEl.textContent = ts
  const dot = document.createElement('span'); dot.className = 'dot'
  const op = document.createElement('span'); op.className = 'op'; op.textContent = evt.op || ''
  const msg = document.createElement('span'); msg.className = 'msg'; msg.textContent = evt.mensagem || ''
  const linha = document.createElement('span'); linha.className = 'linha-ref'
  if (evt.linha && evt.tipo === TIPO.ERRO) {
    const a = document.createElement('a')
    a.title = 'linha na planilha'
    a.textContent = 'linha ' + evt.linha
    linha.appendChild(a)
  }
  row.append(tsEl, dot, op, msg, linha)
  return row
}

function setFiltro(f) {
  state.filtro = f
  document.querySelectorAll('.filter-tab').forEach(t => {
    const on = t.dataset.filter === f
    t.classList.toggle('active', on)
    t.setAttribute('aria-pressed', String(on))
  })
  const list = document.getElementById('logList')
  list.innerHTML = ''
  list.classList.add('bulk')   // desliga a animação row-in durante o re-render em massa
  const matching = state.logs.filter(entryMatchesFiltro)
  const start = Math.max(0, matching.length - MAX_LOG_ROWS)
  const frag = document.createDocumentFragment()
  if (start > 0) {
    const note = document.createElement('div')
    note.className = 'log-trim-note'
    note.textContent = 'Mostrando as últimas ' + MAX_LOG_ROWS + ' linhas — o log completo está no arquivo.'
    frag.appendChild(note)
  }
  for (let i = start; i < matching.length; i++) frag.appendChild(buildLogRow(matching[i]))
  list.appendChild(frag)
  list.scrollTop = list.scrollHeight
  requestAnimationFrame(() => list.classList.remove('bulk'))
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
  // stack bar (largura é valor dinâmico → style.width)
  const stack = document.getElementById('doneStack')
  stack.replaceChildren()
  const addSeg = (cls, pct) => {
    if (pct <= 0) return
    const d = document.createElement('div')
    d.className = cls
    d.style.width = pct + '%'
    stack.appendChild(d)
  }
  if (total > 0) {
    addSeg('b-ok',   r.totals.ok   / total * 100)
    addSeg('b-warn', r.totals.warn / total * 100)
    addSeg('b-err',  r.totals.erro / total * 100)
  } else {
    const empty = document.createElement('div')
    empty.className = 'b-empty'
    stack.appendChild(empty)
  }

  // pt keys card
  const ptKeysCard = document.getElementById('ptKeysCard')
  const ptKeysList = document.getElementById('ptKeysList')
  const ptKeys = state.run.preTransferenciaKeys || []
  if (ptKeys.length > 0) {
    ptKeysCard.classList.remove('hidden')
    ptKeysList.replaceChildren()
    for (const k of ptKeys) {
      const pill = document.createElement('span')
      pill.className = 'key-pill'
      const icon = document.createElement('span')
      icon.className = 'key-icon'
      icon.textContent = '#'
      pill.append(icon, document.createTextNode(String(k)))
      ptKeysList.appendChild(pill)
    }
  } else {
    ptKeysCard.classList.add('hidden')
  }

  // erros card
  const card = document.getElementById('errorsCard')
  const body = document.getElementById('errorsBody')
  body.replaceChildren()
  const errs = (r.errors || []).filter(e => e.tipo === TIPO.ERRO)
  if (errs.length === 0) {
    card.classList.add('hidden')
    return
  }
  card.classList.remove('hidden')
  document.getElementById('errorsSub').textContent = errs.length + ' itens'
  for (const e of errs) {
    const div = document.createElement('div')
    div.className = 'err-item'
    const linha = document.createElement('span'); linha.className = 'linha'; linha.textContent = 'linha ' + (e.linha || '—')
    const ean = document.createElement('span'); ean.className = 'ean'; ean.textContent = e.ean || ''
    const motivo = document.createElement('span'); motivo.className = 'motivo'; motivo.textContent = e.mensagem || ''
    const tagWrap = document.createElement('span')
    const tag = document.createElement('span'); tag.className = 'op-tag'; tag.textContent = e.op || ''
    tagWrap.appendChild(tag)
    div.append(linha, ean, motivo, tagWrap)
    body.appendChild(div)
  }
}

function resetUpload() {
  goToIdle()
}

// Helpers
// ───────────────────────────────────────────────────────────
function formatSize(b) {
  if (!b) return '0 B'
  if (b < 1024) return b + ' B'
  if (b < 1024*1024) return (b/1024).toFixed(1) + ' KB'
  return (b/1024/1024).toFixed(1) + ' MB'
}

function abrirErp() {
  window.open('https://erp.bluesoft.com.br', '_blank')
}

// ───────────────────────────────────────────────────────────
// Tema claro/escuro (persistido em localStorage; default: escuro)
// ───────────────────────────────────────────────────────────
function applyTheme(theme) {
  const isLight = theme === 'light'
  document.documentElement.dataset.theme = isLight ? 'light' : 'dark'
  const btn = document.getElementById('themeToggle')
  if (btn) {
    btn.textContent = isLight ? '☀️' : '🌙'
    btn.setAttribute('aria-pressed', String(isLight))
  }
}

function toggleTheme() {
  const next = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light'
  try { localStorage.setItem('tema', next) } catch {}
  applyTheme(next)
}

// ───────────────────────────────────────────────────────────
// Wiring de eventos (substitui os antigos handlers inline)
// ───────────────────────────────────────────────────────────
const ACTIONS = {
  goToSetup, toggleSecret, abrirErp, cancelSetup, setupCtaClick,
  abrirModalArvore, abrirLogs, resetUpload, iniciarRun, parar,
  reenviarErros, fecharModalArvore, baixarArvore, toggleTheme,
}

function wireEvents() {
  // Cliques com data-action (delegação global)
  document.addEventListener('click', e => {
    const el = e.target.closest('[data-action]')
    if (!el) return
    const fn = ACTIONS[el.dataset.action]
    if (fn) fn()
  })

  // Op-cards: alterna a operação (ignora cliques nos controles extras)
  document.querySelectorAll('.op-card').forEach(card => {
    const activate = e => {
      if (e.target.closest('.op-card-extra')) return
      if (card.getAttribute('aria-disabled') === 'true') return
      toggleOp(card)
    }
    card.addEventListener('click', activate)
    card.addEventListener('keydown', e => {
      if (e.key === ' ' || e.key === 'Enter') { e.preventDefault(); activate(e) }
    })
  })

  // Filtros do log
  document.querySelectorAll('.filter-tab').forEach(tab => {
    tab.addEventListener('click', () => setFiltro(tab.dataset.filter))
  })

  // Setup: inputs revalidam o CTA
  ;['setupTenant', 'setupClientId', 'setupSecret'].forEach(id => {
    document.getElementById(id).addEventListener('input', setupChanged)
  })

  // Idle: escolha de modo (dados/fotos)
  document.querySelectorAll('input[name=idleModo]').forEach(r => {
    r.addEventListener('change', () => onModoChange(r.value))
  })

  // Input de arquivo
  const fileInput = document.getElementById('fileInput')
  fileInput.addEventListener('change', () => onFileChange(fileInput.files[0]))
  fileInput.addEventListener('click', () => { fileInput.value = '' })

  // Ready: tipo de CD e loja de origem
  document.querySelectorAll('input[name=cdTipo]').forEach(r => {
    r.addEventListener('change', () => onCdTipo(r))
  })
  document.getElementById('preOrigemInput').addEventListener('input', refreshReadyCta)

  // Drag & drop na dropzone
  const dz = document.getElementById('dropzone')
  dz.addEventListener('dragover', e => { e.preventDefault(); dz.classList.add('dragover') })
  dz.addEventListener('dragleave', () => dz.classList.remove('dragover'))
  dz.addEventListener('drop', e => {
    e.preventDefault(); dz.classList.remove('dragover')
    const f = e.dataTransfer.files[0]
    if (f) onFileChange(f)
  })
}

applyTheme(document.documentElement.dataset.theme || 'dark')
wireEvents()
boot()
