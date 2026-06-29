"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

type Milestone = {
  id: string;
  number: number;
  title: string;
  description: string;
  state: "open" | "closed";
  due_on: string | null;
  open_issues: number;
  closed_issues: number;
};

type Props = {
  params: Promise<{ owner: string; repo: string }>;
};

function formatDueDate(due_on: string | null): string {
  if (!due_on) return "No due date";
  const d = new Date(due_on);
  return `Due ${d.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" })}`;
}

function dateInputValue(due_on: string | null): string {
  if (!due_on) return "";
  return due_on.slice(0, 10);
}

function toApiDueOn(dateStr: string): string | undefined {
  if (!dateStr) return undefined;
  return `${dateStr}T00:00:00Z`;
}

export default function MilestonesPage({ params }: Props) {
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");
  const [milestones, setMilestones] = useState<Milestone[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stateFilter, setStateFilter] = useState<"open" | "closed" | "all">("open");
  const [creating, setCreating] = useState(false);
  const [createForm, setCreateForm] = useState({ title: "", description: "", due_on: "" });
  const [editingMilestone, setEditingMilestone] = useState<Milestone | null>(null);
  const [editForm, setEditForm] = useState({ title: "", description: "", due_on: "" });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    params.then(({ owner: o, repo: r }) => {
      setOwner(o);
      setRepo(r);
    });
  }, [params]);

  const fetchMilestones = useCallback(async () => {
    if (!owner || !repo) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/milestones?state=all&per_page=100`);
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to load milestones");
      }
      const data = (await res.json()) as Array<{
        id: number | string;
        number: number;
        title: string;
        description?: string | null;
        state: "open" | "closed";
        due_on?: string | null;
        open_issues: number;
        closed_issues: number;
      }>;
      setMilestones(
        data.map((m) => ({
          id: String(m.id),
          number: m.number,
          title: m.title,
          description: m.description ?? "",
          state: m.state,
          due_on: m.due_on ?? null,
          open_issues: m.open_issues,
          closed_issues: m.closed_issues,
        })),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load milestones");
    } finally {
      setLoading(false);
    }
  }, [owner, repo]);

  useEffect(() => {
    fetchMilestones();
  }, [fetchMilestones]);

  const filteredMilestones = milestones.filter((m) => {
    if (stateFilter === "all") return true;
    return m.state === stateFilter;
  });

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!createForm.title.trim()) {
      setError("Title is required");
      return;
    }
    setCreating(true);
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/milestones`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: createForm.title.trim(),
          description: createForm.description,
          due_on: toApiDueOn(createForm.due_on),
        }),
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to create milestone");
      }
      setCreateForm({ title: "", description: "", due_on: "" });
      await fetchMilestones();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create milestone");
    } finally {
      setCreating(false);
    }
  };

  const startEdit = (milestone: Milestone) => {
    setEditingMilestone(milestone);
    setEditForm({
      title: milestone.title,
      description: milestone.description,
      due_on: dateInputValue(milestone.due_on),
    });
  };

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingMilestone) return;
    setSaving(true);
    setError(null);
    try {
      const res = await fetch(
        `/repos/${owner}/${repo}/milestones/${editingMilestone.number}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            title: editForm.title.trim(),
            description: editForm.description,
            due_on: toApiDueOn(editForm.due_on),
            state: editingMilestone.state,
          }),
        },
      );
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to update milestone");
      }
      setEditingMilestone(null);
      await fetchMilestones();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update milestone");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleState = async (m: Milestone) => {
    const nextState = m.state === "open" ? "closed" : "open";
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/milestones/${m.number}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state: nextState }),
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to update milestone state");
      }
      await fetchMilestones();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to update milestone state");
    }
  };

  const handleDelete = async (m: Milestone) => {
    if (!window.confirm(`Delete milestone '${m.title}'?`)) return;
    setError(null);
    try {
      const res = await fetch(`/repos/${owner}/${repo}/milestones/${m.number}`, {
        method: "DELETE",
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => null)) as { message?: string } | null;
        throw new Error(data?.message ?? "Failed to delete milestone");
      }
      await fetchMilestones();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete milestone");
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
          Milestones in{" "}
          <span className="text-[#0969da]">
            {owner}/{repo}
          </span>
        </h1>

        {error && <p className="text-[#d1242f] mb-4">{error}</p>}

        <form
          onSubmit={handleCreate}
          className="bg-white border border-[#d0d7de] rounded-md p-6 mb-6"
        >
          <h2 className="text-lg font-semibold mb-4">Create milestone</h2>
          <div>
            <label htmlFor="create-title" className="block text-sm font-semibold mb-1">
              Title <span className="text-[#d1242f]">*</span>
            </label>
            <input
              id="create-title"
              type="text"
              required
              value={createForm.title}
              onChange={(e) => setCreateForm((f) => ({ ...f, title: e.target.value }))}
              className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
              placeholder="Milestone title"
            />
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
          <div className="mt-4">
            <label htmlFor="create-due-on" className="block text-sm font-semibold mb-1">
              Due date
            </label>
            <input
              id="create-due-on"
              type="date"
              value={createForm.due_on}
              onChange={(e) => setCreateForm((f) => ({ ...f, due_on: e.target.value }))}
              className="px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
            />
          </div>
          <div className="mt-4 flex justify-end">
            <button
              type="submit"
              disabled={creating || !createForm.title.trim()}
              className="px-4 py-1.5 text-sm bg-[#1f883d] text-white rounded-md font-semibold border border-black/10 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {creating ? "Creating…" : "Create milestone"}
            </button>
          </div>
        </form>

        <div className="flex gap-2 mb-4">
          {(["open", "closed", "all"] as const).map((filter) => (
            <button
              key={filter}
              type="button"
              onClick={() => setStateFilter(filter)}
              className={`px-4 py-1.5 text-sm rounded-md font-medium border ${
                stateFilter === filter
                  ? "bg-white border-[#d0d7de] font-semibold shadow-sm"
                  : "border-transparent text-[#656d76] hover:text-[#1f2328]"
              }`}
            >
              {filter === "open" ? "Open" : filter === "closed" ? "Closed" : "All"}
            </button>
          ))}
        </div>

        <div className="bg-white border border-[#d0d7de] rounded-md overflow-hidden">
          <div className="p-4 border-b border-[#d0d7de] font-semibold">
            {filteredMilestones.length} milestone{filteredMilestones.length === 1 ? "" : "s"}
          </div>
          {filteredMilestones.length === 0 ? (
            <p className="p-6 text-sm text-[#656d76]">No milestones yet.</p>
          ) : (
            <ul className="divide-y divide-[#d0d7de]">
              {filteredMilestones.map((m) => {
                const total = m.open_issues + m.closed_issues;
                const progressWidth = total === 0 ? "0%" : `${(m.closed_issues / total) * 100}%`;

                return (
                  <li key={m.id} className="p-4">
                    {editingMilestone?.id === m.id ? (
                      <form onSubmit={handleUpdate} className="space-y-4">
                        <div>
                          <label className="block text-sm font-semibold mb-1">Title</label>
                          <input
                            type="text"
                            required
                            value={editForm.title}
                            onChange={(e) => setEditForm((f) => ({ ...f, title: e.target.value }))}
                            className="w-full px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
                          />
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
                        <div>
                          <label className="block text-sm font-semibold mb-1">Due date</label>
                          <input
                            type="date"
                            value={editForm.due_on}
                            onChange={(e) =>
                              setEditForm((f) => ({ ...f, due_on: e.target.value }))
                            }
                            className="px-3 py-2 border border-[#d0d7de] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-[#0969da]"
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
                            onClick={() => setEditingMilestone(null)}
                            className="px-4 py-1.5 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                          >
                            Cancel
                          </button>
                        </div>
                      </form>
                    ) : (
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <h3 className="text-base font-semibold m-0">{m.title}</h3>
                            <span
                              className={`px-2 py-0.5 rounded-full text-[11px] font-semibold ${
                                m.state === "open"
                                  ? "bg-[#ddf4ff] text-[#0969da]"
                                  : "bg-[#eaeef2] text-[#656d76]"
                              }`}
                            >
                              {m.state === "open" ? "Open" : "Closed"}
                            </span>
                          </div>
                          {m.description && (
                            <p className="text-sm text-[#656d76] mb-2">{m.description}</p>
                          )}
                          <p className="text-sm text-[#656d76] mb-2">{formatDueDate(m.due_on)}</p>
                          <div className="flex items-center gap-2 mb-1">
                            <div className="flex-1 h-1.5 bg-[#eaeef2] rounded-full overflow-hidden">
                              <div
                                style={{ width: progressWidth }}
                                className="bg-[#1a7f37] h-1.5 rounded-full"
                              />
                            </div>
                            <span className="text-xs text-[#656d76] shrink-0">
                              {m.closed_issues} / {total} closed
                            </span>
                          </div>
                        </div>
                        <div className="flex gap-2 shrink-0">
                          <button
                            type="button"
                            onClick={() => handleToggleState(m)}
                            className="px-3 py-1 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                          >
                            {m.state === "open" ? "Close" : "Reopen"}
                          </button>
                          <button
                            type="button"
                            onClick={() => startEdit(m)}
                            className="px-3 py-1 text-sm border border-[#d0d7de] rounded-md bg-white hover:bg-[#f6f8fa]"
                          >
                            Edit
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDelete(m)}
                            className="px-3 py-1 text-sm border border-[#d1242f] text-[#d1242f] rounded-md bg-white hover:bg-[#fff1f0]"
                          >
                            Delete
                          </button>
                        </div>
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
