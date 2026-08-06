import requests

def main():
    print("Hello from Dockerized Python App!")
    response = requests.get("https://httpbin.org/get")
    print(f"Status Code: {response.status_code}")

if __name__ == "__main__":
    main()