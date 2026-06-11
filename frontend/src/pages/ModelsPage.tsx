import { useState, useEffect } from "react";
import { providerConfigsApi } from "../api/client";

const providerLabels: Record<string, string> = {
  "openai-codex": "OpenAI Codex",
  anthropic: "Anthropic (Claude)",
  openai: "OpenAI",
};

const modelHints: Record<string, string[]> = {
  "openai-codex": ["gpt-5.1-codex-max", "gpt-5.1-codex-mini", "gpt-5.3-codex-spark", "gpt-5.5"],
  anthropic: ["claude-sonnet-4-6", "claude-opus-4-7", "claude-haiku-4-5"],
  openai: ["gpt-5", "gpt-5-pro", "gpt-5-mini", "o4-mini"],
};

export default function ModelsPage() {
  const [items, setItems] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: "", provider: "openai-codex", model_id: "gpt-5.1-codex-max", api_key: "", base_url: "" });
  const [submitting, setSubmitting] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [editForm, setEditForm] = useState({ name: "", provider: "openai-codex", model_id: "", api_key: "", base_url: "" });

  const load = async () => {
    try {
      setItems(await providerConfigsApi.list());
      setError("");
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      await providerConfigsApi.create(form);
      setShowForm(false);
      setForm({ name: "", provider: "openai-codex", model_id: "gpt-5.1-codex-max", api_key: "", base_url: "" });
      load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete "${name}"?`)) return;
    await providerConfigsApi.delete(id);
    load();
  };

  const handleEdit = (pc: any) => {
    setEditing(pc.id);
    setEditForm({ name: pc.name, provider: pc.provider, model_id: pc.model_id, api_key: "", base_url: pc.base_url || "" });
  };

  const handleSaveEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const data = { ...editForm };
      if (!data.api_key) delete (data as any).api_key; // don't overwrite with empty
      await providerConfigsApi.update(editing!, data);
      setEditing(null);
      load();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) return <div className="content"><p>Loading...</p></div>;

  const renderProviderForm = (f: any, setF: any) => {
    const hints = modelHints[f.provider] || [];
    return (
      <>
        <div className="form-group">
          <label>Name</label>
          <input value={f.name} onChange={(e: any) => setF({ ...f, name: e.target.value })} required placeholder="My Codex Config" />
        </div>
        <div className="form-group">
          <label>Provider</label>
          <select value={f.provider} onChange={(e: any) => setF({ ...f, provider: e.target.value, model_id: (modelHints[e.target.value] || [""])[0] })}>
            {Object.entries(providerLabels).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </select>
        </div>
        <div className="form-group">
          <label>Model ID</label>
          <input value={f.model_id} onChange={(e: any) => setF({ ...f, model_id: e.target.value })} required list="model-suggestions" />
          <datalist id="model-suggestions">
            {hints.map((m: string) => (<option key={m} value={m} />))}
          </datalist>
        </div>
        <div className="form-group">
          <label>API Key</label>
          <input type="password" value={f.api_key} onChange={(e: any) => setF({ ...f, api_key: e.target.value })} placeholder={editing ? "leave empty to keep current" : "sk-..."} />
        </div>
        <div className="form-group">
          <label>Base URL (optional)</label>
          <input value={f.base_url} onChange={(e: any) => setF({ ...f, base_url: e.target.value })} placeholder="https://api.openai.com/v1" />
        </div>
      </>
    );
  };

  return (
    <div>
      <div className="page-header">
        <h2>Models</h2>
        <button onClick={() => setShowForm(!showForm)} className="btn btn-primary">
          {showForm ? "Cancel" : "+ New Config"}
        </button>
      </div>
      {error && <p style={{ color: "var(--danger)", marginBottom: 12 }}>{error}</p>}

      {showForm && (
        <form className="form card" onSubmit={handleCreate} style={{ marginBottom: 24 }}>
          <h3 style={{ marginBottom: 16 }}>New Model Config</h3>
          {renderProviderForm(form, setForm)}
          <button type="submit" className="btn btn-primary" disabled={submitting}>
            {submitting ? "Creating..." : "Create Config"}
          </button>
        </form>
      )}

      {editing && (
        <form className="form card" onSubmit={handleSaveEdit} style={{ marginBottom: 24 }}>
          <h3 style={{ marginBottom: 16 }}>Edit Model Config</h3>
          {renderProviderForm(editForm, setEditForm)}
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={submitting}>
              {submitting ? "Saving..." : "Save"}
            </button>
            <button type="button" onClick={() => setEditing(null)} className="btn btn-ghost">Cancel</button>
          </div>
        </form>
      )}

      {items.length === 0 ? (
        <div className="empty-state"><p>No model configs yet. Create a config for openai-codex (Codex agent) or anthropic (Claude agent).</p></div>
      ) : (
        items.map((pc: any) => (
          <div key={pc.id} className="card card-row">
            <div>
              <h3>{pc.name} <span className="tag tag-draft">{providerLabels[pc.provider] || pc.provider}</span></h3>
              <p style={{ fontSize: 13, margin: 0 }}>Model: <code>{pc.model_id}</code></p>
              {pc.base_url && <p style={{ fontSize: 11, margin: 0, color: "var(--text-muted)" }}>Base URL: {pc.base_url}</p>}
              <p style={{ fontSize: 11, marginTop: 4, color: "var(--text-muted)" }}>
                Created: {new Date(pc.created_at).toLocaleString()}
              </p>
            </div>
            <div className="card-actions">
              <button onClick={() => handleEdit(pc)} className="btn btn-ghost">Edit</button>
              <button onClick={() => handleDelete(pc.id, pc.name)} className="btn btn-danger">Delete</button>
            </div>
          </div>
        ))
      )}
    </div>
  );
}
