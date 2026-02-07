import { useState } from 'react';
import {
  Globe,
  Plus,
  Trash2,
  RefreshCw,
  Copy,
  Check,
  ShieldCheck,
  ShieldAlert,
  ShieldQuestion,
  ExternalLink,
} from 'lucide-react';
import {
  useDomains,
  useAddDomain,
  useRemoveDomain,
  useVerifyDomain,
} from '@/hooks/useDomains';
import type { DomainInfo, DNSInstruction } from '@/api/domains';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/Dialog';

interface DomainsTabProps {
  deploymentId: string;
}

function VerificationBadge({ status }: { status?: string }) {
  if (!status) return null;

  switch (status) {
    case 'verified':
      return (
        <Badge variant="success" className="gap-1">
          <ShieldCheck className="h-3 w-3" />
          Verified
        </Badge>
      );
    case 'pending':
      return (
        <Badge variant="warning" className="gap-1">
          <ShieldQuestion className="h-3 w-3" />
          Pending
        </Badge>
      );
    case 'failed':
      return (
        <Badge variant="destructive" className="gap-1">
          <ShieldAlert className="h-3 w-3" />
          Failed
        </Badge>
      );
    default:
      return (
        <Badge variant="secondary" className="gap-1">
          {status}
        </Badge>
      );
  }
}

function TypeBadge({ type }: { type: 'auto' | 'custom' }) {
  return type === 'auto' ? (
    <Badge variant="default" className="bg-blue-500 hover:bg-blue-500/80">
      Auto
    </Badge>
  ) : (
    <Badge variant="default" className="bg-purple-500 hover:bg-purple-500/80">
      Custom
    </Badge>
  );
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className="inline-flex items-center rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
      title="Copy to clipboard"
    >
      {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
    </button>
  );
}

function DNSInstructionsTable({ instructions }: { instructions: DNSInstruction[] }) {
  return (
    <div className="mt-3 rounded-md border">
      <div className="bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground">
        DNS Records Required
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/30">
            <tr>
              <th className="px-3 py-1.5 text-left text-xs font-medium text-muted-foreground">Type</th>
              <th className="px-3 py-1.5 text-left text-xs font-medium text-muted-foreground">Name</th>
              <th className="px-3 py-1.5 text-left text-xs font-medium text-muted-foreground">Value</th>
              <th className="px-3 py-1.5 text-left text-xs font-medium text-muted-foreground">Priority</th>
              <th className="w-8"></th>
            </tr>
          </thead>
          <tbody>
            {instructions.map((instruction, idx) => (
              <tr key={idx} className="border-b last:border-0">
                <td className="px-3 py-2 font-mono text-xs">{instruction.type}</td>
                <td className="px-3 py-2 font-mono text-xs">{instruction.name}</td>
                <td className="px-3 py-2 font-mono text-xs">
                  <span className="inline-flex items-center gap-1">
                    {instruction.value}
                    <CopyButton value={instruction.value} />
                  </span>
                </td>
                <td className="px-3 py-2 text-xs text-muted-foreground">{instruction.priority}</td>
                <td></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function DomainRow({
  domain,
  deploymentId,
}: {
  domain: DomainInfo;
  deploymentId: string;
}) {
  const [removeDialogOpen, setRemoveDialogOpen] = useState(false);
  const removeDomain = useRemoveDomain(deploymentId);
  const verifyDomain = useVerifyDomain(deploymentId);

  const showVerifyButton =
    domain.type === 'custom' &&
    domain.verification_status !== 'verified';

  const showRemoveButton = domain.type === 'custom';

  const handleRemoveConfirm = async () => {
    await removeDomain.mutateAsync(domain.hostname);
  };

  const handleVerify = async () => {
    await verifyDomain.mutateAsync(domain.hostname);
  };

  return (
    <>
      <div className="rounded-md border p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              <a
                href={`https://${domain.hostname}`}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-primary hover:underline inline-flex items-center gap-1"
              >
                {domain.hostname}
                <ExternalLink className="h-3 w-3" />
              </a>
              <TypeBadge type={domain.type} />
              <VerificationBadge status={domain.verification_status} />
              {domain.ssl_enabled && (
                <Badge variant="outline" className="gap-1 text-green-600 border-green-300">
                  <ShieldCheck className="h-3 w-3" />
                  SSL
                </Badge>
              )}
            </div>
            {domain.last_check_error && (
              <p className="text-xs text-destructive ml-6">
                {domain.last_check_error}
              </p>
            )}
            {domain.verified_at && (
              <p className="text-xs text-muted-foreground ml-6">
                Verified {new Date(domain.verified_at).toLocaleString()}
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            {showVerifyButton && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleVerify}
                disabled={verifyDomain.isPending}
              >
                {verifyDomain.isPending ? (
                  <>
                    <LoadingSpinner className="mr-1 h-3.5 w-3.5" />
                    Verifying...
                  </>
                ) : (
                  <>
                    <RefreshCw className="mr-1 h-3.5 w-3.5" />
                    Verify
                  </>
                )}
              </Button>
            )}
            {showRemoveButton && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setRemoveDialogOpen(true)}
                disabled={removeDomain.isPending}
                className="text-destructive hover:text-destructive"
              >
                <Trash2 className="mr-1 h-3.5 w-3.5" />
                Remove
              </Button>
            )}
          </div>
        </div>

        {domain.type === 'custom' &&
          domain.verification_status !== 'verified' &&
          domain.instructions &&
          domain.instructions.length > 0 && (
            <DNSInstructionsTable instructions={domain.instructions} />
          )}
      </div>

      <ConfirmDialog
        open={removeDialogOpen}
        onOpenChange={setRemoveDialogOpen}
        title="Remove Domain"
        description={`Are you sure you want to remove "${domain.hostname}"? This will stop routing traffic for this domain to your deployment.`}
        confirmLabel="Remove"
        variant="destructive"
        onConfirm={handleRemoveConfirm}
      />
    </>
  );
}

function AddDomainDialog({
  open,
  onOpenChange,
  deploymentId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  deploymentId: string;
}) {
  const [hostname, setHostname] = useState('');
  const [error, setError] = useState<string | null>(null);
  const addDomain = useAddDomain(deploymentId);

  const handleAdd = async () => {
    setError(null);

    const trimmed = hostname.trim().toLowerCase();
    if (!trimmed) {
      setError('Hostname is required');
      return;
    }

    // Basic hostname validation
    const hostnamePattern = /^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}$/;
    if (!hostnamePattern.test(trimmed)) {
      setError('Please enter a valid hostname (e.g., app.example.com)');
      return;
    }

    try {
      await addDomain.mutateAsync(trimmed);
      onOpenChange(false);
      setHostname('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add domain');
    }
  };

  const handleClose = () => {
    if (!addDomain.isPending) {
      onOpenChange(false);
      setHostname('');
      setError(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Custom Domain</DialogTitle>
          <DialogDescription>
            Point your own domain to this deployment. You will need to configure DNS records after adding the domain.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="hostname">Hostname</Label>
            <Input
              id="hostname"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="app.example.com"
              disabled={addDomain.isPending}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleAdd();
              }}
            />
            <p className="text-xs text-muted-foreground">
              Enter the full hostname you want to use (e.g., app.example.com or www.example.com)
            </p>
          </div>

          {error && (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={addDomain.isPending}>
            Cancel
          </Button>
          <Button onClick={handleAdd} disabled={addDomain.isPending}>
            {addDomain.isPending ? (
              <>
                <LoadingSpinner className="mr-1 h-4 w-4" />
                Adding...
              </>
            ) : (
              <>
                <Plus className="mr-1 h-4 w-4" />
                Add Domain
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function DomainsTab({ deploymentId }: DomainsTabProps) {
  const { data: domains, isLoading, error } = useDomains(deploymentId);
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle className="text-lg">Domains</CardTitle>
              <p className="mt-1 text-sm text-muted-foreground">
                Manage the domains that route traffic to this deployment.
              </p>
            </div>
            <Button size="sm" onClick={() => setAddDialogOpen(true)}>
              <Plus className="mr-1 h-4 w-4" />
              Add Custom Domain
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              Failed to load domains: {error instanceof Error ? error.message : 'Unknown error'}
            </div>
          ) : domains && domains.length > 0 ? (
            <div className="space-y-3">
              {domains.map((domain) => (
                <DomainRow
                  key={domain.hostname}
                  domain={domain}
                  deploymentId={deploymentId}
                />
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No domains configured. The auto-generated domain will appear once the deployment is running.
            </p>
          )}
        </CardContent>
      </Card>

      <AddDomainDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        deploymentId={deploymentId}
      />
    </>
  );
}
