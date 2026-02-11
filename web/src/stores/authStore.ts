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
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
  clearAuth: () => void;
}

// APIGate v0.3.2+ handles auth via JWT tokens at /mod/auth/* endpoints.
// Login/register return flat JSON: { token, user, success }
// /mod/auth/me returns: { user: { id, email, name, plan_id, ... } }

function parseUser(userData: Record<string, unknown>): User {
  const limits = userData.plan_limits as Record<string, unknown> | undefined;
  return {
    id: String(userData.id ?? ''),
    email: String(userData.email ?? ''),
    name: String(userData.name ?? userData.email ?? ''),
    plan_id: String(userData.plan_id ?? 'free'),
    plan_limits: {
      max_deployments: Number(limits?.max_deployments ?? 1),
      max_cpu_cores: Number(limits?.max_cpu_cores ?? 1),
      max_memory_mb: Number(limits?.max_memory_mb ?? 1024),
      max_disk_gb: Number(limits?.max_disk_gb ?? 5),
    },
  };
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      checkAuth: async () => {
        try {
          set({ isLoading: true, error: null });
          const { token } = get();
          if (!token) {
            set({ user: null, isAuthenticated: false, isLoading: false });
            return;
          }
          const response = await fetch('/mod/auth/me', {
            headers: { 'Authorization': `Bearer ${token}` },
          });

          if (response.ok) {
            const data = await response.json();
            const user = parseUser(data.user as Record<string, unknown>);
            set({
              user,
              isAuthenticated: true,
              isLoading: false,
            });
          } else {
            set({
              user: null,
              token: null,
              isAuthenticated: false,
              isLoading: false,
            });
          }
        } catch {
          set({
            user: null,
            token: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      login: async (email, password) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/mod/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
          });

          if (!response.ok) {
            const errorData = await response.json();
            const detail = errorData.error || errorData.errors?.[0]?.detail || 'Login failed';
            throw new Error(detail);
          }

          const data = await response.json();
          const token = data.token;
          if (!token) {
            throw new Error('No token in login response');
          }
          set({ token });

          // Fetch user profile with the new token
          await get().checkAuth();
        } catch (err) {
          set({
            isLoading: false,
            error: err instanceof Error ? err.message : 'Login failed',
          });
          throw err;
        }
      },

      signup: async (email, password) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/mod/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, name: email.split('@')[0] }),
          });

          if (!response.ok) {
            const errorData = await response.json();
            const detail = errorData.error || errorData.errors?.[0]?.detail || 'Signup failed';
            throw new Error(detail);
          }

          // Register returns token directly — no need to auto-login
          const data = await response.json();
          const token = data.token;
          if (!token) {
            throw new Error('No token in register response');
          }
          set({ token });
          await get().checkAuth();
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
          if (token) {
            await fetch('/mod/auth/logout', {
              method: 'POST',
              headers: { 'Authorization': `Bearer ${token}` },
            });
          }
        } finally {
          set({
            user: null,
            token: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      clearError: () => set({ error: null }),

      clearAuth: () =>
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'hoster-auth',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
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
