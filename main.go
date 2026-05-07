package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

// ─── Payloads ────────────────────────────────────────────────────────────────

type PayloadLinhaCompra struct {
	FornecedorKey int `json:"fornecedorKey"`
	DivisaoKey    int `json:"divisaoKey"`
	CompradorKey  int `json:"compradorKey"`
	ProdutoKey    int `json:"produtoKey"`
	LojaKey       int `json:"lojaKey"`
}

type PayloadLinhaCompraCD struct {
	FornecedorKey int   `json:"fornecedorKey"`
	DivisaoKey    int   `json:"divisaoKey"`
	ProdutoKey    int   `json:"produtoKey"`
	LojaKey       []int `json:"lojaKey"`
}

type LinhaLoja struct {
	LojaKey                   int    `json:"lojaKey"`
	QuantidadeEstoqueSeguranca int   `json:"quantidadeEstoqueDeSeguranca"`
	QuantidadeEstoqueMaximo   int    `json:"quantidadeEstoqueMaximo"`
	QuantidadePontoExtra      int    `json:"quantidadePontoExtra"`
	Multiplo                  int    `json:"multiplo"`
	DistribuirPor             string `json:"distribuirPor"`
	QuantidadeAtacado         int    `json:"quantidadeAtacado"`
	CrossDocking              bool   `json:"crossDocking"`
	Remover                   bool   `json:"remover"`
	OperacaoEntreLojaSuspensa bool   `json:"operacaoEntreLojaSuspensa"`
	CompraSuspensa            bool   `json:"compraSuspensa"`
}

type PayloadLinhaLoja struct {
	Linhas []LinhaLoja `json:"linhas"`
}

type PayloadNCM struct {
	NCM string `json:"ncm"`
}

type LogEntry struct {
	Tipo     string `json:"tipo"`
	Mensagem string `json:"mensagem"`
}

// Canal de cancelamento — fechado por /api/cancel para interromper o loop em handleUpload.
var (
	cancelChan chan struct{}
	cancelMu   sync.Mutex
)

// ─── Logger persistente em arquivo ────────────────────────────────────────────

type runLog struct {
	f    *os.File
	path string
	mu   sync.Mutex
}

func newRunLog() *runLog {
	dir, err := logsDir()
	if err != nil {
		return nil
	}
	name := fmt.Sprintf("upload-%s.log", time.Now().Format("20060102-150405"))
	p := filepath.Join(dir, name)
	f, err := os.Create(p)
	if err != nil {
		return nil
	}
	return &runLog{f: f, path: p}
}

func (r *runLog) writeln(format string, args ...any) {
	if r == nil || r.f == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	fmt.Fprintf(r.f, "[%s] ", time.Now().Format("15:04:05.000"))
	fmt.Fprintf(r.f, format, args...)
	fmt.Fprintln(r.f)
}

func (r *runLog) event(tipo, msg string) {
	if r == nil {
		return
	}
	r.writeln("%-6s %s", strings.ToUpper(tipo), msg)
}

func (r *runLog) httpCall(label, method, endpoint string, payload any, status int, body string, err error) {
	if r == nil || r.f == nil {
		return
	}
	// Consultas de GTIN são puro lookup — não logar (ruído em massa).
	if strings.HasPrefix(label, "GTIN") {
		return
	}
	// Respostas de sucesso (2xx) também não precisam de detalhe — log foca em problemas.
	if err == nil && status >= 200 && status < 300 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(r.f, "\n[%s] HTTP   %s | %s %s\n", ts, label, method, endpoint)
	if payload != nil {
		b, _ := json.MarshalIndent(payload, "  ", "  ")
		fmt.Fprintf(r.f, "  request:  %s\n", string(b))
	}
	if err != nil {
		fmt.Fprintf(r.f, "  error:    %v\n", err)
		return
	}
	fmt.Fprintf(r.f, "  status:   %d\n", status)
	fmt.Fprintf(r.f, "  response: %s\n", body)
}

func (r *runLog) close() {
	if r == nil || r.f == nil {
		return
	}
	r.f.Close()
}

func logsDir() (string, error) {
	var base string
	if exe, err := os.Executable(); err == nil {
		base = filepath.Dir(exe)
	} else {
		wd, _ := os.Getwd()
		base = wd
	}
	dir := filepath.Join(base, "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func abrirCaminho(p string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", p).Start()
	case "darwin":
		return exec.Command("open", p).Start()
	default:
		return exec.Command("xdg-open", p).Start()
	}
}

// ─── Auth / HTTP ──────────────────────────────────────────────────────────────

func obterToken(tenant, clientID, clientSecret string) (string, error) {
	authURL := fmt.Sprintf("https://erp.bluesoft.com.br/%s/oauth2/token", tenant)
	resp, err := http.PostForm(authURL, url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {"switch.write"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	})
	if err != nil {
		return "", fmt.Errorf("erro de conexão: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("autenticação falhou (%d): %s", resp.StatusCode, string(body))
	}
	var result map[string]any
	json.Unmarshal(body, &result)
	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("token não encontrado na resposta")
	}
	return token, nil
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Semáforo global: limita a 10 requisições simultâneas (limite da API Bluesoft).
var apiSem = make(chan struct{}, 10)

const maxRetries429 = 6

func postAPI(token, endpoint string, payload any) (int, string, error) {
	apiSem <- struct{}{}
	defer func() { <-apiSem }()

	body, _ := json.Marshal(payload)
	backoff := 500 * time.Millisecond
	var status int
	var respBody string
	for tries := 0; tries < maxRetries429; tries++ {
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := httpClient.Do(req)
		if err != nil {
			return 0, "", err
		}
		rb, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		respBody = string(rb)
		if status != 429 {
			return status, respBody, nil
		}
		time.Sleep(backoff)
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
	return status, respBody, nil
}

func putAPI(token, endpoint string, payload any) (int, string, error) {
	apiSem <- struct{}{}
	defer func() { <-apiSem }()

	body, _ := json.Marshal(payload)
	backoff := 500 * time.Millisecond
	var status int
	var respBody string
	for tries := 0; tries < maxRetries429; tries++ {
		req, _ := http.NewRequest("PUT", endpoint, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := httpClient.Do(req)
		if err != nil {
			return 0, "", err
		}
		rb, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		respBody = string(rb)
		if status != 429 {
			return status, respBody, nil
		}
		time.Sleep(backoff)
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
	return status, respBody, nil
}

func getAPI(token, endpoint string) (int, []byte, error) {
	apiSem <- struct{}{}
	defer func() { <-apiSem }()

	backoff := 500 * time.Millisecond
	var status int
	var body []byte
	for tries := 0; tries < maxRetries429; tries++ {
		req, _ := http.NewRequest("GET", endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := httpClient.Do(req)
		if err != nil {
			return 0, nil, err
		}
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		status = resp.StatusCode
		if status != 429 {
			return status, body, nil
		}
		time.Sleep(backoff)
		if backoff < 8*time.Second {
			backoff *= 2
		}
	}
	return status, body, nil
}

// ─── Consulta GTIN ────────────────────────────────────────────────────────────

type ProdutoInfo struct {
	ProdutoKey    int
	FornecedorKey int
}

// consultarGTINv também devolve status/body para o logger persistente.
func consultarGTINv(tenant, token, gtin string) (*ProdutoInfo, int, string, error) {
	ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/comercial/produtos/gtin/%s", tenant, gtin)
	status, raw, err := getAPI(token, ep)
	body := string(raw)
	if err != nil {
		return nil, status, body, err
	}
	// 404 = produto não cadastrado: devolve info zerada para o chamador tratar como skip.
	if status == 404 {
		return &ProdutoInfo{}, status, body, nil
	}
	if status != 200 {
		return nil, status, body, fmt.Errorf("GTIN: HTTP %d", status)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, status, body, fmt.Errorf("resposta inválida do GTIN")
	}
	produtoKey, _ := result["produtoKey"].(float64)
	fornecedorKey, _ := result["fornecedorPadraoKey"].(float64)
	return &ProdutoInfo{
		ProdutoKey:    int(produtoKey),
		FornecedorKey: int(fornecedorKey),
	}, status, body, nil
}

// ─── Leitura da planilha TOYNG ────────────────────────────────────────────────

type ProdutoLinha struct {
	EAN   string // EAN 13 (unidade) — obrigatório
	DUN   string // DUN 14 (caixa) — opcional, usado quando CD em CX
	NCM   string // opcional, usado pela operação Alterar NCM
	Lojas []int  // lojaKeys com valor > 0
}

func lerPlanilha(caminho string) ([]ProdutoLinha, error) {
	f, err := excelize.OpenFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("não foi possível abrir: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Preencher")
	if err != nil {
		return nil, fmt.Errorf("aba 'Preencher' não encontrada")
	}
	if len(rows) < 8 {
		return nil, fmt.Errorf("planilha com menos de 8 linhas")
	}

	// Linha 8 (índice 7) = cabeçalhos
	header := rows[7]

	// Sempre captura EAN 13, DUN 14 e NCM (DUN/NCM são opcionais).
	eanCol, dunCol, ncmCol := -1, -1, -1
	type lojaCol struct {
		idx  int
		loja int
	}
	var lojaCols []lojaCol

	for i, h := range header {
		h = strings.TrimSpace(h)
		hUp := strings.ToUpper(h)
		if eanCol == -1 && strings.Contains(hUp, "EAN 13") {
			eanCol = i
		} else if dunCol == -1 && strings.Contains(hUp, "DUN 14") {
			dunCol = i
		} else if ncmCol == -1 && strings.Contains(hUp, "NCM") {
			ncmCol = i
		}
		// tenta parsear como inteiro (número de loja)
		if v, err := strconv.Atoi(h); err == nil {
			lojaCols = append(lojaCols, lojaCol{i, v})
		}
	}

	if eanCol == -1 {
		return nil, fmt.Errorf("coluna 'EAN 13' não encontrada no cabeçalho (linha 8)")
	}
	if len(lojaCols) == 0 {
		return nil, fmt.Errorf("nenhuma coluna de loja encontrada")
	}

	// Lojas que nunca recebem parâmetro — sempre filtradas.
	// Inclui números pontuais e o range 200–300 (inativadas).
	excluida := func(l int) bool {
		switch l {
		case 5, 9, 13, 15, 16, 17, 18:
			return true
		}
		return l >= 200 && l <= 300
	}

	// Lista de todas as lojas válidas da planilha (usado quando linha está toda em 0).
	var todasLojas []int
	for _, lc := range lojaCols {
		if !excluida(lc.loja) {
			todasLojas = append(todasLojas, lc.loja)
		}
	}

	var result []ProdutoLinha
	// Dados a partir da linha 9 (índice 8)
	for _, row := range rows[8:] {
		if len(row) == 0 {
			continue
		}
		ean := ""
		if eanCol < len(row) {
			ean = strings.TrimSpace(row[eanCol])
		}
		if ean == "" {
			continue
		}
		dun := ""
		if dunCol >= 0 && dunCol < len(row) {
			dun = strings.TrimSpace(row[dunCol])
		}
		ncm := ""
		if ncmCol >= 0 && ncmCol < len(row) {
			ncm = strings.TrimSpace(row[ncmCol])
		}

		// Colunas de loja com valor > 0
		var lojas []int
		for _, lc := range lojaCols {
			if lc.idx >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[lc.idx])
			if val == "" || val == "0" || val == "-" {
				continue
			}
			v, err := strconv.ParseFloat(val, 64)
			if err != nil || v <= 0 {
				continue
			}
			lojas = append(lojas, lc.loja)
		}

		// Convenção do compras: linha toda em 0 → considerar todas as lojas válidas.
		if len(lojas) == 0 {
			lojas = append(lojas, todasLojas...)
		}

		// Filtra lojas excluídas (sempre).
		filtradas := lojas[:0]
		for _, l := range lojas {
			if !excluida(l) {
				filtradas = append(filtradas, l)
			}
		}
		lojas = filtradas

		result = append(result, ProdutoLinha{EAN: ean, DUN: dun, NCM: ncm, Lojas: lojas})
	}
	return result, nil
}

// ─── Lógica de CD ────────────────────────────────────────────────────────────

// Agrupa lojas por CD: < 30 -> CD3, 300-308 -> CD306
func agruparPorCD(lojas []int) map[int][]int {
	cds := map[int][]int{}
	for _, l := range lojas {
		if l < 30 {
			cds[3] = append(cds[3], l)
		} else if l >= 300 && l <= 308 {
			cds[306] = append(cds[306], l)
		}
		// fora do range: ignora
	}
	return cds
}

// dedupInts remove duplicatas preservando a ordem da primeira aparição.
func dedupInts(xs []int) []int {
	seen := make(map[int]struct{}, len(xs))
	out := make([]int, 0, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

// ─── Processamento principal ──────────────────────────────────────────────────

const compradorKey = 655124
const divisaoKey   = 1

func processarProduto(
	tenant, token string,
	produto ProdutoLinha,
	fazerLinhaCompra, fazerLinhaCompraCD, fazerLinhaLoja, fazerNCM bool,
	cdTipo string,
	rl *runLog,
	send func(tipo, msg string),
) {
	// Operações de loja exigem ao menos uma loja na planilha.
	exigeLojas := fazerLinhaCompra || fazerLinhaCompraCD || fazerLinhaLoja
	if exigeLojas && len(produto.Lojas) == 0 {
		send("info", fmt.Sprintf("⚠  EAN %s: sem lojas ativas, pulando", produto.EAN))
		return
	}

	// Linha de Compra e Sortimento sempre usam EAN 13.
	// Linha CD em UN também usa EAN 13; em CX usa o DUN 14 (lookup separado).
	// Alterar NCM também usa EAN 13.
	precisaUN := fazerLinhaCompra || fazerLinhaLoja || fazerNCM || (fazerLinhaCompraCD && cdTipo != "CX")
	precisaCX := fazerLinhaCompraCD && cdTipo == "CX"

	var infoUN *ProdutoInfo
	if precisaUN {
		i, status, body, err := consultarGTINv(tenant, token, produto.EAN)
		rl.httpCall("GTIN lookup (EAN)", "GET",
			fmt.Sprintf("/comercial/produtos/gtin/%s", produto.EAN), nil, status, body, err)
		if err != nil {
			send("erro", fmt.Sprintf("❌ EAN %s: %v", produto.EAN, err))
			return
		}
		if i.ProdutoKey == 0 {
			send("aviso", fmt.Sprintf("⚠ EAN %s: produto_key não encontrado, pulando", produto.EAN))
			return
		}
		infoUN = i
	}

	// ── 0. Alterar NCM ──
	if fazerNCM && infoUN != nil {
		prefix := fmt.Sprintf("EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey)
		if produto.NCM == "" {
			send("aviso", fmt.Sprintf("⚠ %s | NCM: vazio na planilha, pulando", prefix))
		} else {
			ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/comercial/produtos/%d", tenant, infoUN.ProdutoKey)
			payload := PayloadNCM{NCM: produto.NCM}
			status, body, err := putAPI(token, ep, payload)
			rl.httpCall("Alterar NCM | "+prefix, "PUT", ep, payload, status, body, err)
			if err != nil || (status != 200 && status != 201 && status != 204) {
				msg := apiErrMsg(status, body, err)
				send("erro", fmt.Sprintf("❌ %s | NCM: %s", prefix, msg))
			} else {
				send("ok", fmt.Sprintf("✅ %s | NCM atualizado: %s", prefix, produto.NCM))
			}
		}
	}

	// ── 1. Linha de Compra (paralelo, uma req por loja) ──
	if fazerLinhaCompra && infoUN != nil {
		prefix := fmt.Sprintf("EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey)
		ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadecompra", tenant)

		var wg sync.WaitGroup
		var sucesso, falha int32
		for _, loja := range produto.Lojas {
			wg.Add(1)
			go func(l int) {
				defer wg.Done()
				payload := PayloadLinhaCompra{
					FornecedorKey: infoUN.FornecedorKey,
					DivisaoKey:    divisaoKey,
					CompradorKey:  compradorKey,
					ProdutoKey:    infoUN.ProdutoKey,
					LojaKey:       l,
				}
				status, body, err := postAPI(token, ep, payload)
				rl.httpCall(fmt.Sprintf("Linha Compra | %s | loja %d", prefix, l), "POST", ep, payload, status, body, err)
				if err != nil || (status != 200 && status != 201) {
					msg := apiErrMsg(status, body, err)
					send("erro", fmt.Sprintf("❌ %s | Linha Compra loja %d: %s", prefix, l, msg))
					atomic.AddInt32(&falha, 1)
				} else {
					atomic.AddInt32(&sucesso, 1)
				}
			}(loja)
		}
		wg.Wait()
		if sucesso > 0 {
			send("ok", fmt.Sprintf("✅ %s | Linha Compra: %d/%d lojas OK", prefix, sucesso, len(produto.Lojas)))
		}
	}

	// ── 2. Sortimento / Linha de Loja ──
	if fazerLinhaLoja && infoUN != nil {
		prefix := fmt.Sprintf("EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey)
		linhas := make([]LinhaLoja, len(produto.Lojas))
		for i, l := range produto.Lojas {
			linhas[i] = LinhaLoja{
				LojaKey:       l,
				DistribuirPor: "PALETE",
			}
		}
		ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadeloja/%d", tenant, infoUN.ProdutoKey)
		payload := PayloadLinhaLoja{Linhas: linhas}
		status, body, err := postAPI(token, ep, payload)
		rl.httpCall("Linha Loja | "+prefix, "POST", ep, payload, status, body, err)
		if err != nil || (status != 200 && status != 201) {
			msg := apiErrMsg(status, body, err)
			send("erro", fmt.Sprintf("❌ %s | Linha Loja: %s", prefix, msg))
		} else {
			send("ok", fmt.Sprintf("✅ %s | Linha Loja: OK (%d lojas)", prefix, len(produto.Lojas)))
		}
	}

	// ── 3. Linha de Compra CD ──
	if fazerLinhaCompraCD {
		// Resolve a info correta para o CD: UN reaproveita infoUN; CX faz lookup pelo DUN.
		var infoCD *ProdutoInfo
		gtinCD := produto.EAN
		if precisaCX {
			if produto.DUN == "" {
				send("aviso", fmt.Sprintf("⚠ EAN %s: sem DUN 14 na planilha, pulando Linha CD", produto.EAN))
				return
			}
			i, status, body, err := consultarGTINv(tenant, token, produto.DUN)
			rl.httpCall("GTIN lookup (DUN)", "GET",
				fmt.Sprintf("/comercial/produtos/gtin/%s", produto.DUN), nil, status, body, err)
			if err != nil {
				send("erro", fmt.Sprintf("❌ DUN %s: %v", produto.DUN, err))
				return
			}
			if i.ProdutoKey == 0 {
				send("aviso", fmt.Sprintf("⚠ DUN %s: produto_key não encontrado, pulando Linha CD", produto.DUN))
				return
			}
			infoCD = i
			gtinCD = produto.DUN
		} else {
			infoCD = infoUN
		}
		if infoCD == nil {
			return
		}

		prefix := fmt.Sprintf("%s %s (produto %d)", gtinLabel(precisaCX), gtinCD, infoCD.ProdutoKey)
		cds := agruparPorCD(produto.Lojas)

		var wg sync.WaitGroup
		for cdKey, lojas := range cds {
			wg.Add(1)
			go func(cdKey int, lojas []int) {
				defer wg.Done()
				lojasFinal := dedupInts(append([]int{cdKey}, lojas...))
				payload := PayloadLinhaCompraCD{
					FornecedorKey: infoCD.FornecedorKey,
					DivisaoKey:    divisaoKey,
					ProdutoKey:    infoCD.ProdutoKey,
					LojaKey:       lojasFinal,
				}
				ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadecompra/cd/%d", tenant, cdKey)
				status, body, err := postAPI(token, ep, payload)
				rl.httpCall(fmt.Sprintf("Linha CD%d | %s", cdKey, prefix), "POST", ep, payload, status, body, err)
				if err != nil || (status != 200 && status != 201) {
					msg := apiErrMsg(status, body, err)
					send("erro", fmt.Sprintf("❌ %s | Linha CD%d: %s", prefix, cdKey, msg))
				} else {
					send("ok", fmt.Sprintf("✅ %s | Linha CD%d: OK (%d lojas)", prefix, cdKey, len(lojasFinal)))
				}
			}(cdKey, lojas)
		}
		wg.Wait()
	}
}

func gtinLabel(cx bool) string {
	if cx {
		return "DUN"
	}
	return "EAN"
}

func apiErrMsg(status int, body string, err error) string {
	if err != nil {
		return err.Error()
	}
	var errResp map[string]any
	if json.Unmarshal([]byte(body), &errResp) == nil {
		if msg, ok := errResp["message"].(string); ok {
			return fmt.Sprintf("HTTP %d: %s", status, msg)
		}
	}
	if len(body) > 150 {
		body = body[:150] + "..."
	}
	return fmt.Sprintf("HTTP %d: %s", status, body)
}

// ─── Handlers HTTP ────────────────────────────────────────────────────────────

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming não suportado", 500)
		return
	}

	rl := newRunLog()
	defer rl.close()

	var sendMu sync.Mutex
	send := func(tipo, msg string) {
		sendMu.Lock()
		defer sendMu.Unlock()
		b, _ := json.Marshal(LogEntry{Tipo: tipo, Mensagem: msg})
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		rl.event(tipo, msg)
	}
	if rl != nil {
		rl.writeln("───── Início do upload ─────")
		rl.writeln("Arquivo de log: %s", rl.path)
	}

	// (Re)inicializa o canal de cancelamento para esta sessão de upload.
	cancelMu.Lock()
	cancelChan = make(chan struct{})
	localCancel := cancelChan
	cancelMu.Unlock()

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		send("erro", "Erro ao ler form: "+err.Error())
		send("fim", "")
		return
	}

	tenant            := r.FormValue("tenant")
	clientID          := r.FormValue("clientId")
	clientSecret      := r.FormValue("clientSecret")
	fazerLC           := r.FormValue("linhaCompra") == "true"
	fazerLCCD         := r.FormValue("linhaCompraCD") == "true"
	fazerLL           := r.FormValue("linhaLoja") == "true"
	fazerNCM          := r.FormValue("alterarNCM") == "true"
	cdTipo            := r.FormValue("cdTipo")
	if cdTipo == "" {
		cdTipo = "UN"
	}

	if !fazerLC && !fazerLCCD && !fazerLL && !fazerNCM {
		send("erro", "Selecione pelo menos uma operação")
		send("fim", "")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		send("erro", "Arquivo não recebido: "+err.Error())
		send("fim", "")
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "bluesoft-*.xlsx")
	if err != nil {
		send("erro", "Erro ao criar arquivo temporário")
		send("fim", "")
		return
	}
	defer os.Remove(tmp.Name())
	io.Copy(tmp, file)
	tmp.Close()

	send("info", "🔐 Autenticando...")
	token, err := obterToken(tenant, clientID, clientSecret)
	if err != nil {
		send("erro", "❌ Falha na autenticação: "+err.Error())
		send("fim", "")
		return
	}
	send("ok", "✅ Autenticado com sucesso")

	send("info", "📄 Lendo planilha...")
	produtos, err := lerPlanilha(tmp.Name())
	if err != nil {
		send("erro", "❌ Erro ao ler planilha: "+err.Error())
		send("fim", "")
		return
	}
	send("info", fmt.Sprintf("📋 %d produtos encontrados. Iniciando envio...", len(produtos)))

	ops := []string{}
	if fazerNCM  { ops = append(ops, "Alterar NCM") }
	if fazerLC   { ops = append(ops, "Linha Compra") }
	if fazerLL   { ops = append(ops, "Linha Loja") }
	if fazerLCCD { ops = append(ops, "Linha Compra CD ("+cdTipo+")") }
	send("info", "Operações: "+strings.Join(ops, " | "))

	var okCount, errCount atomic.Int32
	cancelado := false
	for i, p := range produtos {
		select {
		case <-localCancel:
			send("aviso", "⛔ Processamento cancelado pelo usuário.")
			cancelado = true
		default:
		}
		if cancelado {
			break
		}

		processarProduto(tenant, token, p, fazerLC, fazerLCCD, fazerLL, fazerNCM, cdTipo, rl,
			func(tipo, msg string) {
				send(tipo, msg)
				if tipo == "ok" {
					okCount.Add(1)
				} else if tipo == "erro" {
					errCount.Add(1)
				}
			},
		)
		sendMu.Lock()
		prog := map[string]any{"atual": i + 1, "total": len(produtos)}
		b, _ := json.Marshal(prog)
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", b)
		flusher.Flush()
		sendMu.Unlock()
	}

	send("info", "─────────────────────────────────")
	send("info", fmt.Sprintf("Concluído: %d ✅  |  %d ❌", okCount.Load(), errCount.Load()))
	send("fim", fmt.Sprintf(`{"ok":%d,"erros":%d}`, okCount.Load(), errCount.Load()))
}

func handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	cancelMu.Lock()
	if cancelChan != nil {
		select {
		case <-cancelChan:
			// já fechado
		default:
			close(cancelChan)
		}
	}
	cancelMu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func handleAbrirLogs(w http.ResponseWriter, r *http.Request) {
	dir, err := logsDir()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := abrirCaminho(dir); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": dir})
}

func handleEnv(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tenant":   os.Getenv("BLUESOFT_TENANT"),
		"clientId": os.Getenv("client_id"),
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlUI))
}

func main() {
	godotenv.Load()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Não foi possível abrir porta:", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/env", handleEnv)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/cancel", handleCancel)
	http.HandleFunc("/api/abrir-logs", handleAbrirLogs)
	go func() {
		time.Sleep(300 * time.Millisecond)
		abrirNavegador(addr)
	}()
	fmt.Printf("Bluesoft Uploader rodando em %s\n", addr)
	http.Serve(ln, nil)
}

func abrirNavegador(u string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		exec.Command("open", u).Start()
	default:
		exec.Command("xdg-open", u).Start()
	}
}
