import { create } from "zustand";

export interface RequestLogEntry {
  id: string;
  method: string;
  url: string;
  status: number | null;
  ok: boolean;
  durationMs: number;
  startedAt: number;
  requestBody?: unknown;
  responseBody?: unknown;
  errorMessage?: string;
}

const MAX_ENTRIES = 300;

interface RequestLogState {
  entries: RequestLogEntry[];
  paused: boolean;
  add: (entry: RequestLogEntry) => void;
  clear: () => void;
  togglePaused: () => void;
}

export const useRequestLogStore = create<RequestLogState>()((set, get) => ({
  entries: [],
  paused: false,
  add: (entry) => {
    if (get().paused) return;
    set((state) => ({ entries: [entry, ...state.entries].slice(0, MAX_ENTRIES) }));
  },
  clear: () => set({ entries: [] }),
  togglePaused: () => set((state) => ({ paused: !state.paused })),
}));

function newId(): string {
  return typeof crypto.randomUUID === "function"
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function logRequest(entry: Omit<RequestLogEntry, "id">) {
  useRequestLogStore.getState().add({ ...entry, id: newId() });
}
