'use client';
import React, {useContext, useState} from 'react';
import {Context} from '@/app/_app';
import {observer} from "mobx-react-lite";

const LoginForm = () => {
  const [platform, setPlatform] = useState('ios-mobile');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const { store } = useContext(Context);


    const [captchaVerified, setCaptchaVerified] = useState(false);

    const handleCaptchaChange = (value: string | null) => {
      if (value) {
        setCaptchaVerified(true);
      }
    };


  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-100">
      <div className="w-full max-w-md p-8 space-y-6 bg-white rounded-lg shadow-md">
        <h2 className="text-2xl font-bold text-center text-gray-900">Вход</h2>
        <form className="space-y-6">

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className=" text-black w-full px-3 py-2 mt-1 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            />
          </div>
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Пароль
            </label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className=" text-black w-full px-3 py-2 mt-1 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            />
          </div>
          <div>
            {/* <label htmlFor="platform" className="block text-sm font-medium text-gray-700">
              Платформа
            </label>
            <select
              id="platform"
              name="platform"
              required
              value={platform}
              onChange={(e) => setPlatform(e.target.value)}
              className="text-black w-full px-3 py-2 mt-1 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            >
              <option value="" disabled>
                Выберите платформу
              </option>
              <option value="ios-mobile">iOS</option>
              <option value="android-mobile">Android</option>
            </select> */}

        {/* Тут капча важно */}
        <div style={{ transform: "scale(0.9)", transformOrigin: "0 0", maxWidth: "100%" }}>
  {/*<ReCAPTCHA*/}
  {/*  sitekey="6LffSw4rAAAAAENeTm2aejDbLWa2QvbO8eOkjRlL"*/}
  {/*  onChange={handleCaptchaChange}*/}
  {/*/>*/}
</div>


          </div>

          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => store.login(email, password,platform)}
              className="w-full px-4 py-2 text-sm font-medium text-black bg-indigo-600 border border-transparent rounded-md shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Войти
            </button>
          </div>
          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => store.signup( email, password ,platform)}
              className="w-full px-4 py-2 text-sm font-medium text-black bg-green-600 border border-transparent rounded-md shadow-sm hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
            >
              Регистрация
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default observer(LoginForm);