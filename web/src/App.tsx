import { Routes, Route } from 'react-router-dom';
import { Layout } from '@/components/layout/Layout';
import { ProtectedRoute } from '@/components/auth';
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage, SignupPage } from '@/pages/auth';
import { MarketplacePage } from '@/pages/marketplace/MarketplacePage';
import { TemplateDetailPage } from '@/pages/marketplace/TemplateDetailPage';
import { MyDeploymentsPage } from '@/pages/deployments/MyDeploymentsPage';
import { DeploymentDetailPage } from '@/pages/deployments/DeploymentDetailPage';
import { CreatorDashboardPage } from '@/pages/creator/CreatorDashboardPage';
import { MyNodesPage } from '@/pages/nodes/MyNodesPage';
import { NodesTab, AddNodeForm, AddSSHKeyForm } from '@/components/nodes';
import { CloudServersTab, CredentialsTab } from '@/components/nodes';
import { ProvisionNodeForm } from '@/components/cloud';
import { AddCredentialForm } from '@/components/cloud';
import { SSHKeysPage } from '@/pages/ssh-keys/SSHKeysPage';
import { NotFoundPage } from '@/pages/NotFoundPage';

export default function App() {
  return (
    <Routes>
      {/* Landing page - no layout */}
      <Route path="/" element={<LandingPage />} />

      {/* Auth pages - no layout */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />

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

        {/* Nodes - Requires Auth */}
        <Route
          path="nodes"
          element={
            <ProtectedRoute>
              <MyNodesPage />
            </ProtectedRoute>
          }
        >
          <Route index element={<NodesTab />} />
          <Route path="new" element={<AddNodeForm />} />
          <Route path="new-key" element={<AddSSHKeyForm />} />
          <Route path="cloud" element={<CloudServersTab />} />
          <Route path="cloud/new" element={<ProvisionNodeForm />} />
          <Route path="credentials" element={<CredentialsTab />} />
          <Route path="credentials/new" element={<AddCredentialForm />} />
        </Route>
        <Route
          path="ssh-keys"
          element={
            <ProtectedRoute>
              <SSHKeysPage />
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
