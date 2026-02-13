import { Routes, Route, Navigate, useParams } from 'react-router-dom';
import { Layout } from '@/components/layout/Layout';
import { ProtectedRoute } from '@/components/auth';
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage, SignupPage } from '@/pages/auth';
import { MarketplacePage } from '@/pages/marketplace/MarketplacePage';
import { TemplateDetailPage } from '@/pages/marketplace/TemplateDetailPage';
import { CreateTemplatePage } from '@/pages/templates/CreateTemplatePage';
import { EditTemplatePage } from '@/pages/templates/EditTemplatePage';
import { MyDeploymentsPage } from '@/pages/deployments/MyDeploymentsPage';
import { DeploymentDetailPage } from '@/pages/deployments/DeploymentDetailPage';
import { DashboardPage } from '@/pages/dashboard/DashboardPage';
import { MyNodesPage } from '@/pages/nodes/MyNodesPage';
import { NodesTab, AddNodeForm, AddSSHKeyForm } from '@/components/nodes';
import { CloudServersTab, CredentialsTab } from '@/components/nodes';
import { ProvisionNodeForm } from '@/components/cloud';
import { AddCredentialForm } from '@/components/cloud';
import { SSHKeysPage } from '@/pages/ssh-keys/SSHKeysPage';
import { BillingPage } from '@/pages/billing/BillingPage';
import { NotFoundPage } from '@/pages/NotFoundPage';

function MarketplaceRedirect() {
  const { id } = useParams();
  return <Navigate to={`/templates/${id}`} replace />;
}

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

        {/* Dashboard - Requires Auth */}
        <Route
          path="dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />

        {/* Templates - Public (browse all + manage own) */}
        <Route path="templates" element={<MarketplacePage />} />
        <Route path="templates/new" element={<ProtectedRoute><CreateTemplatePage /></ProtectedRoute>} />
        <Route path="templates/:id/edit" element={<ProtectedRoute><EditTemplatePage /></ProtectedRoute>} />
        <Route path="templates/:id" element={<TemplateDetailPage />} />

        {/* Legacy marketplace redirects */}
        <Route path="marketplace" element={<Navigate to="/templates" replace />} />
        <Route path="marketplace/:id" element={<MarketplaceRedirect />} />

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
        {/* Billing - Requires Auth */}
        <Route
          path="billing"
          element={
            <ProtectedRoute>
              <BillingPage />
            </ProtectedRoute>
          }
        />

        {/* 404 */}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
