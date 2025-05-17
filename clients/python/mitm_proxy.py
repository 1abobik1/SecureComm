import json
import base64
import http.server
import socketserver
import requests
from cryptography.hazmat.primitives.asymmetric import rsa, ec
from cryptography.hazmat.primitives import serialization

# Порт, на котором прокси будет принимать запросы от клиента
PROXY_PORT = 8081
# Адрес реального сервера
SERVER_URL = "http://localhost:8080"

# Счётчик этапов подмены (0 - нет подмен, 1 - rsa_pub_client, 2 - ecdsa_pub_client, 3 - nonce1, 4 - signature1)
step = 0

# Генерация фальшивого RSA-ключа
def generate_fake_rsa_key():
    rsa_private_key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
    rsa_public_key = rsa_private_key.public_key()
    der = rsa_public_key.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    return base64.b64encode(der).decode('utf-8')

# Генерация фальшивого ECDSA-ключа
def generate_fake_ecdsa_key():
    ecdsa_private_key = ec.generate_private_key(curve=ec.SECP256R1())
    ecdsa_public_key = ecdsa_private_key.public_key()
    der = ecdsa_public_key.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    return base64.b64encode(der).decode('utf-8')

# Генерация фальшивого nonce
def generate_fake_nonce():
    return base64.b64encode(b"fake_nonce").decode('utf-8')

# Генерация фальшивой подписи
def generate_fake_signature():
    return base64.b64encode(b"fake_signature").decode('utf-8')

# Класс для обработки запросов
class MitMProxy(http.server.SimpleHTTPRequestHandler):
    global step

    def do_POST(self):
        global step
        # Читаем запрос от клиента
        content_length = int(self.headers.get('Content-Length', 0))
        request_data = self.rfile.read(content_length).decode('utf-8')
        request_json = json.loads(request_data)

        # Определяем, какой запрос обрабатываем
        if self.path == "/handshake/init":
            print(f"🌐 Перехват запроса /handshake/init (этап {step})")

            # Поэтапная подмена
            if step == 1:
                print("🛡️ Тестирование подмены rsa_pub_client")
                original_rsa_pub = request_json['rsa_pub_client']
                fake_rsa_pub = generate_fake_rsa_key()
                print(f"🔄 Подмена rsa_pub_client: {original_rsa_pub[:20]}... -> {fake_rsa_pub[:20]}...")
                request_json['rsa_pub_client'] = fake_rsa_pub
            elif step == 2:
                print("🛡️ Тестирование подмены ecdsa_pub_client")
                original_ecdsa_pub = request_json['ecdsa_pub_client']
                fake_ecdsa_pub = generate_fake_ecdsa_key()
                print(f"🔄 Подмена ecdsa_pub_client: {original_ecdsa_pub[:20]}... -> {fake_ecdsa_pub[:20]}...")
                request_json['ecdsa_pub_client'] = fake_ecdsa_pub
            elif step == 3:
                print("🛡️ Тестирование подмены nonce1")
                original_nonce1 = request_json['nonce1']
                fake_nonce1 = generate_fake_nonce()
                print(f"🔄 Подмена nonce1: {original_nonce1[:20]}... -> {fake_nonce1[:20]}...")
                request_json['nonce1'] = fake_nonce1
            elif step == 4:
                print("🛡️ Тестирование подмены signature1")
                original_signature1 = request_json['signature1']
                fake_signature1 = generate_fake_signature()
                print(f"🔄 Подмена signature1: {original_signature1[:20]}... -> {fake_signature1[:20]}...")
                request_json['signature1'] = fake_signature1

            # Пересылаем модифицированный запрос серверу
            try:
                # Передаём все заголовки клиента серверу
                response = requests.post(f"{SERVER_URL}/handshake/init", json=request_json, headers=self.headers)
                if not response.ok:
                    error_msg = f"❌ Ошибка сервера: {response.status_code} {response.reason}"
                    print(f"{error_msg}\nТело ответа: {response.text}")
                    self.send_error(response.status_code, explain=error_msg)
                    step += 1  # Переходим к следующему этапу при ошибке
                    if step > 4:
                        step = 0  # Сбрасываем после всех этапов
                    return
            except requests.RequestException as e:
                error_msg = f"❌ Исключение при запросе: {str(e)}"
                print(f"{error_msg}\nТело запроса: {request_data}")
                self.send_error(502, explain=error_msg)
                step += 1  # Переходим к следующему этапу при ошибке
                if step > 4:
                    step = 0  # Сбрасываем после всех этапов
                return

            # Получаем ответ сервера
            response_json = response.json()
            print("✅ Успешный ответ сервера на /handshake/init")
            step += 1  # Переходим к следующему этапу при успехе
            if step > 4:
                step = 0  # Сбрасываем после всех этапов

        elif self.path == "/handshake/finalize":
            print("🌐 Перехват запроса /handshake/finalize")
            # Подмена encrypted
            original_encrypted = request_json['encrypted']
            fake_encrypted = base64.b64encode(b"fake_data").decode('utf-8')
            print(f"🔄 Подмена encrypted: {original_encrypted[:20]}... -> {fake_encrypted[:20]}...")
            request_json['encrypted'] = fake_encrypted

            # Пересылаем модифицированный запрос серверу
            headers = {"X-Client-ID": self.headers.get("X-Client-ID", "")}
            try:
                response = requests.post(f"{SERVER_URL}/handshake/finalize", json=request_json, headers=headers, timeout=5)
                if not response.ok:
                    error_msg = f"❌ Ошибка сервера: {response.status_code} {response.reason}"
                    print(f"{error_msg}\nТело ответа: {response.text}")
                    self.send_error(response.status_code, explain=error_msg)
                    return
            except requests.RequestException as e:
                error_msg = f"❌ Исключение при запросе: {str(e)}"
                print(f"{error_msg}\nТело запроса: {request_data}")
                self.send_error(502, explain=error_msg)
                return

            # Получаем ответ сервера
            response_json = response.json()
            print("✅ Ответ сервера на /handshake/finalize отправлен клиенту")

        else:
            self.send_response(404)
            self.end_headers()
            return

        # Отправляем ответ клиенту
        self.send_response(response.status_code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(response_json).encode('utf-8'))

# Запуск прокси-сервера
def run_proxy():
    server_address = ('', PROXY_PORT)
    httpd = socketserver.TCPServer(server_address, MitMProxy)
    print(f"🌐 Прокси-сервер запущен на порту {PROXY_PORT}...")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("⏹ Прокси-сервер остановлен пользователем")
        httpd.shutdown()

if __name__ == "__main__":
    run_proxy()