# Bluesoft Uploader

Ferramenta desktop para importar planilhas `.xlsx` para a API Bluesoft ERP.  
Roda no navegador via servidor local — **zero instalação, zero CGO, zero dor de cabeça**.

## Como buildar

### Pré-requisito
- [Go 1.22+](https://go.dev/dl/) instalado no Windows

### 1. Instalar dependências
```bash
go mod tidy
```

### 2. Rodar em dev
```bash
go run .
```
Abre o navegador automaticamente em `http://localhost:{porta}`.

### 3. Gerar o `.exe` final
```bash
go build -ldflags="-H windowsgui" -o BluesoftUploader.exe .
```
A flag `-H windowsgui` esconde o terminal ao abrir o `.exe`.

O executável gerado é **standalone** — basta distribuir o `.exe` + `.env`.

---

## Configuração via `.env`

Copie `.env.example` para `.env` na mesma pasta do `.exe`:

```env
BLUESOFT_TENANT=minipreco
client_id=SEU_CLIENT_ID
client_secret=SEU_CLIENT_SECRET
```

Os campos são pré-preenchidos na interface automaticamente.

---

## Formatos das planilhas

O mapeamento de colunas é configurável na interface.  
Os nomes **padrão** (quando a coluna tem o mesmo nome do campo da API):

### Linha de Compra (Loja)
`POST /api/v2/compras/sortimento/linhadecompra`

| Coluna        | Tipo                                      |
|---------------|-------------------------------------------|
| produtoKey    | inteiro                                   |
| fornecedorKey | inteiro                                   |
| divisaoKey    | inteiro                                   |
| compradorKey  | inteiro                                   |
| lojaKeys      | inteiros separados por vírgula: `3,4,6,7` |

### Linha de Compra CD
`POST /api/compras/sortimento/linhadecompra/cd/{cdKey}`

| Coluna        | Tipo                        |
|---------------|-----------------------------|
| produtoKey    | inteiro                     |
| fornecedorKey | inteiro                     |
| divisaoKey    | inteiro                     |
| cdKey         | inteiro (vai na URL)        |
| lojaKey       | inteiros separados por vírgula |

### Sortimento / Linha de Loja
`POST /api/compras/sortimento/linhadeloja/{produtoKey}`

| Coluna            | Tipo                              |
|-------------------|-----------------------------------|
| produtoKey        | inteiro (vai na URL)              |
| lojaKey           | inteiro                           |
| estoqueSeguranca  | inteiro                           |
| estoqueMaximo     | inteiro                           |
| pontoExtra        | inteiro                           |
| multiplo          | inteiro                           |
| distribuirPor     | PALETE / CAIXA / UNIDADE          |
| quantidadeAtacado | inteiro                           |
| crossDocking      | true/false ou sim/não             |
| remover           | true/false ou sim/não             |
| operacaoSuspensa  | true/false ou sim/não             |
| compraSuspensa    | true/false ou sim/não             |

> Para **Linha de Loja**, cada linha da planilha = uma loja de um produto.
