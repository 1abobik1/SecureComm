from telegram.ext import ApplicationBuilder, CommandHandler, MessageHandler, filters, ConversationHandler, CallbackQueryHandler
from config import TOKEN, EMAIL, PASSWORD, FILE_ID, FILE_CATEGORY
from db import init_db
from handlers import start, help_command, logout, register_start, register_email, register_password, register_cancel, \
    login_start, login_email, login_password, login_cancel, get_file_start, get_file_id, get_file_cancel, \
    delete_many_files_start, delete_many_files_ids, delete_many_files_cancel, get_all_files_category, \
    handle_category_selection, usage_start, usage_cancel, handle_file, echo, handle_download, error_handler

# Запускает бота
def main():
    init_db()
    app = ApplicationBuilder().token(TOKEN).build()
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
    delete_many_files_handler = ConversationHandler(
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
    app.run_polling()

if __name__ == '__main__':
    main()