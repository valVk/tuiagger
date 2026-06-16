import React, { createContext, useContext } from 'react';
import type { HttpClient } from '../types/services.js';
import { FetchHttpClient } from '../types/services.js';

interface Services {
  httpClient: HttpClient;
}

const defaultServices: Services = {
  httpClient: new FetchHttpClient(),
};

const ServicesContext = createContext<Services>(defaultServices);

export function ServicesProvider({ children, services = defaultServices }: { children: React.ReactNode; services?: Services }) {
  return (
    <ServicesContext.Provider value={services}>
      {children}
    </ServicesContext.Provider>
  );
}

export function useServices(): Services {
  return useContext(ServicesContext);
}
