FROM python:3.10-slim

WORKDIR /app


RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# DEVOPS OPTIMIZATION:
# Prevent PyTorch from downloading heavy CUDA/GPU drivers (7GB+).
# Since this container is only used for live inference (predict), CPU-only is enough.
# This pre-installs the light CPU version before stable-baselines3 triggers.

RUN pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY serve_agent.py .
COPY ppo_tanks_model.zip .

EXPOSE 5000

CMD ["python", "serve_agent.py"]