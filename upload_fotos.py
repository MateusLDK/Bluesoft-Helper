import csv
import os
import shutil
import tempfile
import zipfile
from concurrent.futures import ThreadPoolExecutor, as_completed

import boto3
import psycopg2
import rarfile  # pip install rarfile
from dotenv import load_dotenv
from PIL import Image  # pip install pillow
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import FileResponse

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
            url = f"https://{S3_BUCKET}.s3.{S3_REGION}.amazonaws.com/{key}"
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


@app.get("/health")
def health():
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
