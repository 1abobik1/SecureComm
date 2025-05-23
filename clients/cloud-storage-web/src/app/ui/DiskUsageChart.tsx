// components/DiskUsageChart.tsx
import {Pie} from "react-chartjs-2";
import {ArcElement, CategoryScale, Chart as ChartJS, Legend, Tooltip} from 'chart.js';
import {useEffect, useState} from 'react';
import {jwtDecode} from 'jwt-decode';

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


const DiskUsageChart: React.FC<DiskUsageChartProps> = ({ fileCounts }) => {


  const [usage, setUsage] = useState<UsageResponse | null>(null);
    const[loading, setLoading] = useState(true);



  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    const decoded: JwtPayload = jwtDecode(token);
    const userId = decoded.user_id;

    fetch(`http://localhost:8081/user/${userId}/usage`, {
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
  }, []);
  if (loading || !usage) return <div className="text-gray-600">Загрузка...</div>;
  const {current_used_gb,current_used_mb,storage_limit_gb } = usage;
  const ostatok = storage_limit_gb - (current_used_gb + current_used_mb / 1024)

  const chartData = {
    labels: ['Свободно','Занято'],
    datasets: [
      {
        data: [
          fileCounts.text,
          fileCounts.photo,
          fileCounts.video,
          fileCounts.other,
          ostatok,

        ],
        backgroundColor: ['#a0aec0','#4c6ef5'], // Цвет для свободного места
        borderColor: '#fff',
        borderWidth: 1,
      }
    ]
  };

  return (
    <div className="mt-6">
      <h3 className="text-lg mb-4">🗂 Статистика использования</h3>
      <div className="flex items-center justify-between">
        <div className="w-1/2">
          <Pie data={chartData} options={{ responsive: true }} />
        </div>
        <div className="w-1/2 pl-6">
          <p className="text-xl">Свободно места: {ostatok.toFixed(3)} GB</p>
        </div>
      </div>
    </div>
  );
};

export default DiskUsageChart;
