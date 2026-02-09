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

// APIGate handles auth backend via JWT tokens.
// Hoster provides its own branded login/signup pages that call APIGate's /auth/* endpoints.
// Login returns a JWT token; all subsequent requests use Authorization: Bearer.

function parseUserFromAuthMe(data: Record<string, unknown>): User {
  // /auth/me returns JSON:API format: { data: { type, id, attributes, relationships } }
  const jsonApiData = data.data as Record<string, unknown> | undefined;
  const attrs = (jsonApiData?.attributes || {}) as Record<string, unknown>;
  const relationships = (jsonApiData?.relationships || {}) as Record<string, unknown>;
  const planRel = (relationships.plan as Record<string, unknown>)?.data as Record<string, unknown> | undefined;

  return {
    id: (jsonApiData?.id as string) || '',
    email: (attrs.email as string) || '',
    name: (attrs.name as string) || (attrs.email as string) || '',
    plan_id: (planRel?.id as string) || 'free',
    plan_limits: (attrs.plan_limits as PlanLimits) || {
      max_deployments: 1,
      max_cpu_cores: 1,
      max_memory_mb: 1024,
      max_disk_gb: 5,
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
          const response = await fetch('/auth/me', {
            headers: { 'Authorization': `Bearer ${token}` },
          });

          if (response.ok) {
            const data = await response.json();
            const user = parseUserFromAuthMe(data);
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
          const response = await fetch('/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
          });

          if (!response.ok) {
            const errorData = await response.json();
            const detail = errorData.errors?.[0]?.detail || 'Login failed';
            throw new Error(detail);
          }

          const data = await response.json();
          const token = data.data?.attributes?.token;
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
          const response = await fetch('/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
          });

          if (!response.ok) {
            const errorData = await response.json();
            const detail = errorData.errors?.[0]?.detail || 'Signup failed';
            throw new Error(detail);
          }

          // Auto-login after signup to get JWT token
          await get().login(email, password);
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
            await fetch('/auth/logout', {
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
