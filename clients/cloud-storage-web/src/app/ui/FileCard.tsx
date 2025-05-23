import React, {useContext, useRef, useState} from 'react';
import CloudService from '../api/services/CloudServices';
import {ArrowDownTrayIcon, TrashIcon} from '@heroicons/react/24/outline';
import ModalDelete from './ModalDelete';
import TypeFileIcon from './TypeFileIcon';
import {cryptoHelper} from '@/app/api/utils/CryptoHelper';
import PasswordModal, {PasswordModalRef} from '@/app/ui/PasswordModal';
import {Context} from '@/app/_app';

export type FileCardData = {
  name: string;
  created_at: string;
  obj_id: string;
  url: string;
  type: string;
  mime_type: string;
  onDelete: (obj_id: string) => void;
};

const FileCard: React.FC<FileCardData> = ({ obj_id, created_at, name, url, type, onDelete, mime_type }) => {
  const [isModalOpen, setIsModalOpen] = useState<boolean>(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [action, setAction] = useState<'view' | 'download' | null>(null);
  const passwordModalRef = useRef<PasswordModalRef>(null);
  const { store } = useContext(Context);

  const performDownload = async () => {
    try {
      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed to download file');

      const blob = await response.blob();
      const encryptedFile = new File([blob], name, { type: mime_type });
      const decryptedBlob = await cryptoHelper.decryptFile(encryptedFile);

      const downloadUrl = URL.createObjectURL(decryptedBlob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      a.download = name;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(downloadUrl);
      return true;
    } catch (error) {
      console.error('Decryption error:', error);
      throw error;
    }
  };

  const performView = async () => {
    try {
      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed preview');

      const blob = await response.blob();
      const encryptedFile = new File([blob], name, { type: mime_type });
      const decryptedBlob = await cryptoHelper.decryptFile(encryptedFile);
      const viewUrl = URL.createObjectURL(decryptedBlob);

      window.open(viewUrl, '_blank');

      setTimeout(() => {
        URL.revokeObjectURL(viewUrl);
      }, 60000);

      return true;
    } catch (error) {
      console.error('Decryption error:', error);
      throw error;
    }
  };

  const handleDownload = async () => {
    try {
      setDownloadError(null);
      await store.initializeKey();

      if (!store.hasCryptoKey) {
        setAction('download');
        passwordModalRef.current?.open();
        return;
      }

      await performDownload();
    } catch (error) {
      console.error('Download error:', error);
      setDownloadError('Ошибка при скачивании файла');
    }
  };

  const handleView = async () => {
    try {
      setDownloadError(null);
      await store.initializeKey();

      if (!store.hasCryptoKey) {
        setAction('view');
        passwordModalRef.current?.open();
        return;
      }

      await performView();
    } catch (error) {
      console.error('View error:', error);
      setDownloadError('Ошибка при просмотре файла');
    }
  };

  const handlePasswordSubmit = async (password: string) => {
    try {
      const success = await store.decryptAndStoreKey(password);
      if (!success) {
        setDownloadError('Неверный пароль');
        return false;
      }

      if (action === 'view') {
        await performView();
      } else if (action === 'download') {
        await performDownload();
      }

      passwordModalRef.current?.close();
      return true;
    } catch (error) {
      console.error('Password submit error:', error);
      setDownloadError(error.message || 'Ошибка при расшифровке файла');
      return false;
    }
  };

  const handleDelete = async () => {
    try {
      await CloudService.deleteFile(type, obj_id);
      setIsModalOpen(false);
      onDelete(obj_id);
    } catch (error) {
      console.error('Delete error:', error);
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const truncateText = (text: string, maxLength: number): string => {
    return text.length <= maxLength ? text : text.slice(0, maxLength) + '…';
  };

  return (
      <>
        <PasswordModal
            ref={passwordModalRef}
            onSubmit={handlePasswordSubmit}
            title="Для скачивания или просмотра введите пароль"
            description="Этот файл защищен шифрованием. Для доступа требуется ваш пароль."
        />
        <div className="p-4 bg-white border-t border-b border-gray-200 w-full">
          <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
            <div className="flex items-center gap-2 w-full min-w-0">
              <TypeFileIcon type={type} />
              <div className="break-words leading-snug">
                <button onClick={handleView} className="hidden sm:inline break-all leading-snug">
                  {name}
                </button>
                <span className="inline sm:hidden">{truncateText(name, 20)}</span>
              </div>
            </div>
            <div className="flex justify-between items-center sm:justify-end gap-4 w-full sm:w-auto">
              <div className="text text-gray-500">{formatDate(created_at)}</div>
              <div className="flex items-center gap-2">
                <button
                    onClick={handleDownload}
                    className="w-6 h-6 flex items-center justify-center text-blue-500 hover:text-blue-700"
                    aria-label="Скачать файл"
                >
                  <ArrowDownTrayIcon className="w-5 h-5" />
                </button>
                <button
                    onClick={() => setIsModalOpen(true)}
                    className="w-6 h-6 flex items-center justify-center text-red-500 hover:text-red-700"
                    aria-label="Удалить файл"
                >
                  <TrashIcon className="w-5 h-5" />
                </button>
              </div>
            </div>
          </div>
          {downloadError && (
              <div className="text-red-500 text-sm mt-2">{downloadError}</div>
          )}
          {isModalOpen && (
              <ModalDelete
                  message="Вы уверены, что хотите удалить этот файл?"
                  onConfirm={handleDelete}
                  onCancel={() => setIsModalOpen(false)}
              />
          )}
        </div>
      </>
  );
};

export default FileCard;