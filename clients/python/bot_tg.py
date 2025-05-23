import sqlite3
import logging
import base64
from telegram import Update, ReplyKeyboardMarkup, KeyboardButton
from telegram.ext import ApplicationBuilder, CommandHandler, MessageHandler, ContextTypes, ConversationHandler, filters
import requests
from client_http import perform_handshake, perform_finalize, derive_keys
from cryptography.hazmat.primitives import serialization

# Настройка логирования
logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s', level=logging.INFO)
logger = logging.getLogger(__name__)

# Токен бота
TOKEN = "7745265874:AAFKjBo36eDqq08hN30QTqK0v27HDnMlDL0"

# URL API хранилища
AUTH_BASE_URL = "http://localhost:8081"  # Для Auth API (signup, login, logout)
SECURECOMM_BASE_URL = "http://localhost:8080"  # Для SecureComm API (handshake, session)

SIGNUP_URL = f"{AUTH_BASE_URL}/user/signup"
LOGIN_URL = f"{AUTH_BASE_URL}/user/login"
LOGOUT_URL = f"{AUTH_BASE_URL}/user/logout"
HANDSHAKE_INIT_URL = f"{SECURECOMM_BASE_URL}/handshake/init"
HANDSHAKE_FINALIZE_URL = f"{SECURECOMM_BASE_URL}/handshake/finalize"

# Кнопки главного меню
MAIN_MENU_BUTTONS = [
    [KeyboardButton('📝 Зарегистрироваться'), KeyboardButton('🔑 Войти')],
    [KeyboardButton('ℹ️ Помощь'), KeyboardButton('🚪 Выйти')],
]

# Состояния для ConversationHandler
EMAIL, PASSWORD = range(2)

# Инициализация базы данных SQLite
def init_db():
    conn = sqlite3.connect("sessions.db")
    cursor = conn.cursor()
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS sessions (
            telegram_id INTEGER PRIMARY KEY,
            client_id TEXT,
            ks_client TEXT,              -- ks в Base64 от сервера или handshake
            ecdsa_priv_client TEXT       -- ecdsa_priv в Base64 от сервера или handshake
        )
    ''')
    conn.commit()
    conn.close()

# Сохранение сессии в SQLite
def save_session(telegram_id, client_id, ks_client=None, ecdsa_priv_client=None):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute('''
            INSERT OR REPLACE INTO sessions (telegram_id, client_id, ks_client, ecdsa_priv_client)
            VALUES (?, ?, ?, ?)
        ''', (telegram_id, client_id, ks_client, ecdsa_priv_client))
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
            "SELECT client_id, ks_client, ecdsa_priv_client FROM sessions WHERE telegram_id = ?",
            (telegram_id,))
        result = cursor.fetchone()
        if result:
            return {
                "client_id": result[0],
                "ks_client": result[1],
                "ecdsa_priv_client": result[2]
            }
        return None
    except sqlite3.Error as e:
        logger.error(f"Ошибка при получении сессии: {e}")
        return None
    finally:
        conn.close()

# Приветствие с главным меню
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = "Привет! Выберите нужную команду на панели ниже или напишите сообщение."
    reply_markup = ReplyKeyboardMarkup(MAIN_MENU_BUTTONS, resize_keyboard=True)
    await update.message.reply_text(text, reply_markup=reply_markup)

# Команда /help
async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = "🟢 *Меню команд:*\n" \
           "• 📝 Зарегистрироваться – начать регистрацию\n" \
           "• 🔑 Войти – войти в аккаунт\n" \
           "• ℹ️ Помощь – это сообщение\n" \
           "• 🚪 Выйти – выйти из аккаунта\n"
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

        # Выполняем handshake
        handshake_data = perform_handshake(HANDSHAKE_INIT_URL, access_token)
        ks = perform_finalize(HANDSHAKE_FINALIZE_URL, handshake_data, access_token)
        client_id = handshake_data["client_id"]
        ecdsa_priv = handshake_data["ecdsa_priv"]

        # Сохраняем ключи в Base64
        ks_client = base64.b64encode(ks).decode('utf-8')
        ecdsa_priv_bytes = ecdsa_priv.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption()
        )
        ecdsa_priv_client = base64.b64encode(ecdsa_priv_bytes).decode('utf-8')

        # Сохраняем сессию
        save_session(update.effective_user.id, client_id, ks_client, ecdsa_priv_client)

        await update.message.reply_text("Регистрация успешна! Теперь вы можете работать с файлами.")
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
        ks_client = data.get("ks_client")
        ecdsa_priv_client = data.get("ecdsa_priv_client")
        if not access_token or not refresh_token or not ks_client or not ecdsa_priv_client:
            raise ValueError("Сервер не вернул все необходимые данные")
        context.user_data["access_token"] = access_token
        context.user_data["refresh_token"] = refresh_token

        # Проверяем сессию
        session = get_session(update.effective_user.id)
        if session:
            client_id = session["client_id"]
            ks_client_existing = session["ks_client"]
            ecdsa_priv_client_existing = session["ecdsa_priv_client"]
            if ks_client != ks_client_existing or ecdsa_priv_client != ecdsa_priv_client_existing:
                logger.warning("Данные от сервера отличаются от сохранённых. Обновляем сессию.")
                save_session(update.effective_user.id, client_id, ks_client, ecdsa_priv_client)
            else:
                logger.info(f"Сессия для telegram_id {update.effective_user.id} актуальна.")
        else:
            # Выполняем handshake для получения client_id
            handshake_data = perform_handshake(HANDSHAKE_INIT_URL, access_token)
            client_id = handshake_data["client_id"]
            save_session(update.effective_user.id, client_id, ks_client, ecdsa_priv_client)

        ks = base64.b64decode(ks_client)
        ecdsa_priv_bytes = base64.b64decode(ecdsa_priv_client)
        ecdsa_priv = serialization.load_pem_private_key(ecdsa_priv_bytes, password=None)
        k_enc, k_mac = derive_keys(ks)

        await update.message.reply_text("Вход успешен! Теперь вы можете работать с файлами.")
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
    else:
        await update.message.reply_text("Я вас понял! Для выбора действия используйте панели ниже или команды.")

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

    # Добавляем обработчики
    app.add_handler(CommandHandler("start", start))
    app.add_handler(CommandHandler("help", help_command))
    app.add_handler(CommandHandler("logout", logout))
    app.add_handler(register_handler)
    app.add_handler(login_handler)
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, echo))

    print("Бот запущен...")
    app.run_polling()

if __name__ == '__main__':
    main()