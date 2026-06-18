// ───────────────────────────────────────────────────────────
// Wrapper de fetch: timeout + erros normalizados em pt-BR.
// Não usar para o stream /api/run nem para /fotos/upload (longa duração).
// raw:true devolve o Response (para blob/headers) já com timeout e erro tratados.
// ───────────────────────────────────────────────────────────
export async function api(url, opts = {}) {
  const { timeout = 15000, raw = false, ...rest } = opts
  let resp
  try {
    resp = await fetch(url, { ...rest, signal: AbortSignal.timeout(timeout) })
  } catch (e) {
    if (e.name === 'TimeoutError') throw new Error('tempo esgotado — o servidor demorou demais para responder')
    throw new Error('não foi possível conectar ao servidor')
  }
  if (raw) {
    if (!resp.ok) {
      const msg = (await resp.text().catch(() => '')).trim()
      throw new Error(msg || ('erro ' + resp.status))
    }
    return resp
  }
  const ct = resp.headers.get('Content-Type') || ''
  if (ct.includes('json')) {
    let d
    try { d = await resp.json() } catch { throw new Error('resposta inválida do servidor') }
    if (!resp.ok) throw new Error(d.error || ('erro ' + resp.status))
    return d
  }
  const txt = await resp.text()
  if (!resp.ok) throw new Error(txt.trim() || ('erro ' + resp.status))
  return txt
}
