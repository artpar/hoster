import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface PlanLimits {
  max_deployments: number;
  max_cpu_cores: number;
  max_memory_mb: number;
  max_disk_gb: number;
}

interface AuthState {
  userId: string | null;
  planId: string | null;
  planLimits: PlanLimits | null;
  isAuthenticated: boolean;

  // Actions
  setAuth: (userId: string, planId: string, limits: PlanLimits) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      userId: null,
      planId: null,
      planLimits: null,
      isAuthenticated: false,

      setAuth: (userId, planId, planLimits) =>
        set({
          userId,
          planId,
          planLimits,
          isAuthenticated: true,
        }),

      clearAuth: () =>
        set({
          userId: null,
          planId: null,
          planLimits: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'hoster-auth',
    }
  )
);

// Selector hooks for convenience
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useUserId = () => useAuthStore((state) => state.userId);
export const usePlanLimits = () => useAuthStore((state) => state.planLimits);
