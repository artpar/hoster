import { Toaster as SonnerToaster } from 'sonner';

export function Toaster() {
  return (
    <SonnerToaster
      position="bottom-right"
      style={{ zIndex: 9999 }}
      toastOptions={{
        className: 'text-sm',
        duration: 5000,
      }}
    />
  );
}
