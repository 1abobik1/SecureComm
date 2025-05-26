import base64
import hashlib
import hmac
import os
import requests
import time
from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.primitives.hmac import HMAC
from cryptography.hazmat.primitives.padding import PKCS7

# Генерирует пару ключей RSA и ECDSA
def generate_keys():
    rsa_private_key = rsa.generate_private_key(public_exponent=65537, key_size=3072)
    rsa_public_key = rsa_private_key.public_key()
    ecdsa_private_key = ec.generate_private_key(curve=ec.SECP256R1())
    ecdsa_public_key = ecdsa_private_key.public_key()
    return rsa_private_key, rsa_public_key, ecdsa_private_key, ecdsa_public_key

# Сериализует ключ в формат DER (или PEM для приватных) и кодирует в base64
def serialize_key_to_der(key, is_private=False):
    if is_private:
        pem = key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption()
        )
        return pem
    else:
        der = key.public_bytes(
            encoding=serialization.Encoding.DER,
            format=serialization.PublicFormat.SubjectPublicKeyInfo
        )
        return base64.b64encode(der).decode('utf-8')

# Создает ECDSA подпись для данных в формате base64
def sign_ecdsa(ecdsa_private_key, data):
    signature = ecdsa_private_key.sign(data, ec.ECDSA(hashes.SHA256()))
    return base64.b64encode(signature).decode('utf-8')

# Проверяет ECDSA подпись, возвращает True, если подпись верна
def verify_ecdsa(ecdsa_public_key, data, signature_b64):
    signature = base64.b64decode(signature_b64)
    try:
        ecdsa_public_key.verify(signature, data, ec.ECDSA(hashes.SHA256()))
        return True
    except Exception:
        return False

# Генерирует случайный nonce, возвращает в base64 и сырые байты
def generate_nonce(size):
    buf = os.urandom(size)
    return base64.b64encode(buf).decode('utf-8'), buf

# Выполняет начальный этап обмена ключами с сервером
def perform_handshake(init_url, access_token=None):
    rsa_priv, rsa_pub, ecdsa_priv, ecdsa_pub = generate_keys()
    rsa_pub_der = rsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    ecdsa_pub_der = ecdsa_pub.public_bytes(
        encoding=serialization.Encoding.DER,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    rsa_pub_der_b64 = base64.b64encode(rsa_pub_der).decode()
    ecdsa_pub_der_b64 = base64.b64encode(ecdsa_pub_der).decode()
    nonce1_b64, nonce1 = generate_nonce(8)
    data_to_sign = base64.b64decode(rsa_pub_der_b64) + base64.b64decode(ecdsa_pub_der_b64) + base64.b64decode(nonce1_b64)
    signature1 = sign_ecdsa(ecdsa_priv, data_to_sign)
    payload = {
        "rsa_pub_client": rsa_pub_der_b64,
        "ecdsa_pub_client": ecdsa_pub_der_b64,
        "nonce1": nonce1_b64,
        "signature1": signature1
    }
    headers = {"Authorization": f"Bearer {access_token}"} if access_token else {}
    response = requests.post(init_url, json=payload, headers=headers)
    response.raise_for_status()
    server_data = response.json()
    client_id = server_data["client_id"]
    rsa_pub_server_der = base64.b64decode(server_data["rsa_pub_server"])
    ecdsa_pub_server_der = base64.b64decode(server_data["ecdsa_pub_server"])
    nonce2 = base64.b64decode(server_data["nonce2"])
    signature2_b64 = server_data["signature2"]
    rsa_pub_server = serialization.load_der_public_key(rsa_pub_server_der)
    ecdsa_pub_server = serialization.load_der_public_key(ecdsa_pub_server_der)
    data_to_verify = rsa_pub_server_der + ecdsa_pub_server_der + nonce2 + nonce1 + client_id.encode('utf-8')
    if not verify_ecdsa(ecdsa_pub_server, data_to_verify, signature2_b64):
        raise Exception("Ошибка проверки подписи сервера")
    return {
        "client_id": client_id,
        "rsa_priv": rsa_priv,
        "ecdsa_priv": ecdsa_priv,
        "rsa_pub_server": rsa_pub_server,
        "ecdsa_pub_server": ecdsa_pub_server,
        "nonce2": nonce2,
        "nonce1": nonce1,
        "nonce1_b64": nonce1_b64
    }

# Завершает обмен ключами, устанавливает сессионный ключ
def perform_finalize(fin_url, handshake_data, access_token=None, nonce3=None):
    _, ks = generate_nonce(32)
    nonce3_b64, nonce3 = generate_nonce(8) if nonce3 is None else (base64.b64encode(nonce3).decode('utf-8'), nonce3)
    payload = ks + nonce3 + handshake_data["nonce2"]
    data_to_sign = payload
    sig3 = sign_ecdsa(handshake_data["ecdsa_priv"], data_to_sign)
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
    payload = {
        "encrypted": encrypted_b64,
        "signature3": sig3
    }
    headers = {"X-Client-ID": handshake_data["client_id"],
               "Authorization": f"Bearer {access_token}"} if access_token else {
        "X-Client-ID": handshake_data["client_id"]}
    response = requests.post(fin_url, json=payload, headers=headers, timeout=5)
    response.raise_for_status()
    response_data = response.json()
    signature4_b64 = response_data["signature4"]
    data_to_verify = ks + nonce3 + handshake_data["nonce2"]
    if not verify_ecdsa(handshake_data["ecdsa_pub_server"], data_to_verify, signature4_b64):
        raise Exception("Ошибка проверки подписи сервера (signature4)")
    return ks

# Деривирует ключи шифрования и HMAC из сессионного ключа
def derive_keys(ks):
    h_enc = HMAC(ks, hashes.SHA256())
    h_enc.update(b"enc")
    k_enc = h_enc.finalize()
    h_mac = HMAC(ks, hashes.SHA256())
    h_mac.update(b"mac")
    k_mac = h_mac.finalize()
    return k_enc, k_mac

# Шифрует файл с использованием AES-CBC и HMAC
def encrypt_file(file_path, k_enc_b64, k_mac_b64):
    k_enc = base64.b64decode(k_enc_b64)
    k_mac = base64.b64decode(k_mac_b64)
    with open(file_path, "rb") as f:
        plaintext = f.read()
    padder = PKCS7(128).padder()
    padded = padder.update(plaintext) + padder.finalize()
    nonce = os.urandom(16)
    iv = os.urandom(16)
    cipher = Cipher(algorithms.AES(k_enc), modes.CBC(iv))
    encryptor = cipher.encryptor()
    ciphertext = encryptor.update(padded) + encryptor.finalize()
    h = HMAC(k_mac, hashes.SHA256())
    h.update(iv)
    h.update(ciphertext)
    tag = h.finalize()
    pkg = nonce + iv + ciphertext + tag
    return pkg

# Расшифровывает файл, проверяет HMAC и снимает padding
def decrypt_file(encrypted_data, k_enc_b64, k_mac_b64):
    try:
        k_enc = base64.b64decode(k_enc_b64)
        k_mac = base64.b64decode(k_mac_b64)
        if len(encrypted_data) < 16 + 16 + 16 + 32:
            return None
        nonce_bytes = encrypted_data[:16]
        iv = encrypted_data[16:32]
        ciphertext = encrypted_data[32:-32]
        tag = encrypted_data[-32:]
        computed_hmac = hmac.new(k_mac, iv + ciphertext, hashlib.sha256).digest()
        if not hmac.compare_digest(computed_hmac, tag):
            return None
        if len(ciphertext) % 16 != 0:
            return None
        cipher = Cipher(algorithms.AES(k_enc), modes.CBC(iv), backend=default_backend())
        decryptor = cipher.decryptor()
        plaintext = decryptor.update(ciphertext) + decryptor.finalize()
        unpadder = PKCS7(128).unpadder()
        unpadded_data = unpadder.update(plaintext) + unpadder.finalize()
        return unpadded_data
    except Exception:
        return None


# Потоковая загрузка зашифрованного файла
def stream_upload_encrypted_file(file_path, cloud_url, access_token, category, k_enc, k_mac):
    """
    Потоково загружает зашифрованный файл на сервер, используя AES-CBC и HMAC-SHA256.

    Args:
        file_path (str): Путь к файлу для загрузки.
        cloud_url (str): URL эндпоинта для загрузки (e.g., http://localhost:8080/files/one/encrypted).
        access_token (str): Bearer-токен для авторизации.
        category (str): Категория файла (photo, video, text, unknown).
        k_enc (bytes): Ключ шифрования AES (32 байта).
        k_mac (bytes): Ключ HMAC-SHA256 (32 байта).

    Returns:
        dict: JSON-ответ сервера с метаданными и presigned URL.

    Raises:
        Exception: Если загрузка не удалась или сервер вернул ошибку.
    """
    # Проверяем существование файла
    if not os.path.exists(file_path):
        raise Exception(f"Файл {file_path} не существует")

    # Инициализируем AES-CBC и HMAC
    iv = os.urandom(16)
    cipher = Cipher(algorithms.AES(k_enc), modes.CBC(iv))
    encryptor = cipher.encryptor()
    h = HMAC(k_mac, hashes.SHA256())
    h.update(iv)

    # Генерируем nonce
    nonce = os.urandom(16)

    # Определяем размер чанка (100 МБ, как в Go)
    chunk_size = 100 * 1024 * 1024  # 100 MB

    def generate_chunks():
        """Генератор для потокового чтения, шифрования и HMAC."""
        with open(file_path, "rb") as f:
            # Отправляем nonce и IV первыми
            yield nonce
            yield iv

            # Читаем и шифруем файл по чанкам
            padder = PKCS7(128).padder()
            while True:
                chunk = f.read(chunk_size)
                if not chunk:
                    # Конец файла: добавляем padding
                    padded = padder.finalize()
                    if padded:
                        ciphertext = encryptor.update(padded)
                        h.update(ciphertext)
                        yield ciphertext
                    break

                # Добавляем данные в padder
                padded_chunk = padder.update(chunk)
                if padded_chunk:
                    ciphertext = encryptor.update(padded_chunk)
                    h.update(ciphertext)
                    yield ciphertext

                if len(chunk) < chunk_size:
                    # Конец файла: добавляем padding
                    padded = padder.finalize()
                    if padded:
                        ciphertext = encryptor.update(padded)
                        h.update(ciphertext)
                        yield ciphertext
                    break

            # Завершаем шифрование и отправляем HMAC tag
            ciphertext = encryptor.finalize()
            if ciphertext:
                h.update(ciphertext)
                yield ciphertext
            yield h.finalize()

    # Формируем запрос
    headers = {
        "Authorization": f"Bearer {access_token}",
        "X-File-Category": category,
        "X-Orig-Filename": os.path.basename(file_path),
        "X-Orig-Mime": "audio/x-psf",  # Как в Go примере
        "Content-Type": "application/octet-stream"
    }

    # Отправляем потоковый запрос
    try:
        response = requests.post(cloud_url, headers=headers, data=generate_chunks(), timeout=60)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.HTTPError as e:
        error_body = response.text if response else "No response body"
        raise Exception(f"Ошибка загрузки {response.status_code}: {error_body}") from e
    except Exception as e:
        raise Exception(f"Ошибка при загрузке: {e}")


# Тестирует сессию, отправляя зашифрованное сообщение
def perform_session_test(test_url, session, access_token=None, plaintext="Hello, Secure World!"):
    ts = int(time.time() * 1000)
    timestamp = ts.to_bytes(8, byteorder='big')
    nonce = os.urandom(16)
    if isinstance(plaintext, str):
        plaintext_bytes = plaintext.encode('utf-8')
    else:
        plaintext_bytes = plaintext
    blob = timestamp + nonce + plaintext_bytes
    padder = PKCS7(128).padder()
    padded = padder.update(blob) + padder.finalize()
    iv = os.urandom(16)
    cipher = Cipher(algorithms.AES(session["k_enc"]), modes.CBC(iv))
    encryptor = cipher.encryptor()
    ciphertext = encryptor.update(padded) + encryptor.finalize()
    h = HMAC(session["k_mac"], hashes.SHA256())
    h.update(iv)
    h.update(ciphertext)
    tag = h.finalize()
    pkg = iv + ciphertext + tag
    b64msg = base64.b64encode(pkg).decode('utf-8')
    signature = sign_ecdsa(session["ecdsa_priv"], pkg)
    payload = {
        "encrypted_message": b64msg,
        "client_signature": signature
    }
    headers = {
        "X-Client-ID": session["client_id"],
        "Content-Type": "application/json",
        "Authorization": f"Bearer {access_token}" if access_token else ""
    }
    response = requests.post(test_url, json=payload, headers=headers, timeout=30)
    response.raise_for_status()
    response_data = response.json()
    return response_data

# Запускает процесс обмена ключами и тестирования сессии
def main():
    init_url = "http://localhost:8080/handshake/init"
    fin_url = "http://localhost:8080/handshake/finalize"
    test_url = "http://localhost:8080/session/test"
    handshake_data = perform_handshake(init_url)
    ks = perform_finalize(fin_url, handshake_data)
    K_enc, K_mac = derive_keys(ks)
    session = {
        "client_id": handshake_data["client_id"],
        "k_enc": K_enc,
        "k_mac": K_mac,
        "ecdsa_priv": handshake_data["ecdsa_priv"]
    }
    perform_session_test(test_url, session)

if __name__ == "__main__":
    main()