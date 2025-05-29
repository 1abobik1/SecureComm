import sqlite3

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

# Сохраняет сессию пользователя в базе данных
def save_session(telegram_id, client_id, k_enc=None, k_mac=None):
    try:
        conn = sqlite3.connect("sessions.db")
        cursor = conn.cursor()
        cursor.execute('INSERT OR REPLACE INTO sessions (telegram_id, client_id, k_enc, k_mac) VALUES (?, ?, ?, ?)',
                       (telegram_id, client_id, k_enc, k_mac))
        conn.commit()
    finally:
        conn.close()

# Получает сессию пользователя из базы данных
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
    finally:
        conn.close()