'use client';
import {useEffect, useState} from "react";
import {FileData} from "@/app/api/models/FileData";
import CloudService from "../api/services/CloudServices";
import FileCard from "@/app/ui/FileCard";
import TypeFileIcon from "../ui/TypeFileIcon";
import FileUploader from "./FileUploader";
import {Loader2} from 'lucide-react';

export default function TypeBlock({ type }) {
  const [file, setFile] = useState<FileData[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isError, setIsError] = useState<boolean>(false);
  const [timeSort,setTimeSort] = useState<boolean>(false)
  const [nameSortAsc, setNameSortAsc] = useState<boolean>(true);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc'); // сортировка по возрастанию или убыванию
  const [filteredFiles, setFilteredFiles] = useState<FileData[]>([]);


  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await CloudService.getAllCloud(type);
        console.log(response.data)
        const fileData = response.data.file_data;

        if (Array.isArray(fileData)) {
          const files: FileData[] = fileData.map((file: any) => ({
            obj_id: String(file.obj_id),
            name: String(file.name),
            url: String(file.url),
            created_at: String(file.created_at),
            mime_type: String(file.mime_type)
          }));
          setFile(files);
          setFilteredFiles(files);

        } else {
          setFile([]);
        }
      } catch (error) {
        console.error("Ошибка при получении данных:", error);
        setIsError(true);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [type]);

  // Функция сортировки по дате
  const sortFiles = (order: 'asc' | 'desc') => {
    const sortedFiles = [...filteredFiles].sort((a, b) => {
      const dateA = new Date(a.created_at).getTime();
      const dateB = new Date(b.created_at).getTime();
      return order === 'asc' ? dateA - dateB : dateB - dateA;
    });
    setFilteredFiles(sortedFiles);
  };

  // Обработчик изменения сортировки
  const handleSortChange = (order: 'asc' | 'desc') => {
    setSortOrder(order);
    sortFiles(order); // Пересортировываем файлы
  };

  const handleNameSortChange = () => {
    const sorted = [...filteredFiles].sort((a, b) => {
      return nameSortAsc
        ? a.name.localeCompare(b.name)
        : b.name.localeCompare(a.name);
    });
    setFilteredFiles(sorted);
    setNameSortAsc(!nameSortAsc);
  };



  if (isLoading) return(<div className=" inset-0 bg-white/70 backdrop-blur-sm z-10 flex items-center justify-center">
  <div className="flex flex-col items-center">
    <Loader2 className="w-10 h-10 text-blue-500 animate-spin mb-2" />
    <span className="text-gray-700 text-sm">Загрузка файлов...</span>
  </div>
</div>)

  if (isError) return <p>Произошла ошибка при загрузке данных.</p>;

  const handleDelete = (id: string) => {
    setFilteredFiles(prevFiles => prevFiles.filter(file => file.obj_id !== id)); // Убираем удаленный файл из состояния
  };

  return (
    <>



      {filteredFiles.length === 0 ? (
        <div className="p-10 text-center text-gray-600">
          <div className="text-5xl mb-4">📂</div>
          <p className="text-lg">Нет файлов.</p>
          <div className="p-4">
            <FileUploader/>
          </div>

        </div>
      ) : (
        <div className="sm:p-6 p-2 mx-auto bg-white rounded shadow ">
          <h2 className="text-xl font-jetbrains  mb-4 flex items-center gap-2">
            <TypeFileIcon type={type}/><div className="text-blue-500">{type}</div>
          </h2>

          <div className="flex flex-row justify-between">
            <div className="mb-4">
              <button
                onClick={handleNameSortChange}
                className="bg-blue-500 hover:bg-blue-600 text-white py-2 px-4 rounded flex items-center"
              >
                Названию&nbsp;
                {nameSortAsc ? <span className="ml-1">▲</span> : <span className="ml-1">▼</span>}
              </button>
            </div>

            <div className="mb-4 mr-28">
              <button
                onClick={() => {
                  handleSortChange(timeSort ? 'desc' : 'asc');
                  setTimeSort(!timeSort);
                }}
                className="bg-blue-500 hover:bg-blue-600 text-white py-2 px-4 rounded flex items-center"
              >
                Дате&nbsp;
                {timeSort ? <span className="ml-2">▲</span> : <span className="ml-2">▼</span>}
              </button>
            </div>
          </div>

          {filteredFiles.map((item) => (
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
    </>
  );




};
