// A single, unobtrusive toast anchored bottom-center — design.md "Error
// states: Toast/notification". Not locked by any test file directly (App
// owns the toast per the UI Contract; this is Dispatch's own component
// choice), but its copy/behavior IS exercised indirectly through App's
// wiring of AddBar's onTransientError and the move-rollback path.
export interface ToastProps {
  message: string;
  onRetry?: () => void;
  onDismiss: () => void;
}

export function Toast({ message, onRetry, onDismiss }: ToastProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed inset-x-0 bottom-6 z-50 flex justify-center px-4"
    >
      <div className="flex items-center gap-3 rounded-xl bg-raised px-4 py-3 shadow-modal">
        <p className="text-sm text-text">{message}</p>
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="text-sm font-medium text-primary hover:text-primary-dim"
          >
            Retry
          </button>
        )}
        <button
          type="button"
          onClick={onDismiss}
          aria-label="Dismiss notification"
          className="text-sm text-text-dim hover:text-text"
        >
          ×
        </button>
      </div>
    </div>
  );
}
