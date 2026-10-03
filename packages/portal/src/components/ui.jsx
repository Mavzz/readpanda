import { useEffect, useId } from 'react';
import { Link } from 'react-router-dom';
import { CloseIcon, PlusIcon, SearchIcon } from './icons';

// Portal design system v2 components (12d, with desktop states).

const cx = (...parts) => parts.filter(Boolean).join(' ');

// ── Type ──────────────────────────────────────────────────────

export const Eyebrow = ({ className, children, ...rest }) => (
  <span className={cx('text-[11px] leading-none font-bold tracking-[1px] uppercase text-ink-meta', className)} {...rest}>
    {children}
  </span>
);

export const PageHeader = ({ title, subtitle, children }) => (
  <div className="flex flex-wrap items-center justify-between gap-4">
    <div className="flex flex-col gap-1.5 min-w-0">
      <h1 className="m-0 text-[26px] leading-[1.2] font-extrabold tracking-[-0.5px] text-ink-title">{title}</h1>
      {subtitle && <p className="m-0 text-[13px] leading-normal text-ink-holder">{subtitle}</p>}
    </div>
    {children && <div className="flex flex-wrap items-center gap-2.5">{children}</div>}
  </div>
);

// 12c section: one eyebrow header, "See all" only when there's overflow.
export const Section = ({ title, seeAllTo, children }) => (
  <section className="flex flex-col gap-2.5">
    <div className="flex items-baseline justify-between">
      <Eyebrow>{title}</Eyebrow>
      {seeAllTo && <Link to={seeAllTo} className="text-[13px] font-bold text-link">See all</Link>}
    </div>
    {children}
  </section>
);

// ── Buttons ───────────────────────────────────────────────────

// One gold primary per screen. Secondary is a surface-2 pill. Disabled is an
// outline, never a filled pill.
export const Button = ({ variant = 'primary', className, disabled, type = 'button', children, ...rest }) => {
  const base = 'inline-flex items-center justify-center gap-2 h-11 px-5 rounded-full text-sm whitespace-nowrap transition-[background,transform] duration-150';
  const look = disabled
    ? 'bg-transparent shadow-[inset_0_0_0_1.5px_var(--rp-surface-3)] text-ink-disabled font-bold'
    : variant === 'primary'
      ? 'px-[22px] bg-primary-gradient text-on-gold font-extrabold shadow-glow active:scale-[0.98]'
      : 'bg-surface-2 text-ink-title font-bold hover:bg-surface-3';
  return (
    <button type={type} disabled={disabled} className={cx(base, look, className)} {...rest}>
      {children}
    </button>
  );
};

export const IconButton = ({ label, className, children, ...rest }) => (
  <button
    type="button"
    aria-label={label}
    title={label}
    className={cx('w-[38px] h-[38px] shrink-0 rounded-full bg-surface-2 text-ink-title flex items-center justify-center hover:bg-surface-3 disabled:opacity-50', className)}
    {...rest}
  >
    {children}
  </button>
);

// Text action: gold when it's the screen's main move, muted otherwise.
export const TextAction = ({ tone = 'link', className, children, type = 'button', ...rest }) => {
  const color = { link: 'text-link font-extrabold', muted: 'text-ink-holder font-bold hover:text-ink-body', gold: 'text-gold font-extrabold' }[tone];
  return (
    <button type={type} className={cx('bg-transparent p-0 text-[13px] disabled:text-ink-disabled disabled:cursor-default', color, className)} {...rest}>
      {children}
    </button>
  );
};

// ── Fields ────────────────────────────────────────────────────

const fieldBase = 'w-full box-border px-4 border-none rounded-field bg-surface-2 text-sm font-semibold text-ink-title outline-none focus-visible:outline-2 focus-visible:outline-gold';

// Labels are eyebrows. Forms use radius 14 so multi-line fields don't turn
// into lozenges.
export const Field = ({ label, hint, max, value = '', multiline, as, className, children, ...rest }) => {
  const id = useId();
  const Control = as ?? (multiline ? 'textarea' : 'input');
  return (
    <div className={cx('flex flex-col gap-1.5', className)}>
      {label && <label htmlFor={id}><Eyebrow>{label}</Eyebrow></label>}
      <Control
        id={id}
        value={value}
        maxLength={max}
        className={cx(fieldBase, multiline ? 'py-3 min-h-28 leading-normal resize-y' : 'h-11', as === 'select' && 'appearance-none')}
        {...rest}
      >
        {children}
      </Control>
      {(hint || max) && (
        <span className="text-[11px] leading-[1.4] font-semibold text-ink-holder">
          {hint ?? `${String(value).length} of ${max}`}
        </span>
      )}
    </div>
  );
};

// Search placeholder says what you can search.
export const SearchInput = ({ className, onChange, ...rest }) => (
  <label className={cx('flex items-center gap-2.5 h-11 px-4 rounded-full bg-surface-2 box-border', className)}>
    <SearchIcon width={16} height={16} className="text-ink-holder shrink-0" />
    <input
      type="search"
      className="flex-1 min-w-0 border-none bg-transparent outline-none text-[13px] font-medium text-ink-title"
      onChange={(e) => onChange(e.target.value)}
      {...rest}
    />
  </label>
);

// ── Chip, tag, toggle ─────────────────────────────────────────

// Filter chips are single-select, gold when on.
export const Chip = ({ selected, children, ...rest }) => (
  <button
    type="button"
    aria-pressed={selected}
    className={cx(
      'h-[34px] px-3.5 inline-flex items-center rounded-full text-[13px] font-bold transition-colors',
      selected ? 'bg-gold text-on-gold' : 'bg-surface-2 text-ink-body hover:bg-surface-3'
    )}
    {...rest}
  >
    {children}
  </button>
);

// Status is a surface-3 tag. Published gets a gold dot: the ruleset has no green.
export const Tag = ({ dot, muted, children }) => (
  <span className={cx('inline-flex items-center gap-[5px] px-2 py-[3px] rounded-full bg-surface-3 text-[10px] font-bold whitespace-nowrap', muted ? 'text-ink-holder' : 'text-ink-body')}>
    {dot && <span className="w-1.5 h-1.5 rounded-full bg-gold" />}
    {children}
  </span>
);

export const Toggle = ({ on, label, onChange, disabled }) => (
  <button
    type="button"
    role="switch"
    aria-checked={on}
    aria-label={label}
    disabled={disabled}
    onClick={() => onChange(!on)}
    className={cx('relative inline-flex items-center w-11 h-[26px] shrink-0 rounded-full transition-colors disabled:opacity-50', on ? 'bg-gold' : 'bg-surface-3')}
  >
    <span className={cx('w-5 h-5 rounded-full transition-transform', on ? 'translate-x-[21px] bg-on-gold' : 'translate-x-[3px] bg-ink-disabled')} />
  </button>
);

// ── Feedback ──────────────────────────────────────────────────

// Replaces alert(). One sentence, two text actions at most.
export const Banner = ({ message, onRetry, onDismiss, retryLabel = 'Try again' }) => {
  if (!message) return null;
  return (
    <div role="alert" className="bg-surface-2 rounded-[18px] px-4 py-3.5 flex flex-col gap-2.5">
      <span className="text-[13px] leading-normal text-ink-body break-words">{message}</span>
      {(onRetry || onDismiss) && (
        <div className="flex gap-4">
          {onRetry && <TextAction onClick={onRetry}>{retryLabel}</TextAction>}
          {onDismiss && <TextAction tone="muted" onClick={onDismiss}>Dismiss</TextAction>}
        </div>
      )}
    </div>
  );
};

export const Loading = ({ label = 'Loading…' }) => (
  <p className="m-0 text-[13px] text-ink-holder" role="status">{label}</p>
);

// An empty section shows the first step, not a blank box.
export const EmptyState = ({ title, body, action }) => (
  <div className="rounded-card shadow-[inset_0_0_0_1.5px_var(--rp-dashed)] px-6 py-10 flex flex-col items-center gap-2 text-center">
    <span className="text-sm font-extrabold text-ink-title">{title}</span>
    {body && <span className="text-[13px] text-ink-holder max-w-sm">{body}</span>}
    {action && <div className="mt-2">{action}</div>}
  </div>
);

// Dashed "create" slot — the only place dashed lines appear.
export const CreateSlot = ({ label, className, ...rest }) => (
  <button
    type="button"
    className={cx('rounded-card px-[18px] py-4 flex items-center gap-3 bg-transparent shadow-[inset_0_0_0_1.5px_var(--rp-dashed)] hover:bg-surface-1 text-left', className)}
    {...rest}
  >
    <span className="w-[38px] h-[38px] shrink-0 rounded-full bg-surface-2 flex items-center justify-center text-link">
      <PlusIcon width={18} height={18} />
    </span>
    <span className="text-sm font-extrabold text-ink-body">{label}</span>
  </button>
);

// ── Modal (12b) ───────────────────────────────────────────────

// × · centered 16pt title · text action (muted until valid, then gold).
// Centered at 480 wide on a 60% backdrop.
export const Modal = ({ title, onClose, action, width = 480, children }) => {
  useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-4"
      onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="w-full bg-page rounded-panel overflow-hidden flex flex-col max-h-[calc(100vh-32px)]"
        style={{ maxWidth: width }}
      >
        <div className="grid grid-cols-[64px_1fr_64px] items-center h-[52px] px-2 shrink-0">
          <IconButton label="Close" onClick={onClose}><CloseIcon width={14} height={14} strokeWidth={2.5} /></IconButton>
          <span className="text-center text-base font-extrabold text-ink-title truncate">{title}</span>
          {action ? (
            <TextAction
              tone={action.disabled ? 'muted' : 'gold'}
              disabled={action.disabled}
              onClick={action.onClick}
              className="text-sm text-right pr-1.5"
            >
              {action.label}
            </TextAction>
          ) : <span />}
        </div>
        <div className="px-4 pt-1.5 pb-[18px] overflow-y-auto flex flex-col gap-4">{children}</div>
      </div>
    </div>
  );
};

// ── Covers and rows ───────────────────────────────────────────

const COVER_TINTS = [
  ['#3a4a7a', '#1d2748'], ['#7a5a3a', '#3b2a1d'], ['#3a7a6a', '#1d3b35'], ['#6a3a7a', '#2f1d3b'],
  ['#7a3a44', '#3b1d22'], ['#4a6a3a', '#243b1d'], ['#3a5a7a', '#1d2c3b'],
];

const tintFor = (seed = '') => {
  let h = 0;
  for (const ch of String(seed)) h = (h * 31 + ch.charCodeAt(0)) | 0;
  const [a, b] = COVER_TINTS[Math.abs(h) % COVER_TINTS.length];
  return `linear-gradient(160deg, ${a}, ${b})`;
};

// A book cover, or a tinted block when there's no image.
export const Cover = ({ src, title, className, style }) => (
  src ? (
    <img src={src} alt="" loading="lazy" className={cx('object-cover shrink-0 bg-surface-3', className)} style={style} />
  ) : (
    <div aria-hidden className={cx('shrink-0', className)} style={{ background: tintFor(title), ...style }} />
  )
);

// Three fanned covers, as on the curated tile.
export const CoverStack = ({ books = [] }) => {
  const [a, b, c] = [books[2], books[1], books[0]];
  const card = 'absolute w-10 h-[60px] rounded-cover-s';
  return (
    <div className="relative w-16 h-[66px] shrink-0" aria-hidden>
      <Cover src={a?.cover_image_url} title={a?.title ?? 'a'} className={card} style={{ left: 4, top: 4, transform: 'rotate(-8deg)' }} />
      <Cover src={b?.cover_image_url} title={b?.title ?? 'b'} className={card} style={{ left: 14, top: 2, transform: 'rotate(2deg)' }} />
      <Cover src={c?.cover_image_url} title={c?.title ?? 'c'} className={cx(card, 'shadow-[0_6px_14px_rgba(0,0,0,.4)]')} style={{ left: 24, top: 4, transform: 'rotate(10deg)' }} />
    </div>
  );
};

// 12d BookRow, admin flavour: 46×66 cover, title, meta and tags.
export const BookRow = ({ cover, title, meta, tags, trailing }) => (
  <div className="flex items-center gap-3.5 min-w-0">
    <Cover src={cover} title={title} className="w-[46px] h-[66px] rounded-cover-s" />
    <div className="flex-1 min-w-0 flex flex-col gap-[5px]">
      <span className="text-sm leading-[1.3] font-extrabold text-ink-title truncate">{title}</span>
      {meta && <span className="text-xs leading-[1.3] font-semibold text-ink-meta truncate">{meta}</span>}
      {tags && <div className="flex flex-wrap gap-1.5">{tags}</div>}
    </div>
    {trailing}
  </div>
);

// Surface-1 list with hairlines inset to the text.
export const List = ({ children, className }) => (
  <div className={cx('bg-surface-1 rounded-card px-5 py-1', className)}>{children}</div>
);

export const ListItem = ({ children, className, ...rest }) => (
  <div className={cx('py-3 hairline-b last:shadow-none', className)} {...rest}>{children}</div>
);

export const Card = ({ className, children, ...rest }) => (
  <div className={cx('bg-surface-1 rounded-card p-5', className)} {...rest}>{children}</div>
);
