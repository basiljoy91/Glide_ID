'use client'

import { useEffect, useMemo, useState } from 'react'
import { useRouter } from 'next/navigation'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/store/useStore'

type SecuritySettings = {
  enforce_mfa: boolean
  require_mfa_for_roles: string[]
  password_policy: {
    min_length: number
    require_uppercase: boolean
    require_lowercase: boolean
    require_number: boolean
    require_symbol: boolean
    expire_days: number
  }
  trusted_ip_ranges: string[]
  session_timeout_minutes: number
  sso_enabled: boolean
}

type SSOConfiguration = {
  enabled: boolean
  provider: string
  config: Record<string, any>
}

const emptySecurity: SecuritySettings = {
  enforce_mfa: false,
  require_mfa_for_roles: ['org_admin', 'hr'],
  password_policy: {
    min_length: 12,
    require_uppercase: true,
    require_lowercase: true,
    require_number: true,
    require_symbol: true,
    expire_days: 90,
  },
  trusted_ip_ranges: [],
  session_timeout_minutes: 480,
  sso_enabled: false,
}

const emptySSO: SSOConfiguration = {
  enabled: false,
  provider: '',
  config: {},
}

const roleOptions = ['org_admin', 'hr', 'dept_manager']

export default function SecurityPage() {
  const { isAuthenticated, user, token } = useAuthStore()
  const router = useRouter()
  const base = useMemo(() => process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080', [])
  const canManage = user?.role === 'org_admin' || user?.permissions?.includes('security.manage')

  const [security, setSecurity] = useState<SecuritySettings>(emptySecurity)
  const [sso, setSSO] = useState<SSOConfiguration>(emptySSO)
  const [trustedRange, setTrustedRange] = useState('')
  const [configKey, setConfigKey] = useState('')
  const [configValue, setConfigValue] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSavingSecurity, setIsSavingSecurity] = useState(false)
  const [isSavingSSO, setIsSavingSSO] = useState(false)

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push('/admin/login')
      return
    }
    if (!canManage) {
      router.push('/admin/org')
      return
    }
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.id, canManage])

  const authHeaders = () => ({
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  })

  const load = async () => {
    try {
      setIsLoading(true)
      const [securityResp, ssoResp] = await Promise.all([
        fetch(`${base}/api/v1/org/security`, { headers: authHeaders() }),
        fetch(`${base}/api/v1/org/security/sso`, { headers: authHeaders() }),
      ])
      if (!securityResp.ok || !ssoResp.ok) {
        throw new Error('Failed to load security settings')
      }
      const securityData = await securityResp.json()
      const ssoData = await ssoResp.json()
      setSecurity({ ...emptySecurity, ...securityData, password_policy: { ...emptySecurity.password_policy, ...securityData.password_policy } })
      setSSO({ ...emptySSO, ...ssoData, config: ssoData.config || {} })
    } catch (error: any) {
      toast.error(error.message || 'Failed to load security settings')
    } finally {
      setIsLoading(false)
    }
  }

  const saveSecurity = async () => {
    try {
      setIsSavingSecurity(true)
      const resp = await fetch(`${base}/api/v1/org/security`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(security),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to update security settings')
      }
      const data = await resp.json()
      setSecurity({ ...emptySecurity, ...data, password_policy: { ...emptySecurity.password_policy, ...data.password_policy } })
      toast.success('Security settings updated')
    } catch (error: any) {
      toast.error(error.message || 'Failed to update security settings')
    } finally {
      setIsSavingSecurity(false)
    }
  }

  const saveSSO = async () => {
    try {
      setIsSavingSSO(true)
      const resp = await fetch(`${base}/api/v1/org/security/sso`, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(sso),
      })
      if (!resp.ok) {
        const error = await resp.json().catch(() => ({}))
        throw new Error(error.error || 'Failed to update SSO configuration')
      }
      const data = await resp.json()
      setSSO({ ...emptySSO, ...data, config: data.config || {} })
      toast.success('SSO configuration updated')
    } catch (error: any) {
      toast.error(error.message || 'Failed to update SSO configuration')
    } finally {
      setIsSavingSSO(false)
    }
  }

  const addTrustedRange = () => {
    const value = trustedRange.trim()
    if (!value) {
      toast.error('Enter an IP or CIDR range')
      return
    }
    setSecurity((current) => ({
      ...current,
      trusted_ip_ranges: current.trusted_ip_ranges.includes(value)
        ? current.trusted_ip_ranges
        : [...current.trusted_ip_ranges, value],
    }))
    setTrustedRange('')
  }

  const toggleMFARole = (role: string) => {
    setSecurity((current) => ({
      ...current,
      require_mfa_for_roles: current.require_mfa_for_roles.includes(role)
        ? current.require_mfa_for_roles.filter((item) => item !== role)
        : [...current.require_mfa_for_roles, role],
    }))
  }

  const addConfigEntry = () => {
    const key = configKey.trim()
    if (!key) {
      toast.error('Config key is required')
      return
    }
    setSSO((current) => ({ ...current, config: { ...current.config, [key]: configValue } }))
    setConfigKey('')
    setConfigValue('')
  }

  if (!isAuthenticated || !user || !canManage) return null

  return (
    <div className="container mx-auto space-y-6 p-6">
      <div>
        <h1 className="text-3xl font-bold">Security & Access</h1>
        <p className="text-muted-foreground">Configure MFA enforcement, password policy, trusted networks, session lifetime, and staged SSO settings.</p>
      </div>

      {isLoading ? (
        <div className="rounded-lg border p-6 text-sm text-muted-foreground">Loading security settings...</div>
      ) : (
        <>
          <section className="grid gap-6 rounded-lg border bg-card p-6 lg:grid-cols-2">
            <div className="space-y-4">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-semibold">Security Controls</h2>
                  <p className="text-sm text-muted-foreground">These controls are enforced by the backend for password login and active JWT sessions.</p>
                </div>
                <Button onClick={saveSecurity} disabled={isSavingSecurity}>{isSavingSecurity ? 'Saving...' : 'Save Security'}</Button>
              </div>

              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={security.enforce_mfa} onChange={(e) => setSecurity((current) => ({ ...current, enforce_mfa: e.target.checked }))} /> Enforce MFA for selected roles</label>
              <div className="flex flex-wrap gap-2">
                {roleOptions.map((role) => (
                  <button
                    key={role}
                    type="button"
                    onClick={() => toggleMFARole(role)}
                    className={`rounded-full border px-3 py-1 text-sm ${security.require_mfa_for_roles.includes(role) ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground'}`}
                  >
                    {role}
                  </button>
                ))}
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <Input type="number" placeholder="Minimum password length" value={security.password_policy.min_length} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, min_length: Number(e.target.value) || 0 } }))} />
                <Input type="number" placeholder="Password expiry days" value={security.password_policy.expire_days} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, expire_days: Number(e.target.value) || 0 } }))} />
              </div>
              <div className="grid gap-2 text-sm md:grid-cols-2">
                <label className="flex items-center gap-2"><input type="checkbox" checked={security.password_policy.require_uppercase} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, require_uppercase: e.target.checked } }))} /> Require uppercase</label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={security.password_policy.require_lowercase} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, require_lowercase: e.target.checked } }))} /> Require lowercase</label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={security.password_policy.require_number} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, require_number: e.target.checked } }))} /> Require number</label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={security.password_policy.require_symbol} onChange={(e) => setSecurity((current) => ({ ...current, password_policy: { ...current.password_policy, require_symbol: e.target.checked } }))} /> Require symbol</label>
              </div>
              <Input type="number" placeholder="Session timeout minutes" value={security.session_timeout_minutes} onChange={(e) => setSecurity((current) => ({ ...current, session_timeout_minutes: Number(e.target.value) || 0 }))} />

              <div className="space-y-2 rounded-lg border p-4">
                <h3 className="font-medium">Trusted Networks</h3>
                <div className="grid gap-2 md:grid-cols-[1fr_auto]">
                  <Input placeholder="203.0.113.0/24 or 198.51.100.10" value={trustedRange} onChange={(e) => setTrustedRange(e.target.value)} />
                  <Button variant="outline" onClick={addTrustedRange}>Add</Button>
                </div>
                <div className="space-y-2 text-sm">
                  {security.trusted_ip_ranges.length === 0 ? (
                    <div className="rounded-md border border-dashed p-3 text-muted-foreground">No IP allowlist configured. All networks are currently allowed.</div>
                  ) : (
                    security.trusted_ip_ranges.map((entry) => (
                      <div key={entry} className="flex items-center justify-between rounded-md border px-3 py-2">
                        <span>{entry}</span>
                        <Button variant="ghost" onClick={() => setSecurity((current) => ({ ...current, trusted_ip_ranges: current.trusted_ip_ranges.filter((item) => item !== entry) }))}>Remove</Button>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            <div className="space-y-4 rounded-lg border bg-background p-4">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-semibold">SSO Configuration</h2>
                  <p className="text-sm text-muted-foreground">Provider configuration is saved now. End-user SSO sign-in remains intentionally disabled until the backend flow is completed.</p>
                </div>
                <Button onClick={saveSSO} disabled={isSavingSSO}>{isSavingSSO ? 'Saving...' : 'Save SSO'}</Button>
              </div>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={sso.enabled} onChange={(e) => setSSO((current) => ({ ...current, enabled: e.target.checked }))} /> Mark SSO as configured</label>
              <select className="h-10 rounded-md border border-input bg-background px-3 text-sm" value={sso.provider} onChange={(e) => setSSO((current) => ({ ...current, provider: e.target.value }))}>
                <option value="">Select provider</option>
                <option value="oidc">OIDC</option>
                <option value="saml">SAML</option>
                <option value="google">Google Workspace</option>
                <option value="azure">Azure AD</option>
                <option value="okta">Okta</option>
              </select>
              <div className="grid gap-2 md:grid-cols-[1fr_1fr_auto]">
                <Input placeholder="Config key" value={configKey} onChange={(e) => setConfigKey(e.target.value)} />
                <Input placeholder="Config value" value={configValue} onChange={(e) => setConfigValue(e.target.value)} />
                <Button variant="outline" onClick={addConfigEntry}>Add</Button>
              </div>
              <div className="space-y-2 text-sm">
                {Object.keys(sso.config).length === 0 ? (
                  <div className="rounded-md border border-dashed p-3 text-muted-foreground">No provider configuration entries saved yet.</div>
                ) : (
                  Object.entries(sso.config).map(([key, value]) => (
                    <div key={key} className="flex items-center justify-between rounded-md border px-3 py-2">
                      <div>
                        <div className="font-medium">{key}</div>
                        <div className="text-xs text-muted-foreground break-all">{String(value)}</div>
                      </div>
                      <Button variant="ghost" onClick={() => setSSO((current) => {
                        const next = { ...current.config }
                        delete next[key]
                        return { ...current, config: next }
                      })}>Remove</Button>
                    </div>
                  ))
                )}
              </div>
            </div>
          </section>
        </>
      )}
    </div>
  )
}
