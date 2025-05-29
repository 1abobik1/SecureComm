import base64
import sqlite3

import jwt
import os
import requests
from engineio.async_drivers import aiohttp
from telegram import Update, ReplyKeyboardMarkup, KeyboardButton
from telegram.ext import ContextTypes, ConversationHandler
from telegram.error import TimedOut
import asyncio
from tests.client_http import encrypt_file, decrypt_file, perform_finalize, perform_handshake, derive_keys
from datetime import datetime
import pytz
import re
import logging
from config import MAIN_MENU_BUTTONS, FILE_MENU_BUTTONS, EMAIL, PASSWORD, FILE_ID, FILE_CATEGORY, \
    SIGNUP_URL, LOGIN_URL, LOGOUT_URL, HANDSHAKE_INIT_URL, HANDSHAKE_FINALIZE_URL, \
    UPLOAD_FILES_URL, GET_FILE_URL, GET_ALL_FILES_URL, CLOUD_BASE_URL
from db import get_session, save_session
from utils import get_file_category, get_file_extension

# Настройка логирования
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s', level=logging.INFO)
logger = logging.getLogger(__name__)

# Обрабатывает ошибки бота
async def error_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if isinstance(context.error, TimedOut) and update and update.message:
        await update.message.reply_text("Таймаут. Попробуйте снова через несколько секунд.")
    elif update and update.message:
        await update.message.reply_text("Произошла ошибка. Попробуйте позже.")

# Отображает начальное сообщение и меню
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Привет! Выберите действие ниже.",
                                    reply_markup=ReplyKeyboardMarkup(MAIN_MENU_BUTTONS, resize_keyboard=True))

# Отображает справочное сообщение с командами
async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    text = "🟢 *Команды:*\n"
    text += "• 📝 Зарегистрироваться – начать регистрацию\n" \
            "• 🔑 Войти – войти в аккаунт\n" \
            "• ℹ️ Помощь – это сообщение\n"
    if session and context.user_data.get("access_token"):
        text += "• 📤 Загрузить файл – загрузить файл\n" \
                "• 📥 Получить файл – скачать файл по полному ID\n" \
                "• 🗑️ Удалить файлы – удалить один или несколько файлов по полному ID\n" \
                "• 📂 Получить все файлы – список файлов по категории\n" \
                "• 📊 Проверить использование – данные о диске\n" \
                "• 🚪 Выйти – выйти из аккаунта\n"
    await update.message.reply_text(text, reply_markup=ReplyKeyboardMarkup(
        FILE_MENU_BUTTONS if session else MAIN_MENU_BUTTONS, resize_keyboard=True), parse_mode="Markdown")

# Начинает процесс регистрации
async def register_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите email:")
    return EMAIL

# Обрабатывает ввод email при регистрации
async def register_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD

# Завершает регистрацию с паролем
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
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception:
        await update.message.reply_text("Ошибка. Попробуйте снова.")
    return ConversationHandler.END

# Отменяет регистрацию
async def register_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Регистрация отменена.")
    return ConversationHandler.END

# Начинает процесс входа
async def login_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Введите email:")
    return EMAIL

# Обрабатывает ввод email при входе
async def login_email(update: Update, context: ContextTypes.DEFAULT_TYPE):
    context.user_data["email"] = update.message.text
    await update.message.reply_text("Введите пароль:")
    return PASSWORD

# Завершает вход с паролем
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
    except requests.exceptions.ConnectionError:
        await update.message.reply_text("Ошибка подключения к серверу. Проверьте, запущен ли сервер, и попробуйте снова.")
        return EMAIL
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        if "not found" in error_msg.lower() or "invalid" in error_msg.lower() or "incorrect" in error_msg.lower():
            await update.message.reply_text("Неверный email или пароль. Введите email заново:")
            return EMAIL
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception:
        await update.message.reply_text("Ошибка. Попробуйте снова.")
        return EMAIL
    return ConversationHandler.END

# Отменяет вход
async def login_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Вход отменён.")
    return ConversationHandler.END

# Выполняет выход из аккаунта
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
        await start(update, context)
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception:
        await update.message.reply_text("Ошибка. Попробуйте снова.")

# Начинает процесс загрузки файла
async def upload_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Отправьте файл (до 50 МБ).")
    return ConversationHandler.END

# Обрабатывает загрузку файла
async def handle_file(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return
    try:
        if not update.message.document and not update.message.photo and not update.message.video:
            await update.message.reply_text("Отправьте файл, фото или видео.")
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
        if file_size > 20 * 1024 * 1024:
            await update.message.reply_text(f"Файл {original_file_name} > 20 МБ. Используйте сайт.")
            return
        file = await (document.get_file() if update.message.document else
                      photo.get_file() if update.message.photo else
                      video.get_file())
        os.makedirs("uploads", exist_ok=True)
        safe_file_name = original_file_name
        file_path = os.path.join("uploads", safe_file_name)
        for attempt in range(3):
            try:
                await asyncio.wait_for(file.download_to_drive(file_path), timeout=30)
                break
            except asyncio.TimeoutError:
                if attempt == 2:
                    await update.message.reply_text(f"Таймаут для {safe_file_name}. Попробуйте снова.")
                    return
                continue
        loop = asyncio.get_event_loop()
        encrypted_data = await loop.run_in_executor(None, encrypt_file, file_path, session["k_enc"], session["k_mac"])
        if not encrypted_data:
            raise Exception("Ошибка шифрования")
        file_category = get_file_category(mime_type)
        access_token = context.user_data["access_token"]
        encoded_file_name = base64.b64encode(safe_file_name.encode('utf-8')).decode('ascii')
        headers = {
            "Authorization": f"Bearer {access_token}",
            "Content-Type": "application/octet-stream",
            "X-Orig-Filename": encoded_file_name,
            "X-Orig-Mime": mime_type,
            "X-File-Category": file_category
        }
        async with aiohttp.ClientSession() as session:
            async with session.post(UPLOAD_FILES_URL, headers=headers, data=encrypted_data) as response:
                response.raise_for_status()
                response_data = await response.json()
        obj_id = response_data.get("obj_id", "не указан")
        clean_obj_id = obj_id.rstrip('.')
        expected_extension = get_file_extension(mime_type).lstrip('.')
        obj_id_with_ext = f"{clean_obj_id}.{expected_extension}" if not clean_obj_id.lower().endswith(f".{expected_extension}") else clean_obj_id
        download_url = response_data.get("url")
        created_at_raw = response_data.get("created_at", "не указан")
        created_at = datetime.strptime(created_at_raw, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=pytz.UTC).astimezone(pytz.timezone("Europe/Moscow")).strftime("%d.%m.%Y %H:%M:%S") if created_at_raw != "не указан" else created_at_raw
        category_display = {
            "photo": "📸 Фото",
            "video": "📹 Видео",
            "text": "📝 Текст",
            "unknown": "📁 Прочее"
        }.get(file_category, file_category)
        if "file_urls" not in context.user_data:
            context.user_data["file_urls"] = {}
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
            f"• ID: <code>{obj_id_with_ext}</code>\n"
            f"• Тип: <code>{mime_type}</code>\n"
            f"• Категория: <code>{category_display}</code>\n"
            f"• Создан: <code>{created_at}</code>"
        )
        os.remove(file_path)
        await update.message.reply_text(message, parse_mode="HTML")
    except aiohttp.ClientResponseError as e:
        error_msg = (await e.response.json()).get("error", "Ошибка") if e.response else "Ошибка"
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")

# Начинает процесс получения файла
async def get_file_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла:")
    return FILE_ID

# Обрабатывает ввод ID файла для скачивания
async def get_file_id(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_id = update.message.text.strip()
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return ConversationHandler.END
    try:
        file_info = context.user_data.get("file_urls", {}).get(file_id, {})
        download_url = file_info.get("url")
        file_name = file_info.get("name")
        full_obj_id = file_info.get("full_obj_id", file_id)
        file_category = file_info.get("category")
        if not download_url or not full_obj_id or not file_category:
            access_token = context.user_data["access_token"]
            headers = {"Authorization": f"Bearer {access_token}"}
            params = {"id": file_id, "type": file_category or "unknown"}
            async with aiohttp.ClientSession() as session:
                async with session.get(GET_FILE_URL, headers=headers, params=params) as response:
                    if response.status == 200:
                        file_data = await response.json()
                        download_url = file_data.get("url")
                        encoded_name = file_data.get("name", file_id)
                        try:
                            file_name = base64.b64decode(encoded_name).decode('utf-8')
                        except Exception:
                            file_name = file_id
                        full_obj_id = file_data.get("obj_id", file_id)
                        file_category = get_file_category(file_data.get("mime_type", "unknown"))
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
        async with aiohttp.ClientSession() as session:
            async with session.get(download_url) as response:
                response.raise_for_status()
                encrypted_data = await response.read()
        loop = asyncio.get_event_loop()
        decrypted_data = await loop.run_in_executor(None, decrypt_file, encrypted_data, session["k_enc"], session["k_mac"])
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
    except aiohttp.ClientResponseError as e:
        error_msg = (await e.response.json()).get("error", "Ошибка") if e.response else str(e)
        await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END

# Отменяет получение файла
async def get_file_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Получение отменено.")
    return ConversationHandler.END

# Обрабатывает скачивание файла через inline-кнопки
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
        os.makedirs("downloads", exist_ok=True)
        file_response = requests.get(download_url, stream=True)
        file_response.raise_for_status()
        encrypted_data = b"".join(file_response.iter_content(chunk_size=8192))
        decrypted_data = decrypt_file(encrypted_data, session["k_enc"], session["k_mac"])
        if not decrypted_data:
            raise Exception("Ошибка расшифровки")
        file_path = os.path.join("downloads", file_name.encode('ascii', 'ignore').decode('ascii'))
        with open(file_path, "wb") as f:
            f.write(decrypted_data)
        with open(file_path, "rb") as f:
            await query.message.reply_document(document=f, filename=file_name)
        os.remove(file_path)
    except requests.exceptions.HTTPError as e:
        error_msg = file_response.json().get("error", "Ошибка") if 'json' in file_response.headers.get('Content-Type', '') else str(e)
        await query.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        await query.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")

# Начинает процесс удаления файлов
async def delete_many_files_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите ID файла для удаления (полный ID, например: '2/3ac6335b-bd4f-4bcd-a07f-b2c26a316644').")
    return FILE_ID

# Обрабатывает удаление одного или нескольких файлов
async def delete_many_files_ids(update: Update, context: ContextTypes.DEFAULT_TYPE):
    file_ids_input = update.message.text.strip()
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token") or "k_enc" not in session or "k_mac" not in session:
        await update.message.reply_text("Войдите в аккаунт или сессия повреждена.")
        return ConversationHandler.END
    file_ids = [fid.strip() for fid in file_ids_input.replace(',', ' ').split() if fid.strip()]
    if not file_ids:
        await update.message.reply_text("Не введено ни одного ID. Попробуйте снова.")
        return ConversationHandler.END
    uuid_pattern = re.compile(r'^\d+/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\..+$')
    invalid_ids = [fid for fid in file_ids if not uuid_pattern.match(fid)]
    if invalid_ids:
        await update.message.reply_text(f"Некорректный формат ID: {', '.join(invalid_ids)}. Пример: '3/d66d9210-869f-4f3f-bf2b-4ccf53ee2390.jpg'")
        return ConversationHandler.END
    try:
        access_token = context.user_data["access_token"]
        headers = {"Authorization": f"Bearer {access_token}", "Content-Type": "application/json"}
        for file_id in file_ids:
            extension = file_id.split('.')[-1].lower()
            file_category = "unknown"
            if extension in ['jpg', 'jpeg', 'png', 'gif']:
                file_category = "photo"
            elif extension in ['mp4', 'avi', 'mkv']:
                file_category = "video"
            elif extension in ['txt', 'doc', 'docx', 'pdf']:
                file_category = "text"
            params = {"id": file_id, "type": file_category}
            response = requests.delete(GET_FILE_URL, headers=headers, params=params)
            response.raise_for_status()
            await update.message.reply_text(f"Файл с ID {file_id} успешно удалён!")
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        if response.status_code == 404:
            if "bucket name cannot be empty" in error_msg.lower():
                await update.message.reply_text(
                    f"Ошибка: бакет для файла с ID {file_ids[0]} не найден на сервере.")
            else:
                await update.message.reply_text(
                    f"Файл с ID {file_ids[0]} не найден: {error_msg}.")
        elif response.status_code == 400:
            await update.message.reply_text(f"Некорректный запрос: {error_msg}")
        else:
            await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END

# Отменяет удаление файлов
async def delete_many_files_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Удаление отменено.")
    return ConversationHandler.END

# Начинает процесс получения всех файлов
async def get_all_files_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    session = get_session(update.effective_user.id)
    if not session or not context.user_data.get("access_token"):
        await update.message.reply_text("Войдите в аккаунт.")
        return ConversationHandler.END
    await update.message.reply_text("Введите категорию (photo, unknown, video, text):")
    return FILE_CATEGORY

# Отображает кнопки для выбора категории файлов
async def get_all_files_category(update: Update, context: ContextTypes.DEFAULT_TYPE):
    category_buttons = [
        [KeyboardButton('📸 Фото'), KeyboardButton('📹 Видео')],
        [KeyboardButton('📝 Текст'), KeyboardButton('📁 ปрочее')]
    ]
    reply_markup = ReplyKeyboardMarkup(category_buttons, resize_keyboard=True, one_time_keyboard=True)
    await update.message.reply_text("Выберите категорию:", reply_markup=reply_markup)
    return FILE_CATEGORY

# Обрабатывает выбор категории файлов
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
        file_data = response_data.get("file_data")
        if not file_data or not isinstance(file_data, list):
            await update.message.reply_text(f"Файлов в категории {selected_category} нет.",
                                            reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
        else:
            if "file_urls" not in context.user_data:
                context.user_data["file_urls"] = {}
            message = f"📂 *Файлы в категории {selected_category}:*\n"
            context.user_data["file_list"] = file_data
            for idx, file in enumerate(file_data, 1):
                encoded_name = file.get("name", "Без имени")
                try:
                    if len(encoded_name) % 4 == 0 and re.match(r'^[A-Za-z0-9+/=]+$', encoded_name):
                        name = base64.b64decode(encoded_name).decode('utf-8')
                    else:
                        name = encoded_name
                except Exception as e:
                    logger.error(f"Ошибка декодирования имени файла: {encoded_name}, ошибка: {e}")
                    name = encoded_name if encoded_name != "Без имени" else "неизвестный_файл"
                obj_id = file.get("obj_id", "не указан")
                download_url = file.get("url", None)
                created_at_raw = file.get("created_at", "не указан")
                mime_type = file.get("mime_type", "unknown")
                clean_obj_id = obj_id.rstrip('.')
                current_extension = obj_id.split('.')[-1].lower() if '.' in obj_id else ''
                expected_extension = get_file_extension(mime_type).lstrip('.')
                if current_extension == expected_extension:
                    obj_id_with_ext = obj_id
                else:
                    obj_id_with_ext = f"{clean_obj_id}.{expected_extension}" if not clean_obj_id.lower().endswith(
                        f".{expected_extension}") else clean_obj_id
                if created_at_raw != "не указан":
                    created_at_dt = datetime.strptime(created_at_raw, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=pytz.UTC)
                    created_at = created_at_dt.astimezone(pytz.timezone("Europe/Moscow")).strftime("%d.%m.%Y %H:%M")
                else:
                    created_at = created_at_raw
                message += f"📄 {name} | ID: <code>{obj_id_with_ext}</code> | {created_at}\n"
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
        await update.message.reply_text(f"Ошибка: {error_msg}",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    except Exception as e:
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.",
                                        reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    return ConversationHandler.END

# Отменяет получение всех файлов
async def get_all_files_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("", reply_markup=ReplyKeyboardMarkup(FILE_MENU_BUTTONS, resize_keyboard=True))
    return

# Проверяет использование диска
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
        message = f"📊 *Использование для {client_id}:*\n" \
                  f"• Использовано: {response_data['current_used_gb']} GB\n" \
                  f"• План: {response_data['plan_name']}\n" \
                  f"• Лимит: {response_data['storage_limit_gb']} GB"
        await update.message.reply_text(message, parse_mode="Markdown")
    except requests.exceptions.HTTPError as e:
        error_msg = response.json().get("error", "Ошибка")
        if response.status_code == 401:
            await update.message.reply_text("Ошибка авторизации. Войдите заново.")
        elif response.status_code == 404:
            await update.message.reply_text(f"Пользователь {client_id} не найден.")
        else:
            await update.message.reply_text(f"Ошибка: {error_msg}")
    except Exception as e:
        await update.message.reply_text(f"Ошибка: {e}. Попробуйте снова.")
    return ConversationHandler.END

# Отменяет проверку использования
async def usage_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    await update.message.reply_text("Проверка отменена.")
    return ConversationHandler.END

# Обрабатывает текстовые сообщения
async def echo(update: Update, context: ContextTypes.DEFAULT_TYPE):
    txt = update.message.text.lower()
    actions = {
        "зарегистрироваться": register_start, "📝": register_start,
        "вход": login_start, "войти": login_start, "🔑": login_start,
        "помощь": help_command, "ℹ️": help_command,
        "выйти": logout, "🚪": logout,
        "загрузить файл": upload_file_start, "📤": upload_file_start,
        "получить файл": get_file_start, "📥": get_file_start,
        "удалить файлы": delete_many_files_start, "🗑️": delete_many_files_start,
        "получить все файлы": get_all_files_start, "📂": get_all_files_start,
        "проверить тариф": usage_start, "📊": usage_start
    }
    await actions.get(txt.split()[0], lambda u, c: u.message.reply_text("Выберите действие ниже."))(update, context)