FROM python:3.11-slim
WORKDIR /app
RUN pip install --no-cache-dir paho-mqtt boto3
COPY bridge.py .
CMD ["python", "-u", "bridge.py"]
