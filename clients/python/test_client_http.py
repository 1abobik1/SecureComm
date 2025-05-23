import unittest
import base64
import requests
import time
import os
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import padding, ec
from client_http import perform_handshake, perform_finalize, derive_keys, generate_nonce, sign_ecdsa, verify_ecdsa, perform_session_test

class TestHandshakeSecurity(unittest.TestCase):
    def setUp(self):
        self.init_url = "http://localhost:8080/handshake/init"
        self.fin_url = "http://localhost:8080/handshake/finalize"
        self.test_url = "http://localhost:8080/session/test"
        # Задержка перед каждым тестом, чтобы избежать 429
        time.sleep(1)

    def test_replay_attack_init(self):
        # Тест на replay-атаку: повторная отправка того же nonce1
        handshake_data = perform_handshake(self.init_url)
        nonce1_b64 = handshake_data["nonce1_b64"]
        with self.assertRaises(requests.RequestException):
            perform_handshake(self.init_url, nonce1=nonce1_b64)  # Повторяем тот же nonce1

    def test_replay_attack_finalize(self):
        # Тест на replay-атаку: повторная отправка того же nonce3
        handshake_data = perform_handshake(self.init_url)
        _, nonce3 = generate_nonce(8)
        ks = perform_finalize(self.fin_url, handshake_data, nonce3=nonce3)
        try:
            perform_finalize(self.fin_url, handshake_data, nonce3=nonce3)  # Повторяем тот же nonce3
            self.fail("Ожидалась ошибка при повторной отправке nonce3")
        except requests.HTTPError as e:
            # Ожидаем HTTP 409 Conflict как подтверждение защиты от replay-атаки
            self.assertEqual(e.response.status_code, 409, "Ожидался код 409 для повторного nonce3")

    def test_timing_attack_signature(self):
        # Тест на timing-атаку: сравнение времени проверки валидной и невалидной подписи
        handshake_data = perform_handshake(self.init_url)
        # Создаём валидную подпись как байты
        valid_signature_bytes = handshake_data["ecdsa_priv"].sign(
            handshake_data["nonce1"], ec.ECDSA(hashes.SHA256())
        )
        # Создаём невалидную подпись, изменяя последний байт
        invalid_signature_bytes = valid_signature_bytes[:-1] + (b'\x00' if valid_signature_bytes[-1] != 0 else b'\x01')
        # Кодируем обе подписи в base64
        valid_signature = base64.b64encode(valid_signature_bytes).decode('utf-8')
        invalid_signature = base64.b64encode(invalid_signature_bytes).decode('utf-8')

        # Усредняем время для нескольких запусков, чтобы уменьшить шум
        num_runs = 5
        valid_times = []
        invalid_times = []
        for _ in range(num_runs):
            start = time.time_ns()
            verify_ecdsa(handshake_data["ecdsa_pub_server"], handshake_data["nonce1"], valid_signature)
            valid_times.append(time.time_ns() - start)

            start = time.time_ns()
            verify_ecdsa(handshake_data["ecdsa_pub_server"], handshake_data["nonce1"], invalid_signature)
            invalid_times.append(time.time_ns() - start)

        # Среднее время
        avg_valid_time = sum(valid_times) / num_runs
        avg_invalid_time = sum(invalid_times) / num_runs

        # Проверяем, что разница во времени минимальна (порог 1 мс)
        self.assertLess(abs(avg_valid_time - avg_invalid_time), 1_000_000,
                        "Время проверки подписи не константное")

    def test_brute_force_client_id(self):
        # Тест на brute-force: попытка отправки запросов с неверным client_id
        handshake_data = perform_handshake(self.init_url)
        invalid_client_id = "invalid_" + handshake_data["client_id"]
        _, ks = generate_nonce(32)
        nonce3_b64, nonce3 = generate_nonce(8)
        payload = ks + nonce3 + handshake_data["nonce2"]
        sig3 = sign_ecdsa(handshake_data["ecdsa_priv"], payload)
        to_encrypt = payload
        encrypted = handshake_data["rsa_pub_server"].encrypt(
            to_encrypt,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None
            )
        )
        encrypted_b64 = base64.b64encode(encrypted).decode('utf-8')
        payload = {"encrypted": encrypted_b64, "signature3": sig3}
        headers = {"X-Client-ID": invalid_client_id}

        with self.assertRaises(requests.RequestException):
            response = requests.post(self.fin_url, json=payload, headers=headers, timeout=5)
            response.raise_for_status()

    def test_functional_success(self):
        # Функциональный тест: успешное выполнение рукопожатия и сессии
        handshake_data = perform_handshake(self.init_url)
        self.assertIsNotNone(handshake_data["client_id"])
        ks = perform_finalize(self.fin_url, handshake_data)
        self.assertIsNotNone(ks)
        K_enc, K_mac = derive_keys(ks)
        self.assertEqual(len(K_enc), 32)
        self.assertEqual(len(K_mac), 32)
        session = {
            "client_id": handshake_data["client_id"],
            "k_enc": K_enc,
            "k_mac": K_mac,
            "ecdsa_priv": handshake_data["ecdsa_priv"]
        }
        perform_session_test(self.test_url, session)
        # Проверяем, что запрос прошёл без ошибок (ошибки вызовут исключение)

    def test_large_data_session(self):
        # Тест на отправку больших данных через /session/test
        handshake_data = perform_handshake(self.init_url)
        ks = perform_finalize(self.fin_url, handshake_data)
        K_enc, K_mac = derive_keys(ks)
        session = {
            "client_id": handshake_data["client_id"],
            "k_enc": K_enc,
            "k_mac": K_mac,
            "ecdsa_priv": handshake_data["ecdsa_priv"]
        }
        # Тест с 5 МБ
        large_data_5mb = os.urandom(5 * 1024 * 1024)
        perform_session_test(self.test_url, session, large_data_5mb)
        # Тест с 10 МБ
        large_data_10mb = os.urandom(10 * 1024 * 1024)
        perform_session_test(self.test_url, session, large_data_10mb)
        # Проверяем, что запросы прошли без ошибок (ошибки вызовут исключение)

if __name__ == "__main__":
    unittest.main()