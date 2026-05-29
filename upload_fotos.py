import csv
import os
import shutil
import subprocess
import tempfile
import zipfile
from time import sleep

import boto3
import psycopg2
import rarfile  # pip install rarfile
from dotenv import load_dotenv
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import FileResponse

load_dotenv()


def conectarVPN():
    if os.getenv("SKIP_VPN", "false").lower() == "true":
        print("SKIP_VPN=true, pulando conexão VPN")
        return
    print("Conectando ao VPN")
    subprocess.Popen(
        [
            "sudo",
            "openvpn",
            "--config",
            "vpn.ovpn",
            "--daemon",
        ],
    )
    sleep(15)
    print("VPN conectada com sucesso")


def desconectarVPN():
    if os.getenv("SKIP_VPN", "false").lower() == "true":
        return
    print("Desconectando do VPN")
    subprocess.Popen(["sudo", "pkill", "openvpn"])


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

EXTENSOES = (".jpg", ".jpeg", ".png", ".webp")


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

        with open(csv_path, "w", newline="", encoding="utf-8") as f:
            writer = csv.writer(f, delimiter=";")
            writer.writerow(["gtin", "url"])

            for root, dirs, files in os.walk(fotos_dir):
                for arquivo_foto in sorted(files):
                    print(arquivo_foto)
                    if not arquivo_foto.lower().endswith(EXTENSOES):
                        continue
                    caminho = os.path.join(root, arquivo_foto)
                    nome_sem_extensao = os.path.splitext(arquivo_foto)[0]

                    cur.execute(
                        "SELECT produto_key FROM fornecedor_produto WHERE codigo_referencia = %s",
                        (nome_sem_extensao,),
                    )
                    row = cur.fetchone()
                    if row is None:
                        nao_encontrados.append(arquivo_foto)
                        continue
                    produto_key = row[0]

                    cur.execute(
                        "SELECT gtin_principal FROM produto_d WHERE produto_key = %s",
                        (produto_key,),
                    )
                    row = cur.fetchone()
                    if row is None:
                        nao_encontrados.append(arquivo_foto)
                        continue

                    gtin = row[0]
                    caminho = os.path.join(root, arquivo_foto)  # CORRETO
                    key = f"{S3_PREFIX}{arquivo_foto}"
                    ct = (
                        "image/jpeg"
                        if arquivo_foto.lower().endswith((".jpg", ".jpeg"))
                        else "image/png"
                    )
                    print(f"Enviando imagem do item: {gtin}")

                    try:
                        s3.upload_file(
                            caminho, S3_BUCKET, key, ExtraArgs={"ContentType": ct}
                        )
                        url = f"https://{S3_BUCKET}.s3.{S3_REGION}.amazonaws.com/{key}"
                        uploadados.append(key)
                        writer.writerow([gtin, url])
                    except Exception as e:
                        print(f"Erro ao enviar imagem do item {gtin}: {str(e)}")

        cur.close()
        conn.close()

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

    try:
        conectarVPN()
        uvicorn.run(app, host="0.0.0.0", port=8000)

    except Exception as e:
        print(e)

    finally:
        desconectarVPN()
