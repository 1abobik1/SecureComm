'use client'

import { createContext, useContext, useState } from 'react';

const UsageRefreshContext = createContext({
  refreshKey: 0,
  triggerRefresh: () => {},
});

export const UsageRefreshProvider = ({ children }: { children: React.ReactNode }) => {
  const [refreshKey, setRefreshKey] = useState(0);

  const triggerRefresh = () => setRefreshKey(k => k + 1);

  return (
    <UsageRefreshContext.Provider value={{ refreshKey, triggerRefresh }}>
      {children}
    </UsageRefreshContext.Provider>
  );
};

export const useUsageRefresh = () => useContext(UsageRefreshContext);
