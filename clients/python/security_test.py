import requests
import time
import random
import string
import base64
import os
from statistics import mean, stdev
import asyncio
import aiohttp
from cryptography.hazmat.primitives.asymmetric import rsa, ec
from cryptography.hazmat.primitives import serialization, hashes, padding
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.hmac import HMAC
from cryptography.hazmat.primitives.padding import PKCS7

from client_http import perform_handshake, perform_finalize, derive_keys, sign_ecdsa

class SecurityTester:
    def __init__(self, auth_host="http://localhost:8081", crypto_host="http://localhost:8080"):
        self.auth_host = auth_host
        self.crypto_host = crypto_host
        self.email = "35ddвd2вв3@gmail.com"
        self.password = "TestPass123"
        self.access_token = None
        self.session_data = None
        self.register_user()

    def generate_random_string(self, length):
        return ''.join(random.choices(string.ascii_letters + string.digits, k=length))

    def register_user(self):
        payload = {
            "email": self.email,
            "password": self.password,
            "platform": "web"
        }
        try:
            response = requests.post(f"{self.auth_host}/user/signup", json=payload)
            if response.status_code == 200:
                self.access_token = response.json().get("access_token")
                cookies = response.cookies.get_dict()
                self.refresh_token = cookies.get("refresh_token")
                print("Пользователь успешно зарегистрирован.")
            elif response.status_code == 409:
                print(f"Пользователь с email {self.email} уже существует. Пробую логин...")
                self.login_user()
            else:
                print(f"Ошибка регистрации: {response.status_code} - {response.text}")
                self.login_user()
        except Exception as e:
            print(f"Ошибка при регистрации: {e}")
            self.login_user()

    def login_user(self):
        payload = {
            "email": self.email,
            "password": self.password,
            "platform": "web"
        }
        try:
            response = requests.post(f"{self.auth_host}/user/login", json=payload)
            if response.status_code == 200:
                self.access_token = response.json().get("access_token")
                print("Успешный логин.")
            else:
                print(f"Ошибка логина: {response.status_code} - {response.text}")
        except Exception as e:
            print(f"Ошибка при логине: {e}")

    def brute_force_login(self, max_attempts=7):
        print("\n=== Тестирование brute-force на /user/login ===")
        attempt_count = 0
        incorrect_passwords = [self.generate_random_string(8) for _ in range(max_attempts)]

        for password in incorrect_passwords:
            payload = {
                "email": self.email,
                "password": password,
                "platform": "web"
            }
            start_time = time.time()
            try:
                response = requests.post(f"{self.auth_host}/user/login", json=payload)
                duration = (time.time() - start_time) * 1000
                attempt_count += 1

                if response.status_code == 403:
                    print(f"Попытка {attempt_count}/{max_attempts}: Неверный пароль, время: {duration:.2f} мс")
                elif response.status_code == 429:
                    print(f"Попытка {attempt_count}/{max_attempts}: 429 Too Many Requests — сервер ограничил доступ")
                else:
                    print(f"Попытка {attempt_count}/{max_attempts}: Неожиданный статус {response.status_code}")
            except Exception as e:
                print(f"Ошибка при попытке {attempt_count}: {e}")

        print(f"Brute-force тест завершен. Выполнено {attempt_count} попыток.")

    def replay_attack_test(self, attempts=5):
        """Проверяет защиту от replay атак на /session/test с детальным анализом."""
        print("\n=== Тестирование replay attack на /session/test ===")
        if not self.access_token:
            print("Токен недоступен. Пропуск теста.")
            return

        try:
            handshake_data = perform_handshake(f"{self.crypto_host}/handshake/init", self.access_token)
            ks = perform_finalize(f"{self.crypto_host}/handshake/finalize", handshake_data, self.access_token)
            k_enc, k_mac = derive_keys(ks)
            self.session_data = {
                "client_id": handshake_data["client_id"],
                "k_enc": k_enc,
                "k_mac": k_mac,
                "ecdsa_priv": handshake_data["ecdsa_priv"]
            }

            plaintext = "Test replay attack"
            ts = int(time.time() * 1000)
            timestamp = ts.to_bytes(8, byteorder='big')
            nonce = os.urandom(16)
            plaintext_bytes = plaintext.encode('utf-8')
            blob = timestamp + nonce + plaintext_bytes
            padder = PKCS7(128).padder()
            padded = padder.update(blob) + padder.finalize()
            iv = os.urandom(16)
            cipher = Cipher(algorithms.AES(self.session_data["k_enc"]), modes.CBC(iv))
            encryptor = cipher.encryptor()
            ciphertext = encryptor.update(padded) + encryptor.finalize()
            h = HMAC(self.session_data["k_mac"], hashes.SHA256())
            h.update(iv)
            h.update(ciphertext)
            tag = h.finalize()
            pkg = iv + ciphertext + tag
            b64msg = base64.b64encode(pkg).decode('utf-8')
            signature = sign_ecdsa(self.session_data["ecdsa_priv"], pkg)
            payload = {"encrypted_message": b64msg, "client_signature": signature}
            headers = {
                "X-Client-ID": self.session_data["client_id"],
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.access_token}"
            }

            print("\nТест 1: Отправка исходного запроса")
            start_time = time.time()
            response = requests.post(f"{self.crypto_host}/session/test", json=payload, headers=headers)
            duration = (time.time() - start_time) * 1000
            if response.status_code == 200:
                print(f"Первый запрос успешен, время: {duration:.2f} мс")
                print(f"Ответ сервера: {response.json()}")
                print(f"Nonce (base64): {base64.b64encode(nonce).decode('utf-8')}")
            else:
                print(f"Ошибка первого запроса: {response.status_code} - {response.text}")
                return
            time.sleep(5)

            print("\nТест 2: Повтор исходного запроса (ожидается 409 Conflict)")
            for i in range(attempts):
                start_time = time.time()
                try:
                    response = requests.post(f"{self.crypto_host}/session/test", json=payload, headers=headers)
                    duration = (time.time() - start_time) * 1000
                    if response.status_code == 409:
                        print(
                            f"Попытка повтора {i + 1}: Сервер отклонил запрос (409 Conflict), защита от повторов работает, время: {duration:.2f} мс")
                        print(f"Ответ сервера: {response.text}")
                    else:
                        print(
                            f"Попытка повтора {i + 1}: Неожиданный статус {response.status_code}, время: {duration:.2f} мс")
                        print(f"Ответ сервера: {response.text}")
                except Exception as e:
                    print(f"Ошибка при попытке повтора {i + 1}: {e}")
                time.sleep(5)

            print("\nТест 3: Новый запрос с тем же nonce, но новым timestamp")
            new_ts = int(time.time() * 1000)
            new_timestamp = new_ts.to_bytes(8, byteorder='big')
            if not self.session_data:
                print("Ошибка: session_data не инициализировано.")
                return
            blob = new_timestamp + nonce + plaintext_bytes
            padder = PKCS7(128).padder()
            padded = padder.update(blob) + padder.finalize()
            iv = os.urandom(16)
            cipher = Cipher(algorithms.AES(self.session_data["k_enc"]), modes.CBC(iv))
            encryptor = cipher.encryptor()
            ciphertext = encryptor.update(padded) + encryptor.finalize()
            h = HMAC(self.session_data["k_mac"], hashes.SHA256())
            h.update(iv)
            h.update(ciphertext)
            tag = h.finalize()
            pkg = iv + ciphertext + tag
            new_b64msg = base64.b64encode(pkg).decode('utf-8')
            new_signature = sign_ecdsa(self.session_data["ecdsa_priv"], pkg)
            new_payload = {"encrypted_message": new_b64msg, "client_signature": new_signature}
            start_time = time.time()
            try:
                response = requests.post(f"{self.crypto_host}/session/test", json=new_payload, headers=headers)
                duration = (time.time() - start_time) * 1000
                if response.status_code == 409:
                    print(f"Запрос с тем же nonce отклонен (409 Conflict), защита работает, время: {duration:.2f} мс")
                    print(f"Ответ сервера: {response.text}")
                else:
                    print(
                        f"Неожиданный статус {response.status_code} для запроса с тем же nonce, время: {duration:.2f} мс")
                    print(f"Ответ сервера: {response.text}")
            except Exception as e:
                print(f"Ошибка при запросе с тем же nonce: {e}")

            print("\nТест 4: Новый запрос с новым nonce (ожидается успех)")
            new_nonce = os.urandom(16)
            blob = new_timestamp + new_nonce + plaintext_bytes
            padder = PKCS7(128).padder()
            padded = padder.update(blob) + padder.finalize()
            iv = os.urandom(16)
            cipher = Cipher(algorithms.AES(self.session_data["k_enc"]), modes.CBC(iv))
            encryptor = cipher.encryptor()
            ciphertext = encryptor.update(padded) + encryptor.finalize()
            h = HMAC(self.session_data["k_mac"], hashes.SHA256())
            h.update(iv)
            h.update(ciphertext)
            tag = h.finalize()
            pkg = iv + ciphertext + tag
            new_b64msg = base64.b64encode(pkg).decode('utf-8')
            new_signature = sign_ecdsa(self.session_data["ecdsa_priv"], pkg)
            new_payload = {"encrypted_message": new_b64msg, "client_signature": new_signature}
            start_time = time.time()
            try:
                response = requests.post(f"{self.crypto_host}/session/test", json=new_payload, headers=headers)
                duration = (time.time() - start_time) * 1000
                if response.status_code == 200:
                    print(f"Запрос с новым nonce успешен, время: {duration:.2f} мс")
                    print(f"Ответ сервера: {response.json()}")
                else:
                    print(f"Неожиданный статус {response.status_code} для нового nonce, время: {duration:.2f} мс")
                    print(f"Ответ сервера: {response.text}")
            except Exception as e:
                print(f"Ошибка при запросе с новым nonce: {e}")

        except Exception as e:
            print(f"Ошибка при тестировании replay attack: {e}")

    def timing_attack_login(self, attempts=20):
        print("\n=== Тестирование timing attack на /user/login ===")
        correct_times = []
        incorrect_times = []

        payload_correct = {"email": self.email, "password": self.password, "platform": "web"}

        for i in range(attempts):
            start_time = time.time()
            try:
                response = requests.post(f"{self.auth_host}/user/login", json=payload_correct)
                duration = (time.time() - start_time) * 1000
                if response.status_code == 200:
                    correct_times.append(duration)
                    print(f"Попытка {i+1} (правильный пароль): Успех, время: {duration:.2f} мс")
                elif response.status_code == 429:
                    print(f"Попытка {i+1} (правильный пароль): 429 — превышен лимит. Ждём 10 секунд.")
                    time.sleep(10)
                else:
                    print(f"Попытка {i+1} (правильный пароль): Ошибка: {response.status_code} - {response.text}")
            except Exception as e:
                print(f"Ошибка при попытке {i+1} (правильный пароль): {e}")
            time.sleep(1)

        for i in range(attempts):
            payload_incorrect = {"email": self.email, "password": self.generate_random_string(8), "platform": "web"}
            start_time = time.time()
            try:
                response = requests.post(f"{self.auth_host}/user/login", json=payload_incorrect)
                duration = (time.time() - start_time) * 1000
                if response.status_code == 403:
                    incorrect_times.append(duration)
                    print(f"Попытка {i+1} (неправильный пароль): Отклонено, время: {duration:.2f} мс")
                elif response.status_code == 429:
                    print(f"Попытка {i+1} (неправильный пароль): 429 — превышен лимит. Ждём 10 секунд.")
                    time.sleep(10)
                else:
                    print(f"Попытка {i+1} (неправильный пароль): Ошибка: {response.status_code} - {response.text}")
            except Exception as e:
                print(f"Ошибка при попытке {i+1} (неправильный пароль): {e}")
            time.sleep(1)

        if correct_times and incorrect_times:
            correct_mean = mean(correct_times)
            incorrect_mean = mean(incorrect_times)
            print(f"Среднее время правильного входа: {correct_mean:.2f} мс")
            print(f"Среднее время неправильного входа: {incorrect_mean:.2f} мс")
            print(f"Разница: {abs(correct_mean - incorrect_mean):.2f} мс")
        else:
            print("Недостаточно данных для анализа timing attack.")

    def run_all_tests(self):
        self.brute_force_login()
        self.timing_attack_login()
        self.replay_attack_test()

def main():
    tester = SecurityTester()
    if tester.access_token:
        init_url = f"{tester.crypto_host}/handshake/init"
        try:
            handshake_data = perform_handshake(init_url, tester.access_token)
            print(f"Handshake успешно завершен, идентификатор клиента: {handshake_data['client_id']}")
            tester.run_all_tests()
        except Exception as e:
            print(f"Ошибка при выполнении handshake: {e}")
    else:
        print("Не удалось получить access_token. Тесты не выполнены.")

if __name__ == "__main__":
    main()