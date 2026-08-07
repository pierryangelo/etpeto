import { Field as BaseField } from "@base-ui/react/field";
import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { useId, type ReactNode } from "react";

/**
 * The primitives every screen composes.
 *
 * Appearance is decided once, in the `ds-*` shortcuts, and behaviour comes from
 * Base UI. A feature imports these and never writes a style class of its own —
 * that is what keeps two screens from inventing two different buttons.
 */

type ButtonVariant = "primary" | "secondary" | "quiet" | "danger";

const buttonVariants: Record<ButtonVariant, string> = {
  primary: "ds-btn-primary",
  secondary: "ds-btn-secondary",
  quiet: "ds-btn-quiet",
  danger: "ds-btn-danger",
};

export function Button({
  variant = "secondary",
  small,
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  small?: boolean;
}) {
  return (
    <button
      type="button"
      className={`${buttonVariants[variant]} ${small ? "ds-btn-sm" : ""} ${className}`}
      {...props}
    />
  );
}

export function IconButton({
  label,
  icon,
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  icon: string;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className={`ds-icon-btn ${className}`}
      {...props}
    >
      <span className={icon} />
    </button>
  );
}

export function Field({
  label,
  help,
  children,
}: {
  label: string;
  help?: string;
  children: (props: { id: string; className: string }) => ReactNode;
}) {
  const id = useId();
  return (
    <BaseField.Root className="grid gap-1.5">
      <BaseField.Label htmlFor={id} className="ds-label">
        {label}
      </BaseField.Label>
      {children({ id, className: "ds-input" })}
      {help && <span className="ds-help">{help}</span>}
    </BaseField.Root>
  );
}

export function TextField({
  label,
  help,
  value,
  onChange,
  ...rest
}: {
  label: string;
  help?: string;
  value: string;
  onChange: (value: string) => void;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, "value" | "onChange">) {
  return (
    <Field label={label} help={help}>
      {({ id, className }) => (
        <input
          id={id}
          className={className}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          {...rest}
        />
      )}
    </Field>
  );
}

export function TextAreaField({
  label,
  help,
  value,
  onChange,
}: {
  label: string;
  help?: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field label={label} help={help}>
      {({ id }) => (
        <textarea
          id={id}
          className="ds-textarea"
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
      )}
    </Field>
  );
}

export function Switch({
  checked,
  title,
  description,
  onChange,
}: {
  checked: boolean;
  title: string;
  description?: string;
  onChange: (value: boolean) => void;
}) {
  const titleId = useId();
  const descriptionId = useId();
  return (
    // Not a <label>: wrapping the control would fold the description into its
    // accessible name. The switch points at the title instead.
    <div className="flex items-start gap-2.5 border-t border-line pt-2.5">
      <BaseSwitch.Root
        checked={checked}
        onCheckedChange={onChange}
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        className="mt-0.5 h-4.5 w-7.5 flex-none rounded-full border border-line-strong bg-line transition-colors duration-140 data-[checked]:border-accent-strong data-[checked]:bg-accent focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-[var(--color-focus)]"
      >
        <BaseSwitch.Thumb className="block size-3.5 translate-x-0.5 rounded-full bg-white transition-transform duration-140 data-[checked]:translate-x-3.5" />
      </BaseSwitch.Root>
      <span>
        <strong id={titleId} className="block text-12.5px font-600 text-ink">
          {title}
        </strong>
        {description && (
          <span id={descriptionId} className="ds-help">
            {description}
          </span>
        )}
      </span>
    </div>
  );
}

/** A toggle used to pick from a small, visible set. */
export function Chip({
  pressed,
  onToggle,
  children,
}: {
  pressed: boolean;
  onToggle: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      onClick={onToggle}
      className="ds-control h-6 cursor-pointer border-line bg-paper px-2 text-11.5px text-subtle aria-pressed:border-accent-line aria-pressed:bg-accent-soft aria-pressed:text-accent-strong"
    >
      {children}
    </button>
  );
}

export function Badge({
  tone = "muted",
  children,
}: {
  tone?: "live" | "draft" | "muted";
  children: ReactNode;
}) {
  const tones = {
    live: "ds-badge-live",
    draft: "ds-badge-draft",
    muted: "ds-badge-muted",
  } as const;
  return <span className={tones[tone]}>{children}</span>;
}

export function PageHeading({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex items-end justify-between gap-4">
      <div>
        <span className="ds-eyebrow text-accent-strong">{eyebrow}</span>
        <h1 className="ds-title mb-1 mt-1">{title}</h1>
        {description && <p className="m-0 max-w-2xl ds-meta">{description}</p>}
      </div>
      {action}
    </div>
  );
}

export function Panel({
  eyebrow,
  title,
  action,
  children,
}: {
  eyebrow: string;
  title: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="flex min-w-0 flex-col overflow-auto ds-panel">
      <div className="flex items-center justify-between gap-3 border-b border-line px-3 py-2.5">
        <div>
          <span className="ds-eyebrow">{eyebrow}</span>
          <strong className="mt-0.5 block text-12.5px text-ink">{title}</strong>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function Divider() {
  return <div className="ds-divider" aria-hidden="true" />;
}

/** A strip that states an ongoing condition, with its own actions. */
export function NoticeBar({
  tone = "attention",
  children,
  actions,
}: {
  tone?: "attention" | "danger";
  children: ReactNode;
  actions?: ReactNode;
}) {
  const tones = {
    attention: "bg-accent-soft text-accent-strong",
    danger: "bg-paper text-danger",
  } as const;
  return (
    <div
      className={`flex items-center gap-2.5 border-b border-line px-5 py-2 text-12px ${tones[tone]}`}
    >
      {children}
      {actions && (
        <span className="ml-auto flex items-center gap-1.5">{actions}</span>
      )}
    </div>
  );
}

/** The area a form or detail panel occupies beside a tree or list. */
export function DetailPane({ children }: { children: ReactNode }) {
  return (
    <div className="ds-stack min-w-0 flex-1 overflow-auto bg-paper p-5">
      {children}
    </div>
  );
}

/** Groups a destructive action with the consequence it carries. */
export function DangerZone({
  action,
  note,
}: {
  action: ReactNode;
  note: string;
}) {
  return (
    <div className="mt-2 border-t border-line pt-3">
      {action}
      <p className="ds-help mt-1.5">{note}</p>
    </div>
  );
}

/** Actions pinned to the foot of a panel. */
export function PanelActions({ children }: { children: ReactNode }) {
  return (
    <div className="mt-auto flex gap-1.5 border-t border-line p-2">
      {children}
    </div>
  );
}
