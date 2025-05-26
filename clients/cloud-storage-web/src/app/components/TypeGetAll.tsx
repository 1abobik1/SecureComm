'use client';
import React, {useEffect, useState} from 'react';
import {FileData} from '@/app/api/models/FileData';
import CloudService from '../api/services/CloudServices';
import dynamic from 'next/dynamic';
import TypeFileIcon from '../ui/TypeFileIcon';
import FileCard from '../ui/FileCard';
import {Loader2} from 'lucide-react';


export default function TypeGetAll() {
    const [filesByType, setFilesByType] = useState<Record<string, FileData[]>>({});
    const [isLoading, setIsLoading] = useState(true);
    const [isError, setIsError] = useState(false);
    const types = ['text', 'photo', 'video', 'unknown'];

    const DiskUsageChart = dynamic(() => import('../ui/DiskUsageChart'), {
        ssr: false,
        loading: () => (
            <div className="flex justify-center items-center h-20">
                <div className="w-6 h-6 border-4 border-blue-500 border-t-transparent rounded-full animate-spin" />
            </div>
        ),
    });
    useEffect(() => {
        const fetchAllTypes = async () => {
            try {
                // Параллельные запросы для всех типов
                const responses = await Promise.all(
                    types.map((type) => CloudService.getAllCloud(type))
                );

                const result: Record<string, FileData[]> = {};
                responses.forEach((response, index) => {
                    const type = types[index];
                    const fileData = response.data.file_data;

                    if (Array.isArray(fileData)) {
                        result[type] = fileData.map((file) => {
                            let decodedName = file.name; // По умолчанию используем имя как есть
                            try {
                                // Проверяем, является ли строка валидной Base64
                                if (file.name && /^[A-Za-z0-9+/=]+$/.test(file.name)) {
                                    // Декодируем Base64 в строку байтов
                                    const binaryString = atob(file.name);
                                    // Преобразуем строку байтов в массив Uint8Array
                                    const bytes = new Uint8Array(binaryString.length);
                                    for (let i = 0; i < binaryString.length; i++) {
                                        bytes[i] = binaryString.charCodeAt(i);
                                    }
                                    // Декодируем байты как UTF-8
                                    decodedName = new TextDecoder('utf-8').decode(bytes);
                                }
                            } catch (error) {
                                console.warn(`Не удалось раскодировать имя файла: ${file.name}. Оставляем без изменений.`, error);
                            }
                            return {
                                obj_id: String(file.obj_id),
                                name: decodedName,
                                url: file.url,
                                created_at: String(file.created_at),
                                mime_type: String(file.mime_type),
                            };
                        });
                    } else {
                        result[type] = [];
                    }
                });

                setFilesByType(result);
                console.log("Обработанные файлы:", result);
            } catch (error) {
                console.error('Ошибка при получении данных:', error);
                setIsError(true);
            } finally {
                setIsLoading(false);
            }
        };

        fetchAllTypes();
    }, []);

    // Обработчик удаления файла
    const handleDelete = (obj_id: string) => {
        setFilesByType((prev) => {
            const updated = { ...prev };
            Object.keys(updated).forEach((type) => {
                updated[type] = updated[type].filter((file) => file.obj_id !== obj_id);
            });
            return updated;
        });
    };

    if (isLoading) {
        return (
            <div className="inset-0 bg-white/70 backdrop-blur-sm z-10 flex items-center justify-center">
                <div className="flex flex-col items-center">
                    <Loader2 className="w-10 h-10 text-blue-500 animate-spin mb-2" />
                    <span className="text-gray-700 text-sm">Загрузка файлов...</span>
                </div>
            </div>
        );
    }

    if (isError) {
        return <p className="text-red-500">Произошла ошибка при загрузке данных.</p>;
    }

    // Рассчитываем количество файлов по типам
    const fileCounts = types.reduce((acc, type) => {
        acc[type] = filesByType[type]?.length || 0;
        return acc;
    }, {} as Record<string, number>);


    return (
        <div className="grid grid-cols-1 sm:grid-cols-1 lg:grid-cols-1 xl:grid-cols-2 gap-6">
            {types.map((type) => (
                <div
                    key={type}
                    className="bg-white border rounded-xl shadow-md p-4 flex flex-col justify-between w-full h-64"
                >
                    <div className="flex items-center gap-2 mb-3">
                        <TypeFileIcon type={type} />
                        <h2 className="text-lg font-jetbrains text-blue-600 capitalize">{type}</h2>
                    </div>

                    {filesByType[type]?.length === 0 ? (
                        <div className="text-gray-500 text-center flex-1 flex flex-col items-center justify-center">
                            <p className="text-xl">📂 Нет файлов</p>
                        </div>
                    ) : (
                        <div className="space-y-2 overflow-auto flex-1">
                            {filesByType[type].map((item) => (
                                <FileCard
                                    key={item.obj_id}
                                    obj_id={item.obj_id}
                                    name={item.name}
                                    url={item.url}
                                    created_at={item.created_at}
                                    type={type}
                                    mime_type={item.mime_type}
                                    onDelete={handleDelete}
                                />
                            ))}
                        </div>
                    )}
                </div>
            ))}

            <div className="mt-10 p-4">
                <h2 className="text-xl font-semibold mb-3">📁 Все файлы</h2>
                <div className="flex flex-wrap gap-3 overflow-x-auto">
                    {Object.values(filesByType).flat().map((file) => (
                        <button
                            key={file.obj_id}
                            onClick={() => {
                                // Вызываем handleView из FileCard (нужно передать данные в FileCard)
                                // Временное решение: отображаем как ссылку без href
                            }}
                            className="px-3 py-2 bg-gray-100 rounded-lg border shadow text-sm hover:bg-blue-50 transition whitespace-nowrap"
                        >
                            {file.name}
                        </button>
                    ))}
                </div>
            </div>

            <DiskUsageChart fileCounts={fileCounts} />
        </div>
    );
}