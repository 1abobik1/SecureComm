import base64
import jwt
import sqlite3
import logging
import os
import requests
from telegram import Update, ReplyKeyboardMarkup, KeyboardButton, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ApplicationBuilder, CommandHandler, MessageHandler, ContextTypes, ConversationHandler, filters, \
    CallbackQueryHandler
from telegram.error import TimedOut
import asyncio
from client_http import encrypt_file, decrypt_file, perform_finalize, perform_handshake, derive_keys
from datetime import datetime
import pytz
import re

# Настройка логирования
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s', level=logging.INFO)
logger = logging.getLogger(__name__)

# Токен бота
TOKEN = "7745265874:AAFKjBo36eDqq08hN30QTqK0v27HDnMlDL0"

# URL API хранилища
AUTH_BASE_URL = "http://localhost:8081"
SECURECOMM_BASE_URL = "http://localhost:8080"
CLOUD_BASE_URL = "http://localhost:8080"

SIGNUP_URL = f"{AUTH_BASE_URL}/user/signup"
LOGIN_URL = f"{AUTH_BASE_URL}/user/login"
LOGOUT_URL = f"{AUTH_BASE_URL}/user/logout"
HANDSHAKE_INIT_URL = f"{SECURECOMM_BASE_URL}/handshake/init"
HANDSHAKE_FINALIZE_URL = f"{SECURECOMM_BASE_URL}/handshake/finalize"
UPLOAD_FILES_URL = f"{CLOUD_BASE_URL}/files/one/encrypted"
GET_FILE_URL = f"{CLOUD_BASE_URL}/files/one"
DELETE_FILES_URL = f"{CLOUD_BASE_URL}/files/many"  # Используем /files/many
GET_ALL_FILES_URL = f"{CLOUD_BASE_URL}/files/all"

# Кнопки меню
MAIN_MENU_BUTTONS = [
    [KeyboardButton('📝 Зарегистрироваться'), KeyboardButton('🔑 Войти')],
    [KeyboardButton('ℹ️ Помощь')],
]
FILE_MENU_BUTTONS = [
    [KeyboardButton('📤 Загрузить файл'), KeyboardButton('📥 Получить файл')],
    [KeyboardButton('🗑️ Удалить файлы'), KeyboardButton('📂 Получить все файлы')],  # Изменено на "Удалить файлы"
    [KeyboardButton('📊 Проверить использование'), KeyboardButton('🚪 Выйти')],
]

# Состояния для ConversationHandler
EMAIL, PASSWORD = range(2)
FILE_ID = range(1)[0]
FILE_CATEGORY = range(1)[0]


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


# Сохранение и получение сессии
def save_session(telegram_id, client_id, k_enc=None, k_mac=None):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute('INSERT OR REPLACE INTO sessions (telegram_id, client_id, k_enc, k_mac) VALUES (?, ?, ?, ?)',
                       (telegram_id, client_id, k_enc, k_mac))
        conn.commit()
    except sqlite3.Error as e:
        logger.error(f"Ошибка при сохранении сессии: {e}")
        raise
    finally:
        conn.close()


def get_session(telegram_id):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute("SELECT client_id, k_enc, k_mac FROM sessions WHERE telegram_id = ?", (telegram_id,))
        result = cursor.fetchone()
        if result:
            client_id, k_enc, k_mac = result
            return {"client_id": client_id, "k_enc": k_enc, "k_mac": k_mac} if all([client_id, k_enc, k_mac]) else {}
        return {}
    except sqlite3.Error as e:
        logger.error(f"Ошибка при получении сессии: {e}")
        return {}
    finally:
        conn.close()


# Определение категории файла по MIME-типу
def get_file_category(mime_type):
    photo_mimes = ['image/jpeg', 'image/png', 'image/gif', 'image/bmp', 'image/webp']
    video_mimes = ['video/mp4', 'video/mpeg', 'video/avi', 'video/mov', 'video/webm']
    text_mimes = [
        'text/plain',  # .txt
        'text/csv',  # .csv
        'application/pdf',  # .pdf
        'application/msword',  # .doc
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',  # .docx
        'text/x-python',  # .py
        'text/javascript',  # .js
        'text/html'  # .html
    ]

    if mime_type in photo_mimes:
        return 'photo'
    elif mime_type in video_mimes:
        return 'video'
    elif mime_type in text_mimes:
        return 'text'
    return 'unknown'


# Обработчик ошибок
async def error_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    logger.error(f"Update {update} caused error: {context.error}")
    if isinstance(context.error, TimedOut) and update and update.message:
        try:
            await update.message.reply_text("Таймаут. Попробуйте снова через несколько секунд.")
        except Exception as e:
            logger.error(f"Failed to send timeout error: {e}")
    elif update and update.message:
        try:
            await update.message.reply_text("Произошла ошибка. Попробуйте позже.")
        except Exception as e:
            logger.error(f"Failed to send error message: {e}")


# Начало работы
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = "Привет! Выберите действие ниже."
    await update.message.reply_text(text, reply_markup=ReplyKeyboardMarkup(MAIN_MENU_BUTTONS, resize_keyboard=True))


async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    text = "🟢 *Команды:*\n" if session and context.user_data.get("access_token") else "🟢 *Команды:*\n"
    text += "• 📝 Зарегистрироваться – начать регистрацию\n" \
            "• 🔑 Войти – войти в аккаунт\n" \
            "• ℹ️ Помощь – это сообщение\n"
    if session and context.user_data.get("access_token"):
        text += "• 📤 Загрузить файл – загрузить файл\n" \
                "• 📥 Получить файл – скачать файл по полному ID\n" \
                "• 🗑️ Удалить файлы – удалить один или несколько файлов по полному ID (например, '2/3ac6335b-bd4f-4bcd-a07f-b2c26a316644' или 'id1, id2, id3')\n" \
                "• 📂 Получить все файлы – список файлов по категории\n" \
                "• 📊 Проверить использование – данные о диске\n" \
                "• 🚪 Выйти – выйти из аккаунта\n"
    await update.message.reply_text(text, reply_markup=ReplyKeyboardMarkup(
        FILE_MENU_BUTTONS if session else MAIN_MENU_BUTTONS, resize_keyboard=True), parse_mode="Markdown")


# Регистрация
async def register_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите email:")
    return EMAIL


async def register_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD


async def register_password(update: Update, context: ContextTypes.DEFAULT_TYPE):
    email, password = context.user_data.get("email"), update.message.text
    if len(password) < 6:
        await update.message.reply_text("Пароль < 6 символов. Попробуйте снова.")
        return PASSWORD
    context.user_data["password"] = password
    payload = {"email": email, "password": password, "platform": "tg-bot"}

    try:
        response = requests.post(SIGNUP_URL, json=payload)
        response.raise_for_status()
        data = response.json()
        access_token, refresh_token = data.get("access_token"), data.get("refresh_token")
        if not access_token or not refresh_token:
            raise ValueError("Отсутствуют токены")
        context.user_data.update({"access_token": access_token, "refresh_token": refresh_token})

        handshake_data = perform_handshake(HANDSHAKE_INIT_URL, access_token)
        ks = perform_finalize(HANDSHAKE_FINALIZE_URL, handshake_data, access_token)
        client_id = handshake_data["client_id"]
        k_enc, k_mac = derive_keys(ks)
        save_session(update.effective_user.id, client_id, base64.b64encode(k_enc).decode('utf-8'),
                     base64.b64encode(k_mac).decode('utf-8'))

        await update.message.reply_text("Регистрация успешна!",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Регистрация: {e}, {response.text}")
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Регистрация: {e}")
        await update.message.reply_text("Ошибка. Попробуйте снова.")
    return ConversationHandler.END


async def register_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Регистрация отменена.")
    return ConversationHandler.END


async def login_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите email:")
    return EMAIL


async def login_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD


async def login_password(update: Update, context: ContextTypes.DEFAULT_TYPE):
    email, password = context.user_data.get("email"), update.message.text
    payload = {"email": email, "password": password, "platform": "tg-bot"}

    try:
        response = requests.post(LOGIN_URL, json=payload, timeout=10)
        response.raise_for_status()
        data = response.json()
        access_token, refresh_token, k_enc, k_mac = data.get("access_token"), data.get("refresh_token"), data.get("k_enc"), data.get("k_mac")
        if not all([access_token, refresh_token, k_enc, k_mac]):
            raise ValueError("Отсутствуют данные")
        context.user_data.update({"access_token": access_token, "refresh_token": refresh_token})

        decoded_token = jwt.decode(access_token, options={"verify_signature": False})
        client_id = decoded_token.get("client_id") or decoded_token.get("sub") or decoded_token.get("user_id") or decoded_token.get("id")
        if not client_id:
            raise ValueError("client_id не найден")

        save_session(update.effective_user.id, client_id, k_enc, k_mac)
        await update.message.reply_text("Вход успешен!",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except requests.exceptions.ConnectionError as e:
        logger.error(f"Connection error: {e}")
        await update.message.reply_text("Ошибка подключения к серверу. Проверьте, запущен ли сервер, и попробуйте снова.")
        return EMAIL
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Login error: {e}, {response.text}")
        if "not found" in error_msg.lower() or "invalid" in error_msg.lower() or "incorrect" in error_msg.lower():
            await update.message.reply_text("Неверный email или пароль. Введите email заново:")
            return EMAIL
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Login error: {e}")
        await update.message.reply_text("Ошибка. Попробуйте снова.")
        return EMAIL
    return ConversationHandler.END


async def login_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Вход отменён.")
    return ConversationHandler.END


# Выход
async def logout(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session:
        await update.message.reply_text("Вы не вошли.")
        return
    refresh_token = context.user_data.get("refresh_token")
    if not refresh_token:
        await update.message.reply_text("Ошибка: отсутствует refresh_token.")
        return

    try:
        response = requests.post(LOGOUT_URL, json={"platform": "tg-bot", "refresh_token": refresh_token})
        response.raise_for_status()
        context.user_data.clear()
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute("DELETE FROM sessions WHERE telegram_id = ?", (update.effective_user.id,))
        conn.commit()
        conn.close()
        logger.info(f"Сессия {update.effective_user.id} удалена")
        await start(update, context)
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Выход: {e}, {response.text}")
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Выход: {e}")
        await update.message.reply_text("Ошибка. Попробуйте снова.")


# Загрузка файла
async def upload_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Отправьте файл (до 50 МБ).")
    return ConversationHandler.END


async def handle_file(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return

    try:
        if not update.message.document and not update.message.photo and not update.message.video:
            await update.message.reply_text("Отправьте файл, фото или видео.")
            logger.error("No file received")
            return

        if update.message.document:
            document = update.message.document
            original_file_name = document.file_name or f"file_{update.message.message_id}.bin"
            file_size = document.file_size
            mime_type = document.mime_type or 'application/octet-stream'
        elif update.message.photo:
            photo = update.message.photo[-1]
            original_file_name = f"photo_{update.message.message_id}.jpg"
            file_size = photo.file_size
            mime_type = 'image/jpeg'
        elif update.message.video:
            video = update.message.video
            original_file_name = video.file_name or f"video_{update.message.message_id}.mp4"
            file_size = video.file_size
            mime_type = video.mime_type or 'video/mp4'

        if file_size > 50 * 1024 * 1024:
            await update.message.reply_text(f"Файл {original_file_name} > 50 МБ. Используйте сайт.")
            logger.info(f"File {original_file_name} too large: {file_size} bytes")
            return

        file = await (document.get_file() if update.message.document else
                      photo.get_file() if update.message.photo else
                      video.get_file())
        os.makedirs("uploads", exist_ok=True)
        safe_file_name = original_file_name
        file_path = os.path.join("uploads", safe_file_name)
        logger.info(f"Downloading to {file_path}")

        for attempt in range(3):
            try:
                await asyncio.wait_for(file.download_to_drive(file_path), timeout=30)
                logger.info(f"Downloaded {safe_file_name} to {file_path}")
                break
            except asyncio.TimeoutError:
                if attempt == 2:
                    await update.message.reply_text(f"Таймаут для {safe_file_name}. Попробуйте снова.")
                    logger.error(f"Download failed for {safe_file_name}")
                    return
                continue

        encrypted_data = encrypt_file(file_path, session["k_enc"], session["k_mac"])
        if not encrypted_data:
            raise Exception("Ошибка шифрования")

        file_category = get_file_category(mime_type)
        access_token = context.user_data["access_token"]
        # Кодируем имя файла в Base64
        encoded_file_name = base64.b64encode(safe_file_name.encode('utf-8')).decode('ascii')
        headers = {
            "Authorization": f"Bearer {access_token}",
            "Content-Type": "application/octet-stream",
            "X-Orig-Filename": encoded_file_name,
            "X-Orig-Mime": mime_type,
            "X-File-Category": file_category
        }
        response = requests.post(UPLOAD_FILES_URL, headers=headers, data=encrypted_data, timeout=30)
        response.raise_for_status()
        response_data = response.json()
        logger.info(f"Uploaded {safe_file_name}: Server returned - obj_id: {response_data.get('obj_id', 'не указан')}, "
                    f"url: {response_data.get('url', 'не указан')}, created_at: {response_data.get('created_at', 'не указан')}, "
                    f"mime_type: {response_data.get('mime_type', 'не указан')}, name: {response_data.get('name', 'не указан')}")

        obj_id = response_data.get("obj_id", "не указан")
        # Удаляем лишние точки из obj_id
        clean_obj_id = obj_id.rstrip('.')
        # Проверяем, содержит ли obj_id расширение, соответствующее mime_type
        expected_extension = get_file_extension(mime_type).lstrip('.')
        if clean_obj_id.lower().endswith(f".{expected_extension}"):
            obj_id_with_ext = clean_obj_id  # Используем obj_id как есть, без добавления расширения
        else:
            obj_id_with_ext = f"{clean_obj_id}.{expected_extension}"  # Добавляем расширение, если его нет
        download_url = response_data.get("url")
        created_at_raw = response_data.get("created_at", "не указан")
        if created_at_raw != "не указан":
            created_at_dt = datetime.strptime(created_at_raw, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=pytz.UTC)
            created_at = created_at_dt.astimezone(pytz.timezone("Europe/Moscow")).strftime("%d.%m.%Y %H:%M:%S")
        else:
            created_at = created_at_raw

        category_display = {
            "photo": "📸 Фото",
            "video": "📹 Видео",
            "text": "📝 Текст",
            "unknown": "📁 Прочее"
        }.get(file_category, file_category)

        if "file_urls" not in context.user_data:
            context.user_data["file_urls"] = {}
        # Сохраняем полный obj_id с расширением
        context.user_data["file_urls"][obj_id_with_ext] = {
            "full_obj_id": obj_id_with_ext,
            "url": download_url,
            "name": safe_file_name,
            "category": file_category
        }

        message = (
            f"Файл загружен!\n"
            f"📋 Детали:\n"
            f"• Имя: <code>{safe_file_name}</code>\n"
            f"• ID: <code>{obj_id_with_ext}</code>\n"  # Показываем ID с расширением
            f"• Тип: <code>{mime_type}</code>\n"
            f"• Категория: <code>{category_display}</code>\n"
            f"• Создан: <code>{created_at}</code>"
        )
        os.remove(file_path)
        await update.message.reply_text(message, parse_mode="HTML")

    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Upload error: {e}, {response.text}")
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"File processing error: {e}")
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")


# Получение файла
async def get_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла:")
    return FILE_ID


async def get_file_id(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_id = update.message.text.strip()  # Убираем лишние символы (например, точку)
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return ConversationHandler.END

    try:
        # Ищем информацию о файле в context.user_data["file_urls"] по полному ID
        file_info = context.user_data.get("file_urls", {}).get(file_id, {})
        download_url = file_info.get("url")
        file_name = file_info.get("name")
        full_obj_id = file_info.get("full_obj_id", file_id)
        file_category = file_info.get("category")

        # Если информация не найдена, запрашиваем с сервера
        if not download_url or not full_obj_id or not file_category:
            access_token = context.user_data["access_token"]
            headers = {"Authorization": f"Bearer {access_token}"}
            params = {"id": file_id, "type": file_category or "unknown"}
            response = requests.get(GET_FILE_URL, headers=headers, params=params)
            if response.status_code == 200:
                file_data = response.json()
                download_url = file_data.get("url")
                encoded_name = file_data.get("name", file_id)
                try:
                    file_name = base64.b64decode(encoded_name).decode('utf-8')
                except Exception as e:
                    logger.error(f"Ошибка декодирования имени файла {encoded_name}: {e}")
                    file_name = file_id
                full_obj_id = file_data.get("obj_id", file_id)
                file_category = get_file_category(file_data.get("mime_type", "unknown"))

                # Обновляем file_urls с полным obj_id
                if "file_urls" not in context.user_data:
                    context.user_data["file_urls"] = {}
                context.user_data["file_urls"][full_obj_id] = {
                    "full_obj_id": full_obj_id,
                    "url": download_url,
                    "name": file_name,
                    "category": file_category
                }
            else:
                response.raise_for_status()

        if not download_url:
            await update.message.reply_text(f"URL для файла с ID `{file_id}` не найден. Попробуйте загрузить файл снова или используйте 'Получить все файлы'.")
            return ConversationHandler.END

        file_response = requests.get(download_url, stream=True)
        file_response.raise_for_status()
        encrypted_data = b"".join(file_response.iter_content(chunk_size=8192))
        decrypted_data = decrypt_file(encrypted_data, session["k_enc"], session["k_mac"])
        if not decrypted_data:
            raise Exception("Ошибка расшифровки")

        os.makedirs("downloads", exist_ok=True)
        safe_file_name = file_name.encode('ascii', 'ignore').decode('ascii')
        file_path = os.path.join("downloads", safe_file_name)
        with open(file_path, "wb") as f:
            f.write(decrypted_data)
        with open(file_path, "rb") as f:
            await update.message.reply_document(document=f, filename=file_name)
        os.remove(file_path)
        await update.message.reply_text(f"Файл `{file_name}` (ID: `{full_obj_id}`) успешно скачан.")

    except requests.exceptions.HTTPError as e:
        try:
            response = e.response
            error_msg = response.json().get("error", "Ошибка") if response and 'json' in response.headers.get('Content-Type', '') else str(e)
            logger.error(f"Get file error: {e}, {response.text if response else 'No response'}")
        except AttributeError:
            error_msg = str(e)
            logger.error(f"Get file error: {e}, No response available")
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Get file error: {e}")
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END


async def get_file_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Получение отменено.")
    return ConversationHandler.END

def get_file_extension(mime_type):
    mime_to_ext = {
        'image/jpeg': '.jpg',
        'image/png': '.png',
        'image/gif': '.gif',
        'video/mp4': '.mp4',
        'video/avi': '.avi',
        'video/mkv': '.mkv',
        'application/pdf': '.pdf',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document': '.docx',
        'text/plain': '.txt'
    }
    return mime_to_ext.get(mime_type, '.unknown')

# Скачивание через inline-кнопки
async def handle_download(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()

    _, idx, file_category = query.data.split("_")
    idx = int(idx)

    file_data = context.user_data.get("file_list", [])
    if idx < 0 or idx >= len(file_data):
        await query.message.reply_text("Файл не найден. Попробуйте снова.")
        return

    file = file_data[idx]
    obj_id = file.get("obj_id", "не указан")
    file_name = context.user_data["file_urls"].get(obj_id, {}).get("name", obj_id.split("/")[-1])
    download_url = file.get("url")
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await query.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return

    try:
        downloads_dir = "downloads"
        try:
            os.makedirs(downloads_dir, exist_ok=True)
            logger.debug(f"Directory {downloads_dir} created or already exists")
        except OSError as e:
            logger.error(f"Failed to create directory {downloads_dir}: {e}")
            await query.message.reply_text(f"Ошибка: не удалось создать директорию для сохранения файла ({e}).")
            return

        file_response = requests.get(download_url, stream=True)
        file_response.raise_for_status()
        encrypted_data = b"".join(file_response.iter_content(chunk_size=8192))
        decrypted_data = decrypt_file(encrypted_data, session["k_enc"], session["k_mac"])
        if not decrypted_data:
            raise Exception("Ошибка расшифровки")

        file_path = os.path.join(downloads_dir, file_name.encode('ascii', 'ignore').decode('ascii'))
        with open(file_path, "wb") as f:
            f.write(decrypted_data)
        with open(file_path, "rb") as f:
            await query.message.reply_document(document=f, filename=file_name)
        os.remove(file_path)

    except requests.exceptions.HTTPError as e:
        error_msg = file_response.json().get("error", "Ошибка") if 'json' in file_response.headers.get('Content-Type', '') else str(e)
        logger.error(f"Get file error: {e}, {file_response.text}")
        await query.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Get file error: {e}")
        await query.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")


# Удаление файлов (одного или нескольких)
async def delete_many_files_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла для удаления (полный ID, например: '2/3ac6335b-bd4f-4bcd-a07f-b2c26a316644'). Если не уверены в ID, используйте '📂 Получить все файлы' для проверки.")
    return FILE_ID


async def delete_many_files_ids(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_ids_input = update.message.text.strip()
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена. Убедитесь, что client_id, k_enc и k_mac доступны.")
        return ConversationHandler.END

    # Разделяем введенные ID
    file_ids = [fid.strip() for fid in file_ids_input.replace(',', ' ').split() if fid.strip()]
    if not file_ids:
        await update.message.reply_text("Не введено ни одного ID. Попробуйте снова.")
        return ConversationHandler.END

    # Проверка формата ID (полный ID с bucket/ и расширением)
    uuid_pattern = re.compile(r'^\d+/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\..+$')
    invalid_ids = [fid for fid in file_ids if not uuid_pattern.match(fid)]
    if invalid_ids:
        await update.message.reply_text(f"Некорректный формат ID: {', '.join(invalid_ids)}. Пример: '3/d66d9210-869f-4f3f-bf2b-4ccf53ee2390.jpg'")
        return ConversationHandler.END

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}", "Content-Type": "application/json"}

        # Определяем type и готовим данные для удаления
        for file_id in file_ids:
            # Определяем type на основе расширения
            extension = file_id.split('.')[-1].lower()
            file_category = "unknown"
            if extension in ['jpg', 'jpeg', 'png', 'gif']:
                file_category = "photo"
            elif extension in ['mp4', 'avi', 'mkv']:
                file_category = "video"
            elif extension in ['txt', 'doc', 'docx', 'pdf']:
                file_category = "text"
            logger.info(f"File ID {file_id}: initial category={file_category} based on extension")

            # Отправляем полный ID с расширением
            params = {"id": file_id, "type": file_category}
            logger.info(f"Deleting file: DELETE /files/one with params {params}")
            response = requests.delete(GET_FILE_URL, headers=headers, params=params)
            response.raise_for_status()
            await update.message.reply_text(f"Файл с ID {file_id} успешно удалён!")

    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Delete files error: {e}, {response.text}")
        if response.status_code == 404:
            if "bucket name cannot be empty" in error_msg.lower():
                await update.message.reply_text(
                    f"Ошибка: бакет для файла с ID {file_ids[0]} не найден на сервере. "
                    f"Проверьте, существует ли бакет, через '📂 Получить все файлы'."
                )
            else:
                await update.message.reply_text(
                    f"Файл с ID {file_ids[0]} не найден: {error_msg}. "
                    f"Возможно, он был удалён ранее или ID содержит ошибку. Проверьте список файлов через '📂 Получить все файлы'."
                )
        elif response.status_code == 400:
            await update.message.reply_text(f"Некорректный запрос: {error_msg}")
        else:
            await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Delete files error: {e}")
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END

async def delete_many_files_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Удаление отменено.")
    return ConversationHandler.END


# Получение всех файлов
async def get_all_files_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите категорию (photo, unknown, video, text):")
    return FILE_CATEGORY


async def get_all_files_category(update: Update, context: ContextTypes.DEFAULT_TYPE):
    # Кнопки для выбора категории
    category_buttons = [
        [KeyboardButton('📸 Фото'), KeyboardButton('📹 Видео')],
        [KeyboardButton('📝 Текст'), KeyboardButton('📁 Прочее')]
    ]
    reply_markup = ReplyKeyboardMarkup(category_buttons, resize_keyboard=True, one_time_keyboard=True)
    await update.message.reply_text("Выберите категорию:", reply_markup=reply_markup)
    return FILE_CATEGORY


async def handle_category_selection(update: Update, context: ContextTypes.DEFAULT_TYPE):
    selected_category = update.message.text
    category_map = {
        '📸 Фото': 'photo',
        '📹 Видео': 'video',
        '📝 Текст': 'text',
        '📁 Прочее': 'unknown'
    }
    file_category = category_map.get(selected_category)

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}
        params = {"type": file_category}
        response = requests.get(GET_ALL_FILES_URL, headers=headers, params=params)
        response.raise_for_status()
        response_data = response.json()
        logger.info(f"Files for {file_category}: {response_data}")

        # Внутри try-блока после получения file_data
        file_data = response_data.get("file_data")
        if not file_data or not isinstance(file_data, list):
            await update.message.reply_text(f"Файлов в категории {selected_category} нет.",
                                            reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
        else:
            # Инициализируем file_urls, если его нет
            if "file_urls" not in context.user_data:
                context.user_data["file_urls"] = {}

            message = f"📂 *Файлы в категории {selected_category}:*\n"
            context.user_data["file_list"] = file_data
            for idx, file in enumerate(file_data, 1):
                # Декодируем имя файла из Base64
                try:
                    encoded_name = file.get("name", "Без имени")
                    name = base64.b64decode(encoded_name).decode('utf-8')
                except Exception as e:
                    logger.error(f"Ошибка декодирования имени файла {encoded_name}: {e}")
                    name = "ошибка декодирования"

                obj_id = file.get("obj_id", "не указан")
                download_url = file.get("url", None)
                created_at_raw = file.get("created_at", "не указан")
                mime_type = file.get("mime_type", "unknown")
                # Удаляем лишние точки из obj_id
                clean_obj_id = obj_id.rstrip('.')
                extension = get_file_extension(mime_type)
                obj_id_with_ext = f"{clean_obj_id}{extension}"  # Добавляем расширение к obj_id
                if created_at_raw != "не указан":
                    created_at_dt = datetime.strptime(created_at_raw, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=pytz.UTC)
                    created_at = created_at_dt.astimezone(pytz.timezone("Europe/Moscow")).strftime("%d.%m.%Y %H:%M")
                else:
                    created_at = created_at_raw
                message += f"📄 {name} | ID: <code>{obj_id_with_ext}</code> | {created_at}\n"  # Показываем ID с расширением

                # Сохраняем информацию в file_urls с полным obj_id
                if obj_id_with_ext != "не указан" and download_url:
                    context.user_data["file_urls"][obj_id_with_ext] = {
                        "full_obj_id": obj_id_with_ext,
                        "url": download_url,
                        "name": name,
                        "category": file_category
                    }

            await update.message.reply_text(message, parse_mode="HTML",
                                            reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Get all files error: {e}, {response.text}")
        await update.message.reply_text(f"Ошибка: {error_msg}",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except Exception as e:
        logger.error(f"Get all files error: {e}")
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    return ConversationHandler.END

async def get_all_files_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("", reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    return


# Проверка использования диска
async def usage_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    client_id = session.get("client_id")
    if not client_id:
        await update.message.reply_text("ID не найден. Войдите заново.")
        return ConversationHandler.END

    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}
        response = requests.get(f"{CLOUD_BASE_URL}/user/{client_id}/usage", headers=headers)
        response.raise_for_status()
        response_data = response.json()
        logger.info(f"Usage for {client_id}: {response_data}")
        message = f"📊 *Использование для {client_id}:*\n" \
                  f"• Использовано: {response_data['current_used_gb']} GB\n" \
                  f"• План: {response_data['plan_name']}\n" \
                  f"• Лимит: {response_data['storage_limit_gb']} GB"
        await update.message.reply_text(message, parse_mode="Markdown")
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        logger.error(f"Usage error: {e}, {response.text}")
        if response.status_code == 401:
            await update.message.reply_text("Ошибка авторизации. Войдите заново.")
        elif response.status_code == 404:
            await update.message.reply_text(f"Пользователь {client_id} не найден.")
        else:
            await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        logger.error(f"Usage error: {e}")
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END


async def usage_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Проверка отменена.")
    return ConversationHandler.END


# Обработка текстовых сообщений
async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    txt = update.message.text.lower()
    actions = {
        "зарегистрироваться": register_start, "📝": register_start,
        "вход": login_start, "войти": login_start, "🔑": login_start,
        "помощь": help_command, "ℹ️": help_command,
        "выйти": logout, "🚪": logout,
        "загрузить файл": upload_file_start, "📤": upload_file_start,
        "получить файл": get_file_start, "📥": get_file_start,
        "удалить файлы": delete_many_files_start, "🗑️": delete_many_files_start,  # Обновлено
        "получить все файлы": get_all_files_start, "📂": get_all_files_start,
        "проверить тариф": usage_start, "📊": usage_start
    }
    await actions.get(txt.split()[0], lambda u, c: u.message.reply_text("Выберите действие ниже."))(update, context)


def main():
    init_db()
    app = ApplicationBuilder().token(TOKEN).build()

    # ConversationHandlers
    register_handler = ConversationHandler(
        entry_points=[CommandHandler("register", register_start),
                      MessageHandler(filters.Regex('^(📝 Зарегистрироваться)$'), register_start)],
        states={EMAIL: [MessageHandler(filters.TEXT & ~filters.COMMAND, register_email)],
                PASSWORD: [MessageHandler(filters.TEXT & ~filters.COMMAND, register_password)]},
        fallbacks=[CommandHandler("cancel", register_cancel)]
    )
    login_handler = ConversationHandler(
        entry_points=[CommandHandler("login", login_start), MessageHandler(filters.Regex('^(🔑 Войти)$'), login_start)],
        states={EMAIL: [MessageHandler(filters.TEXT & ~filters.COMMAND, login_email)],
                PASSWORD: [MessageHandler(filters.TEXT & ~filters.COMMAND, login_password)]},
        fallbacks=[CommandHandler("cancel", login_cancel)]
    )
    get_file_handler = ConversationHandler(
        entry_points=[MessageHandler(filters.Regex('^(📥 Получить файл)$'), get_file_start)],
        states={FILE_ID: [MessageHandler(filters.TEXT & ~filters.COMMAND, get_file_id)]},
        fallbacks=[CommandHandler("cancel", get_file_cancel)]
    )
    delete_many_files_handler = ConversationHandler(  # Универсальный обработчик
        entry_points=[CommandHandler("deletemany", delete_many_files_start),
                      MessageHandler(filters.Regex('^(🗑️ Удалить файлы)$'), delete_many_files_start)],
        states={FILE_ID: [MessageHandler(filters.TEXT & ~filters.COMMAND, delete_many_files_ids)]},
        fallbacks=[CommandHandler("cancel", delete_many_files_cancel)]
    )
    get_all_files_handler = ConversationHandler(
        entry_points=[MessageHandler(filters.Regex('^(📂 Получить все файлы)$'), get_all_files_category)],
        states={FILE_CATEGORY: [
            MessageHandler(filters.Regex('^(📸 Фото|📹 Видео|📝 Текст|📁 Прочее)$'), handle_category_selection)]},
        fallbacks=[]
    )
    usage_handler = ConversationHandler(
        entry_points=[CommandHandler("usage", usage_start),
                      MessageHandler(filters.Regex('^(📊 Проверить тариф)$'), usage_start)],
        states={},
        fallbacks=[CommandHandler("cancel", usage_cancel)]
    )

    # Обработчики
    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("help", help_command))
    app.add_handler(CommandHandler("logout", logout))
    app.add_handler(register_handler)
    app.add_handler(login_handler)
    app.add_handler(get_file_handler)
    app.add_handler(delete_many_files_handler)
    app.add_handler(get_all_files_handler)
    app.add_handler(usage_handler)
    app.add_handler(MessageHandler(filters.Document.ALL | filters.PHOTO | filters.VIDEO, handle_file))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))
    app.add_handler(CallbackQueryHandler(handle_download, pattern="^download_"))
    app.add_error_handler(error_handler)

    print("Бот запущен...")
    app.run_polling()


if __name__ == '__main__':
    main()