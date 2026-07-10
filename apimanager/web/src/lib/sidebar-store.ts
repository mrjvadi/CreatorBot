import { create } from "zustand";
import { persist } from "zustand/middleware";

interface SidebarState {
  collapsed: boolean;
  toggle: () => void;
}

export const useSidebarStore = create<SidebarState>()(
  persist((set, get) => ({ collapsed: false, toggle: () => set({ collapsed: !get().collapsed }) }), {
    name: "apimanager-sidebar",
  })
);
