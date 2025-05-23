import React from 'react';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';

interface ModalProps {
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

const ModalDelete: React.FC<ModalProps> = ({ message, onConfirm, onCancel }) => {
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm z-50">
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md animate-fade-in">
        <div className="flex items-center mb-4">
          <ExclamationTriangleIcon className="h-6 w-6 text-yellow-500 mr-2" />
          <h2 className="text-lg font-semibold text-gray-800">Удаление файла</h2>
        </div>
        <p className="text-sm text-gray-700 mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 rounded bg-gray-200 text-gray-800 hover:bg-gray-300 transition"
          >
            Отмена
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 rounded bg-red-500 text-white hover:bg-red-600 transition"
          >
            Удалить
          </button>
        </div>
      </div>
    </div>
  );
};

export default ModalDelete;
