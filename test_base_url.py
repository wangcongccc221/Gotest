import requests

# 测试基础连接
urls = [
    "http://111.75.253.33:8899/",
    "http://111.75.253.33:8899/Api/",
    "http://192.168.0.155:29500/",
    "http://192.168.0.155:29500/Api/"
]

for url in urls:
    try:
        print(f"测试: {url}")
        response = requests.get(url, timeout=5)
        print(f"  ✓ 状态码: {response.status_code}")
        print(f"  响应: {response.text[:100]}")
    except requests.exceptions.Timeout:
        print(f"  ✗ 超时")
    except requests.exceptions.ConnectionError:
        print(f"  ✗ 连接失败")
    except Exception as e:
        print(f"  ✗ 错误: {e}")
    print("-" * 50)
