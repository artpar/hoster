import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface PlanLimits {
  max_deployments: number;
  max_cpu_cores: number;
  max_memory_mb: number;
  max_disk_gb: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  plan_id: string;
  plan_limits: PlanLimits;
}

interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (token: string, password: string) => Promise<void>;
  clearError: () => void;
  clearAuth: () => void;
}

function authHeaders(token: string | null): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) {
    headers['X-Auth-Token'] = token;
  }
  return headers;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      checkAuth: async () => {
        const { token } = get();
        if (!token) {
          set({ isAuthenticated: false, isLoading: false, user: null });
          return;
        }
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/api/auth/me', {
            headers: authHeaders(token),
          });

          if (response.ok) {
            const user = await response.json();
            set({
              user,
              isAuthenticated: true,
              isLoading: false,
            });
          } else {
            set({
              token: null,
              user: null,
              isAuthenticated: false,
              isLoading: false,
            });
          }
        } catch {
          set({
            token: null,
            user: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      login: async (email, password) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Login failed');
          }

          const data = await response.json();
          set({
            token: data.token,
            user: data.user,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (err) {
          set({
            isLoading: false,
            error: err instanceof Error ? err.message : 'Login failed',
          });
          throw err;
        }
      },

      signup: async (email, password, name) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/api/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, name }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Signup failed');
          }

          const data = await response.json();
          set({
            token: data.token,
            user: data.user,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (err) {
          set({
            isLoading: false,
            error: err instanceof Error ? err.message : 'Signup failed',
          });
          throw err;
        }
      },

      logout: async () => {
        const { token } = get();
        try {
          await fetch('/api/auth/logout', {
            method: 'POST',
            headers: authHeaders(token),
          });
        } finally {
          set({
            token: null,
            user: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      forgotPassword: async (email) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/api/auth/forgot', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Failed to send reset email');
          }

          set({ isLoading: false });
        } catch (err) {
          set({
            isLoading: false,
            error: err instanceof Error ? err.message : 'Failed to send reset email',
          });
          throw err;
        }
      },

      resetPassword: async (resetToken, password) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/api/auth/reset', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token: resetToken, password }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error?.message || 'Password reset failed');
          }

          set({ isLoading: false });
        } catch (err) {
          set({
            isLoading: false,
            error: err instanceof Error ? err.message : 'Password reset failed',
          });
          throw err;
        }
      },

      clearError: () => set({ error: null }),

      clearAuth: () =>
        set({
          token: null,
          user: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'hoster-auth',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// Selector hooks for convenience
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useUser = () => useAuthStore((state) => state.user);
export const useAuthLoading = () => useAuthStore((state) => state.isLoading);
export const useAuthError = () => useAuthStore((state) => state.error);
