# ReadPanda public site

Static pages anyone can open without signing in. The admin portal
(`packages/portal`) puts every route behind a login, so these can't live there.

- `/` — a short landing page
- `/support` — contact email and common questions (App Store Connect **Support URL**)
- `/privacy` — privacy policy (App Store Connect **Privacy Policy URL**, and linked from the app)
- `/terms` — terms of use, including the community rules App Review Guideline 1.2 asks for (linked from the app)

Plain HTML and one stylesheet; no build step. `vercel.json` turns on clean
URLs, so `privacy.html` is served at `/privacy`.

## Deploy

A separate Vercel project from the portal:

1. Vercel → Add New → Project → import this repo.
2. **Root Directory:** `site`. **Framework Preset:** Other. Leave the build and
   output settings empty.
3. Deploy. Pushes that touch `site/` redeploy it.

## Preview locally

```bash
npx serve site
```

## Keep in step with the app

- The privacy policy describes what the app collects. When the app adds a
  service or starts collecting something new, update `privacy.html` and its
  "Last updated" date, and the App Privacy answers in App Store Connect.
- The support address is also `SUPPORT_EMAIL` in the app
  (readpanda-mobile, specs/help-support).
