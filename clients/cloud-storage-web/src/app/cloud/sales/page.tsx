import React from 'react';

export default function SalesPage() {
  return (
    <div className="max-w-3xl mx-auto p-6 space-y-6">
      <h1 className="text-3xl font-bold">Облачное хранилище от BeerLoveTeam</h1>

      <section>
        <h2 className="text-xl font-semibold">📦 Условия покупки</h2>
        <p className="mt-2 text-gray-700">
          Вы приобретаете доступ к личному облачному хранилищу. После оплаты вы получите ссылку для входа и уникальные учётные данные.
          Хранилище работает 24/7, поддерживает загрузку, скачивание и шифрование файлов.
        </p>
        <p className="mt-1 text-gray-700">
          Доступ предоставляется в течение 5–10 минут после подтверждения перевода.
        </p>
      </section>

      <section>
        <h2 className="text-xl font-semibold">💰 Тарифы</h2>
        <ul className="mt-2 list-disc list-inside text-gray-800">
          <li>10 ГБ — 300₽ / месяц</li>
          <li>50 ГБ — 1000₽ / месяц</li>
          <li>100 ГБ — 1800₽ / месяц</li>
        </ul>
      </section>

      <section>
        <h2 className="text-xl font-semibold">💸 Оплата</h2>
        <p className="mt-2 text-gray-700">
          Перевод на криптокошелёк:
        </p>
        <div className="mt-1 p-3 bg-gray-100 rounded font-mono break-all">
          TON: `UQBD6AuLuV7FOXcTQBmzLwMesIBHBbN7WOWeDZvUfIhGvOP2`
        </div>
        <p className="text-sm text-gray-500 mt-1">* Укажите выбранный тариф в комментарии к переводу.</p>
      </section>

      <section>
        <h2 className="text-xl font-semibold">📨 Контакты</h2>
        <p className="mt-2 text-gray-700">
          По всем вопросам и подтверждению оплаты — Telegram: <a href="https://t.me/sorrysk" className="text-blue-600 underline">@sorrysk</a>
        </p>
      </section>
    </div>
  );
}
