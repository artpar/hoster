export interface DocEntry {
  label: string;
  description: string;
}

export interface StatusDoc extends DocEntry {
  style: string;
}

export interface MetricDoc extends DocEntry {
  unit?: string;
}

export interface PageDoc {
  title: string;
  subtitle: string;
  sections: Record<string, DocEntry>;
  emptyState: DocEntry;
}

export interface EventDoc extends DocEntry {
  severity: 'info' | 'success' | 'warning' | 'error';
}
