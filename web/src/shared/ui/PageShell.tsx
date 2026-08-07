import type { ReactNode } from "react";

/** The frame every content page sits in: header, then a centred column. */
export function PageShell({
  header,
  children,
}: {
  header?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col bg-canvas">
      {header}
      <main className="mx-auto flex w-full max-w-4xl flex-col gap-5 px-5 py-6">
        {children}
      </main>
    </div>
  );
}

/** The full-height frame for a workspace that manages its own scrolling. */
export function WorkspaceShell({
  header,
  children,
}: {
  header: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="grid h-screen grid-rows-[auto_minmax(0,1fr)] overflow-hidden bg-canvas">
      {header}
      <div className="flex min-h-0 flex-col">{children}</div>
    </div>
  );
}
