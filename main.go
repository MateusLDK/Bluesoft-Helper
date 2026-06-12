package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
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

	"github.com/blang/semver"
	"github.com/inconshreveable/go-update"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

const version = "1.4.0"

const fotosVMURL = "http://192.168.0.25:8000/processar-fotos"
const arvoreVMURL = "http://192.168.0.25:8000/arvore"

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
	LojaKey                    int    `json:"lojaKey"`
	QuantidadeEstoqueSeguranca int    `json:"quantidadeEstoqueDeSeguranca"`
	QuantidadeEstoqueMaximo    int    `json:"quantidadeEstoqueMaximo"`
	QuantidadePontoExtra       int    `json:"quantidadePontoExtra"`
	Multiplo                   int    `json:"multiplo"`
	DistribuirPor              string `json:"distribuirPor"`
	QuantidadeAtacado          int    `json:"quantidadeAtacado"`
	CrossDocking               bool   `json:"crossDocking"`
	Remover                    bool   `json:"remover"`
	OperacaoEntreLojaSuspensa  bool   `json:"operacaoEntreLojaSuspensa"`
	CompraSuspensa             bool   `json:"compraSuspensa"`
}

type PayloadLinhaLoja struct {
	Linhas []LinhaLoja `json:"linhas"`
}

type PayloadNCM struct {
	NCM string `json:"ncm"`
}

type PayloadSubgrupo struct {
	SubGrupoProduto int     `json:"subGrupoProduto"`
	ValorMargem     float64 `json:"valorMargem"`
	QuebraMargem    float64 `json:"quebraMargem"`
}

type ItemPreTransferencia struct {
	ProdutoKey                    int  `json:"produtoKey,omitempty"`
	GTIN                          int  `json:"gtin,omitempty"`
	LojaDestinoKey                int  `json:"lojaDestinoKey"`
	Quantidade                    int  `json:"quantidade"`
	DeveInformarOProdutoKeyOuGtin bool `json:"deveInformarOProdutoKeyOuGtin"`
	QuantidadeFracionada          bool `json:"quantidadeFracionada"`
}

type PayloadPreTransferencia struct {
	LojaOrigemKey                             int                    `json:"lojaOrigemKey"`
	QuantidadeDiasSugestao                    int                    `json:"quantidadeDiasSugestao"`
	ALojaDeOrigemNaoPodeEstarNaListaDeDestino int                    `json:"alojaDeOrigemNaoPodeEstarNaListaDeDestino"`
	Produtos                                  []ItemPreTransferencia `json:"produtos"`
}

type LogEntry struct {
	Tipo     string `json:"tipo"`
	Mensagem string `json:"mensagem"`
}

// EventoSSE é o evento estruturado emitido pelo /api/run e /api/retry.
type EventoSSE struct {
	Tipo     string `json:"tipo"`            // ok | erro | aviso | info | fim
	Op       string `json:"op,omitempty"`    // NCM | Compra | Loja | CD3 | CD306 | etc
	Mensagem string `json:"mensagem"`        // texto livre
	EAN      string `json:"ean,omitempty"`   // EAN do produto, quando aplicável
	Linha    int    `json:"linha,omitempty"` // linha da planilha (1-indexed), quando aplicável
}

// FileSession guarda os produtos lidos de uma planilha pra serem reusados em /api/run e /api/retry.
type FileSession struct {
	ID        string
	Filename  string
	Size      int64
	Path      string // arquivo temporário (xlsx)
	Formato   string // "ficha" ou "pt"
	Produtos  []ProdutoLinha
	Avisos    []string
	CreatedAt time.Time
}

// Canal de cancelamento — fechado por /api/cancel para interromper o loop em handleUpload.
var (
	cancelChan chan struct{}
	cancelMu   sync.Mutex
)

// Storage de sessões de upload — produtos lidos da planilha aguardando /api/run.
var sessoes sync.Map // string -> *FileSession

func novoID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func envPath() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), ".env")
	}
	return ".env"
}

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

// ─── Consulta de margem por subgrupo (árvore mercadológica) ──────────────────

type MargemSubgrupo struct {
	ValorMargem  float64
	QuebraMargem float64
	TemMargem    bool // false quando o subgrupo existe mas não tem margensPossiveis
	Existe       bool // false quando o subgrupo não foi encontrado na árvore
}

type arvoreItem struct {
	SubGrupoProdutoKey int    `json:"subGrupoProdutoKey"`
	Tipo               string `json:"tipo"`
	MargensPossiveis   []struct {
		ValorMargem  float64 `json:"valorMargem"`
		QuebraMargem float64 `json:"quebraMargem"`
	} `json:"margensPossiveis"`
}

// consultarMargemSubgrupo busca o subgrupo na árvore mercadológica e devolve a
// primeira margem possível. Confere o subGrupoProdutoKey client-side em vez de
// confiar cegamente no filtro server-side.
func consultarMargemSubgrupo(tenant, token string, key int, rl *runLog) (MargemSubgrupo, error) {
	ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/comercial/arvoremercadologica?tipo=SUBGRUPO&categoriaKey=%d", tenant, key)
	status, raw, err := getAPI(token, ep)
	rl.httpCall(fmt.Sprintf("Subgrupo lookup (key %d)", key), "GET", ep, nil, status, string(raw), err)
	if err != nil {
		return MargemSubgrupo{}, err
	}
	if status != 200 {
		return MargemSubgrupo{}, fmt.Errorf("árvore mercadológica: HTTP %d", status)
	}
	var itens []arvoreItem
	if err := json.Unmarshal(raw, &itens); err != nil {
		return MargemSubgrupo{}, fmt.Errorf("resposta inválida da árvore mercadológica")
	}
	for _, it := range itens {
		if it.Tipo != "SUBGRUPO" || it.SubGrupoProdutoKey != key {
			continue
		}
		m := MargemSubgrupo{Existe: true}
		if len(it.MargensPossiveis) > 0 {
			m.ValorMargem = it.MargensPossiveis[0].ValorMargem
			m.QuebraMargem = it.MargensPossiveis[0].QuebraMargem
			m.TemMargem = true
		}
		return m, nil
	}
	return MargemSubgrupo{Existe: false}, nil
}

// ─── Leitura da planilha TOYNG ────────────────────────────────────────────────

type ProdutoLinha struct {
	LinhaPlanilha int         // número da linha na planilha (1-indexed) — usado pra exibir e retry
	EAN           string      // EAN 13 (unidade) — obrigatório no modelo ficha
	DUN           string      // DUN 14 (caixa) — opcional, usado quando CD em CX
	NCM           string      // opcional, usado pela operação Alterar NCM
	Lojas         []int       // lojaKeys com valor > 0 — usado pelas ops legadas
	Quantidades   map[int]int // loja → quantidade; usado pela Pré-transferência
	LojaOrigem    int         // só preenchido no modelo PT
	CodigoPT      int         // produtoKey direto, quando vem da coluna "codigo" do modelo PT
	GTINPT        int         // gtin direto, quando vem da coluna "barra" do modelo PT
	Subgrupo      int         // subGrupoProdutoKey da coluna "SUBGRUPO" — 0 = ausente
}

// excluida = lojas que nunca recebem parâmetro (sempre filtradas).
// Inclui números pontuais e o range 200–300 (inativadas).
func lojaExcluida(l int) bool {
	switch l {
	case 5, 9, 13, 15, 16, 17, 18:
		return true
	}
	return l >= 200 && l <= 300
}

// lerPlanilha detecta o formato e despacha para o parser apropriado.
// Retorna produtos, avisos, formato ("ficha" | "pt") e erro.
func lerPlanilha(caminho string) ([]ProdutoLinha, []string, string, error) {
	f, err := excelize.OpenFile(caminho)
	if err != nil {
		return nil, nil, "", fmt.Errorf("não foi possível abrir: %v", err)
	}
	defer f.Close()

	// Tentativa 1: ficha de cadastro (aba "Preencher", header em rows[7] com "EAN 13").
	if rows, err := f.GetRows("Preencher"); err == nil && len(rows) >= 8 {
		header := rows[7]
		for _, h := range header {
			if strings.Contains(strings.ToUpper(strings.TrimSpace(h)), "EAN 13") {
				produtos, avisos, perr := lerPlanilhaFicha(rows)
				if perr != nil {
					return nil, nil, "", perr
				}
				return produtos, avisos, "ficha", nil
			}
		}
	}

	// Tentativa 2: modelo PT (primeira aba, header em rows[0] com "codigo"/"barra" + "origem").
	sheets := f.GetSheetList()
	if len(sheets) > 0 {
		if rows, err := f.GetRows(sheets[0]); err == nil && len(rows) >= 1 {
			header := rows[0]
			temIdent, temOrigem := false, false
			for _, h := range header {
				low := strings.ToLower(strings.TrimSpace(h))
				if low == "codigo" || low == "código" || low == "barra" {
					temIdent = true
				}
				if low == "origem" {
					temOrigem = true
				}
			}
			if temIdent && temOrigem {
				produtos, avisos, perr := lerPlanilhaPT(rows)
				if perr != nil {
					return nil, nil, "", perr
				}
				return produtos, avisos, "pt", nil
			}
		}
	}

	return nil, nil, "", fmt.Errorf("formato não reconhecido: aba 'Preencher' com EAN 13 (linha 8) ou modelo de pré-transferência com 'codigo'/'barra' + 'origem' (linha 1)")
}

// lerPlanilhaFicha lê o formato tradicional: aba "Preencher", header na linha 8.
func lerPlanilhaFicha(rows [][]string) ([]ProdutoLinha, []string, error) {
	header := rows[7]

	eanCol, dunCol, ncmCol, subgrupoCol := -1, -1, -1, -1
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
		} else if subgrupoCol == -1 && strings.Contains(hUp, "SUBGRUPO") {
			subgrupoCol = i
		} else if ncmCol == -1 && strings.Contains(hUp, "NCM") {
			ncmCol = i
		}
		if v, err := strconv.Atoi(h); err == nil {
			lojaCols = append(lojaCols, lojaCol{i, v})
		}
	}

	if eanCol == -1 {
		return nil, nil, fmt.Errorf("coluna 'EAN 13' não encontrada no cabeçalho (linha 8)")
	}
	if len(lojaCols) == 0 {
		return nil, nil, fmt.Errorf("nenhuma coluna de loja encontrada")
	}

	var todasLojas []int
	for _, lc := range lojaCols {
		if !lojaExcluida(lc.loja) {
			todasLojas = append(todasLojas, lc.loja)
		}
	}

	var result []ProdutoLinha
	contadores := struct {
		semNCM           int
		expandidas       int
		semSubgrupo      int
		subgrupoInvalido int
	}{}
	for rowIdx, row := range rows[8:] {
		linhaPlanilha := rowIdx + 9
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
		if ncm == "" {
			contadores.semNCM++
		}
		subgrupo := 0
		if subgrupoCol >= 0 {
			raw := ""
			if subgrupoCol < len(row) {
				raw = strings.TrimSpace(row[subgrupoCol])
			}
			if raw == "" {
				contadores.semSubgrupo++
			} else if v, err := strconv.Atoi(raw); err == nil {
				subgrupo = v
			} else {
				contadores.subgrupoInvalido++
			}
		}

		// Lê quantidades + lista de lojas (>0).
		quantidades := map[int]int{}
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
			quantidades[lc.loja] = int(v)
		}

		// Convenção do compras: linha toda em 0 → considerar todas as lojas válidas (qty=1).
		if len(lojas) == 0 {
			for _, l := range todasLojas {
				lojas = append(lojas, l)
				quantidades[l] = 1
			}
			contadores.expandidas++
		}

		// Filtra lojas excluídas.
		filtradas := lojas[:0]
		for _, l := range lojas {
			if !lojaExcluida(l) {
				filtradas = append(filtradas, l)
			} else {
				delete(quantidades, l)
			}
		}
		lojas = filtradas

		result = append(result, ProdutoLinha{
			LinhaPlanilha: linhaPlanilha,
			EAN:           ean,
			DUN:           dun,
			NCM:           ncm,
			Lojas:         lojas,
			Quantidades:   quantidades,
			Subgrupo:      subgrupo,
		})
	}

	var avisos []string
	if contadores.semNCM > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas com NCM ausente", contadores.semNCM))
	}
	if contadores.expandidas > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas sem lojas — expandidas para todas", contadores.expandidas))
	}
	if subgrupoCol >= 0 && contadores.semSubgrupo > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas sem SUBGRUPO", contadores.semSubgrupo))
	}
	if contadores.subgrupoInvalido > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas com SUBGRUPO não numérico — serão puladas", contadores.subgrupoInvalido))
	}
	return result, avisos, nil
}

// lerPlanilhaPT lê o modelo de pré-transferência: header na linha 1, com colunas
// "codigo" ou "barra", "origem" e colunas de loja com quantidades.
func lerPlanilhaPT(rows [][]string) ([]ProdutoLinha, []string, error) {
	header := rows[0]

	codigoCol, barraCol, origemCol := -1, -1, -1
	type lojaCol struct {
		idx  int
		loja int
	}
	var lojaCols []lojaCol

	for i, h := range header {
		h = strings.TrimSpace(h)
		low := strings.ToLower(h)
		switch low {
		case "codigo", "código":
			if codigoCol == -1 {
				codigoCol = i
			}
		case "barra":
			if barraCol == -1 {
				barraCol = i
			}
		case "origem":
			if origemCol == -1 {
				origemCol = i
			}
		}
		if v, err := strconv.Atoi(h); err == nil {
			lojaCols = append(lojaCols, lojaCol{i, v})
		}
	}

	if codigoCol == -1 && barraCol == -1 {
		return nil, nil, fmt.Errorf("coluna 'codigo' ou 'barra' não encontrada na linha 1")
	}
	if origemCol == -1 {
		return nil, nil, fmt.Errorf("coluna 'origem' não encontrada na linha 1")
	}
	if len(lojaCols) == 0 {
		return nil, nil, fmt.Errorf("nenhuma coluna de loja encontrada na linha 1")
	}

	var result []ProdutoLinha
	semOrigem, semIdent := 0, 0
	for rowIdx, row := range rows[1:] {
		linhaPlanilha := rowIdx + 2 // header em 1, dados começam em 2
		if len(row) == 0 {
			continue
		}

		var codigoPT, gtinPT int
		if codigoCol >= 0 && codigoCol < len(row) {
			if v, err := strconv.Atoi(strings.TrimSpace(row[codigoCol])); err == nil && v > 0 {
				codigoPT = v
			}
		}
		if codigoPT == 0 && barraCol >= 0 && barraCol < len(row) {
			if v, err := strconv.Atoi(strings.TrimSpace(row[barraCol])); err == nil && v > 0 {
				gtinPT = v
			}
		}
		if codigoPT == 0 && gtinPT == 0 {
			semIdent++
			continue
		}

		origem := 0
		if origemCol < len(row) {
			if v, err := strconv.Atoi(strings.TrimSpace(row[origemCol])); err == nil {
				origem = v
			}
		}
		if origem == 0 {
			semOrigem++
			continue
		}

		quantidades := map[int]int{}
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
			if lojaExcluida(lc.loja) {
				continue
			}
			lojas = append(lojas, lc.loja)
			quantidades[lc.loja] = int(v)
		}

		if len(lojas) == 0 {
			continue
		}

		result = append(result, ProdutoLinha{
			LinhaPlanilha: linhaPlanilha,
			Lojas:         lojas,
			Quantidades:   quantidades,
			LojaOrigem:    origem,
			CodigoPT:      codigoPT,
			GTINPT:        gtinPT,
		})
	}

	var avisos []string
	if semOrigem > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas sem coluna 'origem' preenchida — ignoradas", semOrigem))
	}
	if semIdent > 0 {
		avisos = append(avisos, fmt.Sprintf("%d linhas sem 'codigo' nem 'barra' — ignoradas", semIdent))
	}
	return result, avisos, nil
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
const divisaoKey = 1

func processarProduto(
	tenant, token string,
	produto ProdutoLinha,
	fazerLinhaCompra, fazerLinhaCompraCD, fazerLinhaLoja, fazerNCM, fazerSubgrupo bool,
	margens map[int]MargemSubgrupo,
	cdTipo string,
	rl *runLog,
	emit func(EventoSSE),
) {
	// Helper para preencher EAN/Linha automaticamente em cada evento.
	send := func(tipo, op, msg string) {
		emit(EventoSSE{Tipo: tipo, Op: op, Mensagem: msg, EAN: produto.EAN, Linha: produto.LinhaPlanilha})
	}

	// Lojas 3 e 306 são sempre obrigatórias em operações não-PT.
	if fazerLinhaCompra || fazerLinhaCompraCD || fazerLinhaLoja || fazerNCM {
		existentes := make(map[int]struct{}, len(produto.Lojas))
		for _, l := range produto.Lojas {
			existentes[l] = struct{}{}
		}
		for _, l := range []int{3, 306} {
			if _, ok := existentes[l]; !ok {
				produto.Lojas = append(produto.Lojas, l)
			}
		}
	}

	// Operações de loja exigem ao menos uma loja na planilha.
	exigeLojas := fazerLinhaCompra || fazerLinhaCompraCD || fazerLinhaLoja
	if exigeLojas && len(produto.Lojas) == 0 {
		send("info", "", fmt.Sprintf("EAN %s: sem lojas ativas, pulando", produto.EAN))
		return
	}

	// Linha de Compra e Sortimento sempre usam EAN 13.
	// Linha CD em UN também usa EAN 13; em CX usa o DUN 14 (lookup separado).
	// Alterar NCM também usa EAN 13.
	precisaUN := fazerLinhaCompra || fazerLinhaLoja || fazerNCM || fazerSubgrupo || (fazerLinhaCompraCD && cdTipo != "CX")
	precisaCX := fazerLinhaCompraCD && cdTipo == "CX"

	var infoUN *ProdutoInfo
	if precisaUN {
		i, status, body, err := consultarGTINv(tenant, token, produto.EAN)
		rl.httpCall("GTIN lookup (EAN)", "GET",
			fmt.Sprintf("/comercial/produtos/gtin/%s", produto.EAN), nil, status, body, err)
		if err != nil {
			send("erro", "GTIN", fmt.Sprintf("EAN %s: %v", produto.EAN, err))
			return
		}
		if i.ProdutoKey == 0 {
			send("aviso", "GTIN", fmt.Sprintf("EAN %s: produto não encontrado no tenant", produto.EAN))
			return
		}
		infoUN = i
	}

	// ── 0. Alterar NCM ──
	if fazerNCM && infoUN != nil {
		op := "NCM"
		if produto.NCM == "" {
			send("aviso", op, fmt.Sprintf("EAN %s: NCM ausente na planilha, pulando", produto.EAN))
		} else {
			ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/comercial/produtos/%d", tenant, infoUN.ProdutoKey)
			payload := PayloadNCM{NCM: produto.NCM}
			status, body, err := putAPI(token, ep, payload)
			rl.httpCall(fmt.Sprintf("NCM | EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey), "PUT", ep, payload, status, body, err)
			if err != nil || (status != 200 && status != 201 && status != 204) {
				send("erro", op, fmt.Sprintf("EAN %s — %s", produto.EAN, apiErrMsg(status, body, err)))
			} else {
				send("ok", op, fmt.Sprintf("EAN %s → NCM %s", produto.EAN, produto.NCM))
			}
		}
	}

	// ── 0b. Alterar Subgrupo ──
	if fazerSubgrupo && infoUN != nil {
		op := "Subgrupo"
		if produto.Subgrupo == 0 {
			send("aviso", op, fmt.Sprintf("EAN %s: SUBGRUPO ausente na planilha, pulando", produto.EAN))
		} else if m, ok := margens[produto.Subgrupo]; !ok || !m.Existe {
			send("erro", op, fmt.Sprintf("EAN %s: subgrupo %d não existe na árvore mercadológica", produto.EAN, produto.Subgrupo))
		} else if !m.TemMargem {
			send("erro", op, fmt.Sprintf("EAN %s: subgrupo %d sem margens cadastradas", produto.EAN, produto.Subgrupo))
		} else {
			ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/comercial/produtos/%d", tenant, infoUN.ProdutoKey)
			payload := PayloadSubgrupo{
				SubGrupoProduto: produto.Subgrupo,
				ValorMargem:     m.ValorMargem,
				QuebraMargem:    m.QuebraMargem,
			}
			status, body, err := putAPI(token, ep, payload)
			rl.httpCall(fmt.Sprintf("Subgrupo | EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey), "PUT", ep, payload, status, body, err)
			if err != nil || (status != 200 && status != 201 && status != 204) {
				send("erro", op, fmt.Sprintf("EAN %s — %s", produto.EAN, apiErrMsg(status, body, err)))
			} else {
				send("ok", op, fmt.Sprintf("EAN %s → subgrupo %d (margem %.2f)", produto.EAN, produto.Subgrupo, m.ValorMargem))
			}
		}
	}

	// ── 1. Linha de Compra (paralelo, uma req por loja) ──
	if fazerLinhaCompra && infoUN != nil {
		op := "Compra"
		ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadecompra", tenant)

		var wg sync.WaitGroup
		var sucesso int32
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
				pb, _ := json.Marshal(payload)
				log.Printf("[Compra] EAN %s | loja %d | payload=%s", produto.EAN, l, pb)
				status, body, err := postAPI(token, ep, payload)
				log.Printf("[Compra] EAN %s | loja %d | status=%d | body=%s", produto.EAN, l, status, body)
				rl.httpCall(fmt.Sprintf("Compra | EAN %s (produto %d) | loja %d", produto.EAN, infoUN.ProdutoKey, l), "POST", ep, payload, status, body, err)
				if err != nil || (status != 200 && status != 201) {
					send("erro", op, fmt.Sprintf("EAN %s loja %d — %s", produto.EAN, l, apiErrMsg(status, body, err)))
				} else {
					atomic.AddInt32(&sucesso, 1)
				}
			}(loja)
		}
		wg.Wait()
		if sucesso > 0 {
			send("ok", op, fmt.Sprintf("EAN %s — %d/%d lojas OK", produto.EAN, sucesso, len(produto.Lojas)))
		}
	}

	// ── 2. Sortimento / Linha de Loja ──
	if fazerLinhaLoja && infoUN != nil {
		op := "Sortimento"
		linhas := make([]LinhaLoja, len(produto.Lojas))
		for i, l := range produto.Lojas {
			linhas[i] = LinhaLoja{
				LojaKey:       l,
				DistribuirPor: "UNIDADE",
				Multiplo:      1,
			}
		}
		ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadeloja/%d", tenant, infoUN.ProdutoKey)
		payload := PayloadLinhaLoja{Linhas: linhas}
		status, body, err := postAPI(token, ep, payload)
		rl.httpCall(fmt.Sprintf("Sortimento | EAN %s (produto %d)", produto.EAN, infoUN.ProdutoKey), "POST", ep, payload, status, body, err)
		if err != nil || (status != 200 && status != 201) {
			send("erro", op, fmt.Sprintf("EAN %s — %s", produto.EAN, apiErrMsg(status, body, err)))
		} else {
			send("ok", op, fmt.Sprintf("EAN %s — %d lojas OK", produto.EAN, len(produto.Lojas)))
		}
	}

	// ── 3. Linha de Compra CD ──
	if fazerLinhaCompraCD {
		var infoCD *ProdutoInfo
		gtinCD := produto.EAN
		if precisaCX {
			if produto.DUN == "" {
				send("aviso", "CD", fmt.Sprintf("EAN %s: sem DUN 14, pulando CD", produto.EAN))
				return
			}
			i, status, body, err := consultarGTINv(tenant, token, produto.DUN)
			rl.httpCall("GTIN lookup (DUN)", "GET",
				fmt.Sprintf("/comercial/produtos/gtin/%s", produto.DUN), nil, status, body, err)
			if err != nil {
				send("erro", "CD", fmt.Sprintf("DUN %s: %v", produto.DUN, err))
				return
			}
			if i.ProdutoKey == 0 {
				send("aviso", "CD", fmt.Sprintf("DUN %s: produto não encontrado, pulando CD", produto.DUN))
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

		cds := agruparPorCD(produto.Lojas)

		var wg sync.WaitGroup
		for cdKey, lojas := range cds {
			wg.Add(1)
			go func(cdKey int, lojas []int) {
				defer wg.Done()
				op := fmt.Sprintf("CD%d", cdKey)
				lojasFinal := dedupInts(append([]int{cdKey}, lojas...))
				payload := PayloadLinhaCompraCD{
					FornecedorKey: infoCD.FornecedorKey,
					DivisaoKey:    divisaoKey,
					ProdutoKey:    infoCD.ProdutoKey,
					LojaKey:       lojasFinal,
				}
				ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/compras/sortimento/linhadecompra/cd/%d", tenant, cdKey)
				status, body, err := postAPI(token, ep, payload)
				rl.httpCall(fmt.Sprintf("CD%d | %s %s (produto %d)", cdKey, gtinLabel(precisaCX), gtinCD, infoCD.ProdutoKey), "POST", ep, payload, status, body, err)
				if err != nil || (status != 200 && status != 201) {
					send("erro", op, fmt.Sprintf("%s %s — %s", gtinLabel(precisaCX), gtinCD, apiErrMsg(status, body, err)))
				} else {
					send("ok", op, fmt.Sprintf("%s %s — %d lojas OK", gtinLabel(precisaCX), gtinCD, len(lojasFinal)))
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

// ─── Pré-transferência ────────────────────────────────────────────────────────

const preTransfChunkSize = 500

// processarPreTransferencia agrupa produtos por loja origem e dispara um POST por chunk.
// formato: "ficha" → usa lojaOrigemUI; "pt" → usa produto.LojaOrigem.
func processarPreTransferencia(
	tenant, token, formato string,
	produtos []ProdutoLinha,
	lojaOrigemUI int,
	rl *runLog,
	emit func(EventoSSE),
	cancel <-chan struct{},
) (preTransfKeys []int) {
	// Agrupa entradas (produtoKey/gtin, destino, qty) por loja origem.
	porOrigem := map[int][]ItemPreTransferencia{}
	for _, p := range produtos {
		var origem int
		if formato == "pt" {
			origem = p.LojaOrigem
		} else {
			origem = lojaOrigemUI
		}
		if origem == 0 {
			emit(EventoSSE{Tipo: "aviso", Op: "PT", Mensagem: fmt.Sprintf("linha %d sem origem definida — pulando", p.LinhaPlanilha), Linha: p.LinhaPlanilha})
			continue
		}
		for _, dest := range p.Lojas {
			if dest == origem {
				continue // origem nunca é destino dela mesma
			}
			qty := p.Quantidades[dest]
			if qty <= 0 {
				continue
			}
			item := ItemPreTransferencia{
				LojaDestinoKey:                dest,
				Quantidade:                    qty,
				DeveInformarOProdutoKeyOuGtin: false,
				QuantidadeFracionada:          false,
			}
			// Identificador do produto:
			// - PT com codigo → produtoKey direto
			// - PT com barra → gtin direto
			// - ficha → gtin do EAN 13
			if p.CodigoPT > 0 {
				item.ProdutoKey = p.CodigoPT
			} else if p.GTINPT > 0 {
				item.GTIN = p.GTINPT
			} else if p.EAN != "" {
				if v, err := strconv.Atoi(p.EAN); err == nil {
					item.GTIN = v
				} else {
					emit(EventoSSE{Tipo: "aviso", Op: "PT", Mensagem: fmt.Sprintf("linha %d EAN inválido (%q) — pulando", p.LinhaPlanilha, p.EAN), Linha: p.LinhaPlanilha})
					continue
				}
			} else {
				emit(EventoSSE{Tipo: "aviso", Op: "PT", Mensagem: fmt.Sprintf("linha %d sem identificador — pulando", p.LinhaPlanilha), Linha: p.LinhaPlanilha})
				continue
			}
			porOrigem[origem] = append(porOrigem[origem], item)
		}
	}

	if len(porOrigem) == 0 {
		emit(EventoSSE{Tipo: "aviso", Op: "PT", Mensagem: "nada a transferir após filtrar destinos"})
		return
	}

	for origem, itens := range porOrigem {
		// Cancelamento entre origens.
		select {
		case <-cancel:
			emit(EventoSSE{Tipo: "aviso", Mensagem: "⛔ cancelado"})
			return
		default:
		}

		emit(EventoSSE{Tipo: "info", Op: "PT", Mensagem: fmt.Sprintf("origem %d — %d entradas em %d chunks", origem, len(itens), (len(itens)+preTransfChunkSize-1)/preTransfChunkSize)})

		for i := 0; i < len(itens); i += preTransfChunkSize {
			select {
			case <-cancel:
				emit(EventoSSE{Tipo: "aviso", Mensagem: "⛔ cancelado"})
				return
			default:
			}
			j := i + preTransfChunkSize
			if j > len(itens) {
				j = len(itens)
			}
			chunk := itens[i:j]
			payload := PayloadPreTransferencia{
				LojaOrigemKey:                             origem,
				QuantidadeDiasSugestao:                    0,
				ALojaDeOrigemNaoPodeEstarNaListaDeDestino: 1,
				Produtos: chunk,
			}
			ep := fmt.Sprintf("https://erp.bluesoft.com.br/%s/api/modulos/estoque/operacoes-entre-lojas/pre-transferencia-multiloja", tenant)
			status, body, err := postAPI(token, ep, payload)
			rl.httpCall(fmt.Sprintf("Pré-transferência | origem %d | chunk %d-%d", origem, i+1, j), "POST", ep, payload, status, body, err)
			if err != nil || (status != 200 && status != 201) {
				emit(EventoSSE{Tipo: "erro", Op: "PT", Mensagem: fmt.Sprintf("origem %d (chunk %d-%d) — %s", origem, i+1, j, apiErrMsg(status, body, err))})
				continue
			}
			// Tenta extrair preTransferenciaMultilojasKey da resposta.
			ref := ""
			var resp map[string]any
			if json.Unmarshal([]byte(body), &resp) == nil {
				if k, ok := resp["preTransferenciaMultilojasKey"].(float64); ok {
					key := int(k)
					ref = fmt.Sprintf(" #%d", key)
					preTransfKeys = append(preTransfKeys, key)
				}
			}
			emit(EventoSSE{Tipo: "ok", Op: "PT", Mensagem: fmt.Sprintf("origem %d — %d entradas OK%s", origem, len(chunk), ref)})
		}
	}
	return
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

// handleSetup retorna o status atual do .env (configurado ou não) sem expor segredos.
func handleSetup(w http.ResponseWriter, r *http.Request) {
	tenant := os.Getenv("BLUESOFT_TENANT")
	clientID := os.Getenv("client_id")
	clientSecret := os.Getenv("client_secret")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"configured": tenant != "" && clientID != "" && clientSecret != "",
		"tenant":     tenant,
	})
}

// handleSetupTest tenta autenticar com as credenciais informadas (não grava nada).
func handleSetupTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Tenant       string `json:"tenant"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := obterToken(req.Tenant, req.ClientID, req.ClientSecret); err != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleSetupSave grava o .env ao lado do executável e atualiza os env vars do processo.
func handleSetupSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Tenant       string `json:"tenant"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	envMap := map[string]string{
		"BLUESOFT_TENANT": req.Tenant,
		"client_id":       req.ClientID,
		"client_secret":   req.ClientSecret,
	}
	if err := godotenv.Write(envMap, envPath()); err != nil {
		http.Error(w, "não foi possível gravar .env: "+err.Error(), 500)
		return
	}
	for k, v := range envMap {
		os.Setenv(k, v)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

// handleUpload recebe a planilha, valida e devolve metadata. Não executa nada.
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "arquivo não recebido", 400)
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "bluesoft-*.xlsx")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	size, _ := io.Copy(tmp, file)
	tmp.Close()

	produtos, avisos, formato, err := lerPlanilha(tmp.Name())
	if err != nil {
		os.Remove(tmp.Name())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	sess := &FileSession{
		ID:        novoID(),
		Filename:  header.Filename,
		Size:      size,
		Path:      tmp.Name(),
		Formato:   formato,
		Produtos:  produtos,
		Avisos:    avisos,
		CreatedAt: time.Now(),
	}
	sessoes.Store(sess.ID, sess)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       sess.ID,
		"filename": sess.Filename,
		"size":     sess.Size,
		"formato":  formato,
		"linhas":   len(produtos),
		"avisos":   avisos,
	})
}

// runOpcoes é o body comum de /api/run e /api/retry.
type runOpcoes struct {
	ID               string `json:"id"`
	LinhaCompra      bool   `json:"linhaCompra"`
	LinhaCompraCD    bool   `json:"linhaCompraCD"`
	LinhaLoja        bool   `json:"linhaLoja"`
	AlterarNCM       bool   `json:"alterarNCM"`
	AlterarSubgrupo  bool   `json:"alterarSubgrupo"`
	PreTransferencia bool   `json:"preTransferencia"`
	CdTipo           string `json:"cdTipo"`
	LojaOrigem       int    `json:"lojaOrigem"`       // só usado quando formato=ficha + PT
	Linhas           []int  `json:"linhas,omitempty"` // só usado em /api/retry
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	executaSSE(w, r, false)
}

func handleRetry(w http.ResponseWriter, r *http.Request) {
	executaSSE(w, r, true)
}

func executaSSE(w http.ResponseWriter, r *http.Request, isRetry bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req runOpcoes
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.CdTipo == "" {
		req.CdTipo = "UN"
	}
	if !req.LinhaCompra && !req.LinhaCompraCD && !req.LinhaLoja && !req.AlterarNCM && !req.AlterarSubgrupo && !req.PreTransferencia {
		http.Error(w, "selecione ao menos uma operação", 400)
		return
	}
	// Pré-transferência é exclusiva (não combina com as outras ops no mesmo run).
	if req.PreTransferencia && (req.LinhaCompra || req.LinhaCompraCD || req.LinhaLoja || req.AlterarNCM || req.AlterarSubgrupo) {
		http.Error(w, "Pré-transferência roda sozinha — desmarque as outras operações", 400)
		return
	}
	val, ok := sessoes.Load(req.ID)
	if !ok {
		http.Error(w, "sessão não encontrada — recarregue o arquivo", 404)
		return
	}
	sess := val.(*FileSession)
	// Modelo PT só aceita Pré-transferência.
	if sess.Formato == "pt" && !req.PreTransferencia {
		http.Error(w, "essa planilha só suporta Pré-transferência", 400)
		return
	}
	// Ficha + PT exige loja origem.
	if req.PreTransferencia && sess.Formato == "ficha" && req.LojaOrigem == 0 {
		http.Error(w, "informe a loja origem para Pré-transferência na ficha de cadastro", 400)
		return
	}

	// Filtra produtos pelas linhas pedidas (só no retry).
	produtos := sess.Produtos
	if isRetry && len(req.Linhas) > 0 {
		set := make(map[int]bool, len(req.Linhas))
		for _, l := range req.Linhas {
			set[l] = true
		}
		filtrados := make([]ProdutoLinha, 0, len(req.Linhas))
		for _, p := range sess.Produtos {
			if set[p.LinhaPlanilha] {
				filtrados = append(filtrados, p)
			}
		}
		produtos = filtrados
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, fok := w.(http.Flusher)
	if !fok {
		http.Error(w, "streaming não suportado", 500)
		return
	}

	rl := newRunLog()
	defer rl.close()

	var sendMu sync.Mutex
	emit := func(evt EventoSSE) {
		sendMu.Lock()
		defer sendMu.Unlock()
		b, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		rl.event(evt.Tipo, evt.Mensagem)
	}
	emitInfo := func(msg string) { emit(EventoSSE{Tipo: "info", Mensagem: msg}) }

	if rl != nil {
		rl.writeln("───── Início do upload ─────")
		rl.writeln("Arquivo: %s", sess.Filename)
		rl.writeln("Log: %s", rl.path)
	}

	cancelMu.Lock()
	cancelChan = make(chan struct{})
	localCancel := cancelChan
	cancelMu.Unlock()

	tenant := os.Getenv("BLUESOFT_TENANT")
	clientID := os.Getenv("client_id")
	clientSecret := os.Getenv("client_secret")
	if tenant == "" || clientID == "" || clientSecret == "" {
		emit(EventoSSE{Tipo: "erro", Mensagem: "credenciais ausentes — configure em /setup"})
		emit(EventoSSE{Tipo: "fim", Mensagem: ""})
		return
	}

	emitInfo("🔐 Autenticando…")
	token, err := obterToken(tenant, clientID, clientSecret)
	if err != nil {
		emit(EventoSSE{Tipo: "erro", Mensagem: "falha na autenticação: " + err.Error()})
		emit(EventoSSE{Tipo: "fim", Mensagem: ""})
		return
	}
	emit(EventoSSE{Tipo: "ok", Op: "Auth", Mensagem: "autenticado"})

	ops := []string{}
	if req.AlterarNCM {
		ops = append(ops, "Alterar NCM")
	}
	if req.AlterarSubgrupo {
		ops = append(ops, "Alterar Subgrupo")
	}
	if req.LinhaCompra {
		ops = append(ops, "Linha Compra")
	}
	if req.LinhaLoja {
		ops = append(ops, "Sortimento")
	}
	if req.LinhaCompraCD {
		ops = append(ops, "Linha CD ("+req.CdTipo+")")
	}
	if req.PreTransferencia {
		ops = append(ops, "Pré-transferência")
	}
	emitInfo("Operações: " + strings.Join(ops, " · "))
	emitInfo(fmt.Sprintf("Processando %d linhas…", len(produtos)))

	var okCount, errCount, warnCount atomic.Int32
	emitContando := func(evt EventoSSE) {
		emit(evt)
		switch evt.Tipo {
		case "ok":
			okCount.Add(1)
		case "erro":
			errCount.Add(1)
		case "aviso":
			warnCount.Add(1)
		}
	}

	var preTransfKeys []int
	if req.PreTransferencia {
		preTransfKeys = processarPreTransferencia(tenant, token, sess.Formato, produtos, req.LojaOrigem, rl, emitContando, localCancel)
		// PT não tem progresso por linha — manda 1/1 só pra UI completar a barra.
		sendMu.Lock()
		prog := map[string]any{"atual": 1, "total": 1}
		b, _ := json.Marshal(prog)
		fmt.Fprintf(w, "event: progress\ndata: %s\n\n", b)
		flusher.Flush()
		sendMu.Unlock()
	} else {
		// Alterar Subgrupo: consulta a árvore uma vez por subgrupo key distinta
		// presente na planilha e cacheia o resultado para o run inteiro.
		margens := map[int]MargemSubgrupo{}
		if req.AlterarSubgrupo {
			emitInfo("🌳 Consultando subgrupos na árvore mercadológica…")
			for _, p := range produtos {
				if p.Subgrupo == 0 {
					continue
				}
				if _, ok := margens[p.Subgrupo]; ok {
					continue
				}
				m, err := consultarMargemSubgrupo(tenant, token, p.Subgrupo, rl)
				if err != nil {
					// Falha de consulta vira "não existe" no mapa — erro por linha no loop.
					emit(EventoSSE{Tipo: "aviso", Op: "Subgrupo", Mensagem: fmt.Sprintf("subgrupo %d: %v", p.Subgrupo, err)})
					margens[p.Subgrupo] = MargemSubgrupo{Existe: false}
					continue
				}
				margens[p.Subgrupo] = m
			}
			emitInfo(fmt.Sprintf("%d subgrupos consultados", len(margens)))
		}

		cancelado := false
		// Linha de Compra deve estar 100% concluída antes de CD e Loja (o sistema
		// só aceita sortimento/CD após o produto já ter linha de compra em todas as lojas).
		needsTwoPasses := req.LinhaCompra && (req.LinhaCompraCD || req.LinhaLoja)
		totalSteps := len(produtos)
		if needsTwoPasses {
			totalSteps = len(produtos) * 2
		}
		// Quando não há dois passes, CD e Loja correm normalmente no único passe.
		passOneCD := !needsTwoPasses && req.LinhaCompraCD
		passOneLoja := !needsTwoPasses && req.LinhaLoja

		for i, p := range produtos {
			select {
			case <-localCancel:
				emit(EventoSSE{Tipo: "aviso", Mensagem: "⛔ cancelado pelo usuário"})
				cancelado = true
			default:
			}
			if cancelado {
				break
			}
			processarProduto(tenant, token, p,
				req.LinhaCompra, passOneCD, passOneLoja, req.AlterarNCM, req.AlterarSubgrupo,
				margens, req.CdTipo, rl, emitContando,
			)
			sendMu.Lock()
			prog := map[string]any{"atual": i + 1, "total": totalSteps}
			b, _ := json.Marshal(prog)
			fmt.Fprintf(w, "event: progress\ndata: %s\n\n", b)
			flusher.Flush()
			sendMu.Unlock()
		}

		if !cancelado && needsTwoPasses {
			emit(EventoSSE{Tipo: "info", Mensagem: "Linha de Compra concluída — iniciando Sortimento e Linha CD…"})
			for i, p := range produtos {
				select {
				case <-localCancel:
					emit(EventoSSE{Tipo: "aviso", Mensagem: "⛔ cancelado pelo usuário"})
					cancelado = true
				default:
				}
				if cancelado {
					break
				}
				processarProduto(tenant, token, p,
					false, req.LinhaCompraCD, req.LinhaLoja, false, false,
					nil, req.CdTipo, rl, emitContando,
				)
				sendMu.Lock()
				prog := map[string]any{"atual": len(produtos) + i + 1, "total": totalSteps}
				b, _ := json.Marshal(prog)
				fmt.Fprintf(w, "event: progress\ndata: %s\n\n", b)
				flusher.Flush()
				sendMu.Unlock()
			}
		}
	}

	resumo := map[string]any{
		"ok":     okCount.Load(),
		"erros":  errCount.Load(),
		"avisos": warnCount.Load(),
	}
	if len(preTransfKeys) > 0 {
		resumo["preTransferenciaKeys"] = preTransfKeys
	}
	rb, _ := json.Marshal(resumo)
	emit(EventoSSE{Tipo: "fim", Mensagem: string(rb)})
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

func handleFotosUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		http.Error(w, "arquivo muito grande ou inválido", 400)
		return
	}
	arquivo, header, err := r.FormFile("arquivo")
	if err != nil {
		http.Error(w, "arquivo não recebido", 400)
		return
	}
	defer arquivo.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	parte, err := mw.CreateFormFile("arquivo", header.Filename)
	if err != nil {
		http.Error(w, "erro interno ao preparar envio", 500)
		return
	}
	if _, err = io.Copy(parte, arquivo); err != nil {
		http.Error(w, "erro interno ao ler arquivo", 500)
		return
	}
	mw.Close()

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest(http.MethodPost, fotosVMURL, &buf)
	if err != nil {
		http.Error(w, "erro interno ao criar requisição", 500)
		return
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Servidor de fotos indisponível. Verifique se a VM está ligada.", 502)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Erro ao processar fotos: %s", strings.TrimSpace(string(msg))), 502)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="importacao_fotos.csv"`)
	if v := resp.Header.Get("X-Nao-Encontrados"); v != "" {
		w.Header().Set("X-Nao-Encontrados", v)
	}
	io.Copy(w, resp.Body)
}

func handleArvoreBaixar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	dep := strings.TrimSpace(r.URL.Query().Get("departamento"))
	if dep == "" {
		http.Error(w, "informe o departamento", 400)
		return
	}

	client := &http.Client{Timeout: 60 * time.Second}
	ep := arvoreVMURL + "?departamento=" + url.QueryEscape(dep)
	req, err := http.NewRequest(http.MethodGet, ep, nil)
	if err != nil {
		http.Error(w, "erro interno ao criar requisição", 500)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Servidor da árvore indisponível. Verifique se a VM está ligada.", 502)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		// O FastAPI devolve {"detail": "..."} — extrai a mensagem limpa quando possível.
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Detail != "" {
			msg = errResp.Detail
		}
		http.Error(w, msg, 502)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="arvore_mercadologica.csv"`)
	io.Copy(w, resp.Body)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlUI))
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

// needsRestart é setado para 1 quando updater() instala uma nova versão.
var needsRestart int32

func updater() {
	current := semver.MustParse(version)

	type ghAsset struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}
	type ghRelease struct {
		TagName string    `json:"tag_name"`
		Assets  []ghAsset `json:"assets"`
	}

	resp, err := http.Get("https://api.github.com/repos/MateusLDK/helper/releases/latest")
	if err != nil {
		log.Println("Erro ao verificar atualização:", err)
		return
	}
	defer resp.Body.Close()

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		log.Println("Erro ao verificar atualização:", err)
		return
	}

	tag := strings.TrimPrefix(strings.TrimPrefix(rel.TagName, "v"), "V")
	latest, err := semver.ParseTolerant(tag)
	if err != nil {
		log.Println("Erro ao verificar atualização:", err)
		return
	}
	if !latest.GT(current) {
		log.Println("✅ Já está na última versão:", version)
		return
	}

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	assetName := fmt.Sprintf("Helper_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)

	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		log.Println("Erro ao verificar atualização: asset não encontrado:", assetName)
		return
	}

	var dlClient *http.Client
	if runtime.GOOS == "windows" {
		// Windows usa .zip — deixa o http.Client descomprimir normalmente
		dlClient = &http.Client{}
	} else {
		// DisableCompression evita que o http.Client descomprima automaticamente o .tar.gz
		dlClient = &http.Client{Transport: &http.Transport{DisableCompression: true}}
	}
	dlResp, err := dlClient.Get(downloadURL)
	if err != nil {
		log.Println("Erro ao baixar atualização:", err)
		return
	}
	defer dlResp.Body.Close()

	var binary io.Reader
	if runtime.GOOS == "windows" {
		data, err := io.ReadAll(dlResp.Body)
		if err != nil {
			log.Println("Erro ao baixar atualização:", err)
			return
		}
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			log.Println("Erro ao descompactar atualização:", err)
			return
		}
		for _, f := range zr.File {
			if strings.HasSuffix(f.Name, ".exe") {
				rc, err := f.Open()
				if err != nil {
					log.Println("Erro ao descompactar atualização:", err)
					return
				}
				defer rc.Close()
				binary = rc
				break
			}
		}
	} else {
		gz, err := gzip.NewReader(dlResp.Body)
		if err != nil {
			log.Println("Erro ao descompactar atualização:", err)
			return
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		if _, err := tr.Next(); err != nil {
			log.Println("Erro ao descompactar atualização:", err)
			return
		}
		binary = tr
	}

	if binary == nil {
		log.Println("Erro ao atualizar: binário não encontrado no arquivo")
		return
	}

	if err := update.Apply(binary, update.Options{}); err != nil {
		log.Println("Erro ao aplicar atualização:", err)
		return
	}

	log.Printf("🔄 Atualizado de %s para %s!", version, latest)
	log.Println("Reinicie o programa para usar a nova versão")
	atomic.StoreInt32(&needsRestart, 1)
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"needsRestart": atomic.LoadInt32(&needsRestart) == 1,
	})
}

func main() {
	godotenv.Load(envPath())
	updater()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Não foi possível abrir porta:", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("http://127.0.0.1:%d", port)
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/setup", handleSetup)
	http.HandleFunc("/api/setup/test", handleSetupTest)
	http.HandleFunc("/api/setup/save", handleSetupSave)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/run", handleRun)
	http.HandleFunc("/api/retry", handleRetry)
	http.HandleFunc("/api/cancel", handleCancel)
	http.HandleFunc("/api/abrir-logs", handleAbrirLogs)
	http.HandleFunc("/fotos/upload", handleFotosUpload)
	http.HandleFunc("/arvore/baixar", handleArvoreBaixar)
	go func() {
		time.Sleep(300 * time.Millisecond)
		abrirNavegador(addr)
	}()
	fmt.Printf("Bluesoft Uploader rodando em %s\n", addr)
	http.Serve(ln, nil)
}
