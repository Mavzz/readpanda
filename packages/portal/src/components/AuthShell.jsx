import { Banner } from './ui';

// Layout shared by sign in and sign up.
const AuthShell = ({ subtitle, error, onSubmit, children, footer }) => (
  <div className="min-h-screen w-full bg-page flex items-center justify-center px-4 py-12">
    <div className="w-full max-w-[400px] flex flex-col gap-6">
      <div className="flex flex-col items-center gap-3 text-center">
        <img src="/assets/readpandaLogo.png" alt="" className="w-12 h-12 rounded-[14px] object-contain bg-ink-title p-1 box-border" />
        <h1 className="m-0 text-[26px] leading-[1.2] font-extrabold tracking-[-0.5px] text-ink-title">ReadPanda admin</h1>
        <p className="m-0 text-[13px] text-ink-holder">{subtitle}</p>
      </div>
      <form
        noValidate
        onSubmit={(e) => { e.preventDefault(); onSubmit(); }}
        className="bg-surface-1 rounded-card p-6 flex flex-col gap-4"
      >
        <Banner message={error} />
        {children}
      </form>
      {footer && <p className="m-0 text-center text-[13px] text-ink-body">{footer}</p>}
    </div>
  </div>
);

export default AuthShell;
