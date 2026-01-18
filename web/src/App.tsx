import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from '@/components/layout/Layout';
import { MarketplacePage } from '@/pages/marketplace/MarketplacePage';
import { TemplateDetailPage } from '@/pages/marketplace/TemplateDetailPage';
import { MyDeploymentsPage } from '@/pages/deployments/MyDeploymentsPage';
import { DeploymentDetailPage } from '@/pages/deployments/DeploymentDetailPage';
import { CreatorDashboardPage } from '@/pages/creator/CreatorDashboardPage';
import { NotFoundPage } from '@/pages/NotFoundPage';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        {/* Redirect root to marketplace */}
        <Route index element={<Navigate to="/marketplace" replace />} />

        {/* Marketplace - Public */}
        <Route path="marketplace" element={<MarketplacePage />} />
        <Route path="marketplace/:id" element={<TemplateDetailPage />} />

        {/* Deployments - Requires Auth */}
        <Route path="deployments" element={<MyDeploymentsPage />} />
        <Route path="deployments/:id" element={<DeploymentDetailPage />} />

        {/* Creator Dashboard - Requires Auth */}
        <Route path="creator" element={<CreatorDashboardPage />} />

        {/* 404 */}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
