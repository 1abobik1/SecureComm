// components/DiskUsageChart.tsx
import {Pie} from "react-chartjs-2";
import {ArcElement, CategoryScale, Chart as ChartJS, Legend, Tooltip} from 'chart.js';
import {useEffect, useState} from 'react';
import {jwtDecode} from 'jwt-decode';
import {useUsageRefresh} from "@/app/components/UsageRefreshContext";

ChartJS.register(ArcElement, CategoryScale, Tooltip, Legend);

interface UsageResponse {
  current_used_gb: number;
  current_used_mb: number;
  current_used_kb: number;
  storage_limit_gb: number;
  plan_name: string;
}

interface JwtPayload {
  user_id: number;
}

interface DiskUsageChartProps {
  fileCounts: Record<string, number>;
}

// Функция для форматирования размера
const formatSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} КБ`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} МБ`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} ГБ`;
};

// Функция для конвертации всех единиц в байты
const toBytes = (gb: number, mb: number, kb: number): number => {
  return gb * 1024 * 1024 * 1024 + mb * 1024 * 1024 + kb * 1024;
};

const DiskUsageChart: React.FC<DiskUsageChartProps> = ({ fileCounts }) => {
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const { refreshKey } = useUsageRefresh();

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    const decoded: JwtPayload = jwtDecode(token);
    const userId = decoded.user_id;

    fetch(`http://localhost:8080/user/${userId}/usage`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
        .then(res => {
          if (!res.ok) throw new Error(`Ошибка: ${res.status}`);
          return res.json();
        })
        .then(data => {
          setUsage(data);
        })
        .catch(err => {
          console.error('Ошибка загрузки использования:', err);
        })
        .finally(() => {
          setLoading(false);
        });
  }, [refreshKey]);

  if (loading || !usage) return <div className="text-gray-600">Загрузка...</div>;

  const { current_used_gb, current_used_mb, current_used_kb, storage_limit_gb } = usage;

  // Конвертируем все в байты для точных расчетов
  const usedBytes = toBytes(current_used_gb, current_used_mb, current_used_kb);
  const totalBytes = storage_limit_gb * 1024 * 1024 * 1024;
  const freeBytes = totalBytes - usedBytes;

  // Форматируем значения для отображения
  const formattedUsed = formatSize(usedBytes);
  const formattedFree = formatSize(freeBytes);
  const formattedTotal = formatSize(totalBytes);

  const chartData = {
    labels: ['Занято', 'Свободно'],
    datasets: [
      {
        data: [usedBytes, freeBytes],
        backgroundColor: ['#4c6ef5', '#a0aec0'],
        borderColor: '#fff',
        borderWidth: 1,
      }
    ]
  };

  const options = {
    responsive: true,
    plugins: {
      legend: {
        position: 'bottom' as const,
      },
      tooltip: {
        callbacks: {
          label: function(context: any) {
            const label = context.label || '';
            const value = context.raw || 0;
            return `${label}: ${formatSize(value)}`;
          }
        }
      }
    }
  };

  return (
      <div className="mt-6 p-4 bg-white rounded-lg shadow">
        <h3 className="text-lg font-semibold mb-4">🗂 Статистика использования</h3>
        <div className="flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="w-full md:w-1/2 h-64">
            <Pie data={chartData} options={options} />
          </div>
          <div className="w-full md:w-1/2 space-y-4">
            <div className="bg-gray-50 p-4 rounded-lg">
              <h4 className="font-medium text-gray-700 mb-2">Общее пространство</h4>
              <p className="text-2xl font-bold text-blue-600">{formattedTotal}</p>
            </div>

            <div className="bg-blue-50 p-4 rounded-lg">
              <h4 className="font-medium text-gray-700 mb-2">Использовано</h4>
              <p className="text-2xl font-bold text-blue-600">{formattedUsed}</p>
              <div className="mt-2">
                <div className="h-2 w-full bg-gray-200 rounded-full">
                  <div
                      className="h-2 bg-blue-600 rounded-full"
                      style={{ width: `${(usedBytes / totalBytes) * 100}%` }}
                  ></div>
                </div>
                <p className="text-sm text-gray-500 mt-1">
                  {((usedBytes / totalBytes) * 100).toFixed(1)}% от общего объема
                </p>
              </div>
            </div>

            <div className="bg-green-50 p-4 rounded-lg">
              <h4 className="font-medium text-gray-700 mb-2">Свободно</h4>
              <p className="text-2xl font-bold text-green-600">{formattedFree}</p>
            </div>
          </div>
        </div>
      </div>
  );
};

export default DiskUsageChart;