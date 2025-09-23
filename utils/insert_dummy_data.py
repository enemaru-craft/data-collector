import time
import random
import math
import json
import ssl
import requests
import os
from paho.mqtt import client as mqtt


START_TIME = time.time()
# ======================
# 設定
# ======================
# RESTエンドポイント（デバイス登録用）
REGISTER_URL = os.getenv("REGISTER_URL")

# MQTT設定
MQTT_ENDPOINT = os.getenv("MQTT_ENDPOINT")
MQTT_PORT = 8883

CERT_PATH = "../stg-infrastructure/modules/iot/stg_iot_cert.pem"
PRIVATE_KEY_PATH = "../stg-infrastructure/modules/iot/stg_iot_private.key"
CA_PATH = "../stg-infrastructure/AmazonRootCA1.pem"

TOPIC = "register/power"

SESSION_ID = "28"

# 愛知県豊田市付近の緯度経度範囲
LAT_RANGE = (35.0560, 35.1250)
LON_RANGE = (137.0900, 137.1700)


# ======================
# デバイスリスト
# ======================
devices = [
    # Solar
    {"deviceId": f"M5-{SESSION_ID}-solar-1", "deviceType": "solar"},
    {"deviceId": f"M5-{SESSION_ID}-solar-2", "deviceType": "solar"},

    # Geothermal
    {"deviceId": f"M5-{SESSION_ID}-geothermal-1", "deviceType": "geothermal"},
    {"deviceId": f"M5-{SESSION_ID}-geothermal-2", "deviceType": "geothermal"},

    # Hydrogen
    {"deviceId": f"M5-{SESSION_ID}-hydrogen-1", "deviceType": "hydrogen"},
    {"deviceId": f"M5-{SESSION_ID}-hydrogen-2", "deviceType": "hydrogen"},

    # Wind
    {"deviceId": f"M5-{SESSION_ID}-wind-1", "deviceType": "wind"},
    {"deviceId": f"M5-{SESSION_ID}-wind-2", "deviceType": "wind"},
]


# ======================
# デバイス登録関数
# ======================
def register_devices():
    print("Registering devices...")
    for device in devices:
        payload = {
            "sessionId": SESSION_ID,
            "deviceId": device["deviceId"],
            "deviceType": device["deviceType"]
        }
        try:
            response = requests.post(REGISTER_URL, json=payload)
            print(
                f"[REGISTER] {device['deviceId']} -> Status: {response.status_code}, Response: {response.text}")
        except requests.exceptions.RequestException as e:
            print(f"[ERROR] Failed to register {device['deviceId']}: {e}")
    print("All devices registered!\n")


# ======================
# MQTTクライアント設定
# ======================
def connect_mqtt():
    client = mqtt.Client()

    # TLS認証設定
    client.tls_set(
        ca_certs=CA_PATH,
        certfile=CERT_PATH,
        keyfile=PRIVATE_KEY_PATH,
        tls_version=ssl.PROTOCOL_TLSv1_2
    )

    # 接続
    print("Connecting to MQTT broker...")
    client.connect(MQTT_ENDPOINT, MQTT_PORT, keepalive=60)
    client.loop_start()
    print("MQTT Connected!\n")

    return client


# ======================
# 補助関数
# ======================
def random_location():
    """豊田市付近のランダム座標生成"""
    lat = random.uniform(*LAT_RANGE)
    lon = random.uniform(*LON_RANGE)
    return round(lat, 6), round(lon, 6)


def generate_power(device_type):
    """デバイス種別ごとに時間変動 + ノイズつきの発電量生成"""
    elapsed = time.time() - START_TIME  # 起動からの経過秒数

    # 時間変動: サイン波で上下に揺らす
    # 周期は約5分(300秒)
    base_wave = math.sin((elapsed / 300.0) * math.pi * 2)

    # 各発電種別ごとの基準レンジ
    if device_type == "solar":
        base = 150 + 100 * base_wave  # 中心150、±100変動
        noise = random.uniform(-20, 20)
    elif device_type == "geothermal":
        base = 300 + 150 * base_wave
        noise = random.uniform(-30, 30)
    elif device_type == "hydrogen":
        base = 80 + 40 * base_wave
        noise = random.uniform(-10, 10)
    elif device_type == "wind":
        base = 200 + 100 * base_wave
        noise = random.uniform(-25, 25)
    else:
        base = 50 + 20 * base_wave
        noise = random.uniform(-5, 5)

    # 最終値 = 基本値 + ノイズ
    power = base + noise

    # 0以下にはならないようクリップ
    return max(0, round(power, 2))


def publish_power(client, device):
    """発電量をMQTTで送信"""
    lat, lon = random_location()
    payload = {
        "sessionId": SESSION_ID,
        "deviceId": device["deviceId"],
        "deviceType": device["deviceType"],
        "power": generate_power(device["deviceType"]),
        "gpsLat": str(lat),
        "gpsLon": str(lon)
    }
    client.publish(TOPIC, json.dumps(payload), qos=1)
    print(f"[PUBLISH] {device['deviceId']}: {payload}")


# ======================
# メイン処理
# ======================
def main():
    # 1. デバイス登録
    register_devices()

    # 2. MQTT接続
    mqtt_client = connect_mqtt()

    # 3. データ送信ループ
    print("Start publishing power data...\n")
    while True:
        current_time = time.time()

        for device in devices:
            if device["deviceType"] == "wind":
                # 風力は30秒ごと
                if int(current_time) % 30 == 0:
                    publish_power(mqtt_client, device)
            else:
                # それ以外は3秒ごと
                if int(current_time) % 3 == 0:
                    publish_power(mqtt_client, device)

        time.sleep(1)  # ループ間隔1秒


if __name__ == "__main__":
    main()
