#!/usr/bin/env python3
"""
MQTT → DB Bridge (ローカル環境専用)
Mosquitto の MQTT メッセージを受信し、PostgreSQL と DynamoDB (LocalStack) へ直接書き込む。

LocalStack Community 版では Lambda コンテナ起動がタイムアウトするため、
Lambda を経由せず直接 DB へ書き込む方式を採用している。
power-data-registration-lambda と同等のロジックを実装。
"""
import os
import json
import time
import traceback

import boto3
import psycopg2
import paho.mqtt.client as mqtt

# ---------- 設定 ----------
MQTT_BROKER = os.getenv("MQTT_BROKER", "mosquitto")
MQTT_PORT = int(os.getenv("MQTT_PORT", "1883"))
MQTT_TOPIC = os.getenv("MQTT_TOPIC", "register/power")

# DynamoDB (LocalStack)
AWS_ENDPOINT_URL = os.getenv("AWS_ENDPOINT_URL", "http://localstack:4566")
AWS_REGION = os.getenv("AWS_REGION", "ap-northeast-1")
DYNAMODB_TABLE = os.getenv("DYNAMODB_TABLE", "tmp_table")

# PostgreSQL
DB_HOST = os.getenv("DB_HOST", "postgres")
DB_PORT = int(os.getenv("DB_PORT", "5432"))
DB_USER = os.getenv("DB_USER", "postgres")
DB_PASSWORD = os.getenv("DB_PASSWORD", "postgres")
DB_NAME = os.getenv("DB_NAME", "stg")

# ---------- DynamoDB クライアント ----------
dynamodb_client = boto3.client(
    "dynamodb",
    endpoint_url=AWS_ENDPOINT_URL,
    region_name=AWS_REGION,
    aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
    aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
)

# ---------- PostgreSQL ----------
pg_conn = None


def get_pg_conn():
    """PostgreSQL コネクションを取得 (リトライ付き)"""
    for attempt in range(10):
        try:
            conn = psycopg2.connect(
                host=DB_HOST,
                port=DB_PORT,
                user=DB_USER,
                password=DB_PASSWORD,
                dbname=DB_NAME,
                sslmode="disable",
            )
            conn.autocommit = False
            return conn
        except Exception as e:
            print(
                f"[MQTT Bridge] DB connection attempt {attempt+1}/10 failed: {e}")
            time.sleep(2)
    raise RuntimeError("Failed to connect to PostgreSQL after 10 attempts")


def ensure_pg_conn():
    """PostgreSQL コネクションが有効か確認し、切れていたら再接続"""
    global pg_conn
    try:
        if pg_conn is not None and not pg_conn.closed:
            pg_conn.rollback()
            return
    except Exception:
        pass
    pg_conn = get_pg_conn()
    print("[MQTT Bridge] PostgreSQL connected")


def register_power_to_postgres(session_id, device_id, gps_lat, gps_lon, power):
    """
    power_logs テーブルに発電データを書き込む
    (power-data-registration-lambda/model/register.go の RegisterNewPowerLog と同等)
    """
    ensure_pg_conn()
    cur = pg_conn.cursor()
    try:
        # session_devices から session_device_id を取得
        cur.execute(
            "SELECT id FROM session_devices WHERE session_id = %s AND device_id = %s",
            (session_id, device_id),
        )
        row = cur.fetchone()
        if row is None:
            print(
                f"[MQTT Bridge] session_device not found: session={session_id}, device={device_id}")
            pg_conn.rollback()
            return False

        session_device_id = row[0]

        # devices から device_type を取得
        cur.execute(
            "SELECT device_type FROM devices WHERE device_id = %s",
            (device_id,),
        )
        row = cur.fetchone()
        if row is None:
            print(f"[MQTT Bridge] device not found: device_id={device_id}")
            pg_conn.rollback()
            return False

        device_type = row[0]

        # power_logs に INSERT
        cur.execute(
            """
            INSERT INTO power_logs (session_device_id, timestamp, power, gps_lat, gps_lon, device_type)
            VALUES (%s, NOW(), %s, %s, %s, %s)
            """,
            (session_device_id, power, gps_lat, gps_lon, device_type),
        )
        pg_conn.commit()
        return True
    except Exception as e:
        pg_conn.rollback()
        raise e
    finally:
        cur.close()


def register_power_to_dynamodb(session_id, device_id, gps_lat, gps_lon, power):
    """
    DynamoDB (tmp_table) に最新の発電データを upsert する
    (power-data-registration-lambda/model/register.go の RegisterNewPowerLogToDynamoDB と同等)
    """
    dynamodb_client.put_item(
        TableName=DYNAMODB_TABLE,
        Item={
            "session_id": {"S": session_id},
            "device_id": {"S": device_id},
            "gps_lat": {"S": str(gps_lat)},
            "gps_lon": {"S": str(gps_lon)},
            "power": {"N": str(power)},
        },
    )


# ---------- MQTT コールバック ----------
def on_connect(client, userdata, flags, rc, properties=None):
    print(f"[MQTT Bridge] Connected to {MQTT_BROKER}:{MQTT_PORT} (rc={rc})")
    client.subscribe(MQTT_TOPIC)
    print(f"[MQTT Bridge] Subscribed to {MQTT_TOPIC}")


def on_message(client, userdata, msg):
    payload_str = msg.payload.decode("utf-8", errors="replace")
    print(f"[MQTT Bridge] Received on {msg.topic}: {payload_str[:200]}")

    try:
        payload = json.loads(payload_str)

        session_id = payload.get("sessionId", "")
        device_id = payload.get("deviceId", "")
        device_type = payload.get("deviceType", "")
        power = float(payload.get("power", 0))
        gps_lat = str(payload.get("gpsLat", ""))
        gps_lon = str(payload.get("gpsLon", ""))

        if not session_id or not device_id or not device_type:
            print("[MQTT Bridge] Missing required fields, skipping")
            return

        # 1. PostgreSQL に書き込み
        pg_ok = register_power_to_postgres(
            session_id, device_id, gps_lat, gps_lon, power)
        if pg_ok:
            print(
                f"[MQTT Bridge] PostgreSQL OK (session={session_id}, device={device_id}, power={power})")
        else:
            print(f"[MQTT Bridge] PostgreSQL skipped (device not registered)")

        # 2. DynamoDB に書き込み (常に最新値を上書き)
        register_power_to_dynamodb(
            session_id, device_id, gps_lat, gps_lon, power)
        print(
            f"[MQTT Bridge] DynamoDB OK (session={session_id}, device={device_id}, power={power})")

    except Exception as e:
        print(f"[MQTT Bridge] Error: {e}")
        traceback.print_exc()


# ---------- メイン ----------
def main():
    print(
        f"[MQTT Bridge] Starting bridge: {MQTT_BROKER}:{MQTT_PORT} -> PostgreSQL + DynamoDB (direct write)")
    ensure_pg_conn()

    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(MQTT_BROKER, MQTT_PORT, 60)
    client.loop_forever()


if __name__ == "__main__":
    main()
