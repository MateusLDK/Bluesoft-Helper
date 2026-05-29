FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    unrar \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY upload_fotos.py .

EXPOSE 8000

CMD ["python", "upload_fotos.py"]
