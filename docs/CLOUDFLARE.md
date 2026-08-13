# Cloudflare Pages — landing (docs/)

## Goal

GitHub `main` + `docs/**` → Cloudflare Pages project **`speedmap`** → `https://speedmap.buhaiov.com`

No GitHub Pages. DNS already on Cloudflare.

## 1. Create API token (Cloudflare Dashboard)

1. https://dash.cloudflare.com/profile/api-tokens → **Create Token**
2. Template **Edit Cloudflare Workers** (or Custom):
   - **Account** → **Cloudflare Pages** → **Edit**
   - **Account** → **Account Settings** → **Read** (optional but useful)
3. Account Resources: include your account
4. Create → **copy token once**

## 2. Account ID

Dashboard → any zone / Workers → right sidebar **Account ID**.

## 3. Put secrets in GitHub (not in git)

```bash
cd ~/development/SpeedMap

# paste token when prompted (hidden)
gh secret set CLOUDFLARE_API_TOKEN --repo dimonchoo/SpeedMap

gh secret set CLOUDFLARE_ACCOUNT_ID --repo dimonchoo/SpeedMap
# paste Account ID
```

Verify:

```bash
gh secret list --repo dimonchoo/SpeedMap
# must show CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID
```

## 4. Create Pages project (once)

```bash
npx wrangler@latest pages project create speedmap --production-branch=main
# login / use same token if asked
```

Or UI: **Workers & Pages** → Create → Pages → Direct Upload / empty project named `speedmap`.

## 5. Custom domain

CF Pages project **speedmap** → Custom domains → add `speedmap.buhaiov.com`  
(CF will set DNS. Remove broken tunnel/route that returns plain `404 page not found`.)

## 6. Deploy

```bash
gh workflow run "Deploy landing to Cloudflare Pages" --repo dimonchoo/SpeedMap
# or push any change under docs/
```

## Wrong patterns

| Do | Don't |
|----|--------|
| Secrets in GitHub Actions | Token in repo / workflow YAML |
| CF Pages as origin | Point hostname at empty tunnel |
| One project `speedmap` | Fight GitHub Pages lock |
