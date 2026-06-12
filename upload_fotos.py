import csv
import io
import os
import shutil
import tempfile
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed

import boto3
import psycopg2
import rarfile  # pip install rarfile
import requests
from apscheduler.schedulers.background import BackgroundScheduler
from dotenv import load_dotenv
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import FileResponse, Response
from PIL import Image  # pip install pillow

load_dotenv()

app = FastAPI()

DB_CONFIG = {
    "dbname": os.getenv("DB_NAME"),
    "user": os.getenv("DB_USER"),
    "password": os.getenv("DB_PASSWORD"),
    "host": os.getenv("DB_HOST"),
    "port": os.getenv("DB_PORT"),
}

S3_BUCKET = os.getenv("S3_BUCKET")
S3_PREFIX = os.getenv("S3_PREFIX", "fotos-bluesoft/")
S3_REGION = os.getenv("S3_REGION", "us-east-1")

EXTENSOES = (".jpg", ".jpeg", ".png", ".webp", ".bmp", ".gif", ".tif", ".tiff")

S3_MAX_WORKERS = int(os.getenv("S3_MAX_WORKERS", "10"))

# tempo de validade do link presigned, em segundos (default: 7 dias, máximo da AWS)
S3_URL_EXPIRACAO = int(os.getenv("S3_URL_EXPIRACAO", str(7 * 24 * 60 * 60)))


def ping():
    url = "http://localhost:8787/ping/upload-fotos-service"
    try:
        r = requests.post(url, timeout=3)
        print(f"OK [{r.status_code}] {url}")
        print(r.text.strip())
    except requests.exceptions.ConnectionError:
        print("X  Nao consegui conectar — o Palantir NAO esta rodando.")
        print(f"   Alvo: {url}")
        print("   Suba o servidor em OUTRO terminal e deixe aberto:")
        print("       cd ~/Documentos/python/LOTR && python3 palantir.py")
        sys.exit(1)
    except requests.exceptions.Timeout:
        print(f"X  Timeout falando com {url} (servidor travado?).")
        sys.exit(1)


def converter_para_jpg(caminho_origem, caminho_destino):
    """Converte qualquer imagem para JPG, achatando transparência sobre fundo branco."""
    with Image.open(caminho_origem) as img:
        if img.mode in ("RGBA", "LA", "P"):
            img = img.convert("RGBA")
            fundo = Image.new("RGB", img.size, (255, 255, 255))
            fundo.paste(img, mask=img.split()[-1])
            img = fundo
        else:
            img = img.convert("RGB")
        img.save(caminho_destino, "JPEG", quality=90)


@app.post("/processar-fotos")
async def processar_fotos(arquivo: UploadFile = File(...)):
    print("Arquivo recebido")
    tmp_dir = tempfile.mkdtemp()
    csv_path = os.path.join(tmp_dir, "importacao_fotos.csv")

    try:
        # salva o zip/rar recebido
        arquivo_path = os.path.join(tmp_dir, arquivo.filename)
        with open(arquivo_path, "wb") as f:
            f.write(await arquivo.read())

        # descomprime
        fotos_dir = os.path.join(tmp_dir, "fotos")
        os.makedirs(fotos_dir)

        nome = arquivo.filename.lower()
        if nome.endswith(".zip"):
            with zipfile.ZipFile(arquivo_path, "r") as z:
                z.extractall(fotos_dir)
        elif nome.endswith(".rar"):
            with rarfile.RarFile(arquivo_path, "r") as r:
                r.extractall(fotos_dir)
        else:
            raise HTTPException(
                status_code=400, detail="Formato não suportado. Envie .zip ou .rar"
            )

        print(f"Arquivos encontrados: {os.listdir(fotos_dir)}")

        # conecta DB e S3
        print("Conectando ao DB e ao S3")
        conn = psycopg2.connect(**DB_CONFIG)
        cur = conn.cursor()
        s3 = boto3.client(
            "s3",
            region_name=S3_REGION,
            aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID"),
            aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY"),
        )

        nao_encontrados = []
        uploadados = []

        # 1) coleta todas as fotos válidas antes de consultar o banco
        fotos = []
        for root, dirs, files in os.walk(fotos_dir):
            for arquivo_foto in sorted(files):
                if not arquivo_foto.lower().endswith(EXTENSOES):
                    continue
                fotos.append(
                    {
                        "caminho": os.path.join(root, arquivo_foto),
                        "arquivo_foto": arquivo_foto,
                        "nome_sem_extensao": os.path.splitext(arquivo_foto)[0],
                    }
                )

        # 2) uma query só: resolve os gtins de todas as referências de uma vez
        referencias = list({f["nome_sem_extensao"] for f in fotos})
        cur.execute(
            """
            SELECT fp.codigo_referencia, pd.gtin_principal
            FROM fornecedor_produto fp
            JOIN produto_d pd ON pd.produto_key = fp.produto_key
            WHERE fp.codigo_referencia = ANY(%s)
            """,
            (referencias,),
        )
        gtin_por_ref = {ref: gtin for ref, gtin in cur.fetchall()}

        cur.close()
        conn.close()

        # 3) upload paralelo no S3
        def enviar(foto):
            gtin = gtin_por_ref[foto["nome_sem_extensao"]]
            arquivo_jpg = f"{foto['nome_sem_extensao']}.jpg"
            jpg_path = os.path.join(
                os.path.dirname(foto["caminho"]), f"__convertido__{arquivo_jpg}"
            )
            converter_para_jpg(foto["caminho"], jpg_path)
            key = f"{S3_PREFIX}{arquivo_jpg}"
            print(f"Enviando imagem do item: {gtin}")
            s3.upload_file(
                jpg_path, S3_BUCKET, key, ExtraArgs={"ContentType": "image/jpeg"}
            )
            url = s3.generate_presigned_url(
                "get_object",
                Params={"Bucket": S3_BUCKET, "Key": key},
                ExpiresIn=S3_URL_EXPIRACAO,
            )
            return arquivo_jpg, key, gtin, url

        a_enviar = []
        for foto in fotos:
            if foto["nome_sem_extensao"] not in gtin_por_ref:
                nao_encontrados.append(foto["arquivo_foto"])
            else:
                a_enviar.append(foto)

        linhas = []
        with ThreadPoolExecutor(max_workers=S3_MAX_WORKERS) as executor:
            futures = {executor.submit(enviar, foto): foto for foto in a_enviar}
            for future in as_completed(futures):
                foto = futures[future]
                try:
                    arquivo_foto, key, gtin, url = future.result()
                    uploadados.append(key)
                    linhas.append((arquivo_foto, gtin, url))
                except Exception as e:
                    gtin = gtin_por_ref.get(foto["nome_sem_extensao"])
                    print(f"Erro ao enviar imagem do item {gtin}: {str(e)}")

        # 4) escreve o CSV depois (csv.writer não é thread-safe), em ordem estável
        linhas.sort(key=lambda linha: linha[0])
        with open(csv_path, "w", newline="", encoding="utf-8") as f:
            writer = csv.writer(f, delimiter=";")
            writer.writerow(["gtin", "url"])
            for _, gtin, url in linhas:
                writer.writerow([gtin, url])

        headers = {}
        if nao_encontrados:
            headers["X-Nao-Encontrados"] = ",".join(nao_encontrados)

        return FileResponse(
            csv_path,
            media_type="text/csv",
            filename="importacao_fotos.csv",
            headers=headers,
            background=None,  # não deleta antes de servir
        )

    except HTTPException:
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise
    except Exception as e:
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise HTTPException(status_code=500, detail=str(e))

    # cleanup acontece depois do FileResponse ser enviado
    # pra garantir, usa um endpoint de health pra confirmar que tá vivo
    # e considera um background task pra limpar o tmp_dir


@app.get("/arvore")
def baixar_arvore(departamento: str):
    departamento = (departamento or "").strip()
    if not departamento:
        raise HTTPException(status_code=400, detail="Informe o departamento")

    conn = None
    cur = None
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        cur = conn.cursor()
        cur.execute(
            """
            SELECT departamento, secao, grupo, subgrupo, subgrupo_produto_key
            FROM categoria
            WHERE departamento = %s
            ORDER BY secao, grupo, subgrupo
            """,
            (departamento,),
        )
        linhas = cur.fetchall()
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        if cur is not None:
            cur.close()
        if conn is not None:
            conn.close()

    if not linhas:
        raise HTTPException(
            status_code=404, detail=f"Departamento '{departamento}' não encontrado"
        )

    buf = io.StringIO()
    buf.write("﻿")  # BOM para acentos abrirem certo no Excel
    writer = csv.writer(buf, delimiter=";")
    writer.writerow(
        ["departamento", "secao", "grupo", "subgrupo", "subgrupo_produto_key"]
    )
    for linha in linhas:
        writer.writerow(linha)

    return Response(
        content=buf.getvalue(),
        media_type="text/csv; charset=utf-8",
        headers={
            "Content-Disposition": 'attachment; filename="arvore_mercadologica.csv"'
        },
    )


@app.get("/health")
def health():
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn

    ping()

    scheduler = BackgroundScheduler()
    scheduler.add_job(ping, "interval", minutes=60)
    scheduler.start()

    uvicorn.run(app, host="0.0.0.0", port=8000)
