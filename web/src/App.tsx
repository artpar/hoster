import { Routes, Route } from 'react-router-dom';
import { Layout } from '@/components/layout/Layout';
import { ProtectedRoute } from '@/components/auth';
import { LandingPage } from '@/pages/LandingPage';
import { MarketplacePage } from '@/pages/marketplace/MarketplacePage';
import { TemplateDetailPage } from '@/pages/marketplace/TemplateDetailPage';
import { MyDeploymentsPage } from '@/pages/deployments/MyDeploymentsPage';
import { DeploymentDetailPage } from '@/pages/deployments/DeploymentDetailPage';
import { CreatorDashboardPage } from '@/pages/creator/CreatorDashboardPage';
import { LoginPage, SignupPage, ForgotPasswordPage, ResetPasswordPage } from '@/pages/auth';
import { NotFoundPage } from '@/pages/NotFoundPage';

export default function App() {
  return (
    <Routes>
      {/* Landing page - no layout */}
      <Route path="/" element={<LandingPage />} />

      {/* Auth routes - no layout */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/forgot-password" element={<ForgotPasswordPage />} />
      <Route path="/reset-password" element={<ResetPasswordPage />} />

      {/* Main app routes with layout */}
      <Route element={<Layout />}>

        {/* Marketplace - Public */}
        <Route path="marketplace" element={<MarketplacePage />} />
        <Route path="marketplace/:id" element={<TemplateDetailPage />} />

        {/* Deployments - Requires Auth */}
        <Route
          path="deployments"
          element={
            <ProtectedRoute>
              <MyDeploymentsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="deployments/:id"
          element={
            <ProtectedRoute>
              <DeploymentDetailPage />
            </ProtectedRoute>
          }
        />

        {/* Creator Dashboard - Requires Auth */}
        <Route
          path="creator"
          element={
            <ProtectedRoute>
              <CreatorDashboardPage />
            </ProtectedRoute>
          }
        />

        {/* 404 */}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
