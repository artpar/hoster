import { Link, Outlet, useLocation } from 'react-router-dom';
import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { useNodes } from '@/hooks/useNodes';
import { useCloudProvisions } from '@/hooks/useCloudProvisions';
import { useCloudCredentials } from '@/hooks/useCloudCredentials';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { cn } from '@/lib/cn';
import { pages } from '@/docs/registry';

const pageDocs = pages.nodes;

const tabs = [
  { to: '/nodes', label: 'Nodes', matchPrefixes: ['/nodes/new', '/nodes/new-key'] },
  { to: '/nodes/cloud', label: 'Cloud Servers', matchPrefixes: ['/nodes/cloud'] },
  { to: '/nodes/credentials', label: 'Credentials', matchPrefixes: ['/nodes/credentials'] },
] as const;

function isTabActive(tab: typeof tabs[number], pathname: string): boolean {
  if (pathname === tab.to) return true;
  return tab.matchPrefixes.some((prefix) => pathname.startsWith(prefix));
}

export function MyNodesPage() {
  const location = useLocation();
  const { data: nodes } = useNodes();
  const { data: provisions } = useCloudProvisions();
  const { data: credentials } = useCloudCredentials();
  const [guideOpen, setGuideOpen] = useState(false);

  const activeProvisions = provisions?.filter(
    (p) => p.attributes.status !== 'ready' && p.attributes.status !== 'destroyed'
  ) || [];

  const counts: Record<string, number> = {
    '/nodes': nodes?.length || 0,
    '/nodes/cloud': activeProvisions.length,
    '/nodes/credentials': credentials?.length || 0,
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{pageDocs.title}</h1>
        <p className="text-muted-foreground">
          Servers where your deployments run
        </p>
      </div>

      {/* Tab Bar */}
      <div className="mb-6 border-b">
        <nav className="-mb-px flex gap-0">
          {tabs.map((tab) => {
            const active = isTabActive(tab, location.pathname);
            const count = counts[tab.to] || 0;
            return (
              <Link
                key={tab.to}
                to={tab.to}
                className={cn(
                  'inline-flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium transition-colors',
                  active
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:border-muted-foreground/30 hover:text-foreground'
                )}
              >
                {tab.label}
                {count > 0 && (
                  <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-muted px-1.5 text-xs font-medium">
                    {count}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Outlet renders the active child route */}
      <Outlet />

      {/* Node Setup Guide (collapsible) */}
      <Card className="mt-6">
        <CardHeader
          className="cursor-pointer select-none"
          onClick={() => setGuideOpen(!guideOpen)}
        >
          <div className="flex items-center gap-2">
            {guideOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            <CardTitle className="text-lg">Node Setup Guide</CardTitle>
          </div>
        </CardHeader>
        {guideOpen && (
          <CardContent className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Before adding an existing server, ensure it is properly configured:
            </p>
            <div className="rounded-md bg-secondary/50 p-4 font-mono text-xs">
              <p className="text-muted-foreground mb-2"># 1. Create deploy user with Docker access</p>
              <p>sudo useradd -m -s /bin/bash deploy</p>
              <p>sudo usermod -aG docker deploy</p>
              <p className="text-muted-foreground mt-4 mb-2"># 2. Set up SSH key authentication</p>
              <p>sudo mkdir -p /home/deploy/.ssh</p>
              <p>echo "YOUR_PUBLIC_KEY" | sudo tee /home/deploy/.ssh/authorized_keys</p>
              <p>sudo chmod 700 /home/deploy/.ssh</p>
              <p>sudo chmod 600 /home/deploy/.ssh/authorized_keys</p>
              <p>sudo chown -R deploy:deploy /home/deploy/.ssh</p>
            </div>
            <p className="text-sm text-muted-foreground">
              Or use <strong>Create Cloud Server</strong> to automatically provision and configure a node on AWS, DigitalOcean, or Hetzner.
            </p>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
