#!/usr/bin/env python3
"""
MQTT → Lambda Bridge
Mosquitto の MQTT メッセージを受信し、LocalStack の Lambda を invoke する
"""
import os
import json
import boto3
import paho.mqtt.client as mqtt

MQTT_BROKER = os.getenv("MQTT_BROKER", "mosquitto")
MQTT_PORT = int(os.getenv("MQTT_PORT", "1883"))
MQTT_TOPIC = os.getenv("MQTT_TOPIC", "register/power")
AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localstack:4566")
AWS_REGION = os.getenv("AWS_REGION", "ap-northeast-1")
LAMBDA_FUNCTION_NAME = os.getenv(
    "LAMBDA_FUNCTION_NAME", "stg_power_data_registration_lambda")

lambda_client = boto3.client(
    "lambda",
    endpoint_url=AWS_ENDPOINT_URL,
    region_name=AWS_REGION,
    aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
    aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
)


def on_connect(client, userdata, flags, rc, properties=None):
    print(f"[MQTT Bridge] Connected to {MQTT_BROKER}:{MQTT_PORT} (rc={rc})")
    client.subscribe(MQTT_TOPIC)
    print(f"[MQTT Bridge] Subscribed to {MQTT_TOPIC}")


def on_message(client, userdata, msg):
    print(f"[MQTT Bridge] Received on {msg.topic}: {msg.payload[:200]}")
    try:
        # AWS IoT ルールと同様の形式で Lambda へ渡す
        payload = json.loads(msg.payload.decode("utf-8"))
        payload["topic"] = msg.topic
        response = lambda_client.invoke(
            FunctionName=LAMBDA_FUNCTION_NAME,
            InvocationType="RequestResponse",
            Payload=json.dumps(payload),
        )
        result = response["Payload"].read().decode("utf-8")
        print(f"[MQTT Bridge] Lambda response: {result[:500]}")
    except Exception as e:
        print(f"[MQTT Bridge] Error invoking Lambda: {e}")


def main():
    print(
        f"[MQTT Bridge] Starting bridge: {MQTT_BROKER}:{MQTT_PORT} -> {LAMBDA_FUNCTION_NAME}")
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(MQTT_BROKER, MQTT_PORT, 60)
    client.loop_forever()


if __name__ == "__main__":
    main()
