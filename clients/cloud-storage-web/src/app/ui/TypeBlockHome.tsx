'use client';

import React from 'react';
import TypeFileIcon from './TypeFileIcon';
import { Loader2 } from 'lucide-react';
import { useEffect, useState } from "react";
import { FileData } from "@/app/api/models/FileData";
import CloudService from "../api/services/CloudServices";


const TypeBlockHome = ({ type }: { type: string }) => {

const [filesByType, setFilesByType] = useState<Record<string, FileData[]>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isError, setIsError] = useState(false);
  const types = ['text', 'photo', 'video', 'unknown'];

useEffect(() => {
    const fetchAllTypes = async () => {
      try {
        const result: Record<string, FileData[]> = {};

        for (const type of types) {
          const response = await CloudService.getAllCloud(type);
          const fileData = response.data.file_data;

          if (Array.isArray(fileData)) {
            result[type] = fileData.map((file: any) => ({
              obj_id: String(file.obj_id),
              name: String(file.name),
              url: String(file.url),
              created_at: String(file.created_at),
              mime_type: String(file.mime_type)
            }));
          } else {
            result[type] = [];
          }
        }

        setFilesByType(result);
      } catch (error) {
        console.error("Ошибка при получении данных:", error);
        setIsError(true);
      } finally {
        setIsLoading(false);
      }
    };

    fetchAllTypes();
  }, []);

  if (isLoading) return (
    <div className="inset-0 bg-white/70 backdrop-blur-sm z-10 flex items-center justify-center">
      <div className="flex flex-col items-center">
        <Loader2 className="w-10 h-10 text-blue-500 animate-spin mb-2" />
        <span className="text-gray-700 text-sm">Загрузка файлов...</span>
      </div>
    </div>
  );

  if (isError) return <p>Произошла ошибка при загрузке данных.</p>;


  
    return (
        <>
        {filesByType[type]?.length === 0 ? (
          <div className="text-gray-500 text-center flex-1 flex flex-col items-center justify-center">
            <p className="text-xl">📂 Нет файлов</p>
          </div>
        ) : (
          <div className="space-y-2 overflow-auto flex-1">
            {filesByType[type].map((item) => (
              <div
                key={item.obj_id}
                className="text-xl border rounded p-2 flex justify-between items-center"
              ><a href={item.url} target="_blank" rel="noopener noreferrer" className="text-blue-500 text-sm">
                <span className="truncate max-w-[150px]">{item.name}</span>
                </a>
              </div>
            ))}
          </div>
        )}

        
      
       </>
    );
};

export default TypeBlockHome;