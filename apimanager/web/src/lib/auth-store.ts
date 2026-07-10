import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Role = "user" | "admin" | "owner" | string;

interface AuthUser {
  id: string;
  role: Role;
}

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  setSession: (p: { accessToken: string; refreshToken: string; user: AuthUser }) => void;
  setAccessToken: (token: string) => void;
  logout: () => void;
  isAdmin: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      setSession: ({ accessToken, refreshToken, user }) =>
        set({ accessToken, refreshToken, user }),
      setAccessToken: (accessToken) => set({ accessToken }),
      logout: () => set({ accessToken: null, refreshToken: null, user: null }),
      isAdmin: () => {
        const role = get().user?.role;
        return role === "admin" || role === "owner";
      },
    }),
    {
      name: "apimanager-auth",
      partialize: (state) => ({
        refreshToken: state.refreshToken,
        accessToken: state.accessToken,
        user: state.user,
      }),
    }
  )
);
