import base64
import jwt
import sqlite3
import logging
import os
import requests
from telegram import Update, ReplyKeyboardMarkup, KeyboardButton, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ApplicationBuilder, CommandHandler, MessageHandler, ContextTypes, ConversationHandler, filters
from telegram.error import TimedOut
import asyncio
from client_http import encrypt_file,perform_finalize,perform_handshake,derive_keys

# Настройка логирования
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s', level=logging.INFO)
logger = logging.getLogger(__name__)

# Токен бота
TOKEN = "7745265874:AAFKjBo36eDqq08hN30QTqK0v27HDnMlDL0"

# URL API хранилища (хардкорные значения)
AUTH_BASE_URL = "http://localhost:8081"  # Для Auth API (signup, login, logout)
SECURECOMM_BASE_URL = "http://localhost:8080"  # Для SecureComm API (handshake, session)
CLOUD_BASE_URL = "http://localhost:8080"  # Для Cloud API (MinIO)

SIGNUP_URL = f"{AUTH_BASE_URL}/user/signup"
LOGIN_URL = f"{AUTH_BASE_URL}/user/login"
LOGOUT_URL = f"{AUTH_BASE_URL}/user/logout"
HANDSHAKE_INIT_URL = f"{SECURECOMM_BASE_URL}/handshake/init"
HANDSHAKE_FINALIZE_URL = f"{SECURECOMM_BASE_URL}/handshake/finalize"
UPLOAD_FILES_URL = f"{CLOUD_BASE_URL}/files/one/encrypted"  # Новый эндпоинт
GET_FILE_URL = f"{CLOUD_BASE_URL}/files/one"
DELETE_FILE_URL = f"{CLOUD_BASE_URL}/files/one"
GET_ALL_FILES_URL = f"{CLOUD_BASE_URL}/files/all"
DELETE_FILES_URL = f"{CLOUD_BASE_URL}/files/many"

# Кнопки главного меню (до входа)
MAIN_MENU_BUTTONS = [
    [KeyboardButton('📝 Зарегистрироваться'), KeyboardButton('🔑 Войти')],
    [KeyboardButton('ℹ️ Помощь')],
]

# Кнопки меню после входа
FILE_MENU_BUTTONS = [
    [KeyboardButton('📤 Загрузить файл'), KeyboardButton('📥 Получить файл')],
    [KeyboardButton('🗑️ Удалить файл'), KeyboardButton('📂 Получить все файлы')],
    [KeyboardButton('🚪 Выйти')],
]

# Состояния для ConversationHandler
EMAIL, PASSWORD = range(2)
FILE_ID, FILE_TYPE, FILE_CATEGORY = range(3)

# Инициализация базы данных SQLite
def init_db():
    conn = sqlite3.connect("sessions.db")
    cursor = conn.cursor()
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS sessions (
            telegram_id INTEGER PRIMARY KEY,
            client_id TEXT,
            k_enc TEXT,
            k_mac TEXT
        )
    ''')
    conn.commit()
    conn.close()
# Сохранение сессии в SQLite
def save_session(telegram_id, client_id, k_enc=None, k_mac=None):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute('''
            INSERT OR REPLACE INTO sessions (telegram_id, client_id, k_enc, k_mac)
            VALUES (?, ?, ?, ?)
        ''', (telegram_id, client_id, k_enc, k_mac))
        conn.commit()
    except sqlite3.Error as e:
        logger.error(f"Ошибка при сохранении сессии: {e}")
        raise
    finally:
        conn.close()

# Получение сессии из SQLite
def get_session(telegram_id):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute(
            "SELECT client_id, k_enc, k_mac FROM sessions WHERE telegram_id = ?",
            (telegram_id,))
        result = cursor.fetchone()
        if result:
            return {
                "client_id": result[0],
                "k_enc": result[1],
                "k_mac": result[2]
            }
        return None
    except sqlite3.Error as e:
        logger.error(f"Ошибка при получении сессии: {e}")
        return None
    finally:
        conn.close()

# Глобальный обработчик ошибок
async def error_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    logger.error(f"Update {update} caused error: {context.error}")
    if isinstance(context.error, TimedOut):
        if update and update.message:
            try:
                await update.message.reply_text(
                    "Произошла ошибка из-за таймаута. Пожалуйста, попробуйте снова через несколько секунд."
                )
            except Exception as e:
                logger.error(f"Failed to send timeout error message: {str(e)}")
    else:
        if update and update.message:
            try:
                await update.message.reply_text(
                    "Произошла ошибка при обработке запроса. Попробуйте снова позже."
                )
            except Exception as e:
                logger.error(f"Failed to send error message: {str(e)}")

# Приветствие с главным меню
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = "Привет! Выберите нужную команду на панели ниже или напишите сообщение."
    reply_markup = ReplyKeyboardMarkup(MAIN_MENU_BUTTONS, resize_keyboard=True)
    await update.message.reply_text(text, reply_markup=reply_markup)

# Команда /help
async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if session and context.user_data.get("access_token"):
        text = "🟢 *Меню команд:*\n" \
               "• 📤 Загрузить файл – загрузить новый файл\n" \
               "• 📥 Получить файл – скачать файл по ID\n" \
               "• 🗑️ Удалить файл – удалить файл по ID\n" \
               "• 📂 Получить все файлы – получить файлы по категории\n" \
               "• 🚪 Выйти – выйти из аккаунта\n"
        reply_markup = ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True)
    else:
        text = "🟢 *Меню команд:*\n" \
               "• 📝 Зарегистрироваться – начать регистрацию\n" \
               "• 🔑 Войти – войти в аккаунт\n" \
               "• ℹ️ Помощь – это сообщение\n"
        reply_markup = ReplyKeyboardMarkup(MAIN_MENU_BUTTONS, resize_keyboard=True)
    await update.message.reply_text(text, reply_markup=reply_markup, parse_mode="Markdown")

# Регистрация через API
async def register_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите ваш email:")
    return EMAIL

async def register_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD

async def register_password(update: Update, context: ContextTypes.DEFAULT_TYPE):
    email = context.user_data["email"]
    password = update.message.text
    if len(password) < 6:
        await update.message.reply_text("Пароль должен быть не короче 6 символов. Попробуйте снова.")
        return PASSWORD
    context.user_data["password"] = password
    payload = {"email": email, "password": password, "platform": "tg-bot"}

    try:
        response = requests.post(SIGNUP_URL, json=payload)
        response.raise_for_status()
        try:
            data = response.json()
        except ValueError:
            logger.error(f"Не удалось разобрать JSON ответа: {response.text}")
            raise ValueError("Сервер вернул невалидный JSON")
        access_token = data.get("access_token")
        refresh_token = data.get("refresh_token")
        if not access_token or not refresh_token:
            raise ValueError("Сервер не вернул access_token или refresh_token")
        context.user_data["access_token"] = access_token
        context.user_data["refresh_token"] = refresh_token

        # Выполняем handshake для получения client_id, k_enc и k_mac
        handshake_data = perform_handshake(HANDSHAKE_INIT_URL, access_token)
        ks = perform_finalize(HANDSHAKE_FINALIZE_URL, handshake_data, access_token)
        client_id = handshake_data["client_id"]

        # Деривируем ключи k_enc и k_mac
        k_enc, k_mac = derive_keys(ks)
        k_enc_b64 = base64.b64encode(k_enc).decode('utf-8')
        k_mac_b64 = base64.b64encode(k_mac).decode('utf-8')

        # Сохраняем сессию
        save_session(update.effective_user.id, client_id, k_enc_b64, k_mac_b64)

        await update.message.reply_text("Регистрация успешна! Теперь вы можете работать с файлами.", reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка регистрации: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка регистрации: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при регистрации: {e}")
        await update.message.reply_text("Произошла ошибка. Попробуйте снова.")

    return ConversationHandler.END

async def register_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Регистрация отменена.")
    return ConversationHandler.END

# Вход через API
async def login_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите ваш email:")
    return EMAIL

async def login_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD


async def login_password(update: Update, context: ContextTypes.DEFAULT_TYPE):
    email = context.user_data["email"]
    password = update.message.text
    context.user_data["password"] = password
    payload = {"email": email, "password": password, "platform": "tg-bot"}

    try:
        response = requests.post(LOGIN_URL, json=payload)
        response.raise_for_status()
        try:
            data = response.json()
        except ValueError:
            logger.error(f"Не удалось разобрать JSON ответа: {response.text}")
            raise ValueError("Сервер вернул невалидный JSON")

        access_token = data.get("access_token")
        refresh_token = data.get("refresh_token")
        k_enc = data.get("k_enc")  # Получаем k_enc в Base64
        k_mac = data.get("k_mac")  # Получаем k_mac в Base64
        if not access_token or not refresh_token or not k_enc or not k_mac:
            raise ValueError("Сервер не вернул access_token, refresh_token, k_enc или k_mac")
        context.user_data["access_token"] = access_token
        context.user_data["refresh_token"] = refresh_token

        # Извлекаем client_id из access_token
        try:
            decoded_token = jwt.decode(access_token, options={"verify_signature": False})
            client_id = decoded_token.get("client_id")  # Проверяем client_id
            if not client_id:
                # Если client_id не найден, проверяем альтернативные поля
                client_id = decoded_token.get("sub") or decoded_token.get("user_id") or decoded_token.get("id")
                if not client_id:
                    raise ValueError("client_id не найден в access_token под ожидаемыми полями")
        except jwt.InvalidTokenError as e:
            logger.error(f"Ошибка декодирования access_token: {e}")
            raise ValueError("Не удалось декодировать access_token")

        # Сохраняем сессию
        save_session(update.effective_user.id, client_id, k_enc, k_mac)

        await update.message.reply_text("Вход успешен! Теперь вы можете работать с файлами.",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
        return ConversationHandler.END

    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка входа: {e}, Ответ сервера: {response.text}")
        if "incorrect password or email" in error_msg or "Invalid credentials" in error_msg or "User not found" in error_msg:
            await update.message.reply_text("Неверный email или пароль. Попробуйте снова с ввода email.")
            return EMAIL
        await update.message.reply_text(f"Ошибка входа: {error_msg}")
        return ConversationHandler.END
    except Exception as e:
        logger.error(f"Ошибка при входе: {e}")
        await update.message.reply_text("Произошла ошибка. Попробуйте снова с ввода email.")
        return EMAIL

async def login_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Вход отменён.")
    return ConversationHandler.END

# Выход из аккаунта
async def logout(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session:
        await update.message.reply_text("Вы не вошли в аккаунт.")
        return
    refresh_token = context.user_data.get("refresh_token")
    if not refresh_token:
        await update.message.reply_text("Ошибка: refresh_token не найден.")
        return
    try:
        payload = {"platform": "tg-bot", "refresh_token": refresh_token}
        response = requests.post(LOGOUT_URL, json=payload)
        response.raise_for_status()
        context.user_data.clear()
        try:
            conn = sqlite3.connect("sessions.db")
            cursor = conn.cursor()
            cursor.execute("DELETE FROM sessions WHERE telegram_id = ?", (update.effective_user.id,))
            conn.commit()
            logger.info(f"Данные пользователя с telegram_id {update.effective_user.id} удалены после логаута.")
        except sqlite3.Error as e:
            logger.error(f"Ошибка при удалении данных пользователя: {e}")
        finally:
            conn.close()
        await update.message.reply_text("Вы успешно вышли из аккаунта! Локальные данные пользователя удалены.")
        await start(update, context)
    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка выхода: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка выхода: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при выходе: {e}")
        await update.message.reply_text("Произошла ошибка. Попробуйте снова.")

# Обработка загрузки файла
async def upload_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Пожалуйста, войдите в аккаунт, чтобы работать с файлами.")
        return
    await update.message.reply_text("Пожалуйста, отправьте файл для загрузки (до 50 МБ).")
    return ConversationHandler.END

async def handle_file(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Пожалуйста, войдите в аккаунт, чтобы работать с файлами.")
        return

    try:
        # Проверяем, есть ли файл или фото в сообщении
        if not update.message.document and not update.message.photo and not update.message.video:
            await update.message.reply_text("Ошибка: файл или фото не получено. Пожалуйста, отправьте файл или фотографию видео.")
            logger.error("No document or photo received in message")
            return

        # Получаем объект файла или фото
        if update.message.document:
            document = update.message.document
            file_name = document.file_name or f"file_{update.message.message_id}.bin"
            file_size = document.file_size
        elif update.message.photo:
            photo = update.message.photo[-1]  # Берем фото с максимальным разрешением
            file_name = f"photo_{update.message.message_id}.jpg"
            file_size = photo.file_size
        elif update.message.video:
            video = update.message.video
            file_name = video.file_name or f"video_{update.message.message_id}.mp4"
            file_size = video.file_size

        max_file_size = 50 * 1024 * 1024  # 50 МБ в байтах

        # Проверяем размер файла
        if file_size > max_file_size:
            url = "https://example.com/upload"
            text = f"Файл {file_name} превышает 50 МБ ({file_size / (1024 * 1024):.2f} МБ). " \
                   "Пожалуйста, загрузите файл через наш сайт:"
            keyboard = [[InlineKeyboardButton('🔗 Загрузить на сайте', url=url)]]
            reply_markup = InlineKeyboardMarkup(keyboard)
            await update.message.reply_text(text, reply_markup=reply_markup)
            logger.info(f"File {file_name} too large: {file_size} bytes")
            return

        # Скачиваем файл или фото локально
        file = await (document.get_file() if update.message.document else
                      photo.get_file() if update.message.photo else
                      video.get_file())
        os.makedirs("uploads", exist_ok=True)
        file_path = os.path.join("uploads", file_name)

        for attempt in range(3):
            try:
                await asyncio.wait_for(file.download_to_drive(file_path), timeout=30)
                logger.info(f"File {file_name} successfully downloaded to {file_path}")
                break
            except asyncio.TimeoutError:
                logger.warning(f"Download attempt {attempt + 1} failed for {file_name}: Timeout")
                if attempt == 2:
                    await update.message.reply_text(
                        f"Не удалось скачать файл {file_name} из-за таймаута. Попробуйте снова."
                    )
                    logger.error(f"Failed to download {file_name} after 3 attempts")
                    return
                continue

        # Шифруем файл перед отправкой
        encrypted_data = encrypt_file(file_path, session["k_enc"], session["k_mac"])
        if not encrypted_data:
            raise Exception("Ошибка шифрования файла")

        # Отправляем зашифрованный файл
        access_token = context.user_data["access_token"]
        headers = {
            "Authorization": f"Bearer {access_token}",
            "Content-Type": "application/octet-stream",
            "X-Orig-Filename": file_name,
            "X-Orig-Mime": (document.mime_type if update.message.document else
                            "image/jpeg" if update.message.photo else
                            "video/mp4" if update.message.video else
                            "application/octet-stream"),
            "X-File-Category": "unknown"
        }
        response = requests.post(UPLOAD_FILES_URL, headers=headers, data=encrypted_data)

        response.raise_for_status()
        response_data = response.json()
        logger.info(f"File {file_name} uploaded to MinIO: {response_data}")

        # Удаляем локальный файл
        os.remove(file_path)

        await update.message.reply_text(f"Файл {file_name} успешно загружен! Ответ сервера: {response_data}")

    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка загрузки файла: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка загрузки файла: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при обработке файла {file_name if 'file_name' in locals() else 'unknown'}: {str(e)}")
        await update.message.reply_text(f"Ошибка при обработке файла: {str(e)}. Попробуйте снова.")

# Получение одного файла
async def get_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Пожалуйста, войдите в аккаунт, чтобы работать с файлами.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла:")
    return FILE_ID

async def get_file_id(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["file_id"] = update.message.text
    await update.message.reply_text("Введите категорию файла (photo, unknown, video, text):")
    return FILE_TYPE

async def get_file_type(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_id = context.user_data["file_id"]
    file_type = update.message.text.lower()
    if file_type not in ["photo", "unknown", "video", "text"]:
        await update.message.reply_text("Неверная категория. Допустимые категории: photo, unknown, video, text. Попробуйте снова.")
        return FILE_TYPE

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}
        params = {"id": file_id, "type": file_type}
        response = requests.get(GET_FILE_URL, headers=headers, params=params)
        response.raise_for_status()
        response_data = response.json()
        await update.message.reply_text(f"Ссылка на скачивание файла: {response_data}")
    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка получения файла: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка получения файла: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при получении файла: {str(e)}")
        await update.message.reply_text(f"Ошибка при получении файла: {str(e)}. Попробуйте снова.")

    return ConversationHandler.END

async def get_file_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Получение файла отменено.")
    return ConversationHandler.END

# Удаление одного файла
async def delete_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Пожалуйста, войдите в аккаунт, чтобы работать с файлами.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла для удаления:")
    return FILE_ID

async def delete_file_id(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["file_id"] = update.message.text
    await update.message.reply_text("Введите категорию файла (photo, unknown, video, text):")
    return FILE_TYPE

async def delete_file_type(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_id = context.user_data["file_id"]
    file_type = update.message.text.lower()
    if file_type not in ["photo", "unknown", "video", "text"]:
        await update.message.reply_text("Неверная категория. Допустимые категории: photo, unknown, video, text. Попробуйте снова.")
        return FILE_TYPE

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}
        params = {"id": file_id, "type": file_type}
        response = requests.delete(DELETE_FILE_URL, headers=headers, params=params)
        response.raise_for_status()
        response_data = response.json()
        await update.message.reply_text(f"Файл успешно удалён! Ответ сервера: {response_data}")
    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка удаления файла: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка удаления файла: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при удалении файла: {str(e)}")
        await update.message.reply_text(f"Ошибка при удалении файла: {str(e)}. Попробуйте снова.")

    return ConversationHandler.END

async def delete_file_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Удаление файла отменено.")
    return ConversationHandler.END

# Получение всех файлов категории
async def get_all_files_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Пожалуйста, войдите в аккаунт, чтобы работать с файлами.")
        return ConversationHandler.END
    await update.message.reply_text("Введите категорию файлов (photo, unknown, video, text):")
    return FILE_CATEGORY

async def get_all_files_category(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_category = update.message.text.lower()
    if file_category not in ["photo", "unknown", "video", "text"]:
        await update.message.reply_text("Неверная категория. Допустимые категории: photo, unknown, video, text. Попробуйте снова.")
        return FILE_CATEGORY

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}
        params = {"type": file_category}
        response = requests.get(GET_ALL_FILES_URL, headers=headers, params=params)
        response.raise_for_status()
        response_data = response.json()
        await update.message.reply_text(f"Ссылки на файлы категории {file_category}: {response_data}")
    except requests.exceptions.HTTPError as e:
        error_msg = "Неизвестная ошибка"
        try:
            error_msg = response.json().get("error", "Неизвестная ошибка")
        except ValueError:
            error_msg = f"Сервер вернул невалидный JSON: {response.text}"
        logger.error(f"Ошибка получения файлов: {e}, Ответ сервера: {response.text}")
        await update.message.reply_text(f"Ошибка получения файлов: {error_msg}")
    except Exception as e:
        logger.error(f"Ошибка при получении файлов: {str(e)}")
        await update.message.reply_text(f"Ошибка при получении файлов: {str(e)}. Попробуйте снова.")

    return ConversationHandler.END

async def get_all_files_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Получение файлов отменено.")
    return ConversationHandler.END

# Обработка текстовых сообщений
async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    txt = update.message.text.lower()
    if "зарегистрироваться" in txt or "📝" in txt:
        await register_start(update, context)
    elif "вход" in txt or "войти" in txt or "🔑" in txt:
        await login_start(update, context)
    elif "помощь" in txt or "ℹ️" in txt:
        await help_command(update, context)
    elif "выйти" in txt or "🚪" in txt:
        await logout(update, context)
    elif "загрузить файл" in txt or "📤" in txt:
        await upload_file_start(update, context)
    elif "получить файл" in txt or "📥" in txt:
        await get_file_start(update, context)
    elif "удалить файл" in txt or "🗑️" in txt:
        await delete_file_start(update, context)
    elif "получить все файлы" in txt or "📂" in txt:
        await get_all_files_start(update, context)
    else:
        await update.message.reply_text("Я вас понял! Для выбора действия используйте панель ниже или команды.")

def main():
    init_db()
    app = ApplicationBuilder().token(TOKEN).build()

    # ConversationHandler для регистрации
    register_handler = ConversationHandler(
        entry_points=[CommandHandler("register", register_start),
                      MessageHandler(filters.Regex('^(📝 Зарегистрироваться)$'), register_start)],
        states={
            EMAIL: [MessageHandler(filters.TEXT & ~filters.COMMAND, register_email)],
            PASSWORD: [MessageHandler(filters.TEXT & ~filters.COMMAND, register_password)],
        },
        fallbacks=[CommandHandler("cancel", register_cancel)]
    )

    # ConversationHandler для входа
    login_handler = ConversationHandler(
        entry_points=[CommandHandler("login", login_start), MessageHandler(filters.Regex('^(🔑 Войти)$'), login_start)],
        states={
            EMAIL: [MessageHandler(filters.TEXT & ~filters.COMMAND, login_email)],
            PASSWORD: [MessageHandler(filters.TEXT & ~filters.COMMAND, login_password)],
        },
        fallbacks=[CommandHandler("cancel", login_cancel)]
    )

    # ConversationHandler для получения файла
    get_file_handler = ConversationHandler(
        entry_points=[MessageHandler(filters.Regex('^(📥 Получить файл)$'), get_file_start)],
        states={
            FILE_ID: [MessageHandler(filters.TEXT & ~filters.COMMAND, get_file_id)],
            FILE_TYPE: [MessageHandler(filters.TEXT & ~filters.COMMAND, get_file_type)],
        },
        fallbacks=[CommandHandler("cancel", get_file_cancel)]
    )

    # ConversationHandler для удаления файла
    delete_file_handler = ConversationHandler(
        entry_points=[MessageHandler(filters.Regex('^(🗑️ Удалить файл)$'), delete_file_start)],
        states={
            FILE_ID: [MessageHandler(filters.TEXT & ~filters.COMMAND, delete_file_id)],
            FILE_TYPE: [MessageHandler(filters.TEXT & ~filters.COMMAND, delete_file_type)],
        },
        fallbacks=[CommandHandler("cancel", delete_file_cancel)]
    )

    # ConversationHandler для получения всех файлов
    get_all_files_handler = ConversationHandler(
        entry_points=[MessageHandler(filters.Regex('^(📂 Получить все файлы)$'), get_all_files_start)],
        states={
            FILE_CATEGORY: [MessageHandler(filters.TEXT & ~filters.COMMAND, get_all_files_category)],
        },
        fallbacks=[CommandHandler("cancel", get_all_files_cancel)]
    )

    # Добавляем обработчики
    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("help", help_command))
    app.add_handler(CommandHandler("logout", logout))
    app.add_handler(register_handler)
    app.add_handler(login_handler)
    app.add_handler(get_file_handler)
    app.add_handler(delete_file_handler)
    app.add_handler(get_all_files_handler)
    app.add_handler(MessageHandler(filters.Document.ALL, handle_file))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    app.add_error_handler(error_handler)

    print("Бот запущен...")
    app.run_polling()

if __name__ == '__main__':
    main()