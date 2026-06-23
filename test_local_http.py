import requests
import json

# 测试几个可能的地址
test_cases = [
    {
        "name": "本地 Go 后端 (REST API)",
        "url": "http://127.0.0.1:18080/Api/FruitInfo/GetFruitInfo",
        "method": "POST",
        "data": {}
    },
    {
        "name": "本地 Go 后端 (根路径)",
        "url": "http://127.0.0.1:18080/",
        "method": "GET",
        "data": None
    },
    {
        "name": "外网验证码接口",
        "url": "http://111.75.253.33:8899/Api/FruitSorting/CrateProjectPassword",
        "method": "POST",
        "data": {"FPhone": "19874027381"}
    },
    {
        "name": "局域网验证码接口",
        "url": "http://192.168.10.29:8899/Api/FruitSorting/CrateProjectPassword",
        "method": "POST",
        "data": {"FPhone": "19874027381"}
    }
]

def test_http(test_case):
    print(f"\n{'='*60}")
    print(f"测试: {test_case['name']}")
    print(f"URL: {test_case['url']}")
    print(f"方法: {test_case['method']}")

    try:
        if test_case['method'] == 'GET':
            response = requests.get(test_case['url'], timeout=3)
        else:
            response = requests.post(
                test_case['url'],
                headers={"Content-Type": "application/json"},
                json=test_case['data'],
                timeout=3
            )

        print(f"✓ 状态码: {response.status_code}")
        print(f"响应头: {dict(response.headers)}")

        # 尝试解析 JSON
        try:
            result = response.json()
            print(f"响应 JSON:")
            print(json.dumps(result, indent=2, ensure_ascii=False))
        except:
            content = response.text[:200]
            print(f"响应内容 (前200字符):")
            print(content)

        return True

    except requests.exceptions.Timeout:
        print("✗ 超时 (3秒)")
        return False
    except requests.exceptions.ConnectionError as e:
        print(f"✗ 连接失败: {str(e)[:100]}")
        return False
    except Exception as e:
        print(f"✗ 错误: {str(e)[:100]}")
        return False

if __name__ == "__main__":
    print("开始测试 HTTP 连接...")

    success_count = 0
    for test_case in test_cases:
        if test_http(test_case):
            success_count += 1

    print(f"\n{'='*60}")
    print(f"测试完成: {success_count}/{len(test_cases)} 个接口可用")
