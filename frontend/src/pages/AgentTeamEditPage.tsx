import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { agentTeamsApi, templatesApi, promptsApi, skillsApi, toolsApi, providerConfigsApi } from '../api/client'

export default function AgentTeamEditPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id

  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const [templates, setTemplates] = useState<any[]>([])
  const [prompts, setPrompts] = useState<any[]>([])
  const [skills, setSkills] = useState<any[]>([])
  const [tools, setTools] = useState<any[]>([])
  const [providerConfigs, setProviderConfigs] = useState<any[]>([])

  const [name, setName] = useState('')
  const [templateId, setTemplateId] = useState('')
  const [repoUrl, setRepoUrl] = useState('')
  const [gitUsername, setGitUsername] = useState('')
  const [gitPassword, setGitPassword] = useState('')
  const [branch, setBranch] = useState('main')
  const [teamPromptIds, setTeamPromptIds] = useState<string[]>([])
  const [teamSkillIds, setTeamSkillIds] = useState<string[]>([])
  const [members, setMembers] = useState<any[]>([])

  useEffect(() => {
    Promise.all([
      templatesApi.list(),
      promptsApi.list(),
      skillsApi.list(),
      toolsApi.list(),
      providerConfigsApi.list(),
    ]).then(([t, p, s, tl, pc]) => {
      setTemplates(t); setPrompts(p); setSkills(s); setTools(tl); setProviderConfigs(pc)
    }).catch(() => {})

    if (isEdit) {
      agentTeamsApi.get(id!).then((team: any) => {
        setName(team.name)
        setTemplateId(team.template_id)
        setRepoUrl(team.repo_url)
        setGitUsername(team.git_username || '')
        setGitPassword(team.git_password || '')
        setBranch(team.branch || 'main')
        setTeamPromptIds(team.prompt_ids || [])
        setTeamSkillIds(team.skill_ids || [])
        setMembers((team.members || []).map((m: any) => ({
          ...m,
          prompt_ids: m.prompt_ids || [],
          skill_ids: m.skill_ids || [],
          tool_ids: m.tool_ids || [],
        })))
        setLoading(false)
      }).catch((e: any) => { setError(e.message); setLoading(false) })
    } else {
      setLoading(false)
    }
  }, [id])

  const addMember = () => {
    setMembers([...members, {
      name: '', role: 'worker', agent_template_id: '', provider_config_id: '',
      system_prompt_override: '', sequence: members.length,
      prompt_ids: [], skill_ids: [], tool_ids: [],
    }])
  }

  const removeMember = (idx: number) => {
    setMembers(members.filter((_, i) => i !== idx))
  }

  const updateMember = (idx: number, field: string, value: any) => {
    const updated = [...members]
    updated[idx] = { ...updated[idx], [field]: value }
    setMembers(updated)
  }

  const toggleMemberRef = (idx: number, field: string, refId: string) => {
    const updated = [...members]
    const arr = updated[idx][field] as string[]
    if (arr.includes(refId)) {
      updated[idx] = { ...updated[idx], [field]: arr.filter((id: string) => id !== refId) }
    } else {
      updated[idx] = { ...updated[idx], [field]: [...arr, refId] }
    }
    setMembers(updated)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault(); setSaving(true); setError('')
    try {
      const data = {
        name, template_id: templateId,
        repo_url: repoUrl, git_username: gitUsername, git_password: gitPassword, branch,
        prompt_ids: teamPromptIds, skill_ids: teamSkillIds,
        members: members.map((m, i) => ({ ...m, sequence: i })),
      }
      if (isEdit) {
        await agentTeamsApi.update(id!, data)
      } else {
        await agentTeamsApi.create(data)
      }
      navigate('/agent-teams')
    } catch (e: any) { setError(e.message) }
    finally { setSaving(false) }
  }

  if (loading) return <div className="content"><p>Loading...</p></div>

  return (
    <div>
      <div className="page-header">
        <h2>{isEdit ? 'Edit Agent Team' : 'New Agent Team'}</h2>
        <button onClick={() => navigate('/agent-teams')} className="btn btn-ghost">Back</button>
      </div>
      {error && <p style={{ color: 'var(--danger)', marginBottom: 12 }}>{error}</p>}

      <form className="form" onSubmit={handleSave}>
        <div className="card" style={{ marginBottom: 16 }}>
          <h3 style={{ marginBottom: 16 }}>Team Info</h3>
          <div className="form-group">
            <label>Team Name</label>
            <input value={name} onChange={e => setName(e.target.value)} required />
          </div>
          <div className="form-group">
            <label>Dockerfile Template</label>
            <select value={templateId} onChange={e => setTemplateId(e.target.value)} required>
              <option value="">Select template...</option>
              {templates.map((t: any) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </div>
          <div className="form-group">
            <label>Repository URL</label>
            <input value={repoUrl} onChange={e => setRepoUrl(e.target.value)} placeholder="https://github.com/user/repo.git" />
          </div>
          <div className="form-group">
            <label>Git Username (optional)</label>
            <input value={gitUsername} onChange={e => setGitUsername(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Git Password / Token (optional)</label>
            <input type="password" value={gitPassword} onChange={e => setGitPassword(e.target.value)} />
          </div>
          <div className="form-group">
            <label>Branch</label>
            <input value={branch} onChange={e => setBranch(e.target.value)} />
          </div>
        </div>

        <div className="card" style={{ marginBottom: 16 }}>
          <h3 style={{ marginBottom: 16 }}>Team-level Prompts & Skills</h3>
          <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 12 }}>
            These are shared across all members. Member-specific settings can override.
          </p>

          <div className="form-group">
            <label>Prompts</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {prompts.map((p: any) => (
                <label key={p.id} style={{
                  display: 'flex', alignItems: 'center', gap: 4, padding: '4px 10px',
                  border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer', fontSize: 13,
                  background: teamPromptIds.includes(p.id) ? 'var(--primary-light)' : 'transparent',
                }}>
                  <input type="checkbox" checked={teamPromptIds.includes(p.id)}
                    onChange={() => setTeamPromptIds(teamPromptIds.includes(p.id) ? teamPromptIds.filter(pid => pid !== p.id) : [...teamPromptIds, p.id])} />
                  {p.name}
                </label>
              ))}
            </div>
          </div>
          <div className="form-group">
            <label>Skills</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
              {skills.map((s: any) => (
                <label key={s.id} style={{
                  display: 'flex', alignItems: 'center', gap: 4, padding: '4px 10px',
                  border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer', fontSize: 13,
                  background: teamSkillIds.includes(s.id) ? 'var(--primary-light)' : 'transparent',
                }}>
                  <input type="checkbox" checked={teamSkillIds.includes(s.id)}
                    onChange={() => setTeamSkillIds(teamSkillIds.includes(s.id) ? teamSkillIds.filter(sid => sid !== s.id) : [...teamSkillIds, s.id])} />
                  {s.name}
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="card" style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
            <h3>Members</h3>
            <button type="button" onClick={addMember} className="btn btn-primary">+ Add Member</button>
          </div>

          {members.length === 0 && <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>No members yet. Add at least one member (leader or worker).</p>}

          {members.map((m, idx) => (
            <div key={idx} style={{
              border: '1px solid var(--border)', borderRadius: 6, padding: 16, marginBottom: 12,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                <h4 style={{ margin: 0 }}>Member #{idx + 1}</h4>
                <button type="button" onClick={() => removeMember(idx)} style={{ background: 'none', border: 'none', color: 'var(--danger)', cursor: 'pointer', fontSize: 20 }}>&times;</button>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div className="form-group">
                  <label>Name</label>
                  <input value={m.name} onChange={e => updateMember(idx, 'name', e.target.value)} required placeholder="e.g. code-reviewer" />
                </div>
                <div className="form-group">
                  <label>Role</label>
                  <select value={m.role} onChange={e => updateMember(idx, 'role', e.target.value)}>
                    <option value="worker">Worker</option>
                    <option value="leader">Leader</option>
                  </select>
                </div>
                <div className="form-group">
                  <label>Template</label>
                  <select value={m.agent_template_id} onChange={e => updateMember(idx, 'agent_template_id', e.target.value)}>
                    <option value="">Select template...</option>
                    {templates.map((t: any) => <option key={t.id} value={t.id}>{t.name}</option>)}
                  </select>
                </div>
                <div className="form-group">
                  <label>Provider Config</label>
                  <select value={m.provider_config_id} onChange={e => updateMember(idx, 'provider_config_id', e.target.value)}>
                    <option value="">Select config...</option>
                    {providerConfigs.map((pc: any) => <option key={pc.id} value={pc.id}>{pc.name} ({pc.provider})</option>)}
                  </select>
                </div>
              </div>

              <div className="form-group" style={{ marginTop: 8 }}>
                <label>System Prompt Override (optional)</label>
                <textarea value={m.system_prompt_override || ''} onChange={e => updateMember(idx, 'system_prompt_override', e.target.value)}
                  rows={3} style={{ width: '100%' }} placeholder="Custom system prompt for this member..." />
              </div>

              <div style={{ marginTop: 12 }}>
                <details>
                  <summary style={{ cursor: 'pointer', fontSize: 13, color: 'var(--text-muted)' }}>Additional Prompts/Skills/Tools</summary>
                  <div style={{ marginTop: 8 }}>
                    <div className="form-group">
                      <label>Prompts</label>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        {prompts.map((p: any) => (
                          <label key={p.id} style={{
                            display: 'flex', alignItems: 'center', gap: 4, padding: '2px 8px',
                            border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer', fontSize: 12,
                            background: (m.prompt_ids || []).includes(p.id) ? 'var(--primary-light)' : 'transparent',
                          }}>
                            <input type="checkbox" checked={(m.prompt_ids || []).includes(p.id)}
                              onChange={() => toggleMemberRef(idx, 'prompt_ids', p.id)} />
                            {p.name}
                          </label>
                        ))}
                      </div>
                    </div>
                    <div className="form-group">
                      <label>Skills</label>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        {skills.map((s: any) => (
                          <label key={s.id} style={{
                            display: 'flex', alignItems: 'center', gap: 4, padding: '2px 8px',
                            border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer', fontSize: 12,
                            background: (m.skill_ids || []).includes(s.id) ? 'var(--primary-light)' : 'transparent',
                          }}>
                            <input type="checkbox" checked={(m.skill_ids || []).includes(s.id)}
                              onChange={() => toggleMemberRef(idx, 'skill_ids', s.id)} />
                            {s.name}
                          </label>
                        ))}
                      </div>
                    </div>
                    <div className="form-group">
                      <label>Tools</label>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        {tools.map((t: any) => (
                          <label key={t.id} style={{
                            display: 'flex', alignItems: 'center', gap: 4, padding: '2px 8px',
                            border: '1px solid var(--border)', borderRadius: 4, cursor: 'pointer', fontSize: 12,
                            background: (m.tool_ids || []).includes(t.id) ? 'var(--primary-light)' : 'transparent',
                          }}>
                            <input type="checkbox" checked={(m.tool_ids || []).includes(t.id)}
                              onChange={() => toggleMemberRef(idx, 'tool_ids', t.id)} />
                            {t.name}
                          </label>
                        ))}
                      </div>
                    </div>
                  </div>
                </details>
              </div>
            </div>
          ))}
        </div>

        <div className="form-actions">
          <button type="submit" className="btn btn-primary" disabled={saving}>
            {saving ? 'Saving...' : (isEdit ? 'Update Team' : 'Create Team')}
          </button>
          <button type="button" onClick={() => navigate('/agent-teams')} className="btn btn-ghost">Cancel</button>
        </div>
      </form>
    </div>
  )
}
