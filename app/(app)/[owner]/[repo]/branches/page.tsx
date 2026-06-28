import Link from "next/link";
import { cookies } from "next/headers";
import { notFound, redirect } from "next/navigation";
import CreateBranchForm, {
  BranchDeleteButton,
  type BranchItem,
} from "@/components/repo/CreateBranchForm";
import { apiClient, isApiError } from "@/lib/api-client";

interface RepoMetadata {
  name: string;
  full_name: string;
  description: string | null;
  default_branch: string;
  owner: { login: string };
}

export default async function BranchesPage({
  params,
}: {
  params: Promise<{ owner: string; repo: string }>;
}) {
  const { owner, repo } = await params;

  const cookieStore = await cookies();
  const token = cookieStore.get("authToken")?.value;
  if (!token) {
    redirect("/login");
  }
  const authOpts = { token };

  let metadata: RepoMetadata;
  try {
    metadata = await apiClient.getRepo<RepoMetadata>(owner, repo, authOpts);
  } catch (err) {
    if (isApiError(err) && err.status === 401) redirect("/login");
    if (isApiError(err) && err.status === 404) notFound();
    throw err;
  }

  let branches: BranchItem[];
  try {
    branches = await apiClient.getBranches<BranchItem[]>(owner, repo, authOpts);
  } catch (err) {
    if (isApiError(err) && err.status === 401) redirect("/login");
    if (isApiError(err) && err.status === 404) notFound();
    throw err;
  }

  return (
    <div className="min-h-screen bg-[#f6f8fa]">
      <header className="h-16 bg-white/85 backdrop-blur border-b border-[color:var(--border)] sticky top-0 z-[100]">
        <div className="max-w-[1280px] mx-auto px-6 flex items-center justify-between h-full">
          <Link href="/dashboard" className="text-lg font-extrabold flex items-center gap-2">
            <span>🐙</span> OpenHub
          </Link>
          <Link
            href="/dashboard"
            className="px-2 py-1 rounded-full text-xs font-medium bg-[color:var(--primary-light)] text-[color:var(--primary)]"
          >
            {owner}
          </Link>
        </div>
      </header>

      <div className="max-w-[1280px] mx-auto px-6 py-6">
        <nav className="text-sm mb-4">
          <Link href={`/${owner}`} className="text-[#0969da] no-underline hover:underline">
            {owner}
          </Link>
          <span className="text-[#57606a]"> / </span>
          <Link
            href={`/${owner}/${repo}`}
            className="text-[#0969da] no-underline hover:underline"
          >
            {repo}
          </Link>
          <span className="text-[#57606a]"> / </span>
          <span className="text-[#24292f] font-semibold">Branches</span>
        </nav>

        <h1 className="text-2xl font-semibold mb-6">Branches</h1>

        <CreateBranchForm owner={owner} repo={repo} branches={branches} />

        <div className="bg-white border border-[#d0d7de] rounded-lg overflow-hidden">
          {branches.length === 0 ? (
            <p className="p-6 text-sm text-[#57606a] m-0">No branches yet</p>
          ) : (
            <ul className="list-none m-0 p-0 divide-y divide-[#d0d7de]">
              {branches.map((branch) => {
                const isDefault = branch.name === metadata.default_branch;
                return (
                  <li
                    key={branch.name}
                    className="flex items-center gap-3 px-4 py-3 flex-wrap"
                  >
                    <span className="font-semibold text-[#0969da]">{branch.name}</span>
                    <code className="text-xs text-[#57606a] bg-[#f6f8fa] px-2 py-0.5 rounded">
                      {branch.commit.sha.slice(0, 7)}
                    </code>
                    {isDefault && (
                      <span className="px-2 py-0.5 text-xs font-medium bg-[#ddf4ff] text-[#0969da] rounded-full">
                        Default
                      </span>
                    )}
                    <div className="ml-auto">
                      <BranchDeleteButton
                        owner={owner}
                        repo={repo}
                        branch={branch.name}
                        disabled={isDefault}
                      />
                    </div>
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
