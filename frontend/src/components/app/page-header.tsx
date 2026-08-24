/**
 * A consistent title block for every page.
 *
 * Using one component means every page gets identical spacing and type sizes.
 * If each page styled its own heading, they would slowly drift apart — that is
 * how an app starts to look homemade.
 *
 * `action` is a slot: the caller passes in whatever button belongs here.
 * The header does not need to know what the button does.
 */
export function PageHeader({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 border-b border-border px-4 py-5 sm:flex-row sm:items-center sm:justify-between lg:px-6">
      <div className="min-w-0">
        {/* h1 because this is the main heading of the page. Screen reader
            users navigate by headings, so the level must reflect the real
            structure, not just how big we want the text to look. */}
        <h1 className="truncate text-xl font-semibold tracking-tight">
          {title}
        </h1>
        {description && (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        )}
      </div>

      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
}
