import { create } from 'zustand';

interface WsState {
  connected: boolean;
  reconnecting: boolean;
  setConnected: (connected: boolean) => void;
  setReconnecting: (reconnecting: boolean) => void;
}

export const useWsStore = create<WsState>((set) => ({
  connected: false,
  reconnecting: false,
  setConnected: (connected) => set({ connected }),
  setReconnecting: (reconnecting) => set({ reconnecting }),
}));
