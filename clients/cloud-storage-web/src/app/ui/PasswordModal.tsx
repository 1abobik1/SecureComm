'use client';

import React, {forwardRef, useContext, useImperativeHandle, useState} from 'react';
import {X} from 'lucide-react';
import {observer} from 'mobx-react-lite';
import {Context} from '@/app/_app';

export interface PasswordModalRef {
    open: () => void;
    close: () => void;
}

interface PasswordModalProps {
    onSubmit?: (password: string) => Promise<boolean>;
    title?: string;
    description?: string;
}

// eslint-disable-next-line react/display-name
const PasswordModal = observer(forwardRef<PasswordModalRef, PasswordModalProps>(
    ({ onSubmit, title = 'Введите пароль', description }, ref) => {
        const { store } = useContext(Context);
        const [isVisible, setIsVisible] = useState(false);
        const [password, setPassword] = useState('');
        const [error, setError] = useState<string | null>(null);
        const [isLoading, setIsLoading] = useState(false);

        useImperativeHandle(ref, () => ({
            open: () => {
                setPassword('');
                setError(null);
                setIsVisible(true);
            },
            close: () => setIsVisible(false)
        }));

        const handleSubmit = async (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            setIsLoading(true);

            try {
                let success = false;

                if (onSubmit) {
                    success = await onSubmit(password);
                } else {
                    success = await store.decryptAndStoreKey(password);
                }

                if (success) {
                    setIsVisible(false);
                } else {
                    setError("Неверный пароль");
                }
            } catch (err) {
                console.error("Decryption error:", err);
                setError("Ошибка при расшифровке ключа");
            } finally {
                setIsLoading(false);
            }
        };

        const handleClose = () => {
            setIsVisible(false);
        };

        if (!isVisible) return null;

        return (
            <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
                <div className="relative bg-white p-6 rounded shadow-md max-w-sm w-full">
                    <button
                        onClick={handleClose}
                        className="absolute top-2 right-2 text-gray-500 hover:text-gray-800"
                        disabled={isLoading}
                    >
                        <X size={20} />
                    </button>

                    <h2 className="text-lg font-semibold mb-2">{title}</h2>
                    {description && <p className="text-sm text-gray-600 mb-4">{description}</p>}

                    <form onSubmit={handleSubmit}>
                        <input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full p-2 border rounded mb-3"
                            placeholder="Пароль"
                            autoFocus
                            disabled={isLoading}
                            required
                        />

                        {error && <p className="text-red-500 text-sm mb-2">{error}</p>}

                        <button
                            type="submit"
                            className="w-full bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded disabled:opacity-50"
                            disabled={isLoading}
                        >
                            {isLoading ? 'Проверка...' : 'Подтвердить'}
                        </button>
                    </form>
                </div>
            </div>
        );
    }
));

PasswordModal.displayName = 'PasswordModal';
export default PasswordModal;