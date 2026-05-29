# Bluesoft Uploader

Ferramenta desktop para importar planilhas `.xlsx` para a API Bluesoft ERP e para **importar fotos de produtos** em lote.  
Roda no navegador via servidor local — **zero instalação, zero CGO, zero dor de cabeça**.

São duas partes:

- **App Go** (`main.go` / `ui.go`) — interface desktop que importa as planilhas `.xlsx` e atua como proxy para o serviço de fotos.
- **Serviço de fotos** (`upload_fotos.py`) — API Python (FastAPI) que recebe um `.zip`/`.rar` de imagens, resolve os GTINs no banco e sobe as fotos para o S3. Roda separadamente (Docker), normalmente numa VM. Veja [Importação de fotos](#importação-de-fotos-de-produtos).

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

## Configuração das credenciais

As credenciais do Bluesoft são preenchidas **direto na interface** (tela "configurar credenciais"). O app gera/atualiza sozinho um `.env` na mesma pasta do `.exe` com os valores informados:

```env
BLUESOFT_TENANT=minipreco
client_id=SEU_CLIENT_ID
client_secret=SEU_CLIENT_SECRET
```

Não é preciso criar esse arquivo manualmente — ao salvar na interface, os campos passam a vir pré-preenchidos nas próximas vezes.

> O `.env.example` deste repositório é do **serviço de fotos** (banco + S3), não do app Go. Veja [Importação de fotos](#importação-de-fotos-de-produtos).

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

---

## Importação de fotos de produtos

Na tela inicial, selecione **"Importar fotos de produtos"** e envie um `.zip` ou `.rar` com as imagens. Cada arquivo de imagem deve ser nomeado com o **código de referência do produto** (ex: `7891234.jpg`). Enquanto processa, um overlay **"Aguarde, processando…"** cobre a tela; ao final, o navegador baixa um CSV `importacao_fotos.csv` (`gtin;url`) com as fotos enviadas. Imagens cujo código não foi encontrado no banco são listadas na própria tela.

### Como funciona

O app Go encaminha o arquivo para o serviço de fotos (`POST /processar-fotos`). O serviço:

1. Descompacta o `.zip`/`.rar`.
2. **Resolve todos os GTINs numa única consulta** ao banco (JOIN entre `fornecedor_produto` e `produto_d` com `WHERE codigo_referencia = ANY(...)`), em vez de uma query por foto.
3. **Sobe as imagens para o S3 em paralelo** (`ThreadPoolExecutor`), respeitando `S3_MAX_WORKERS`.
4. Devolve o CSV; os códigos não encontrados vão no header `X-Nao-Encontrados`.

> A URL do serviço usada pelo app está em `fotosVMURL` (`main.go`). Ajuste se o serviço rodar em outro host/porta.

> **Um único serviço atende vários helpers.** Não é necessário que cada máquina suba o seu — basta ter uma instância rodando (ex: numa VM) e apontar o `fotosVMURL` de todos os apps para ela.
>
> O S3 e o banco são os **do host que roda o serviço** (configurados no `.env` dele). Todos os helpers que apontam para esse serviço sobem as fotos no **mesmo bucket** e consultam o **mesmo banco**. Ou seja: quem não tem S3/banco configurado em lugar nenhum não consegue importar fotos — é preciso ter ao menos um host com essas credenciais.

### Rodar o serviço de fotos

Pré-requisito: [Docker](https://docs.docker.com/get-docker/) (e `unrar` já vem na imagem para suporte a `.rar`).

```bash
docker compose up -d --build
```

Sobe o serviço em `0.0.0.0:8000`. Para rodar em dev sem Docker:

```bash
pip install -r requirements.txt
python upload_fotos.py
```

### Configuração do serviço (`.env`)

O serviço de fotos lê suas credenciais de um `.env` (veja `.env.example`):

```env
DB_NAME=
DB_USER=
DB_PASSWORD=
DB_HOST=
DB_PORT=5432

S3_BUCKET=
S3_PREFIX=fotos-bluesoft/
S3_REGION=us-east-1
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=

# nº de uploads simultâneos para o S3 (padrão: 10)
S3_MAX_WORKERS=10
```
