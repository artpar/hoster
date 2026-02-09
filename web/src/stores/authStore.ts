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
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  checkAuth: () => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
  clearAuth: () => void;
}

// APIGate handles auth via cookies — no token management needed.
// Login/signup are handled by redirecting to /portal (APIGate's auth UI).

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      checkAuth: async () => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/auth/me', {
            credentials: 'include',
          });

          if (response.ok) {
            const data = await response.json();
            // APIGate returns user data — map to our User type
            const user: User = {
              id: data.id || data.user_id || '',
              email: data.email || '',
              name: data.name || data.email || '',
              plan_id: data.plan_id || 'free',
              plan_limits: data.plan_limits || {
                max_deployments: 1,
                max_cpu_cores: 1,
                max_memory_mb: 1024,
                max_disk_gb: 5,
              },
            };
            set({
              user,
              isAuthenticated: true,
              isLoading: false,
            });
          } else {
            set({
              user: null,
              isAuthenticated: false,
              isLoading: false,
            });
          }
        } catch {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      logout: async () => {
        try {
          await fetch('/auth/logout', {
            method: 'POST',
            credentials: 'include',
          });
        } finally {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
          });
        }
      },

      clearError: () => set({ error: null }),

      clearAuth: () =>
        set({
          user: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'hoster-auth',
      partialize: (state) => ({
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
