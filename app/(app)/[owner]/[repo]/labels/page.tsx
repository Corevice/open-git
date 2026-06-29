"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

type Label = {
  id: string;
  name: string;
  color: string;
  description: string;
};

type Props = {
  params: Promise<{ owner: string; repo: string }>;
};

function labelStyle(color: string): React.CSSProperties {
  const hex = color.startsWith("#") ? color : `#${color}`;
  return { backgroundColor: hex, color: "#fff" };
}

export default function LabelsPage({ params }: Props) {
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [labels, setLabels] = useState<Label[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", color: "0366d6", description: "" });
  const [editingLabel, setEditingLabel] = useState<Label | null>(null);
  const [editForm, setEditForm] = useState({ name: "", color: "", description: "" });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    params.then(({ owner: o, repo: r }) => {
      setOwner(o);
      setRepo(r);
    });
  }, [params]);

  const fetchLabels = useCallback(async () => {
    if (!owner || !repo) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/labels?per_page=100`);
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to load labels");
      }
      const data = (await res.json()) as Array<{
        id: number | string;
        name: string;
        color: string;
        description?: string | null;
      }>;
      setLabels(
        data.map((label) => ({
          id: String(label.id),
          name: label.name,
          color: label.color,
          description: label.description ?? "",
        })),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load labels");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    fetchLabels();
  }, [fetchLabels]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createForm.name.trim()) {
      setError("Name is required");
      return;
    }
    setCreating(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/labels`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: createForm.name.trim(),
          color: createForm.color,
          description: createForm.description,
        }),
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to create label");
      }
      setCreateForm({ name: "", color: "0366d6", description: "" });
      await fetchLabels();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create label");
    } finally {
      setCreating(false);
    }
  };

  const startEdit = (label: Label) => {
    setEditingLabel(label);
    setEditForm({
      name: label.name,
      color: label.color.replace(/^#/, ""),
      description: label.description,
    });
  };

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingLabel) return;
    setSaving(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/labels/${editingLabel.name}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          new_name: editForm.name.trim() !== editingLabel.name ? editForm.name.trim() : undefined,
          color: editForm.color,
          description: editForm.description,
        }),
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to update label");
      }
      setEditingLabel(null);
      await fetchLabels();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update label");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (label: Label) => {
    if (!window.confirm(`Delete label '${label.name}'?`)) return;
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/labels/${label.name}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to delete label");
      }
      await fetchLabels();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete label");
    }
  };

  if (!owner || !repo) return null;

  if (loading) {
    return (
      <div className="min-h-screen bg-[#f6f8fa] flex items-center justify-center">
        <span
          className="inline-block h-8 w-8 animate-spin rounded-full border-2 border-[#0969da] border-t-transparent"
          aria-label="Loading"
        />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa] text-[#1f2328]">
      <div className="max-w-[960px] mx-auto px-6 py-6">
        <div className="mb-4">
          <Link href={`/${owner}/${repo}`} className="text-sm text-[#0969da] hover:underline">
            ← Back to repository
          </Link>
        </div>

        <h1 className="text-2xl font-semibold mb-6">
          Labels in{" "}
          <span className="text-[#0969da]">
            {owner}/{repo}
          </span>
        </h1>

        {error && <p className="text-[#d1242f] mb-4">{error}</p>}

        <form
          onSubmit={handleCreate}
          className="bg-white border border-[#d0d7de] rounded-md p-6 mb-6"
        >
          <h2 className="text-lg font-semibold mb-4">Create label</h2>
          <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
            <div>
              <label htmlFor="create-name" className="block text-sm font-semibold mb-1">
                Name <span className="text-[#d1242f]">*</span>
              </label>
              <input
                id="create-name"
                type="text"
                required
                value={createForm.name}
                onChange={(e) => setCreateForm((f) => ({ ...f, name: e.target.value }))}
                className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                placeholder="Label name"
              />
            </div>
            <div>
              <label htmlFor="create-color" className="block text-sm font-semibold mb-1">
                Color
              </label>
              <input
                id="create-color"
                type="color"
                value={"#" + createForm.color}
                onChange={(e) =>
                  setCreateForm((f) => ({ ...f, color: e.target.value.replace("#", "") }))
                }
                className="h-[38px] w-full cursor-pointer border border-[#d0d7de] rounded-md"
              />
            </div>
          </div>
          <div className="mt-4">
            <label htmlFor="create-description" className="block text-sm font-semibold mb-1">
              Description
            </label>
            <textarea
              id="create-description"
              value={createForm.description}
              onChange={(e) => setCreateForm((f) => ({ ...f, description: e.target.value }))}
              className="w-full min-h-[80px] px-3 py-2 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
              placeholder="Optional description"
            />
          </div>
          <div className="mt-4 flex justify-end">
            <button
              type="submit"
              disabled={creating || !createForm.name.trim()}
              className="px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {creating ? "Creating…" : "Create label"}
            </button>
          </div>
        </form>

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
          <div className="p-4 border-b border-[#d0d7de] font-semibold">
            {labels.length} label{labels.length === 1 ? "" : "s"}
          </div>
          {labels.length === 0 ? (
            <p className="p-6 text-sm text-[#656d76]">No labels yet.</p>
          ) : (
            <ul className="divide-y divide-[#d0d7de]">
              {labels.map((label) => (
                <li key={label.id} className="p-4">
                  {editingLabel?.id === label.id ? (
                    <form onSubmit={handleUpdate} className="space-y-4">
                      <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
                        <div>
                          <label className="block text-sm font-semibold mb-1">Name</label>
                          <input
                            type="text"
                            required
                            value={editForm.name}
                            onChange={(e) => setEditForm((f) => ({ ...f, name: e.target.value }))}
                            className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                          />
                        </div>
                        <div>
                          <label className="block text-sm font-semibold mb-1">Color</label>
                          <input
                            type="color"
                            value={"#" + editForm.color}
                            onChange={(e) =>
                              setEditForm((f) => ({ ...f, color: e.target.value.replace("#", "") }))
                            }
                            className="h-[38px] w-full cursor-pointer border border-[#d0d7de] rounded-md"
                          />
                        </div>
                      </div>
                      <div>
                        <label className="block text-sm font-semibold mb-1">Description</label>
                        <textarea
                          value={editForm.description}
                          onChange={(e) =>
                            setEditForm((f) => ({ ...f, description: e.target.value }))
                          }
                          className="w-full min-h-[80px] px-3 py-2 border border-[#d0d7de] rounded-md text-sm resize-y focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                        />
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="submit"
                          disabled={saving}
                          className="px-4 py-1.5 text-sm bg-[#0969da] text-white rounded-md font-semibold disabled:opacity-50"
                        >
                          {saving ? "Saving…" : "Save"}
                        </button>
                        <button
                          type="button"
                          onClick={() => setEditingLabel(null)}
                          className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                        >
                          Cancel
                        </button>
                      </div>
                    </form>
                  ) : (
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <span
                          className="inline-block px-2 py-0.5 rounded-full text-[11px] font-semibold mb-2"
                          style={labelStyle(label.color)}
                        >
                          {label.name}
                        </span>
                        {label.description && (
                          <p className="text-sm text-[#656d76]">{label.description}</p>
                        )}
                      </div>
                      <div className="flex gap-2 shrink-0">
                        <button
                          type="button"
                          onClick={() => startEdit(label)}
                          className="px-3 py-1 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(label)}
                          className="px-3 py-1 text-sm border border-[#d1242f] text-[#d1242f] rounded-md bg-white hover:bg-[#fff1f0]"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
