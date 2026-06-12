// ───────────────────────────────────────────────────────────
// Toasts — notificações efêmeras no canto inferior direito.
// ───────────────────────────────────────────────────────────
const TOAST_ICON = { erro: '✕', ok: '✓', aviso: '!', info: 'ℹ' }

export function toast(msg, opts = {}) {
  const { kind = 'info', duration = 6000 } = opts
  const wrap = document.getElementById('toasts')
  const el = document.createElement('div')
  el.className = 'toast ' + kind
  el.setAttribute('role', kind === 'erro' ? 'alert' : 'status')

  const icon = document.createElement('span')
  icon.className = 'toast-icon'
  icon.textContent = TOAST_ICON[kind] || TOAST_ICON.info

  const text = document.createElement('span')
  text.className = 'toast-msg'
  text.textContent = msg

  const close = document.createElement('button')
  close.className = 'toast-x'
  close.setAttribute('aria-label', 'fechar')
  close.textContent = '×'

  el.append(icon, text, close)
  wrap.appendChild(el)

  let timer
  const dismiss = () => {
    if (el.dataset.leaving) return
    el.dataset.leaving = '1'
    el.classList.add('leaving')
    el.addEventListener('animationend', () => el.remove(), { once: true })
  }
  const arm = () => { timer = setTimeout(dismiss, duration) }
  close.addEventListener('click', dismiss)
  el.addEventListener('mouseenter', () => clearTimeout(timer))
  el.addEventListener('mouseleave', arm)
  arm()
  return el
}
