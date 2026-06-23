import requests
import json

# 测试获取验证码接口
url = "http://111.75.253.33:8899/Api/FruitSorting/CrateProjectPassword"
headers = {
    "Content-Type": "application/json"
}
data = {
    "FPhone": "19874027381"
}

try:
    print(f"正在请求: {url}")
    print(f"请求体: {json.dumps(data, ensure_ascii=False)}")
    print("-" * 50)

    response = requests.post(url, headers=headers, json=data, timeout=10)

    print(f"状态码: {response.status_code}")
    print(f"响应头: {dict(response.headers)}")
    print("-" * 50)
    print(f"响应内容:")

    # 尝试解析 JSON
    try:
        result = response.json()
        print(json.dumps(result, indent=2, ensure_ascii=False))

        # 检查返回码
        if result.get("returnCode") == 1:
            print("\n✓ 验证码发送成功！请查收手机短信")
        else:
            print(f"\n✗ 发送失败: {result.get('returnMessage', '未知错误')}")
    except json.JSONDecodeError:
        print(response.text)

except requests.exceptions.Timeout:
    print("✗ 请求超时，服务器未响应")
except requests.exceptions.ConnectionError:
    print("✗ 连接失败，无法访问服务器")
except Exception as e:
    print(f"✗ 发生错误: {str(e)}")
