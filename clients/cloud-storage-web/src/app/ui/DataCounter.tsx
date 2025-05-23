import {useEffect, useState} from 'react';
import {jwtDecode} from 'jwt-decode';
import {useUsageRefresh} from '@/app/components/UsageRefreshContext';

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

export default function DataCounter() {
  const [usage, setUsage] = useState<UsageResponse | null>(null);
  const [loading, setLoading] = useState(true);
const { refreshKey } = useUsageRefresh();


  const fetchUsage = async () => {
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
  } ;

    useEffect(() => {
    setLoading(true);
    fetchUsage();
  }, [refreshKey]);

  if (loading || !usage) return <div className="text-gray-600">Загрузка...</div>;

  const { current_used_gb, storage_limit_gb, plan_name,current_used_mb } = usage;

  const totalMbUsed = current_used_gb * 1024 + current_used_mb;
const totalMbLimit = storage_limit_gb * 1024;
const percentUsed = (totalMbUsed / totalMbLimit) * 100;




  return (
    <div className="p-2 max-w">
      <h3 className="text-lg font-jetbrains text-blue-600 mb-2">Тариф - {plan_name}:</h3>
      <div className="w-full h-10 bg-gray-200 rounded-lg flex items-center overflow-hidden">
        <div
          className={`h-full text-white  flex items-center transition-all duration-500 ${
            percentUsed > 80 ? 'bg-red-500' : 'bg-blue-500'
          }`}
          style={{ width: `${percentUsed}%` }}
        >

        </div>







      </div>
      <div className="text-sm font-semibold text-gray-800">
        Занято {current_used_gb}.{Math.round(current_used_mb)} ГБ из {usage.storage_limit_gb} ГБ
      </div>
    </div>
  );
}
