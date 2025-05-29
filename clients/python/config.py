from telegram import KeyboardButton

# Токен бота и URL-адреса сервисов
TOKEN = "7745265874:AAFKjBo36eDqq08hN30QTqK0v27HDnMlDL0"
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
DELETE_FILES_URL = f"{CLOUD_BASE_URL}/files/many"
GET_ALL_FILES_URL = f"{CLOUD_BASE_URL}/files/all"

# Кнопки главного меню
MAIN_MENU_BUTTONS = [
    [KeyboardButton('📝 Зарегистрироваться'), KeyboardButton('🔑 Войти')],
    [KeyboardButton('ℹ️ Помощь')],
]

# Кнопки меню файлов
FILE_MENU_BUTTONS = [
    [KeyboardButton('📤 Загрузить файл'), KeyboardButton('📥 Получить файл')],
    [KeyboardButton('🗑️ Удалить файлы'), KeyboardButton('📂 Получить все файлы')],
    [KeyboardButton('📊 Проверить использование'), KeyboardButton('🚪 Выйти')],
]

# Состояния для ConversationHandler
EMAIL, PASSWORD = range(2)
FILE_ID = range(1)[0]
FILE_CATEGORY = range(1)[0]