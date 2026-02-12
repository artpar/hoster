import { useEffect, useMemo, useState, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Activity, DollarSign, Layers, FileText, CreditCard, CheckCircle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { useDeployments } from '@/hooks/useDeployments';
import { useTemplates } from '@/hooks/useTemplates';
import { StatusBadge } from '@/components/common/StatusBadge';
import { LoadingPage } from '@/components/common/LoadingSpinner';
import { useUser } from '@/stores/authStore';
import { api } from '@/api/client';

interface UsageEvent {
  type: string;
  id: string;
  attributes: Record<string, unknown>;
}

interface Invoice {
  type: string;
  id: string;
  attributes: {
    period_start: string;
    period_end: string;
    items: string;
    subtotal_cents: number;
    tax_cents: number;
    total_cents: number;
    currency: string;
    status: string;
    stripe_session_id: string | null;
    stripe_payment_url: string | null;
    paid_at: string | null;
    created_at: string;
  };
}

export function BillingPage() {
  const user = useUser();
  const [searchParams, setSearchParams] = useSearchParams();
  const { data: deployments, isLoading: deploymentsLoading } = useDeployments();
  const { data: templates, isLoading: templatesLoading } = useTemplates();
  const [usageEvents, setUsageEvents] = useState<UsageEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(true);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [invoicesLoading, setInvoicesLoading] = useState(true);
  const [paying, setPaying] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const fetchInvoices = useCallback(() => {
    api.get<Invoice[]>('/invoices')
      .then((data) => {
        const list = Array.isArray(data.data) ? data.data : data.data ? [data.data] : [];
        setInvoices(list);
        setInvoicesLoading(false);
      })
      .catch(() => setInvoicesLoading(false));
  }, []);

  useEffect(() => {
    if (!user) return;
    const token = JSON.parse(localStorage.getItem('hoster-auth') || '{}')?.state?.token;
    if (!token) { setEventsLoading(false); return; }

    fetch(`/api/v1/meter?user_id=${user.id}&page[size]=50`, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
      .then((r) => r.json())
      .then((data) => {
        setUsageEvents(data.data || []);
        setEventsLoading(false);
      })
      .catch(() => setEventsLoading(false));
  }, [user]);

  useEffect(() => {
    fetchInvoices();
  }, [fetchInvoices]);

  // Handle payment verification on return from Stripe
  useEffect(() => {
    const payment = searchParams.get('payment');
    const sessionId = searchParams.get('session_id');

    if (payment === 'success' && sessionId) {
      const token = JSON.parse(localStorage.getItem('hoster-auth') || '{}')?.state?.token;
      if (token) {
        fetch(`/api/v1/billing/verify-payment?session_id=${sessionId}`, {
          headers: { 'Authorization': `Bearer ${token}` },
        })
          .then((r) => r.json())
          .then((data) => {
            if (data.data?.paid) {
              setSuccessMessage('Payment successful! Invoice has been marked as paid.');
            } else {
              setSuccessMessage('Payment is being processed. It may take a moment to update.');
            }
            fetchInvoices();
          })
          .catch(() => {
            setSuccessMessage('Payment received. Verifying status...');
            fetchInvoices();
          });
      }
      // Clean URL params
      setSearchParams({});
    } else if (payment === 'cancelled') {
      setError('Payment was cancelled.');
      setSearchParams({});
    }
  }, [searchParams, setSearchParams, fetchInvoices]);

  const isLoading = deploymentsLoading || templatesLoading || eventsLoading;

  const { runningDeployments, monthlyCost, deploymentCosts } = useMemo(() => {
    const allDeployments = deployments ?? [];
    const allTemplates = templates ?? [];
    const running = allDeployments.filter((d) => d.attributes.status === 'running');

    const costs = running.map((d) => {
      const tmpl = allTemplates.find((t) => t.id === String(d.attributes.template_id));
      return {
        deployment: d,
        template: tmpl,
        monthlyCents: tmpl?.attributes.price_monthly_cents ?? 0,
      };
    });

    const total = costs.reduce((sum, c) => sum + c.monthlyCents, 0);

    return {
      runningDeployments: running,
      monthlyCost: total,
      deploymentCosts: costs,
    };
  }, [deployments, templates]);

  const handlePayInvoice = async (invoiceId: string) => {
    setPaying(invoiceId);
    setError(null);
    try {
      const token = JSON.parse(localStorage.getItem('hoster-auth') || '{}')?.state?.token;
      const resp = await fetch(`/api/v1/invoices/${invoiceId}/pay`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          success_url: `${window.location.origin}/billing?payment=success`,
          cancel_url: `${window.location.origin}/billing?payment=cancelled`,
        }),
      });
      if (!resp.ok) {
        const err = await resp.json();
        throw new Error(err.error?.detail || err.errors?.[0]?.detail || `Failed (${resp.status})`);
      }
      const data = await resp.json();
      if (data.data?.checkout_url) {
        window.location.href = data.data.checkout_url;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create checkout');
      setPaying(null);
    }
  };

  if (isLoading) return <LoadingPage />;

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Billing & Usage</h1>
        <p className="text-muted-foreground">
          Your running deployments, costs, invoices, and usage activity.
        </p>
      </div>

      {successMessage && (
        <div className="mb-4 flex items-center gap-2 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          <CheckCircle className="h-4 w-4" />
          {successMessage}
          <button onClick={() => setSuccessMessage(null)} className="ml-auto text-green-600 hover:text-green-800">&times;</button>
        </div>
      )}

      {error && (
        <div className="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {error}
          <button onClick={() => setError(null)} className="ml-2 text-red-600 hover:text-red-800">&times;</button>
        </div>
      )}

      {/* Summary Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${(monthlyCost / 100).toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground">
              From {runningDeployments.length} running deployment{runningDeployments.length !== 1 ? 's' : ''}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Deployments</CardTitle>
            <Layers className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{runningDeployments.length}</div>
            <p className="text-xs text-muted-foreground">
              {(deployments ?? []).length} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Usage Events</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{usageEvents.length}</div>
            <p className="text-xs text-muted-foreground">Recent tracked events</p>
          </CardContent>
        </Card>
      </div>

      {/* Invoices */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-lg">Invoices</CardTitle>
        </CardHeader>
        <CardContent>
          {invoicesLoading ? (
            <p className="py-4 text-center text-sm text-muted-foreground">Loading invoices...</p>
          ) : invoices.length === 0 ? (
            <div className="py-8 text-center">
              <FileText className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No invoices yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {invoices.map((invoice) => {
                const attrs = invoice.attributes;
                const items = parseItems(attrs.items);
                return (
                  <div
                    key={invoice.id}
                    className="flex items-center justify-between rounded-md border p-4"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium">
                          Invoice {invoice.id.slice(0, 12)}...
                        </p>
                        <InvoiceStatusBadge status={attrs.status} />
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {formatDate(attrs.period_start)} — {formatDate(attrs.period_end)}
                      </p>
                      {items.length > 0 && (
                        <p className="mt-1 text-xs text-muted-foreground">
                          {items.map((i) => i.deployment_name).join(', ')}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-lg font-semibold">
                        ${(attrs.total_cents / 100).toFixed(2)}
                      </span>
                      {(attrs.status === 'draft' || attrs.status === 'failed') && (
                        <Button
                          size="sm"
                          onClick={() => handlePayInvoice(invoice.id)}
                          disabled={paying === invoice.id}
                        >
                          <CreditCard className="mr-1 h-4 w-4" />
                          {paying === invoice.id ? 'Processing...' : 'Pay Now'}
                        </Button>
                      )}
                      {attrs.status === 'paid' && attrs.paid_at && (
                        <span className="text-xs text-green-600">
                          Paid {formatDate(attrs.paid_at)}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Active Deployments with Costs */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-lg">Running Deployments</CardTitle>
        </CardHeader>
        <CardContent>
          {deploymentCosts.length === 0 ? (
            <div className="py-8 text-center">
              <Layers className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No running deployments</p>
              <Link to="/marketplace" className="mt-1 text-sm text-primary hover:underline">
                Browse the marketplace
              </Link>
            </div>
          ) : (
            <div className="space-y-2">
              {deploymentCosts.map(({ deployment, template, monthlyCents }) => (
                <Link
                  key={deployment.id}
                  to={`/deployments/${deployment.id}`}
                  className="flex items-center justify-between rounded-md border p-3 transition-colors hover:bg-accent/50"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{deployment.attributes.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {template?.attributes.name ?? 'Unknown template'}
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-medium">
                      {monthlyCents === 0 ? 'Free' : `$${(monthlyCents / 100).toFixed(2)}/mo`}
                    </span>
                    <StatusBadge status={deployment.attributes.status} />
                  </div>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Usage Event History */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Usage History</CardTitle>
        </CardHeader>
        <CardContent>
          {usageEvents.length === 0 ? (
            <div className="py-8 text-center">
              <Activity className="mx-auto h-8 w-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm text-muted-foreground">No usage events recorded yet</p>
            </div>
          ) : (
            <div className="space-y-2">
              {usageEvents.slice(0, 20).map((event) => {
                const attrs = event.attributes;
                const eventType = String(attrs.event_type || '');
                const isDeploymentEvent = eventType.startsWith('deployment.');
                const isApiRequest = !isDeploymentEvent && !!attrs.method;

                return (
                  <div
                    key={event.id}
                    className="flex items-center justify-between rounded-md border p-3"
                  >
                    <div className="min-w-0 flex items-center gap-3">
                      {isApiRequest ? (
                        <>
                          <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                            {String(attrs.method)}
                          </span>
                          <p className="truncate text-sm text-muted-foreground">
                            {String(attrs.path || 'request')}
                          </p>
                        </>
                      ) : (
                        <div>
                          <p className="text-sm font-medium">
                            {formatEventType(eventType)}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {String(attrs.resource_id || event.id)}
                          </p>
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-3">
                      {isApiRequest && (
                        <span className={`text-xs font-medium ${
                          Number(attrs.status_code || 200) < 400 ? 'text-green-600' : 'text-red-600'
                        }`}>
                          {String(attrs.status_code || '')}
                        </span>
                      )}
                      <span className="text-xs text-muted-foreground">
                        {formatTimestamp(String(attrs.timestamp || ''))}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function InvoiceStatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    draft: 'bg-gray-100 text-gray-700',
    pending: 'bg-yellow-100 text-yellow-800',
    paid: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
  };
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors[status] || 'bg-gray-100 text-gray-700'}`}>
      {status}
    </span>
  );
}

interface InvoiceLineItem {
  deployment_id: string;
  deployment_name: string;
  template_name: string;
  monthly_cents: number;
  description: string;
}

function parseItems(itemsStr: string): InvoiceLineItem[] {
  try {
    return JSON.parse(itemsStr || '[]');
  } catch {
    return [];
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  try {
    return new Date(dateStr).toLocaleDateString();
  } catch {
    return dateStr;
  }
}

function formatEventType(type: string): string {
  const labels: Record<string, string> = {
    'deployment.created': 'Deployment Created',
    'deployment.started': 'Deployment Started',
    'deployment.stopped': 'Deployment Stopped',
    'deployment.deleted': 'Deployment Deleted',
  };
  return labels[type] || type;
}

function formatTimestamp(ts: string): string {
  if (!ts) return '';
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
}
