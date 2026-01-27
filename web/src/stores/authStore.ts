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

  // Derived state for convenience
  userId: string | null;
  planId: string | null;
  planLimits: PlanLimits | null;

  // Actions
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (token: string, password: string) => Promise<void>;
  clearError: () => void;
  setAuth: (userId: string, planId: string, limits: PlanLimits) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,
      error: null,

      // Derived state
      userId: null,
      planId: null,
      planLimits: null,

      checkAuth: async () => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/auth/me', {
            credentials: 'include',
          });

          if (response.ok) {
            const user = await response.json();
            set({
              user,
              isAuthenticated: true,
              isLoading: false,
              userId: user.id,
              planId: user.plan_id,
              planLimits: user.plan_limits,
            });
          } else {
            set({
              user: null,
              isAuthenticated: false,
              isLoading: false,
              userId: null,
              planId: null,
              planLimits: null,
            });
          }
        } catch {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            userId: null,
            planId: null,
            planLimits: null,
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
            credentials: 'include',
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Login failed');
          }

          await get().checkAuth();
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
          const response = await fetch('/auth/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, name }),
            credentials: 'include',
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Signup failed');
          }

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
            userId: null,
            planId: null,
            planLimits: null,
          });
        }
      },

      forgotPassword: async (email) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/auth/forgot', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Failed to send reset email');
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

      resetPassword: async (token, password) => {
        try {
          set({ isLoading: true, error: null });
          const response = await fetch('/auth/reset', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token, password }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Password reset failed');
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

      // Legacy setAuth for backward compatibility
      setAuth: (userId, planId, planLimits) =>
        set({
          userId,
          planId,
          planLimits,
          isAuthenticated: true,
          user: {
            id: userId,
            email: '',
            name: '',
            plan_id: planId,
            plan_limits: planLimits,
          },
        }),

      clearAuth: () =>
        set({
          user: null,
          userId: null,
          planId: null,
          planLimits: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'hoster-auth',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        userId: state.userId,
        planId: state.planId,
        planLimits: state.planLimits,
      }),
    }
  )
);

// Selector hooks for convenience
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useUserId = () => useAuthStore((state) => state.userId);
export const usePlanLimits = () => useAuthStore((state) => state.planLimits);
export const useUser = () => useAuthStore((state) => state.user);
export const useAuthLoading = () => useAuthStore((state) => state.isLoading);
export const useAuthError = () => useAuthStore((state) => state.error);

// Session recovery: Check auth when window regains focus
// This helps recover sessions that may have been restored by APIGate
if (typeof window !== 'undefined') {
  window.addEventListener('focus', () => {
    const state = useAuthStore.getState();
    // Only check if we think we're unauthenticated
    // This avoids spamming the server when already authenticated
    if (!state.isAuthenticated) {
      state.checkAuth();
    }
  });
}
