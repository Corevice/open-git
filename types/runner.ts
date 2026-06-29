export interface Runner {
  id: string;
  name: string;
  status: "online" | "offline" | "busy";
  labels: string[];
  runner_type: "act" | "official";
  last_seen_at: string | null;
}

export interface RegistrationTokenResponse {
  token: string;
  expires_at: string;
}

export interface RunnerListResponse {
  runners: Runner[];
}

export interface RegisterRunnerRequest {
  registration_token: string;
  name: string;
  labels: string[];
  os: string;
  arch: string;
  runner_type: "act" | "official";
}

export interface WorkflowJobResponse {
  id: string;
  assigned_runner_id: string | null;
  runner_type: "act" | "official" | null;
}
