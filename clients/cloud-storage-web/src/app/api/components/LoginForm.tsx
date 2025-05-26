'use client';
import React, {useContext, useState} from 'react';
import {Context} from '@/app/api/store/context';
import {observer} from "mobx-react-lite";
import {motion} from 'framer-motion';
import {FiLock, FiLogIn, FiMail, FiUserPlus} from 'react-icons/fi';

const LoginForm = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const store = useContext(Context);

  const handleLogin = async () => {
    setIsLoading(true);
    try {
      await store.login(email, password);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSignup = async () => {
    setIsLoading(true);
    try {
      await store.signup(email, password);
    } finally {
      setIsLoading(false);
    }
  };

  return (
      <div className="min-h-screen bg-gradient-to-br from-indigo-50 to-blue-100 flex items-center justify-center p-4">
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
            className="w-full max-w-md"
        >
          <div className="bg-white rounded-2xl shadow-xl overflow-hidden">
            <div className="bg-gradient-to-r from-indigo-600 to-blue-500 p-6 text-center">
              <h2 className="text-2xl font-bold text-white">Добро пожаловать</h2>
              <p className="text-indigo-100 mt-1">Введите свои данные для продолжения</p>
            </div>

            <div className="p-6 space-y-6">
              <div className="space-y-4">
                <div>
                  <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
                    Email
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <FiMail className="h-5 w-5 text-gray-400" />
                    </div>
                    <input
                        id="email"
                        name="email"
                        type="email"
                        autoComplete="email"
                        required
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-700"
                        placeholder="your@email.com"
                    />
                  </div>
                </div>

                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                    Пароль
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <FiLock className="h-5 w-5 text-gray-400" />
                    </div>
                    <input
                        id="password"
                        name="password"
                        type="password"
                        autoComplete="current-password"
                        required
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-gray-700"
                        placeholder="••••••••"
                    />
                  </div>
                </div>
              </div>

              <div className="space-y-4">
                <motion.button
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    type="button"
                    onClick={handleLogin}
                    disabled={isLoading}
                    className="w-full flex justify-center items-center px-4 py-3 bg-gradient-to-r from-indigo-600 to-blue-500 text-white font-medium rounded-lg shadow-md hover:shadow-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-70"
                >
                  {isLoading ? (
                      <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                  ) : (
                      <FiLogIn className="mr-2" />
                  )}
                  Войти
                </motion.button>

                <motion.button
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    type="button"
                    onClick={handleSignup}
                    disabled={isLoading}
                    className="w-full flex justify-center items-center px-4 py-3 bg-gradient-to-r from-green-600 to-emerald-500 text-white font-medium rounded-lg shadow-md hover:shadow-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-70"
                >
                  {isLoading ? (
                      <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                  ) : (
                      <FiUserPlus className="mr-2" />
                  )}
                  Регистрация
                </motion.button>
              </div>
            </div>
          </div>
        </motion.div>
      </div>
  );
};

export default observer(LoginForm);