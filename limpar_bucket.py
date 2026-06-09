import os
import requests
from datetime import datetime, timedelta, timezone

import boto3
from dotenv import load_dotenv

load_dotenv()

S3_BUCKET = os.getenv("S3_BUCKET")
S3_PREFIX = os.getenv("S3_PREFIX", "fotos-bluesoft/")
S3_REGION = os.getenv("S3_REGION", "us-east-1")
IDADE_HORAS = 24  # apaga objetos mais velhos que isso

def ping(url:str):
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


def limpar_bucket():
    s3 = boto3.client("s3", region_name=S3_REGION)
    corte = datetime.now(timezone.utc) - timedelta(hours=IDADE_HORAS)

    paginator = s3.get_paginator("list_objects_v2")
    pages = paginator.paginate(Bucket=S3_BUCKET, Prefix=S3_PREFIX)

    para_deletar = []
    for page in pages:
        for obj in page.get("Contents", []):
            if obj["LastModified"] < corte:
                para_deletar.append({"Key": obj["Key"]})

    if not para_deletar:
        print("Nada pra deletar.")
        return

    # delete_objects aceita até 1000 por vez
    for i in range(0, len(para_deletar), 1000):
        batch = para_deletar[i : i + 1000]
        s3.delete_objects(Bucket=S3_BUCKET, Delete={"Objects": batch})
        print(f"Deletados: {len(batch)} objetos")


if __name__ == "__main__":
    
    ping("http://localhost:8787/ping/upload-fotos-cleanup/start")
    try:
        limpar_bucket()
        ping("http://localhost:8787/ping/upload-fotos-cleanup")
    except Exception as e:
        ping("http://localhost:8787/ping/upload-fotos-cleanup/fail")
