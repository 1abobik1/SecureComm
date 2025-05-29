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

# Счётчик этапов подмены (0 - нет подмен, 1 - rsa_pub_client, 2 - ecdsa_pub_client, 3 - nonce1, 4 - signature1,
# 5 - encrypted_message, 6 - client_signature)
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

# Генерация фальшивого зашифрованного сообщения
def generate_fake_encrypted_message():
    return base64.b64encode(b"fake_encrypted_message").decode('utf-8')

# Класс для обработки запросов
class MitMProxy(http.server.SimpleHTTPRequestHandler):
    global step

    def do_POST(self):
        global step
        # Читаем запрос от клиента
        content_length = int(self.headers.get('Content-Length', 0))
        request_data = self.rfile.read(content_length).decode('utf-8')
        request_json = json.loads(request_data)

        print(f"\tОжидание запроса на этапе {step}...")

        # Определяем, какой запрос обрабатываем
        if self.path == "/handshake/init":
            print(f"\tПерехват запроса /handshake/init (этап {step})")

            # Поэтапная подмена
            if step == 1:
                print("\t\tТестирование подмены rsa_pub_client")
                original_rsa_pub = request_json.get('rsa_pub_client', 'отсутствует')
                fake_rsa_pub = generate_fake_rsa_key()
                print(f"\t\tПодмена rsa_pub_client: {original_rsa_pub[:20]}... -> {fake_rsa_pub[:20]}...")
                request_json['rsa_pub_client'] = fake_rsa_pub
            elif step == 2:
                print("\t\tТестирование подмены ecdsa_pub_client")
                original_ecdsa_pub = request_json.get('ecdsa_pub_client', 'отсутствует')
                fake_ecdsa_pub = generate_fake_ecdsa_key()
                print(f"\t\tПодмена ecdsa_pub_client: {original_ecdsa_pub[:20]}... -> {fake_ecdsa_pub[:20]}...")
                request_json['ecdsa_pub_client'] = fake_ecdsa_pub
            elif step == 3:
                print("\t\tТестирование подмены nonce1")
                original_nonce1 = request_json.get('nonce1', 'отсутствует')
                fake_nonce1 = generate_fake_nonce()
                print(f"\t\tПодмена nonce1: {original_nonce1[:20]}... -> {fake_nonce1[:20]}...")
                request_json['nonce1'] = fake_nonce1
            elif step == 4:
                print("\t\tТестирование подмены signature1")
                original_signature1 = request_json.get('signature1', 'отсутствует')
                fake_signature1 = generate_fake_signature()
                print(f"\t\tПодмена signature1: {original_signature1[:20]}... -> {fake_signature1[:20]}...")
                request_json['signature1'] = fake_signature1
            elif step not in [0, 1, 2, 3, 4]:
                print(f"\t\tПредупреждение: запрос /handshake/init на этапе {step} не ожидался")

            # Пересылаем модифицированный запрос серверу
            try:
                response = requests.post(f"{SERVER_URL}/handshake/init", json=request_json, headers=self.headers)
                if not response.ok:
                    error_msg = f"Ошибка сервера: {response.status_code} {response.reason}"
                    print(f"\t\t{error_msg}")
                    print(f"\t\tТело ответа: {response.text}")
                    self.send_error(response.status_code, explain=error_msg)
                    step += 1
                    if step > 6:
                        step = 0
                    print(f"\t\tТекущий этап: {step}")
                    print("\t----")
                    return
            except requests.RequestException as e:
                error_msg = f"Исключение при запросе: {str(e)}"
                print(f"\t\t{error_msg}")
                print(f"\t\tТело запроса: {request_data}")
                self.send_error(502, explain=error_msg)
                step += 1
                if step > 6:
                    step = 0
                print(f"\t\tТекущий этап: {step}")
                print("\t----")
                return

            # Получаем ответ сервера
            response_json = response.json()
            print("\t\tУспешный ответ сервера на /handshake/init")
            step += 1  # Переходим к следующему этапу при успехе
            if step > 6:
                step = 0  # Сбрасываем после всех этапов
            print(f"\t\tТекущий этап: {step}")
            print("\t----")

        elif self.path == "/handshake/finalize":
            print(f"\tПерехват запроса /handshake/finalize (этап {step})")
            # Подмена encrypted
            original_encrypted = request_json.get('encrypted', 'отсутствует')
            fake_encrypted = base64.b64encode(b"fake_data").decode('utf-8')
            print(f"\t\tПодмена encrypted: {original_encrypted[:20]}... -> {fake_encrypted[:20]}...")
            request_json['encrypted'] = fake_encrypted

            # Пересылаем модифицированный запрос серверу
            headers = {"X-Client-ID": self.headers.get("X-Client-ID", "")}
            try:
                response = requests.post(f"{SERVER_URL}/handshake/finalize", json=request_json, headers=headers, timeout=5)
                if not response.ok:
                    error_msg = f"Ошибка сервера: {response.status_code} {response.reason}"
                    print(f"\t\t{error_msg}")
                    print(f"\t\tТело ответа: {response.text}")
                    self.send_error(response.status_code, explain=error_msg)
                    step += 1
                    if step > 6:
                        step = 0
                    print(f"\t\tТекущий этап: {step}")
                    print("\t----")
                    return
            except requests.RequestException as e:
                error_msg = f"Исключение при запросе: {str(e)}"
                print(f"\t\t{error_msg}")
                print(f"\t\tТело запроса: {request_data}")
                self.send_error(502, explain=error_msg)
                step += 1  # Переходим к следующему этапу при ошибке
                if step > 6:
                    step = 0  # Сбрасываем после всех этапов
                print(f"\t\tТекущий этап: {step}")
                print("\t----")
                return

            # Получаем ответ сервера
            response_json = response.json()
            print("\t\tОтвет сервера на /handshake/finalize отправлен клиенту")
            step += 1
            if step > 6:
                step = 0
            print(f"\t\tТекущий этап: {step}")
            print("\t----")

        elif self.path == "/session/test":
            print(f"\tПерехват запроса /session/test (этап {step})")

            # Поэтапная подмена
            if step == 5:
                print("\t\tТестирование подмены encrypted_message")
                original_encrypted_message = request_json.get('encrypted_message', 'отсутствует')
                fake_encrypted_message = generate_fake_encrypted_message()
                print(f"\t\tПодмена encrypted_message: {original_encrypted_message[:20]}... -> {fake_encrypted_message[:20]}...")
                request_json['encrypted_message'] = fake_encrypted_message
            elif step == 6:
                print("\t\tТестирование подмены client_signature")
                original_signature = request_json.get('client_signature', 'отсутствует')
                fake_signature = generate_fake_signature()
                print(f"\t\tПодмена client_signature: {original_signature[:20]}... -> {fake_signature[:20]}...")
                request_json['client_signature'] = fake_signature
            else:
                print(f"\t\tПредупреждение: запрос /session/test на этапе {step} не ожидался")

            # Пересылаем модифицированный запрос серверу
            headers = {
                "X-Client-ID": self.headers.get("X-Client-ID", ""),
                "Content-Type": self.headers.get("Content-Type", "application/json")
            }
            try:
                response = requests.post(f"{SERVER_URL}/session/test", json=request_json, headers=headers, timeout=30)
                if not response.ok:
                    error_msg = f"Ошибка сервера: {response.status_code} {response.reason}"
                    print(f"\t\t{error_msg}")
                    print(f"\t\tТело ответа: {response.text}")
                    self.send_error(response.status_code, explain=error_msg)
                    step += 1
                    if step > 6:
                        step = 0
                    print(f"\t\tТекущий этап: {step}")
                    print("\t----")
                    return
            except requests.RequestException as e:
                error_msg = f"Исключение при запросе: {str(e)}"
                print(f"\t\t{error_msg}")
                print(f"\t\tТело запроса: {request_data}")
                self.send_error(502, explain=error_msg)
                step += 1
                if step > 6:
                    step = 0
                print(f"\t\tТекущий этап: {step}")
                print("\t----")
                return

            # Получаем ответ сервера
            response_json = response.json()
            print("\t\tУспешный ответ сервера на /session/test")
            step += 1  # Переходим к следующему этапу при успехе
            if step > 6:
                step = 0  # Сбрасываем после всех этапов
            print(f"\t\tТекущий этап: {step}")
            print("\t----")

        else:
            print(f"\tНеизвестный путь: {self.path}")
            self.send_response(404)
            self.end_headers()
            print(f"\t\tТекущий этап: {step}")
            print("\t----")
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
    print(f"\tПрокси-сервер запущен на порту {PROXY_PORT}...")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\tПрокси-сервер остановлен пользователем")
        httpd.shutdown()

if __name__ == "__main__":
    run_proxy()