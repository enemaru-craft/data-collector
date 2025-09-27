import requests

# エンドポイントURL
url = "https://hogehoge.ap-northeast-1.amazonaws.com/register-new-power-generation-module"

# 変数（セッションIDやM5の識別子）
session_id = "23"  # ← ここに実際のセッションIDを入れる

devices = [
    # Solar
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-solar-1",
        "deviceType": "solar"
    },
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-solar-2",
        "deviceType": "solar"
    },

    # Geothermal
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-geothermal-1",
        "deviceType": "geothermal"
    },
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-geothermal-2",
        "deviceType": "geothermal"
    },

    # Hydrogen
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-hydrogen-1",
        "deviceType": "hydrogen"
    },
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-hydrogen-2",
        "deviceType": "hydrogen"
    },

    # Wind
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-wind-1",
        "deviceType": "wind"
    },
    {
        "sessionId": session_id,
        "deviceId": f"M5-{session_id}-wind-2",
        "deviceType": "wind"
    }
]

# 各デバイスをPOST
for device in devices:
    try:
        response = requests.post(url, json=device)
        # ステータスコードとレスポンスを確認
        print(f"POST {device['deviceId']} -> Status: {response.status_code}")
        print("Response:", response.text)
    except requests.exceptions.RequestException as e:
        print(f"Error posting {device['deviceId']}: {e}")
