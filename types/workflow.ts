export interface WorkflowRun {
  id: number;
  name: string;
  run_number: number;
  status: string;
  conclusion: string | null;
  head_branch: string;
  head_sha: string;
  event?: string;
  run_started_at?: string;
  started_at?: string;
  completed_at?: string;
  created_at?: string;
  updated_at?: string;
  actor?: { login: string };
  triggering_actor?: { login: string };
}

export interface WorkflowJob {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  steps?: WorkflowStep[];
  started_at?: string;
  completed_at?: string;
}

export interface WorkflowStep {
  number: number;
  name: string;
  status: string;
  conclusion: string | null;
  started_at?: string;
  completed_at?: string;
}

export interface Artifact {
  id: number;
  name: string;
  size_in_bytes: number;
  expired?: boolean;
  created_at?: string;
  expires_at?: string;
}

export interface WorkflowRunsResponse {
  workflow_runs: WorkflowRun[];
  total_count?: number;
}
