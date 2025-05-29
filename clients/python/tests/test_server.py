import base64
# python test_server.py --file-size 1000 (размер тестового файла в МБ)
import requests
import os
import random
import string
import tempfile
import time
import argparse
from ..client_http import perform_handshake, perform_finalize, derive_keys, stream_upload_encrypted_file

class ServerTester:
    def __init__(self, auth_host="http://localhost:8081", crypto_host="http://localhost:8080"):
        self.auth_host = auth_host
        self.crypto_host = crypto_host
        self.email = self.generate_random_email()
        self.password = "TestPass123"
        self.access_token = None

    def generate_random_email(self):
        """Генерирует случайный email для регистрации."""
        name = ''.join(random.choices(string.ascii_lowercase + string.digits, k=10))
        return f"{name}@test.com"

    def register_user(self):
        """Регистрирует нового пользователя и возвращает access_token."""
        payload = {
            "email": self.email,
            "password": self.password,
            "platform": "web"
        }
        try:
            response = requests.post(f"{self.auth_host}/user/signup", json=payload)
            if response.status_code == 200:
                self.access_token = response.json().get("access_token")
                print(f"Регистрация успешна: {self.email}, access_token получен")
                return True
            else:
                print(f"Ошибка регистрации: {response.status_code} - {response.text}")
                return False
        except Exception as e:
            print(f"Ошибка при регистрации: {e}")
            return False

    def create_test_file(self, file_size_mb=50):
        """Создаёт временный тестовый файл заданного размера в МБ."""
        try:
            # Валидация размера файла
            if file_size_mb <= 0:
                raise ValueError("Размер файла должен быть больше 0 МБ")
            if file_size_mb > 10000:  # Ограничение в 1 ГБ для разумного тестирования
                raise ValueError("Размер файла не должен превышать 1000 МБ")

            # Используем tempfile для создания временного файла
            with tempfile.NamedTemporaryFile(delete=False, suffix=".txt") as temp_file:
                temp_file_path = temp_file.name

            # Заполняем файл случайными данными заданного размера чанками
            file_size_bytes = int(file_size_mb * 1024 * 1024)  # Преобразуем в int
            chunk_size = 1024 * 1024  # 1 МБ
            with open(temp_file_path, "wb") as f:
                remaining_bytes = file_size_bytes
                while remaining_bytes > 0:
                    chunk = os.urandom(min(chunk_size, remaining_bytes))
                    f.write(chunk)
                    remaining_bytes -= len(chunk)

            print(f"Создан тестовый файл: {temp_file_path} ({file_size_mb} МБ)")
            return temp_file_path
        except Exception as e:
            print(f"Ошибка создания тестового файла: {e}")
            return None

    def test_server(self, file_size_mb=50):
        """Основной тест: регистрация, handshake и загрузка файла заданного размера."""
        print(f"\n=== Запуск теста сервера (размер файла: {file_size_mb} МБ) ===")

        # Step 1: Регистрация пользователя
        if not self.register_user() or not self.access_token:
            print("Тест прерван: не удалось зарегистрировать пользователя")
            return False

        # Step 2: Выполняем handshake
        try:
            init_url = f"{self.crypto_host}/handshake/init"
            fin_url = f"{self.crypto_host}/handshake/finalize"
            handshake_data = perform_handshake(init_url, self.access_token)
            print(f"Handshake инициализирован: client_id={handshake_data['client_id']}")

            ks = perform_finalize(fin_url, handshake_data, self.access_token)
            k_enc, k_mac = derive_keys(ks)
            print(f"Handshake завершён, ключи получены: k_enc={base64.b64encode(k_enc).decode('utf-8')[:10]}..., k_mac={base64.b64encode(k_mac).decode('utf-8')[:10]}...")

        except Exception as e:
            print(f"Ошибка при handshake: {e}")
            return False

        # Step 3: Создаём тестовый файл
        file_path = self.create_test_file(file_size_mb)
        if not file_path:
            print("Тест прерван: не удалось создать тестовый файл")
            return False

        # Step 4: Загружаем зашифрованный файл
        try:
            cloud_url = f"{self.crypto_host}/files/one/encrypted"
            category = "unknown"
            start_time = time.time()
            response = stream_upload_encrypted_file(
                file_path=file_path,
                cloud_url=cloud_url,
                access_token=self.access_token,
                category=category,
                k_enc=k_enc,
                k_mac=k_mac
            )
            duration = (time.time() - start_time) * 1000
            # Рассчитываем скорость загрузки
            speed_mbps = (file_size_mb / (duration / 1000)) if duration > 0 else 0
            print(f"Загрузка файла успешна: {duration:.2f} ms")
            print(f"Скорость загрузки: {speed_mbps:.2f} МБ/с")
            print(f"Ответ сервера: {response}")

            # Валидация ответа
            if "obj_id" in response and "url" in response and "name" in response:
                print("Тест пройден успешно: Сервер вернул корректный JSON с метаданными")
                return True
            else:
                print("Тест не пройден: неверный формат ответа сервера")
                return False

        except Exception as e:
            print(f"Ошибка при загрузке: {e}")
            return False

        finally:
            # Cleanup: удаляем тестовый файл
            try:
                if 'file_path' in locals():
                    os.remove(file_path)
                    print(f"Тестовый файл удалён: {file_path}")
            except Exception as e:
                print(f"Ошибка при удалении файла: {e}")

def main():
    parser = argparse.ArgumentParser(description="Тестирование сервера SecureComm")
    parser.add_argument("--file-size", type=float, default=1, help="Размер тестового файла в МБ (по умолчанию 1)")
    args = parser.parse_args()

    tester = ServerTester()
    success = tester.test_server(args.file_size)
    print(f"\n=== Тест {'пройден' if success else 'не пройден'} ====")

if __name__ == "__main__":
    main()
#ективность. Предыдущий тест с 1 МБ прошел успешно, и с 100 МБ тест должен завершиться аналогично, если сервер и MinIO настроены корректно. Запустите исправленный скрипт и сообщите, если возникнут другие проблемы или потребуется дополнительная функциональность, например, тестирование скачивания файла или других категорий.